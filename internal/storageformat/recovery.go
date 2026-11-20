// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"unicode/utf8"

	"filippo.io/edwards25519"
	"golang.org/x/crypto/argon2"
)

const recoveryPackageSize = 255
const recoveryHeaderSize = 33

// ErrRecoveryAuthentication covers a wrong passphrase and an unauthenticated
// package body. Neither case yields recovery credentials.
var ErrRecoveryAuthentication = errors.New("recovery package authentication failed")

// RecoveryPackage holds an unlocked HSRP v1 export on the owner's device.
// The endpoint is a hint. A provider connection must authenticate ProviderKey
// before the device requests a read challenge.
type RecoveryPackage struct {
	Catalog      CatalogContext
	Generation   uint64
	ProviderKey  [ed25519.PublicKeySize]byte
	Endpoint     netip.AddrPort
	RecoverySeed [ed25519.SeedSize]byte
}

// WithRecoveryPackage reads exactly one local export and calls use only after
// authenticating and validating it. Put all provider and destination I/O in
// that owner-side continuation. It must not log the package, send its secrets
// to a provider or retain the package after returning.
func WithRecoveryPackage(src io.Reader, passphrase []byte, use func(*RecoveryPackage) error) error {
	if src == nil || use == nil {
		return errors.New("recovery requires a package reader and a local continuation")
	}
	raw, err := io.ReadAll(io.LimitReader(src, recoveryPackageSize+1))
	if err != nil {
		return fmt.Errorf("read recovery package: %w", err)
	}
	if len(raw) != recoveryPackageSize {
		return errors.New("invalid recovery package length")
	}
	if string(raw[:4]) != "HSRP" || raw[4] != 1 {
		return errors.New("unsupported recovery package format")
	}
	if !utf8.Valid(passphrase) {
		return errors.New("recovery passphrase must be valid UTF-8")
	}

	// HSRP v1 fixes Argon2id's work factors. The header supplies only salt
	// and nonce. Authenticate all 33 header bytes as associated data.
	key := argon2.IDKey(passphrase, raw[5:21], 3, 64*1024, 4, 32)
	gcm, err := newGCM([32]byte(key))
	clear(key)
	if err != nil {
		return err
	}
	payload, err := gcm.Open(nil, raw[21:recoveryHeaderSize], raw[recoveryHeaderSize:], raw[:recoveryHeaderSize])
	if err != nil {
		return ErrRecoveryAuthentication
	}
	pkg, err := parseRecoveryPayload(payload)
	clear(payload)
	if err != nil {
		return err
	}
	defer func() { *pkg = RecoveryPackage{} }()
	return use(pkg)
}

// The fixed envelope and full GCM tag guarantee a 206-byte payload here.
func parseRecoveryPayload(payload []byte) (*RecoveryPackage, error) {
	generation := binary.BigEndian.Uint64(payload[32:40])
	port := binary.BigEndian.Uint16(payload[76:78])
	if generation == 0 || port == 0 {
		return nil, errors.New("invalid recovery generation or endpoint port")
	}
	if !validRecoveryPin(payload[40:72]) || !validRecoveryPin(payload[142:174]) {
		return nil, errors.New("invalid Ed25519 public key in recovery package")
	}
	return &RecoveryPackage{
		Catalog: CatalogContext{
			FolderID:         [16]byte(payload[0:16]),
			AgreementID:      [16]byte(payload[16:32]),
			CatalogLocator:   [32]byte(payload[174:206]),
			FolderReadKey:    [32]byte(payload[110:142]),
			CatalogVerifyKey: [ed25519.PublicKeySize]byte(payload[142:174]),
		},
		Generation:   generation,
		ProviderKey:  [ed25519.PublicKeySize]byte(payload[40:72]),
		Endpoint:     netip.AddrPortFrom(netip.AddrFrom4([4]byte(payload[72:76])), port),
		RecoverySeed: [ed25519.SeedSize]byte(payload[78:110]),
	}, nil
}

func validRecoveryPin(raw []byte) bool {
	point, err := new(edwards25519.Point).SetBytes(raw)
	// SetBytes accepts some noncanonical encodings. A pinned identity has
	// one encoding and cannot be a point of order dividing eight.
	if err != nil || !bytes.Equal(point.Bytes(), raw) {
		return false
	}
	return point.MultByCofactor(point).Equal(edwards25519.NewIdentityPoint()) == 0
}
