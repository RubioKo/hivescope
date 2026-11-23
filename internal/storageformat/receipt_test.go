// SPDX-License-Identifier: AGPL-3.0-only

//go:build (linux && !android) || windows

package storageformat

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestReceiptSaveRetriesWithoutReplacingConflictingEvidence(t *testing.T) {
	dir := t.TempDir()
	fingerprint, receipt := [32]byte{1}, [132]byte{2}
	save := func() error { return SaveVerifiedReceipt(dir, fingerprint, receipt) }
	for i := 0; i < 2; i++ {
		if err := save(); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(dir, hex.EncodeToString(fingerprint[:])+".receipt")
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, receipt[:]) {
		t.Fatalf("saved receipt differs: %v", err)
	}
	other := append(bytes.Clone(got), 0)
	if err := os.WriteFile(path, other, 0600); err != nil {
		t.Fatal(err)
	}
	if err := save(); err == nil {
		t.Fatal("accepted a different existing receipt")
	}
	got, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(got, other) {
		t.Fatalf("changed the existing receipt: %v", err)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 {
		t.Fatalf("expected only the original entry: %v, %v", files, err)
	}
}
