// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package lan

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
)

func TestPublishedPairDamageAndRevokedAccess(t *testing.T) {
	host, hostSigner := grantTestKey(t)
	authority, authoritySigner := grantTestKey(t)
	writer, writerSigner := grantTestKey(t)
	recovery, recoverySigner := grantTestKey(t)
	device, _ := grantTestKey(t)
	now := time.Now()
	a := CatalogAgreement{ProviderKey: host, AuthorityKey: authority, FolderID: [16]byte{1},
		AgreementID: [16]byte{2}, Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour)}
	sealedObject, sealedCatalog := bytes.Repeat([]byte{1}, 21), bytes.Repeat([]byte{2}, 250)
	object := UploadItem{Locator: [32]byte{1}, Size: uint64(len(sealedObject)), Digest: sha256.Sum256(sealedObject)}
	catalog := UploadItem{Locator: [32]byte{2}, Size: uint64(len(sealedCatalog)), Digest: sha256.Sum256(sealedCatalog)}
	m, err := NewUploadManifest(host, writer, a.FolderID, a.AgreementID, [16]byte{3}, object, catalog)
	if err != nil {
		t.Fatal(err)
	}
	dir, storeID := filepath.Join(t.TempDir(), "provider"), [32]byte{4}
	limits := grantstore.UploadLimits{MaxRequests: 1, HostBytes: 64 << 20, AgreementBytes: 64 << 20, RequesterBytes: 64 << 20}
	p, err := CreateUploadProvider(dir, storeID, a, writer, limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	ref := uploadReference(m)
	if _, code := p.admitUpload(writer, reserveOp, m[:], nil); code != 0 {
		t.Fatalf("reservation refused: %d", code)
	}
	for kind, data := range map[byte][]byte{catalogKind: sealedCatalog, objectKind: sealedObject} {
		w, _, code := p.beginPut(writer, append(ref[:], kind))
		if code != 0 || w == nil {
			t.Fatalf("could not stage kind %d: %d", kind, code)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Finish(); err != nil {
			t.Fatal(err)
		}
	}
	sign := func(data []byte) ([]byte, error) { return ed25519.Sign(hostSigner, data), nil }
	if state, code := p.admitUpload(writer, finalizeOp, ref[:], sign); code != 0 || state.Code != 2 {
		t.Fatalf("publish: state=%d error=%d", state.Code, code)
	}
	assertRefusal := func(kind byte, locator [32]byte, prover ed25519.PrivateKey, want byte) {
		t.Helper()
		request := append([]byte{kind}, a.FolderID[:]...)
		request = append(request, a.AgreementID[:]...)
		request = binary.BigEndian.AppendUint64(request, 7)
		request = append(request, locator[:]...)
		nonce := [32]byte{9}
		proof := ed25519.Sign(prover, readTranscript(host, device, request, nonce))
		var wire bytes.Buffer
		send := func(op byte, b []byte) error { return writeFrame(&wire, op, b) }
		if err := p.releaseRead(send, request, nonce, device, proof, p.generation, time.Now().Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		op, payload, err := readFrame(&wire, 1)
		if err != nil || op != errorOp || !bytes.Equal(payload, []byte{want}) || wire.Len() != 0 {
			t.Fatalf("expected only error %d, got op=%x payload=%x err=%v", want, op, payload, err)
		}
	}
	assertRefusal(catalogKind, catalog.Locator, recoverySigner, 2) // publication grants no read authority
	if err := p.LoadGrant(grantTestRecord(a, recovery, 7, authoritySigner)); err != nil {
		t.Fatal(err)
	}
	// Damage only the catalog while preserving its recorded length.
	sealedCatalog[0] ^= 1
	if err := os.WriteFile(filepath.Join(dir, "item-1-1"), sealedCatalog, 0600); err != nil {
		t.Fatal(err)
	}
	for kind, locator := range map[byte][32]byte{catalogKind: catalog.Locator, objectKind: object.Locator} {
		assertRefusal(kind, locator, writerSigner, 2)
		assertRefusal(kind, locator, recoverySigner, 7)
	}
	assertRefusal(catalogKind, object.Locator, recoverySigner, 2)
	if err := p.LoadGrant(grantTestRecord(a, [32]byte{}, 8, authoritySigner)); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenUploadProvider(dir, storeID, a, writer, limits)
	if err != nil {
		t.Fatal(err)
	}
	p = reopened
	if p.generation != 8 {
		t.Fatalf("reopened generation %d, want 8", p.generation)
	}
	assertRefusal(catalogKind, catalog.Locator, recoverySigner, 2)
	assertRefusal(objectKind, object.Locator, recoverySigner, 2)
}
