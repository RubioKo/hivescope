// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

// CatalogAgreement is the provider's independently provisioned enrollment.
type CatalogAgreement struct {
	ProviderKey, AuthorityKey [32]byte
	FolderID, AgreementID     [16]byte
	Start, Expiry             time.Time
}

func (a CatalogAgreement) readable(now time.Time) bool {
	return !now.Before(a.Start) && now.Before(a.Expiry.Add(7*24*time.Hour))
}

// CommittedCatalog refers to a catalog in the provider's local committed index.
// Digest is the expected SHA-256 from that index, independent of Sealed.
type CommittedCatalog struct {
	Locator, Digest [32]byte
	Sealed          []byte
}

// CatalogProvider serves the prototype's single committed catalog. Grant state
// is process-local. A restarted provider needs an independently trusted source
// for its latest signed record before it may serve reads.
type CatalogProvider struct {
	agreement CatalogAgreement
	locator   [32]byte
	sealed    []byte

	mu         sync.Mutex // serializes grant changes and queuing authorized data
	record     [198]byte
	generation uint64
	recovery   [32]byte // zero for missing or revoked grant
}

func NewCatalogProvider(agreement CatalogAgreement, catalog CommittedCatalog) (*CatalogProvider, error) {
	if !storageformat.ValidEd25519Key(agreement.ProviderKey[:]) || !storageformat.ValidEd25519Key(agreement.AuthorityKey[:]) {
		return nil, errors.New("invalid provider or folder authority public key")
	}
	if agreement.Start.IsZero() || !agreement.Expiry.After(agreement.Start) {
		return nil, errors.New("invalid enrolled agreement term")
	}
	p := &CatalogProvider{agreement: agreement, locator: catalog.Locator}
	if len(catalog.Sealed) >= catalogMinSize && len(catalog.Sealed) <= catalogMaxSize && sha256.Sum256(catalog.Sealed) == catalog.Digest {
		p.sealed = bytes.Clone(catalog.Sealed)
	}
	return p, nil
}

// LoadGrant verifies the current locally supplied record. The caller must
// supply the highest authoritative record when initializing the provider.
// This method does not implement durable GRANT_UPDATE or issue an ACK.
func (p *CatalogProvider) LoadGrant(record []byte) error {
	const domain = "HiveScope/recovery-grant/v1\x00"
	a := p.agreement
	if len(record) != 198 || string(record[:28]) != domain || [32]byte(record[28:60]) != a.ProviderKey ||
		[16]byte(record[60:76]) != a.FolderID || [16]byte(record[76:92]) != a.AgreementID || record[101] != 1 ||
		!ed25519.Verify(a.AuthorityKey[:], record[:134], record[134:]) {
		return errors.New("grant signature or enrollment mismatch")
	}
	generation := binary.BigEndian.Uint64(record[92:100])
	key := [32]byte(record[102:134])
	switch record[100] {
	case 1:
		if generation == math.MaxUint64 || !storageformat.ValidEd25519Key(key[:]) {
			return errors.New("invalid active grant")
		}
	case 2:
		if key != [32]byte{} {
			return errors.New("revoked grant includes a recovery key")
		}
	default:
		return errors.New("unknown recovery grant state")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !a.readable(time.Now()) {
		return errors.New("agreement outside read window")
	}
	if generation == 0 || generation < p.generation || (generation == p.generation && !bytes.Equal(record, p.record[:])) {
		return errors.New("stale or conflicting recovery grant")
	}
	p.record = [198]byte(record)
	p.generation = generation
	p.recovery = key
	return nil
}
