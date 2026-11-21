// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

type interruptedGrantStore struct {
	uncertainGrantStore
	intent []byte
	fail   bool
}

func (s *interruptedGrantStore) PendingRevocation() ([]byte, error) { return s.intent, nil }
func (s *interruptedGrantStore) CompleteRevocation(record []byte) error {
	if s.fail || !bytes.Equal(record, s.intent) {
		return errors.New("could not finish this revocation")
	}
	s.record, s.intent = bytes.Clone(record), nil
	return nil
}

func TestRestoreRequiresAnAuthenticPendingRevocation(t *testing.T) {
	authorityKey, authority := grantTestKey(t)
	providerKey, _ := grantTestKey(t)
	recoveryKey, _ := grantTestKey(t)
	_, otherAuthority := grantTestKey(t)
	now := time.Now()
	a := CatalogAgreement{AuthorityKey: authorityKey, ProviderKey: providerKey, Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour)}
	active := grantTestRecord(a, recoveryKey, 7, authority)
	revoked := grantTestRecord(a, [32]byte{}, 8, authority)
	otherScope := a
	otherScope.FolderID[0] ^= 1
	for _, intent := range [][]byte{
		active, {}, grantTestRecord(a, [32]byte{}, 8, otherAuthority),
		grantTestRecord(otherScope, [32]byte{}, 8, authority),
	} {
		p := &ReadProvider{agreement: a}
		store := &interruptedGrantStore{uncertainGrantStore: uncertainGrantStore{active}, intent: intent}
		if err := p.restoreGrant(store); err == nil || p.store != nil || p.recovery != [32]byte{} {
			t.Fatal("invalid intent admitted the former or an unauthenticated grant")
		}
	}
	p := &ReadProvider{agreement: a}
	store := &interruptedGrantStore{uncertainGrantStore: uncertainGrantStore{active}, intent: revoked}
	if err := p.restoreGrant(store); err != nil {
		t.Fatal(err)
	}
	for _, different := range [][]byte{active, grantTestRecord(a, recoveryKey, 8, authority), grantTestRecord(a, recoveryKey, 9, authority)} {
		if !errors.Is(p.LoadGrant(different), ErrGrantConflict) {
			t.Fatal("accepted a different signed update while revocation was pending")
		}
	}
	// A second interruption cannot turn either the cached or the durable
	// active row back into permission. Another reopen may retry the same bytes.
	store.fail = true
	if !errors.Is(p.applyGrant(revoked, true), ErrGrantUnavailable) || !p.offline || !p.pendingRevocation {
		t.Fatal("reported success despite failing to finish the retry")
	}
	store.fail = false
	p = &ReadProvider{agreement: a}
	if err := p.restoreGrant(store); err != nil {
		t.Fatal(err)
	}
	if err := p.applyGrant(revoked, true); err != nil || p.pendingRevocation || p.recovery != [32]byte{} {
		t.Fatalf("exact retry did not finish with the credential revoked: %v", err)
	}
	if err := p.applyGrant(revoked, true); err != nil {
		t.Fatalf("repeated confirmation failed: %v", err)
	}
}
