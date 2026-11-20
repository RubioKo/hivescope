// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func grantTestKey(t *testing.T) ([32]byte, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return [32]byte(pub), priv
}

func grantTestRecord(a CatalogAgreement, key [32]byte, gen uint64, authority ed25519.PrivateKey) []byte {
	record := append([]byte("HiveScope/recovery-grant/v1\x00"), a.ProviderKey[:]...)
	record = append(record, a.FolderID[:]...)
	record = append(record, a.AgreementID[:]...)
	record = binary.BigEndian.AppendUint64(record, gen)
	state := byte(1)
	if key == [32]byte{} {
		state = 2
	}
	record = append(record, state, 1)
	record = append(record, key[:]...)
	return append(record, ed25519.Sign(authority, record)...)
}

func TestCatalogReadWindow(t *testing.T) {
	start := time.Unix(2_000_000_000, 0)
	a := CatalogAgreement{Start: start, Expiry: start.Add(time.Hour)}
	graceEnd := a.Expiry.Add(7 * 24 * time.Hour)
	if a.readable(start.Add(-time.Nanosecond)) || !a.readable(start) || !a.readable(a.Expiry) ||
		!a.readable(graceEnd.Add(-time.Nanosecond)) || a.readable(graceEnd) {
		t.Fatal("incorrect agreement or seven-day read grace boundary")
	}
}

func TestCatalogLoadGrant(t *testing.T) {
	authorityPub, authority := grantTestKey(t)
	providerKey, _ := grantTestKey(t)
	recovery, wrongSigner := grantTestKey(t)
	now := time.Now()
	a := CatalogAgreement{ProviderKey: providerKey, AuthorityKey: authorityPub, Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour)}
	p, err := NewCatalogProvider(a, CommittedCatalog{})
	if err != nil {
		t.Fatal(err)
	}
	active := grantTestRecord(a, recovery, 7, authority)
	if err := p.LoadGrant(active); err != nil {
		t.Fatal(err)
	}
	if err := p.LoadGrant(active); err != nil {
		t.Fatalf("identical current record rejected: %v", err)
	}
	for _, tc := range []struct {
		name   string
		record []byte
	}{
		{"wrong signing authority", grantTestRecord(a, recovery, 8, wrongSigner)},
		{"superseded generation", grantTestRecord(a, recovery, 6, authority)},
		{"conflict at current generation", grantTestRecord(a, [32]byte{}, 7, authority)},
		{"maximum generation reserved for revocation", grantTestRecord(a, recovery, math.MaxUint64, authority)},
		{"small order key", grantTestRecord(a, [32]byte{1}, 8, authority)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if p.LoadGrant(tc.record) == nil || p.generation != 7 || p.recovery != recovery {
				t.Fatal("invalid update changed active authorization")
			}
		})
	}
	for _, offset := range []int{0, 28, 60, 76, 100, 101} {
		record := grantTestRecord(a, recovery, 8, authority)
		record[offset] ^= 4
		copy(record[134:], ed25519.Sign(authority, record[:134]))
		if p.LoadGrant(record) == nil || p.generation != 7 {
			t.Fatalf("accepted signed update with incorrect field at offset %d", offset)
		}
	}
	if err := p.LoadGrant(grantTestRecord(a, [32]byte{}, 8, authority)); err != nil {
		t.Fatal(err)
	}
	if p.LoadGrant(active) == nil || p.generation != 8 || p.recovery != [32]byte{} {
		t.Fatal("old active grant reopened a revoked grant")
	}
}

func TestCatalogReadDenialBoundaries(t *testing.T) {
	provider, _ := grantTestKey(t)
	authorityPub, authority := grantTestKey(t)
	recovery, signer := grantTestKey(t)
	replacement, _ := grantTestKey(t)
	now := time.Now()
	a := CatalogAgreement{ProviderKey: provider, AuthorityKey: authorityPub, Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour)}
	sealed := make([]byte, 250)
	item := CommittedCatalog{Locator: [32]byte{3}, Digest: sha256.Sum256(sealed), Sealed: sealed}
	p, err := NewCatalogProvider(a, item)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.LoadGrant(grantTestRecord(a, recovery, 7, authority)); err != nil {
		t.Fatal(err)
	}
	request := make([]byte, 73)
	request[0] = 1
	binary.BigEndian.PutUint64(request[33:41], 7)
	copy(request[41:], item.Locator[:])
	nonce := [32]byte{9}
	for _, tc := range []struct {
		name  string
		alter func(*CatalogProvider, []byte)
		code  byte
	}{
		{"unknown folder", func(_ *CatalogProvider, r []byte) { r[1]++ }, 2},
		{"unknown agreement", func(_ *CatalogProvider, r []byte) { r[17]++ }, 2},
		{"wrong generation", func(_ *CatalogProvider, r []byte) { r[40]++ }, 2},
		{"unknown locator", func(_ *CatalogProvider, r []byte) { r[41]++ }, 2},
		{"closed grace", func(p *CatalogProvider, _ []byte) { p.agreement.Expiry = now.Add(-8 * 24 * time.Hour) }, 2},
		{"missing grant", func(p *CatalogProvider, _ []byte) { p.recovery = [32]byte{} }, 2},
		{"damaged committed item", func(p *CatalogProvider, _ []byte) { p.sealed = nil }, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.Clone(request)
			tc.alter(p, r)
			t.Cleanup(func() { p.agreement, p.recovery, p.sealed = a, recovery, sealed })
			proof := ed25519.Sign(signer, readTranscript(provider, replacement, r, nonce))
			var reply bytes.Buffer
			if err := p.releaseCatalog(&reply, r, nonce, replacement, proof, 7, now.Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
			want := []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, tc.code}
			if !bytes.Equal(reply.Bytes(), want) {
				t.Fatal("expected only terminal error, with zero catalog bytes")
			}
		})
	}
	proof := ed25519.Sign(signer, readTranscript(provider, replacement, request, nonce))
	var reply bytes.Buffer
	if err := p.releaseCatalog(&reply, request, nonce, replacement, proof, 7, now); err != nil ||
		!bytes.Equal(reply.Bytes(), []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, 2}) {
		t.Fatal("expired challenge released data")
	}
}
