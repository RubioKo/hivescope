// SPDX-License-Identifier: AGPL-3.0-only

package conformance

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/iotest"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

func TestRecoveryUnlockBeforeProviderOrDestination(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("local atomic publication is available on Linux and Windows")
	}
	const phrase = "HiveScope fixture 1"
	export := recoveryFixture(t, phrase, 0).seal()
	damaged := bytes.Clone(export)
	damaged[len(damaged)-1] ^= 1
	f := hiveFixture()
	object, catalog := fiveByteSnapshot(t)
	for _, tc := range []struct {
		name   string
		input  []byte
		phrase string
		opened bool
	}{
		{"wrong passphrase", export, "incorrect fixture passphrase", false},
		{"damaged tag", damaged, phrase, false},
		{"correct passphrase", export, phrase, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, state := t.TempDir(), t.TempDir()
			dest := filepath.Join(dir, "selected.bin")
			if err := os.WriteFile(dest, []byte("keep\n"), 0600); err != nil {
				t.Fatal(err)
			}

			var providerReads, destinationCalls int
			// This stand-in receives only the endpoint, provider identity pin
			// and requested locator. Both responses are sealed reference items.
			readItem := func(endpoint netip.AddrPort, pin, locator [32]byte) io.Reader {
				t.Helper()
				providerReads++
				if endpoint != netip.MustParseAddrPort("192.0.2.10:4001") || pin != [32]byte(mustHex(t, readFixtureProviderPublic)) {
					t.Fatal("unlocked contact details differ from the fixture")
				}
				switch locator {
				case [32]byte(repeated(f.catalogLocator, 32)):
					return bytes.NewReader(catalog)
				case [32]byte(repeated(f.objectLocator, 32)):
					return bytes.NewReader(object)
				default:
					t.Fatal("unknown read locator")
					return nil
				}
			}
			save := func(snapshot *storageformat.Snapshot, src io.Reader) error {
				destinationCalls++
				return snapshot.SaveRecoverableObject(src, dest, true, state)
			}
			err := storageformat.WithRecoveryPackage(iotest.OneByteReader(bytes.NewReader(tc.input)), []byte(tc.phrase), func(p *storageformat.RecoveryPackage) error {
				pub := ed25519.NewKeyFromSeed(p.RecoverySeed[:]).Public().(ed25519.PublicKey)
				if p.Generation != 7 || !bytes.Equal(pub, mustHex(t, readFixtureRecoveryPublic)) || p.Catalog != f.context(t) {
					t.Fatal("unlocked grant or catalog context differs from the reference")
				}
				catalogSrc := readItem(p.Endpoint, p.ProviderKey, p.Catalog.CatalogLocator)
				snapshot, err := storageformat.OpenCatalog(catalogSrc, p.Catalog)
				if err != nil {
					return err
				}
				return save(snapshot, readItem(p.Endpoint, p.ProviderKey, snapshot.ObjectLocator()))
			})
			if tc.opened {
				if err != nil || providerReads != 2 || destinationCalls != 1 {
					t.Fatalf("valid unlock: provider reads %d, destination calls %d, error %v", providerReads, destinationCalls, err)
				}
				requireFileBytes(t, dest, f.plaintext)
				return
			}
			if !errors.Is(err, storageformat.ErrRecoveryAuthentication) || providerReads != 0 || destinationCalls != 0 {
				t.Fatalf("rejected unlock: provider reads %d, destination calls %d, error %v", providerReads, destinationCalls, err)
			}
			requireFileBytes(t, dest, []byte("keep\n"))
			expectEmptyDir(t, state)
		})
	}
}

func requireRecoveryRejected(t *testing.T, src io.Reader, passphrase string) {
	t.Helper()
	err := storageformat.WithRecoveryPackage(src, []byte(passphrase), func(*storageformat.RecoveryPackage) error {
		t.Fatal("invalid input reached the unlocked recovery operation")
		return nil
	})
	if err == nil {
		t.Fatal("invalid input returned success")
	}
}

func TestRecoveryReaderRequiresCompleteV1Frame(t *testing.T) {
	const phrase = "HiveScope fixture 1"
	example := recoveryFixture(t, phrase, 0).seal()
	badMagic, badVersion := bytes.Clone(example), bytes.Clone(example)
	badMagic[0] ^= 1
	badVersion[4]++
	for _, tc := range []struct {
		name string
		src  io.Reader
	}{
		{"truncated", bytes.NewReader(example[:len(example)-1])},
		{"trailing byte", io.MultiReader(bytes.NewReader(example), bytes.NewReader([]byte{0}))},
		{"unknown magic", bytes.NewReader(badMagic)},
		{"unknown version", bytes.NewReader(badVersion)},
		{"error while checking EOF", io.MultiReader(bytes.NewReader(example), iotest.ErrReader(io.ErrClosedPipe))},
	} {
		t.Run(tc.name, func(t *testing.T) { requireRecoveryRejected(t, tc.src, phrase) })
	}
	// An import must not buffer the rest of an arbitrarily long input.
	excess := bytes.NewReader(make([]byte, 2048))
	requireRecoveryRejected(t, io.MultiReader(bytes.NewReader(example), excess), phrase)
	if excess.Len() < 2047 {
		t.Fatal("reader consumed more than one byte beyond the v1 frame")
	}
}

func TestRecoveryReaderValidatesAuthenticatedFields(t *testing.T) {
	// Little-endian y = p + 3 has a noncanonical encoding even though its
	// reduction gives a curve point. y = 2 has no corresponding curve point.
	noncanonical := repeated(0xff, 32)
	noncanonical[0], noncanonical[31] = 0xf0, 0x7f
	offCurve, identity := make([]byte, 32), make([]byte, 32)
	offCurve[0], identity[0] = 2, 1
	for i, tc := range []struct {
		field  string
		offset int
		value  []byte
	}{
		{"zero generation", 32, make([]byte, 8)},
		{"zero port", 76, make([]byte, 2)},
		{"off-curve provider key", 40, offCurve},
		{"noncanonical catalog key", 142, noncanonical},
		{"identity provider key", 40, identity},
		{"small-order catalog key", 142, make([]byte, 32)},
	} {
		t.Run(tc.field, func(t *testing.T) {
			const phrase = "HiveScope fixture 1"
			f := recoveryFixture(t, phrase, byte(i+1))
			copy(f.payload[tc.offset:], tc.value)
			requireRecoveryRejected(t, bytes.NewReader(f.seal()), phrase)
		})
	}
}

func TestRecoveryReaderPreservesPassphraseEncoding(t *testing.T) {
	const phrase = " caf\u00e9 "
	export := recoveryFixture(t, phrase, 20).seal()
	want := errors.New("operation reached after exact UTF-8 unlock")
	err := storageformat.WithRecoveryPackage(bytes.NewReader(export), []byte(phrase), func(*storageformat.RecoveryPackage) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("exact passphrase failed or callback error lost: %v", err)
	}
	requireRecoveryRejected(t, bytes.NewReader(export), "caf\u00e9")
	requireRecoveryRejected(t, bytes.NewReader(export), " cafe\u0301 ")
	// An independently sealed fixture with non-UTF-8 password bytes proves
	// that a matching tag alone cannot make that password a valid input.
	const invalid = "pass\xff"
	requireRecoveryRejected(t, bytes.NewReader(recoveryFixture(t, invalid, 21).seal()), invalid)
}
