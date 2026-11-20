// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android

package storageformat

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestComparedFileRejectsSpecialEntries(t *testing.T) {
	parent := t.TempDir()
	dir, err := openPublishDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	if err := os.WriteFile(filepath.Join(parent, "real"), []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(parent, "link")); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(parent, "fifo"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"link", "fifo"} {
		if f, err := openComparedFile(dir, name); err == nil {
			f.Close()
			t.Errorf("opened %s for comparison", name)
		}
	}
}

func TestComparedFileDetectsConcurrentEditAndReplacement(t *testing.T) {
	parent := t.TempDir()
	dir, err := openPublishDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	path := filepath.Join(parent, "chosen")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := openComparedFile(dir, "chosen")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	before, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("longer"), 0600); err != nil {
		t.Fatal(err)
	}
	if stable, err := comparedFileUnchanged(dir, "chosen", before); stable || err != nil {
		t.Fatalf("after in-place edit: stable=%t err=%v", stable, err)
	}

	before, err = f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(parent, "new")
	if err := os.WriteFile(replacement, []byte("longer"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
	if stable, err := comparedFileUnchanged(dir, "chosen", before); stable || err != nil {
		t.Fatalf("after replacing name with identical bytes: stable=%t err=%v", stable, err)
	}
}
