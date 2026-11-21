// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The LAN provider validates the signature and scope before calling Replace.
// This layer treats the bytes outside the generation field as opaque.
func recordAt(generation uint64) []byte {
	r := make([]byte, recordSize)
	binary.BigEndian.PutUint64(r[92:100], generation)
	return r
}

func newStore(t *testing.T) (string, [32]byte, *Store) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "grant-state")
	binding := [32]byte{0x34}
	s, err := Create(dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return dir, binding, s
}

func TestExclusiveEnrollmentAndUnsignedGeneration(t *testing.T) {
	dir, binding, s := newStore(t)
	if second, err := Open(dir, binding); err == nil {
		second.Close()
		t.Fatal("second process owner admitted")
	}
	if second, err := Create(dir, binding); err == nil {
		second.Close()
		t.Fatal("recreated an existing private store")
	}
	if got, err := s.Record(); err != nil || got != nil {
		t.Fatalf("new grant should be explicitly empty: bytes=%d err=%v", len(got), err)
	}
	first, final := recordAt(7), recordAt(math.MaxUint64)
	if err := s.Replace(nil, first); err != nil {
		t.Fatal(err)
	}
	if err := s.Replace(nil, final); err == nil {
		t.Fatal("replaced a stale grant")
	}
	if err := s.Replace(first, final); err != nil {
		t.Fatal(err)
	}
	if err := s.Replace(final, first); err == nil {
		t.Fatal("accepted a lower unsigned generation")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if other, err := Open(dir, [32]byte{0x35}); err == nil {
		other.Close()
		t.Fatal("reopened under a different enrollment")
	}
	reopened, err := Open(dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if got, err := reopened.Record(); err != nil || !bytes.Equal(got, final) {
		t.Fatalf("highest generation did not survive: err=%v", err)
	}
}

func TestOlderDatabaseRequiresReconciliation(t *testing.T) {
	dir, binding, s := newStore(t)
	active, revoked := recordAt(7), recordAt(8)
	if err := s.Replace(nil, active); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(dir, indexName)
	older, err := os.ReadFile(index) // no transaction pending, DELETE journal mode
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Replace(active, revoked); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(index, older, 0600); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(dir, binding); err == nil {
		reopened.Close()
		t.Fatal("a rolled-back index served the superseded grant")
	}
}

func TestCheckpointFailureDisablesStore(t *testing.T) {
	dir, binding, s := newStore(t)
	first, next := recordAt(7), recordAt(8)
	if err := s.Replace(nil, first); err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(dir, "checkpoint.next")
	if err := os.WriteFile(pending, []byte("already occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Replace(first, next); err == nil {
		t.Fatal("unconfirmed checkpoint reported success")
	}
	if got, err := s.Record(); err == nil || len(got) != 0 {
		t.Fatal("uncertain update left earlier grant active")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(pending); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(dir, binding); err == nil {
		reopened.Close()
		t.Fatal("accepted disagreement between SQLite and the checkpoint")
	}
}

func TestGrantStateBackingChecks(t *testing.T) {
	dir, binding, s := newStore(t)
	first := recordAt(7)
	if err := s.Replace(nil, first); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, anchorName)); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Record(); err == nil || len(got) != 0 {
		t.Fatal("cached grant survived loss of its durable checkpoint")
	}
	if err := s.Replace(first, recordAt(8)); err == nil {
		t.Fatal("damaged live store accepted another update")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(dir, binding); err == nil {
		reopened.Close()
		t.Fatal("missing checkpoint was recreated")
	}
	index := filepath.Join(dir, indexName)
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	if reopened, err := Open(dir, binding); err == nil {
		reopened.Close()
		t.Fatal("missing index was recreated")
	}
	if _, err := os.Stat(index); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("Open wrote a missing index")
	}
}

func TestUnexpectedLinksFailClosed(t *testing.T) {
	for _, link := range []string{"symlink-index", "hardlink-index", "symlink-journal"} {
		t.Run(link, func(t *testing.T) {
			dir, binding, s := newStore(t)
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			index := filepath.Join(dir, indexName)
			other := filepath.Join(filepath.Dir(dir), "external")
			var err error
			switch link {
			case "symlink-index":
				if err := os.Rename(index, other); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink(other, index)
			case "hardlink-index":
				err = os.Link(index, other)
			case "symlink-journal":
				err = os.Symlink(other, index+"-journal")
			}
			if err != nil {
				t.Fatal(err)
			}
			if reopened, err := Open(dir, binding); err == nil {
				reopened.Close()
				t.Fatal("accepted unexpected link")
			}
		})
	}
}
