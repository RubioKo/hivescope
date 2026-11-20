// SPDX-License-Identifier: AGPL-3.0-only

package conformance

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

func publicationFixture(t *testing.T, fixture snapshotFixture) (*storageformat.Snapshot, []byte) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("local publication currently targets Linux and Windows")
	}
	object := fixture.sealObject(t)
	catalog := fixture.sealCatalog(t, fixture.entry(object))
	snapshot, err := storageformat.OpenCatalog(bytes.NewReader(catalog), fixture.context(t))
	if err != nil {
		t.Fatal(err)
	}
	return snapshot, object
}

func requirePublishedBytes(t *testing.T, path string, want []byte, saveErr error) {
	t.Helper()
	if saveErr != nil {
		t.Fatalf("save failed: %v", saveErr)
	}
	requireFileBytes(t, path, want)
}

func requireFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s differs from the expected %d bytes (read %d)", filepath.Base(path), len(want), len(got))
	}
}

func TestSaveAtChosenDestination(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "chosen.bin")
	// The display name belongs to a different, existing local file.
	unrelated := filepath.Join(dir, "hive.txt")
	keep := []byte("unrelated local file\n")
	if err := os.WriteFile(unrelated, keep, 0600); err != nil {
		t.Fatal(err)
	}

	snapshot, object := publicationFixture(t, hiveFixture())
	err := snapshot.SaveObject(bytes.NewReader(object), dest, false)
	requirePublishedBytes(t, dest, []byte("Hive\n"), err)
	if runtime.GOOS == "linux" {
		info, err := os.Stat(dest)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("published plaintext is accessible beyond the owner: %v", info.Mode())
		}
	}
	requireFileBytes(t, unrelated, keep)

	empty, sealedEmpty := publicationFixture(t, emptyFixture())
	err = empty.SaveObject(bytes.NewReader(sealedEmpty), dest, false)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("create-only collision returned %v, want file exists", err)
	}
	requireFileBytes(t, dest, []byte("Hive\n"))
	err = empty.SaveObject(bytes.NewReader(sealedEmpty), dest, true)
	requirePublishedBytes(t, dest, nil, err)
	edge := edgeFixture()
	multi, sealedMulti := publicationFixture(t, edge)
	err = multi.SaveObject(bytes.NewReader(sealedMulti), dest, true)
	requirePublishedBytes(t, dest, edge.plaintext, err)
	requireFileBytes(t, unrelated, keep)
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 2 {
		t.Fatalf("expected only the two named files: %v, %v", entries, err)
	}
}

func TestFailedVerificationPreservesDestination(t *testing.T) {
	snapshot, object := publicationFixture(t, edgeFixture())
	altered := bytes.Clone(object)
	altered[len(altered)-1] ^= 1
	for _, tc := range []struct {
		name   string
		sealed []byte
	}{
		{"incomplete final block", object[:len(object)-1]},
		{"altered final tag", altered},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			dest := filepath.Join(dir, "existing.bin")
			previous := []byte("keep\n")
			if err := os.WriteFile(dest, previous, 0600); err != nil {
				t.Fatal(err)
			}
			err := snapshot.SaveObject(bytes.NewReader(tc.sealed), dest, true)
			if err == nil || errors.Is(err, storageformat.ErrPublicationUncertain) {
				t.Fatalf("invalid object needs a definite failure before publication, got %v", err)
			}
			requireFileBytes(t, dest, previous)
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("expected only the preserved destination: %v, %v", entries, err)
			}
		})
	}
}

func TestConcurrentCreateOnly(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "shared.bin")
	fixtures := []snapshotFixture{hiveFixture(), emptyFixture()}
	start := make(chan struct{})
	type result struct {
		i   int
		err error
	}
	results := make(chan result, len(fixtures))
	for i, f := range fixtures {
		snapshot, object := publicationFixture(t, f)
		go func() {
			<-start
			results <- result{i, snapshot.SaveObject(bytes.NewReader(object), dest, false)}
		}()
	}
	close(start)
	completed := make([]result, 0, len(fixtures))
	for range fixtures {
		completed = append(completed, <-results)
	}
	winner := -1
	for _, r := range completed {
		if errors.Is(r.err, fs.ErrExist) {
			continue
		}
		if winner >= 0 {
			t.Fatalf("two create-only operations reported publication: %v", r.err)
		}
		requirePublishedBytes(t, dest, fixtures[r.i].plaintext, r.err)
		winner = r.i
	}
	if winner < 0 {
		t.Fatal("neither creator published")
	}
	requireFileBytes(t, dest, fixtures[winner].plaintext)
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected only the winning destination: %v, %v", entries, err)
	}
}

// Supply EOF only after another application has created the destination.
type eofAction func()

func (action eofAction) Read([]byte) (int, error) {
	action()
	return 0, io.EOF
}

func TestCreateOnlyKeepsLateArrival(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "late.bin")
	lateBytes := []byte("created during verification\n")
	snapshot, object := publicationFixture(t, hiveFixture())
	src := io.MultiReader(bytes.NewReader(object), eofAction(func() {
		if err := os.WriteFile(dest, lateBytes, 0600); err != nil {
			t.Fatal(err)
		}
	}))
	err := snapshot.SaveObject(src, dest, false)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("late-arriving destination returned %v, want file exists", err)
	}
	requireFileBytes(t, dest, lateBytes)
}

func TestSaveRejectsInvalidDestination(t *testing.T) {
	snapshot, object := publicationFixture(t, hiveFixture())
	dir := t.TempDir()
	previous := []byte("still inside directory\n")
	file := filepath.Join(dir, "keep.bin")
	if err := os.WriteFile(file, previous, 0600); err != nil {
		t.Fatal(err)
	}
	destinations := []string{"", dir, filepath.Join(dir, "absent", "output")}
	if runtime.GOOS == "windows" {
		destinations = append(destinations, file+":stream", file+".")
	}
	for _, dest := range destinations {
		err := snapshot.SaveObject(bytes.NewReader(object), dest, true)
		if err == nil || errors.Is(err, storageformat.ErrPublicationUncertain) {
			t.Errorf("unexpected publication result at %q: %v", dest, err)
		}
	}
	requireFileBytes(t, file, previous)
}
