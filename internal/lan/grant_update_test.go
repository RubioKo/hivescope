// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"io"
	"testing"
	"time"
)

func TestGrantAckRequiresExactDigestAndEOF(t *testing.T) {
	digest := sha256.Sum256([]byte("synthetic signed record"))
	var ack bytes.Buffer
	if err := writeFrame(&ack, grantAckOp, digest[:]); err != nil {
		t.Fatal(err)
	}
	wire := ack.Bytes()
	if err := verifyGrantAck(bytes.NewReader(wire), digest); err != nil {
		t.Fatal(err)
	}
	wrong := bytes.Clone(wire)
	wrong[len(wrong)-1] ^= 1
	for _, response := range [][]byte{nil, wire[:len(wire)-1], wrong, append(bytes.Clone(wire), 0)} {
		if err := verifyGrantAck(bytes.NewReader(response), digest); err == nil {
			t.Fatal("confirmed an incomplete or mismatched grant acknowledgment")
		}
	}
}

// A transaction can reach the disk and still fail to confirm its checkpoint.
type uncertainGrantStore struct{ record []byte }

func (s *uncertainGrantStore) Record() ([]byte, error) { return s.record, nil }
func (s *uncertainGrantStore) Close() error            { return nil }
func (s *uncertainGrantStore) Replace(_, next []byte) error {
	s.record = bytes.Clone(next)
	return errors.New("synthetic checkpoint failure")
}

func TestGrantUpdateFailsClosed(t *testing.T) {
	providerKey, _ := grantTestKey(t)
	authorityKey, authority := grantTestKey(t)
	recoveryKey, _ := grantTestKey(t)
	now := time.Now()
	a := CatalogAgreement{ProviderKey: providerKey, AuthorityKey: authorityKey, Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour)}
	p, err := NewCatalogProvider(a, CommittedCatalog{})
	if err != nil {
		t.Fatal(err)
	}
	initial := grantTestRecord(a, recoveryKey, 7, authority)
	if err := p.LoadGrant(initial); err != nil {
		t.Fatal(err)
	}
	revocation, err := a.SignRevocation(8, [ed25519.SeedSize]byte(authority.Seed()))
	if err != nil || !bytes.Equal(revocation, grantTestRecord(a, [32]byte{}, 8, authority)) {
		t.Fatalf("revocation differs from the independent signed record: %v", err)
	}
	checkReply := func(record, extra []byte, want byte) {
		t.Helper()
		var reply bytes.Buffer
		stream := struct {
			io.Reader
			io.Writer
		}{bytes.NewReader(extra), &reply}
		if err := p.serveGrantUpdate(stream, record); err != nil {
			t.Fatal(err)
		}
		op, payload, err := readFrame(&reply, sha256.Size)
		if err != nil || op != errorOp || !bytes.Equal(payload, []byte{want}) || reply.Len() != 0 {
			t.Fatalf("expected only error %d; op=%d payload=%x err=%v", want, op, payload, err)
		}
	}
	checkReply(revocation, nil, 7) // An in-memory provider cannot confirm durability.
	s := &uncertainGrantStore{record: initial}
	p.store = s
	checkReply(revocation, []byte{1}, 1) // A second frame or trailing data forbids mutation.
	badSig := bytes.Clone(revocation)
	badSig[len(badSig)-1] ^= 1
	checkReply(badSig, nil, 2)
	if p.generation != 7 || !bytes.Equal(s.record, initial) || p.offline {
		t.Fatal("invalid update changed authorization")
	}
	checkReply(revocation, nil, 7)
	checkReply(initial, nil, 7) // No cached retry ACK on an uncertain provider.
	if !p.offline || !bytes.Equal(s.record, revocation) {
		t.Fatal("an ambiguous commit did not take the provider offline")
	}
	if _, code := p.authorizedItem(nil, [32]byte{}, [32]byte{}, nil, 7, now.Add(time.Minute)); code != 7 {
		t.Fatal("old authorization remained usable after a storage failure")
	}
}
