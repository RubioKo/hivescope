// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"math"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

const (
	grantDomain     = "HiveScope/recovery-grant/v1\x00"
	grantRecordSize = 198
)

var (
	ErrGrantConflict    = errors.New("stale or conflicting grant generation")
	ErrGrantDenied      = errors.New("provider denied the grant update")
	ErrGrantUnavailable = errors.New("durable grant state unavailable")
	errInvalidGrant     = errors.New("invalid grant record")
)

type signedGrant struct {
	record     [grantRecordSize]byte
	generation uint64
	key        [32]byte // zero for a revocation
}

func (a CatalogAgreement) verifyGrant(raw []byte) (signedGrant, error) {
	if len(raw) != grantRecordSize {
		return signedGrant{}, errInvalidGrant
	}
	record := [grantRecordSize]byte(raw)
	if string(record[:28]) != grantDomain {
		return signedGrant{}, errInvalidGrant
	}
	if [32]byte(record[28:60]) != a.ProviderKey || [16]byte(record[60:76]) != a.FolderID ||
		[16]byte(record[76:92]) != a.AgreementID || record[101] != 1 ||
		!ed25519.Verify(a.AuthorityKey[:], record[:134], record[134:]) {
		return signedGrant{}, ErrGrantDenied
	}
	grant := signedGrant{record, binary.BigEndian.Uint64(record[92:100]), [32]byte(record[102:134])}
	if grant.generation == 0 {
		return signedGrant{}, errInvalidGrant
	}
	switch record[100] {
	case 1:
		if grant.generation == math.MaxUint64 || !storageformat.ValidEd25519Key(grant.key[:]) {
			return signedGrant{}, errInvalidGrant
		}
	case 2:
		if grant.key != [32]byte{} {
			return signedGrant{}, errInvalidGrant
		}
	default:
		return signedGrant{}, errInvalidGrant
	}
	return grant, nil
}

// SignRevocation creates the v1 record with the enrolled folder authority.
// The owner retains these exact signed bytes before submitting or retrying.
func (a CatalogAgreement) SignRevocation(generation uint64, authoritySeed [ed25519.SeedSize]byte) ([]byte, error) {
	key := ed25519.NewKeyFromSeed(authoritySeed[:])
	defer clear(key)
	if generation == 0 || a.AuthorityKey != [32]byte(key[ed25519.SeedSize:]) {
		return nil, errors.New("revocation requires the enrolled authority and a positive generation")
	}
	var record [grantRecordSize]byte
	copy(record[:28], grantDomain)
	copy(record[28:60], a.ProviderKey[:])
	copy(record[60:76], a.FolderID[:])
	copy(record[76:92], a.AgreementID[:])
	binary.BigEndian.PutUint64(record[92:100], generation)
	record[100], record[101] = 2, 1 // revoked, agreement read scope
	copy(record[134:], ed25519.Sign(key, record[:134]))
	return record[:], nil
}
