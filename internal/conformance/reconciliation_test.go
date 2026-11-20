// SPDX-License-Identifier: AGPL-3.0-only

//go:build windows || (linux && !android)

package conformance

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/iotest"
	"time"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

func requireReopenResult(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("reopen returned %v, want %v", got, want)
	}
	if want == storageformat.ErrPublicationUncertain && errors.Is(got, storageformat.ErrDestinationDiffers) {
		t.Fatalf("reported a conflict without a verified comparison: %v", got)
	}
	if want == storageformat.ErrDestinationDiffers && errors.Is(got, storageformat.ErrPublicationUncertain) {
		t.Fatalf("stable difference was also reported as uncertain: %v", got)
	}
}

func TestReopenPreservesSelectedFile(t *testing.T) {
	dest, state := filepath.Join(t.TempDir(), "selected.txt"), t.TempDir()
	saved, object := publicationFixture(t, hiveFixture())
	if err := saved.SaveRecoverableObject(bytes.NewReader(object), dest, false, state); err != nil {
		t.Fatal(err)
	}
	reopened, sealed := publicationFixture(t, hiveFixture())
	reopen := func() error { return reopened.ReopenPublication(bytes.NewReader(sealed), dest, state) }
	stamp := time.Unix(1_600_000_000, 0)
	verifyNoRewrite := func(want error) {
		t.Helper()
		if err := os.Chtimes(dest, stamp, stamp); err != nil {
			t.Fatal(err)
		}
		requireReopenResult(t, reopen(), want)
		info, err := os.Stat(dest)
		if err != nil {
			t.Fatal(err)
		}
		if !info.ModTime().Equal(stamp) {
			t.Fatalf("reopening changed the destination modification time to %v", info.ModTime())
		}
	}
	verifyNoRewrite(nil)
	requireFileBytes(t, dest, []byte("Hive\n"))

	// The plaintext length and name still match after this local edit.
	edit := []byte("Hive!")
	if err := os.WriteFile(dest, edit, 0600); err != nil {
		t.Fatal(err)
	}
	verifyNoRewrite(storageformat.ErrDestinationDiffers)
	requireFileBytes(t, dest, edit)
}

func TestReopenComparesEveryByteAndBothLengths(t *testing.T) {
	for _, fixture := range []snapshotFixture{emptyFixture(), edgeFixture()} {
		t.Run(fixture.name, func(t *testing.T) {
			dir, state := t.TempDir(), t.TempDir()
			dest := filepath.Join(dir, "chosen.bin")
			snapshot, object := publicationFixture(t, fixture)
			if err := snapshot.SaveRecoverableObject(bytes.NewReader(object), dest, false, state); err != nil {
				t.Fatal(err)
			}
			reopened, sealed := publicationFixture(t, fixture)
			reopen := func() error { return reopened.ReopenPublication(bytes.NewReader(sealed), dest, state) }
			requireReopenResult(t, reopen(), nil)
			requireFileBytes(t, dest, fixture.plaintext)

			checkEdit := func(edit []byte) {
				t.Helper()
				if err := os.WriteFile(dest, edit, 0600); err != nil {
					t.Fatal(err)
				}
				requireReopenResult(t, reopen(), storageformat.ErrDestinationDiffers)
				requireFileBytes(t, dest, edit)
			}
			checkEdit(append(bytes.Clone(fixture.plaintext), '!'))
			if n := len(fixture.plaintext); n > 0 {
				checkEdit(fixture.plaintext[:n-1])
				lastChanged := bytes.Clone(fixture.plaintext)
				lastChanged[n-1] ^= 1 // first byte of the second authenticated block
				checkEdit(lastChanged)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(dest) {
				t.Fatalf("unexpected files after comparison: %v, %v", entries, err)
			}
		})
	}
}

func TestReopenRequiresOriginalSnapshotAndDestination(t *testing.T) {
	dir, state := t.TempDir(), t.TempDir()
	dest := filepath.Join(dir, "chosen.bin")
	fixture := hiveFixture()
	snapshot, object := publicationFixture(t, fixture)
	if err := snapshot.SaveRecoverableObject(bytes.NewReader(object), dest, false, state); err != nil {
		t.Fatal(err)
	}

	// Another valid signed catalog, with the same name, length and plaintext.
	other := fixture
	other.versionID, other.versionKey, other.catalogSalt, other.objectLocator = 0x74, 0x84, 0x94, 0xd7
	substitute, sealedOther := publicationFixture(t, other)
	err := substitute.ReopenPublication(bytes.NewReader(sealedOther), dest, state)
	requireReopenResult(t, err, storageformat.ErrPublicationUncertain)
	requireFileBytes(t, dest, fixture.plaintext)

	// Even the very same object and catalog fields sealed with a new salt
	// produce a different signed catalog from the one selected at save time.
	resealed := fixture
	resealed.catalogSalt = 0x95
	substitute, sealedOther = publicationFixture(t, resealed)
	err = substitute.ReopenPublication(bytes.NewReader(sealedOther), dest, state)
	requireReopenResult(t, err, storageformat.ErrPublicationUncertain)

	// A different selected path must not be confirmed even if it also matches.
	otherPath := filepath.Join(dir, "another.bin")
	if err := os.WriteFile(otherPath, fixture.plaintext, 0600); err != nil {
		t.Fatal(err)
	}
	err = snapshot.ReopenPublication(bytes.NewReader(object), otherPath, state)
	requireReopenResult(t, err, storageformat.ErrPublicationUncertain)
	requireFileBytes(t, otherPath, fixture.plaintext)

	offline := errors.New("source unavailable")
	damaged := bytes.Clone(object)
	damaged[len(damaged)-1] ^= 1
	for _, tc := range []struct {
		name string
		src  io.Reader
	}{
		{"unavailable source", iotest.ErrReader(offline)},
		{"damaged authenticated object", bytes.NewReader(damaged)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := snapshot.ReopenPublication(tc.src, dest, state)
			requireReopenResult(t, err, storageformat.ErrPublicationUncertain)
			requireFileBytes(t, dest, fixture.plaintext)
		})
	}

	err = snapshot.ReopenPublication(bytes.NewReader(object), dest, state)
	requireReopenResult(t, err, nil)
	if err := os.Remove(dest); err != nil {
		t.Fatal(err)
	}
	err = snapshot.ReopenPublication(bytes.NewReader(object), dest, state)
	requireReopenResult(t, err, storageformat.ErrPublicationUncertain)
	if _, err := os.Lstat(dest); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("reopen recreated missing destination: %v", err)
	}
}

