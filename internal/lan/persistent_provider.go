// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
)

type grantPersistence interface {
	Replace(previous, next []byte) error
	Record() ([]byte, error)
	PendingRevocation() ([]byte, error)
	CompleteRevocation(signed []byte) error
	Close() error
}

// Include the independent store identity, public enrollment and exact term.
// The network address may legitimately change after a restart.
func (a CatalogAgreement) storeBinding(id [32]byte) ([32]byte, error) {
	if id == [32]byte{} {
		return [32]byte{}, errors.New("missing enrolled grant store identity")
	}
	b := append([]byte("HiveScope/local-grant-store/v1\x00"), id[:]...)
	b = append(b, a.ProviderKey[:]...)
	b = append(b, a.AuthorityKey[:]...)
	b = append(b, a.FolderID[:]...)
	b = append(b, a.AgreementID[:]...)
	for _, instant := range []time.Time{a.Start, a.Expiry} {
		b = binary.BigEndian.AppendUint64(b, uint64(instant.Unix()))
		b = binary.BigEndian.AppendUint32(b, uint32(instant.Nanosecond()))
	}
	return sha256.Sum256(b), nil
}

// CreatePersistentReadProvider enrolls a new private directory and commits
// the first signed grant before the provider can serve. Never use creation
// as a fallback when reopening a provisioned agreement fails.
func CreatePersistentReadProvider(dir string, storeID [32]byte, agreement CatalogAgreement,
	catalog CommittedCatalog, object CommittedObject, initial []byte) (*ReadProvider, error) {
	p, err := NewReadProvider(agreement, catalog, object)
	if err != nil {
		return nil, err
	}
	binding, err := agreement.storeBinding(storeID)
	if err != nil {
		return nil, err
	}
	grant, err := agreement.verifyGrant(initial)
	if err != nil {
		return nil, err
	}
	if !agreement.readable(time.Now()) {
		return nil, ErrGrantDenied
	}
	store, err := grantstore.Create(dir, binding)
	if err != nil {
		return nil, err
	}
	if err := store.Replace(nil, grant.record[:]); err != nil {
		return nil, errors.Join(err, store.Close())
	}
	p.store = store
	p.record, p.generation, p.recovery = grant.record, grant.generation, grant.key
	return p, nil
}

// OpenPersistentReadProvider reopens existing enrollment after local unlock.
// It authenticates the current grant or a pending restrictive update. Sealed
// items are still provisioned separately in this stage of the prototype.
func OpenPersistentReadProvider(dir string, storeID [32]byte, agreement CatalogAgreement,
	catalog CommittedCatalog, object CommittedObject) (*ReadProvider, error) {
	p, err := NewReadProvider(agreement, catalog, object)
	if err != nil {
		return nil, err
	}
	binding, err := agreement.storeBinding(storeID)
	if err != nil {
		return nil, err
	}
	store, err := grantstore.Open(dir, binding)
	if err != nil {
		return nil, err
	}
	if err := p.restoreGrant(store); err != nil {
		return nil, errors.Join(err, store.Close())
	}
	return p, nil
}

// A pending intent can only disable access. The previous SQLite row never
// becomes a live read grant while the authority's revocation awaits a retry.
func (p *ReadProvider) restoreGrant(store grantPersistence) error {
	raw, err := store.PendingRevocation()
	if err != nil {
		return err
	}
	pending := raw != nil
	if !pending {
		raw, err = store.Record()
		if err != nil {
			return err
		}
	}
	grant, err := p.agreement.verifyGrant(raw)
	if err != nil {
		return err
	}
	if pending && grant.record[100] != 2 {
		return errors.New("pending grant is not a signed revocation")
	}
	p.pendingRevocation = pending
	p.store = store
	p.record, p.generation, p.recovery = grant.record, grant.generation, grant.key
	return nil
}

// Close stops this instance from authorizing additional frames before it
// releases exclusive ownership of its persistent grant store.
func (p *ReadProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offline = true
	if p.store == nil {
		return nil
	}
	err := p.store.Close()
	p.store = nil
	return err
}
