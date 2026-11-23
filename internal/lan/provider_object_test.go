// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"runtime"
	"testing"
	"time"
)

func TestReadStopsWhenVerifiedItemBecomesShort(t *testing.T) {
	// The open descriptor was verified at admission, but the volume stops
	// supplying bytes during the second frame.
	item := &readableItem{
		ReadCloser: io.NopCloser(bytes.NewReader(make([]byte, maxDataPayload+1))),
		size:       maxDataPayload + 2,
	}
	var wire bytes.Buffer
	send := func(op byte, payload []byte) error { return writeFrame(&wire, op, payload) }
	if err := sendItem(send, item); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected a truncated item error, got %v", err)
	}
	op, size, err := readFrame(&wire, 8)
	if err != nil || op != readSizeOp || len(size) != 8 || binary.BigEndian.Uint64(size) != item.size {
		t.Fatalf("invalid READ_SIZE: op=%x size=%x err=%v", op, size, err)
	}
	op, data, err := readFrame(&wire, maxDataPayload)
	if err != nil || op != dataOp || len(data) != maxDataPayload || wire.Len() != 0 {
		t.Fatal("expected one complete DATA frame and no END after read failure")
	}
}

func TestCommittedObjectRead(t *testing.T) {
	providerKey, _ := grantTestKey(t)
	authorityKey, authority := grantTestKey(t)
	recoveryKey, recovery := grantTestKey(t)
	replacement, _ := grantTestKey(t)
	now := time.Now()
	agreement := CatalogAgreement{ProviderKey: providerKey, AuthorityKey: authorityKey, Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour)}
	catalog := CommittedCatalog{Locator: [32]byte{1}, Sealed: make([]byte, 250)}
	catalog.Digest = sha256.Sum256(catalog.Sealed)
	object := CommittedObject{Locator: [32]byte{2}, Sealed: bytes.Repeat([]byte{0x5a}, maxDataPayload+1)}
	object.Digest = sha256.Sum256(object.Sealed)

	collision := object
	collision.Locator = catalog.Locator
	if _, err := NewReadProvider(agreement, catalog, collision); err == nil {
		t.Fatal("catalog and object shared a locator")
	}
	p, err := NewReadProvider(agreement, catalog, object)
	if err != nil {
		t.Fatal(err)
	}
	active := grantTestRecord(agreement, recoveryKey, 7, authority)
	if err := p.LoadGrant(active); err != nil {
		t.Fatal(err)
	}
	request := make([]byte, 73)
	binary.BigEndian.PutUint64(request[33:41], 7)
	nonce := [32]byte{3}
	signed := func(kind byte, locator [32]byte) []byte {
		request[0] = kind
		copy(request[41:], locator[:])
		return ed25519.Sign(recovery, readTranscript(providerKey, replacement, request, nonce))
	}
	check := func(provider *ReadProvider, kind byte, locator [32]byte, want byte) {
		t.Helper()
		proof := signed(kind, locator)
		item, code := provider.authorizedItem(request, nonce, replacement, proof, 7, now.Add(time.Minute))
		if item != nil {
			item.Close()
			t.Fatalf("read kind %d returned content for a refused read", kind)
		}
		if code != want {
			t.Fatalf("read kind %d: code %d, want %d", kind, code, want)
		}
	}
	check(p, objectKind, catalog.Locator, 2)
	check(p, catalogKind, object.Locator, 2)

	catalogOnly, err := NewCatalogProvider(agreement, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalogOnly.LoadGrant(active); err != nil {
		t.Fatal(err)
	}
	check(catalogOnly, objectKind, [32]byte{}, 2)

	tooShort := object
	tooShort.Sealed = tooShort.Sealed[:15] // even an empty file requires a full tag
	tooShort.Digest = sha256.Sum256(tooShort.Sealed)
	shortPair, err := NewReadProvider(agreement, catalog, tooShort)
	if err != nil {
		t.Fatal(err)
	}
	if err := shortPair.LoadGrant(active); err != nil {
		t.Fatal(err)
	}
	check(shortPair, objectKind, object.Locator, 7)

	// The initial provider owns immutable bytes. Provisioning a second
	// provider with the changed input and original digest marks the pair damaged.
	object.Sealed[0] ^= 0xff
	damaged, err := NewReadProvider(agreement, catalog, object)
	if err != nil {
		t.Fatal(err)
	}
	if err := damaged.LoadGrant(active); err != nil {
		t.Fatal(err)
	}
	check(damaged, objectKind, object.Locator, 7)
	check(damaged, catalogKind, catalog.Locator, 7)
	check(damaged, objectKind, [32]byte{9}, 2)

	proof := signed(objectKind, object.Locator)
	revoked := grantTestRecord(agreement, [32]byte{}, 8, authority)
	applied := make(chan error, 1)
	var ops []byte
	var data []byte
	send := func(op byte, payload []byte) error {
		ops = append(ops, op)
		if op == errorOp && !bytes.Equal(payload, []byte{2}) {
			return errors.New("expected DENIED during streaming")
		}
		if op == dataOp {
			data = append(data, payload...)
			if len(data) != maxDataPayload {
				return errors.New("unexpected DATA after first complete frame")
			}
			go func() { applied <- p.LoadGrant(revoked) }()
			// Pause the transport inside the first write until the superseding
			// grant is actually waiting. A writer waiting on RWMutex prevents
			// subsequent RLocks, which makes the interleaving deterministic.
			deadline := time.Now().Add(5 * time.Second)
			for p.mu.TryRLock() {
				p.mu.RUnlock()
				if time.Now().After(deadline) {
					return errors.New("revocation did not reach the active read")
				}
				runtime.Gosched()
			}
		}
		return nil
	}
	if err := p.releaseRead(send, request, nonce, replacement, proof, 7, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ops, []byte{readSizeOp, dataOp, errorOp}) || len(data) != maxDataPayload || data[0] != 0x5a {
		t.Fatalf("expected first immutable DATA then denial: ops=%x, bytes=%d", ops, len(data))
	}
	if err := <-applied; err != nil {
		t.Fatal(err)
	}
}
