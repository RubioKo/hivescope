// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestComparedFileBlocksWriterOnNTFS(t *testing.T) {
	parent := t.TempDir()
	dir, err := openPublishDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	path := filepath.Join(parent, "chosen")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := openComparedFile(dir, "chosen")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.WriteFile(path, []byte("changed"), 0600); !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		t.Fatalf("concurrent write: got %v, want sharing violation", err)
	}
	before, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	if same, err := comparedFileUnchanged(dir, "chosen", before); !same || err != nil {
		t.Fatalf("stable regular file: same=%t err=%v", same, err)
	}
}

func TestComparedFileReadsReadOnlyDestination(t *testing.T) {
	parent := t.TempDir()
	dir, err := openPublishDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	path := filepath.Join(parent, "chosen")
	if err := os.WriteFile(path, []byte("manual edit"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0600)
	f, err := openComparedFile(dir, "chosen")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, err := io.ReadAll(f); err != nil || string(got) != "manual edit" {
		t.Fatalf("read-only comparison: got %q, %v", got, err)
	}
	if err := f.Sync(); err == nil {
		t.Fatal("read-only handle unexpectedly permitted a durability confirmation")
	}
}

func TestComparedFileRejectsNTFSFileSymlink(t *testing.T) {
	parent := t.TempDir()
	dir, err := openPublishDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	target := filepath.Join(parent, "real")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(parent, "link")); err != nil {
		if errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD) {
			t.Skip("this account cannot create Windows file symlinks")
		}
		t.Fatal(err)
	}
	if f, err := openComparedFile(dir, "link"); err == nil {
		f.Close()
		t.Fatal("opened an NTFS file symlink for comparison")
	}
}
