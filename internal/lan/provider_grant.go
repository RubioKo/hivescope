// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"crypto/sha256"
	"errors"
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

// ReadProvider serves the prototype's single committed pair. Persistent
// providers serialize durable updates and read admission under the same lock.
type ReadProvider struct {
	agreement CatalogAgreement
	catalog   CommittedCatalog
	object    *CommittedObject // absent on catalog-only providers

	mu         sync.RWMutex     // a grant change excludes writes of individual frames
	store      grantPersistence // nil on an in-memory provider
	offline    bool             // closed or awaiting reconciliation after a storage error
	record     [grantRecordSize]byte
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

// LoadGrant verifies a locally supplied signed record. On a persistent
// provider it uses the same commit path as a network update.
func (p *ReadProvider) LoadGrant(record []byte) error {
	return p.applyGrant(record, false)
}

func (p *ReadProvider) applyGrant(raw []byte, requireDurable bool) error {
	grant, err := p.agreement.verifyGrant(raw)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.offline || (requireDurable && p.store == nil) {
		return ErrGrantUnavailable
	}
	if !p.agreement.readable(time.Now()) {
		return ErrGrantDenied
	}
	if grant.generation < p.generation || (grant.generation == p.generation && grant.record != p.record) {
		return ErrGrantConflict
	}
	if grant.generation == p.generation {
		if p.store != nil {
			current, err := p.store.Record()
			if err != nil || !bytes.Equal(current, grant.record[:]) {
				p.offline = true
				return errors.Join(ErrGrantUnavailable, err)
			}
		}
		return nil
	}
	if p.store != nil {
		var previous []byte
		if p.generation != 0 {
			previous = p.record[:]
		}
		if err := p.store.Replace(previous, grant.record[:]); err != nil {
			// The durable outcome can be uncertain. No cached grant may
			// authorize another frame until a new provider reconciles it.
			p.offline = true
			return errors.Join(ErrGrantUnavailable, err)
		}
	}
	p.record, p.generation, p.recovery = grant.record, grant.generation, grant.key
	return nil
}
