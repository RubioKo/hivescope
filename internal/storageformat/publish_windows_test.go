// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsPublishName(t *testing.T) {
	for _, name := range []string{"report.pdf:stream", "old. ", "readme.", "trailing ", "NUL.txt", "CON", "COM1.out", "new\nfile", `..\other`, "x?y"} {
		if err := validatePublishName(name); !errors.Is(err, fs.ErrInvalid) {
			t.Errorf("invalid name %q: got %v", name, err)
		}
	}
	for _, name := range []string{"recovered.txt", "copia-音楽.bin", ".hidden", "April notes.pdf"} {
		if err := validatePublishName(name); err != nil {
			t.Errorf("valid name %q: got %v", name, err)
		}
	}
}

func TestWindowsSameDirectoryRename(t *testing.T) {
	dir, err := openPublishDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	stage := filepath.Join(dir.Name(), "owned-stage")
	target := filepath.Join(dir.Name(), "chosen-name")
	if err := os.WriteFile(target, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stage, []byte("verified"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishStaged(dir, "owned-stage", "chosen-name", false); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("create-only collision: got %v, want ErrExist", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "previous" {
		t.Fatalf("destination after collision: %q, %v", got, err)
	}
	if err := publishStaged(dir, "owned-stage", "chosen-name", true); err != nil {
		t.Fatal(err)
	}
	if err := syncPublishDir(dir); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "verified" {
		t.Fatalf("destination after replace: %q, %v", got, err)
	}
	if _, err := os.Lstat(stage); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("source name after rename: got %v, want ErrNotExist", err)
	}
}
