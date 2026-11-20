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

// CommittedItem refers to sealed bytes in the provider's local committed index.
// Digest is the expected SHA-256 from that index, independent of Sealed.
type CommittedItem struct {
	Locator, Digest [32]byte
	Sealed          []byte
}

type CommittedCatalog = CommittedItem
type CommittedObject = CommittedItem

// An invalid committed item keeps its membership but cannot be served.
func (item CommittedItem) owned(minSize, maxSize int) CommittedItem {
	if len(item.Sealed) < minSize || len(item.Sealed) > maxSize {
		item.Sealed = nil
		return item
	}
	item.Sealed = bytes.Clone(item.Sealed)
	if sha256.Sum256(item.Sealed) != item.Digest {
		item.Sealed = nil
	}
	return item
}

// ReadProvider serves the prototype's single committed pair. Grant state
// is process-local. A restarted provider needs an independently trusted source
// for its latest signed record before it may serve reads.
type ReadProvider struct {
	agreement CatalogAgreement
	catalog   CommittedCatalog
	object    *CommittedObject // absent on catalog-only providers

	mu         sync.RWMutex // a grant change excludes writes of individual frames
	record     [198]byte
	generation uint64
	recovery   [32]byte // zero for missing or revoked grant
}

type CatalogProvider = ReadProvider

// NewCatalogProvider provisions only a catalog. An object request still
// receives a fresh challenge and cannot disclose object membership.
func NewCatalogProvider(agreement CatalogAgreement, catalog CommittedCatalog) (*CatalogProvider, error) {
	if !storageformat.ValidEd25519Key(agreement.ProviderKey[:]) || !storageformat.ValidEd25519Key(agreement.AuthorityKey[:]) {
		return nil, errors.New("invalid provider or folder authority public key")
	}
	if agreement.Start.IsZero() || !agreement.Expiry.After(agreement.Start) {
		return nil, errors.New("invalid enrolled agreement term")
	}
	return &ReadProvider{agreement: agreement, catalog: catalog.owned(catalogMinSize, catalogMaxSize)}, nil
}

// NewReadProvider provisions an immutable committed catalog and object. Their
// opaque locators and digests come from the provider's own index.
func NewReadProvider(agreement CatalogAgreement, catalog CommittedCatalog, object CommittedObject) (*ReadProvider, error) {
	if catalog.Locator == object.Locator {
		return nil, errors.New("committed catalog and object must have distinct locators")
	}
	p, err := NewCatalogProvider(agreement, catalog)
	if err != nil {
		return nil, err
	}
	// At most 64 MiB of plaintext, plus one 16-byte tag per 4 MiB block.
	owned := object.owned(16, (64<<20)+16*16)
	p.object = &owned
	return p, nil
}

// LoadGrant verifies the current locally supplied record. The caller must
// supply the highest authoritative record when initializing the provider.
// This method does not implement durable GRANT_UPDATE or issue an ACK.
func (p *ReadProvider) LoadGrant(record []byte) error {
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