func TestReopenRequiresIntactDedicatedState(t *testing.T) {
	dir, state := t.TempDir(), t.TempDir()
	dest := filepath.Join(dir, "chosen.bin")
	snapshot, object := publicationFixture(t, hiveFixture())
	save := func(path, privateState string) error {
		return snapshot.SaveRecoverableObject(bytes.NewReader(object), path, false, privateState)
	}
	if err := save(dest, dir); err == nil {
		t.Fatal("accepted the destination directory as private restore state")
	}
	expectEmptyDir(t, dir)
	if err := save(dest, state); err != nil {
		t.Fatal(err)
	}
	recordPath := func() string {
		t.Helper()
		entries, err := os.ReadDir(state)
		if err != nil || len(entries) != 1 || !entries[0].Type().IsRegular() {
			t.Fatalf("expected only a restore record in the state directory: %v, %v", entries, err)
		}
		return filepath.Join(state, entries[0].Name())
	}
	record := recordPath()
	original, err := os.ReadFile(record)
	if err != nil || len(original) == 0 {
		t.Fatalf("read restore record: length %d, error %v", len(original), err)
	}
	second := filepath.Join(dir, "second.bin")
	if err := save(second, state); err == nil {
		t.Fatal("overwrote the restore record of an earlier operation")
	}
	if _, err := os.Lstat(second); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("published a second destination after a restore record collision: %v", err)
	}
	requireFileBytes(t, recordPath(), original)
	reopen := func() error { return snapshot.ReopenPublication(bytes.NewReader(object), dest, state) }
	requireReopenResult(t, reopen(), nil)
	requireFileBytes(t, recordPath(), original)

	damaged := bytes.Clone(original)
	damaged[len(damaged)-1] ^= 1
	for _, bad := range [][]byte{damaged, original[:len(original)-1], append(bytes.Clone(original), 0)} {
		if err := os.WriteFile(record, bad, 0600); err != nil {
			t.Fatal(err)
		}
		requireReopenResult(t, reopen(), storageformat.ErrPublicationUncertain)
		requireFileBytes(t, dest, hiveFixture().plaintext)
		requireFileBytes(t, recordPath(), bad)
	}
	if err := os.Remove(record); err != nil {
		t.Fatal(err)
	}
	requireReopenResult(t, reopen(), storageformat.ErrPublicationUncertain)
	expectEmptyDir(t, state)
	requireFileBytes(t, dest, hiveFixture().plaintext)
}
