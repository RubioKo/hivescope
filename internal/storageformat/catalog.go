// SPDX-License-Identifier: AGPL-3.0-only

// Package storageformat reads HiveScope's authenticated v1 storage formats.
package storageformat

import (
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"
)

// CatalogContext holds the trusted identifiers and keys from the unlocked
// recovery package.
type CatalogContext struct {
	FolderID         [16]byte
	AgreementID      [16]byte
	CatalogLocator   [32]byte
	FolderReadKey    [32]byte
	CatalogVerifyKey [ed25519.PublicKeySize]byte
}

// Snapshot describes one file version authenticated by OpenCatalog.
type Snapshot struct {
	catalogSignature [ed25519.SignatureSize]byte
	folderID         [16]byte
	agreementID      [16]byte
	catalogLocator   [32]byte
	catalogVerifyKey [ed25519.PublicKeySize]byte
	versionID        [16]byte
	versionKey       [32]byte
	objectLocator    [32]byte
	objectDigest     [sha256.Size]byte
	size             uint64
	name             string
}

// Name returns display text. Escape control characters when presenting it
// and choose the local destination independently.
func (s *Snapshot) Name() string { return s.name }

// Size is the authenticated plaintext length in bytes.
func (s *Snapshot) Size() uint64 { return s.size }

// ObjectLocator identifies the sealed object named in the catalog.
func (s *Snapshot) ObjectLocator() [32]byte { return s.objectLocator }

// FromCatalog reports whether this snapshot was verified for this signed scope.
// An uninitialized snapshot cannot identify an object for a network request.
func (s *Snapshot) FromCatalog(ctx CatalogContext) bool {
	return s != nil && s.name != "" && s.folderID == ctx.FolderID && s.agreementID == ctx.AgreementID &&
		s.catalogLocator == ctx.CatalogLocator && s.catalogVerifyKey == ctx.CatalogVerifyKey
}

// OpenCatalog verifies a complete HSC1 catalog before exposing its entry.
// The end of src must coincide with the end of the signature.
func OpenCatalog(src io.Reader, ctx CatalogContext) (*Snapshot, error) {
	const tagSize = 16
	var header [40]byte
	if _, err := io.ReadFull(src, header[:]); err != nil {
		return nil, fmt.Errorf("read catalog header: %w", err)
	}
	if string(header[:4]) != "HSC1" {
		return nil, errors.New("unsupported catalog format")
	}
	entryLen := binary.BigEndian.Uint32(header[36:])
	if entryLen < 123 || entryLen > 377 {
		return nil, errors.New("catalog entry length out of range")
	}
	sealedLen := int(entryLen) + tagSize
	bodyLen := sealedLen + ed25519.SignatureSize
	body, err := io.ReadAll(io.LimitReader(src, int64(bodyLen+1)))
	if err != nil {
		return nil, fmt.Errorf("read catalog body: %w", err)
	}
	if len(body) != bodyLen {
		return nil, errors.New("catalog length does not match header")
	}
	sealed, signature := body[:sealedLen], body[sealedLen:]
	context := slices.Concat(ctx.FolderID[:], ctx.AgreementID[:], ctx.CatalogLocator[:])
	signed := slices.Concat([]byte("HiveScope/catalog-sign/v1\x00"), context, header[:], sealed)
	if !ed25519.Verify(ctx.CatalogVerifyKey[:], signed, signature) {
		return nil, errors.New("invalid catalog signature")
	}
	key, err := hkdf.Key(sha256.New, ctx.FolderReadKey[:], header[4:36], "HiveScope/catalog-key/v1", 32)
	if err != nil {
		return nil, fmt.Errorf("derive catalog key: %w", err)
	}
	gcm, err := newGCM([32]byte(key))
	clear(key)
	if err != nil {
		return nil, err
	}
	aad := slices.Concat([]byte("HiveScope/catalog/v1\x00"), context, header[:])
	entry, err := gcm.Open(nil, make([]byte, gcm.NonceSize()), sealed, aad)
	if err != nil {
		return nil, fmt.Errorf("authenticate catalog entry: %w", err)
	}
	defer clear(entry)

	// The authenticated entry has 90 fixed prefix bytes, the display name
	// and a 32-byte digest. Its enclosing length has already been bounded.
	size := binary.BigEndian.Uint64(entry[80:88])
	if size > 64<<20 {
		return nil, errors.New("snapshot exceeds the v1 file size limit")
	}
	nameLen := int(binary.BigEndian.Uint16(entry[88:90]))
	name := entry[90 : len(entry)-sha256.Size]
	if nameLen < 1 || nameLen > 255 || len(name) != nameLen || !utf8.Valid(name) {
		return nil, errors.New("invalid catalog display name")
	}
	return &Snapshot{
		catalogSignature: [ed25519.SignatureSize]byte(signature),
		folderID:         ctx.FolderID,
		agreementID:      ctx.AgreementID,
		catalogLocator:   ctx.CatalogLocator,
		catalogVerifyKey: ctx.CatalogVerifyKey,
		versionID:        [16]byte(entry[:16]),
		objectLocator:    [32]byte(entry[16:48]),
		versionKey:       [32]byte(entry[48:80]),
		size:             size,
		name:             string(name),
		objectDigest:     [sha256.Size]byte(entry[90+nameLen:]),
	}, nil
}
