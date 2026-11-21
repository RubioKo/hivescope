// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type putTrial struct {
	dir     string
	binding [32]byte
	store   *Store
	m       [276]byte
	id      [16]byte
	fp      [32]byte
	object  []byte
	catalog []byte
	state   UploadState
}

func reservePutTrial(t *testing.T) *putTrial {
	t.Helper()
	dir, binding, s := newStore(t)
	tr := &putTrial{
		dir: dir, binding: binding, store: s, m: testManifest(1),
		object: bytes.Repeat([]byte{0xa7}, 21), catalog: bytes.Repeat([]byte{0x81}, 250),
	}
	objectDigest, catalogDigest := sha256.Sum256(tr.object), sha256.Sum256(tr.catalog)
	copy(tr.m[172:204], objectDigest[:])
	copy(tr.m[244:], catalogDigest[:])
	tr.id, tr.fp = [16]byte(tr.m[116:132]), sha256.Sum256(tr.m[:])
	if err := s.InitUploads([96]byte(tr.m[20:116])); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	var err error
	tr.state, err = s.ReserveUpload(tr.m, uploadLimits(), now, now+900)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func (tr *putTrial) begin(t *testing.T, kind byte) *PutWriter {
	t.Helper()
	w, state, err := tr.store.BeginPut(tr.id, tr.fp, kind, uploadLimits(), time.Now())
	if err != nil || w == nil || state.Code != 1 {
		t.Fatalf("begin missing item %d: writer=%v state=%+v err=%v", kind, w != nil, state, err)
	}
	return w
}

func (tr *putTrial) reopen(t *testing.T) {
	t.Helper()
	if err := tr.store.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(tr.dir, tr.binding)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.CheckUploads([96]byte(tr.m[20:116])); err != nil {
		t.Fatal(err)
	}
	tr.store = s
}

func (tr *putTrial) expectMask(t *testing.T, mask byte) {
	t.Helper()
	want := tr.state
	want.Mask = mask
	got, err := tr.store.UploadStatus(tr.id, tr.fp, time.Now().Unix())
	if err != nil || got != want {
		t.Fatalf("status got %+v, err %v, want %+v", got, err, want)
	}
}

func TestPutWriterKeepsOtherSlotAcrossAbort(t *testing.T) {
	tr := reservePutTrial(t)
	w := tr.begin(t, 2)
	if w.Size() != uint64(len(tr.object)) {
		t.Fatal("object writer accepted a different manifest size")
	}
	if _, err := w.Write(tr.object); err != nil {
		t.Fatal(err)
	}
	if state, err := w.Finish(); err != nil || state.Mask != 2 {
		t.Fatal("object staging failed", state, err)
	}
	objectFile := filepath.Join(tr.dir, w.name)
	first, err := os.Stat(objectFile)
	if err != nil {
		t.Fatal(err)
	}
	w = tr.begin(t, 1)
	if _, err := w.Write(tr.catalog[:125]); err != nil {
		t.Fatal(err)
	}
	if second, _, err := tr.store.BeginPut(tr.id, tr.fp, 1, uploadLimits(), time.Now()); !errors.Is(err, ErrUploadBusy) || second != nil {
		t.Fatal("two writers acquired the same reservation", err)
	}
	tr.expectMask(t, 2)
	if _, err := os.Lstat(filepath.Join(tr.dir, w.name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("partial catalog has a named slot", err)
	}
	tr.reopen(t) // closes the unfinished writer
	if _, err := w.Write(tr.catalog[125:]); !errors.Is(err, os.ErrClosed) {
		t.Fatal("closed writer resumed against reopened store", err)
	}
	tr.expectMask(t, 2)
	if again, state, err := tr.store.BeginPut(tr.id, tr.fp, 2, UploadLimits{}, time.Now()); err != nil || again != nil || state.Mask != 2 {
		t.Fatal("already-staged object requested a new writer", state, err)
	}
	w = tr.begin(t, 1)
	if _, err := w.Write(tr.catalog); err != nil {
		t.Fatal(err)
	}
	if state, err := w.Finish(); err != nil || state.Mask != 3 {
		t.Fatal("retry did not complete both staged items", state, err)
	}
	tr.reopen(t)
	tr.expectMask(t, 3)
	final, err := os.Stat(objectFile)
	if err != nil || !os.SameFile(first, final) {
		t.Fatal("retry replaced or removed the staged object", err)
	}
	for path, want := range map[string][]byte{objectFile: tr.object, filepath.Join(tr.dir, w.name): tr.catalog} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("staged file differs from sealed source: %v", err)
		}
	}
}

func TestPutRejectsBadBytesAndReducedLimits(t *testing.T) {
	tr := reservePutTrial(t)
	tiny := uploadLimits()
	tiny.HostBytes = 1
	if w, _, err := tr.store.BeginPut(tr.id, tr.fp, 2, tiny, time.Now()); !errors.Is(err, ErrUploadCapacity) || w != nil {
		t.Fatal("lower host cap allowed new writes")
	}
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"short", tr.object[:20]},
		{"long", append(bytes.Clone(tr.object), 1)},
		{"wrong digest", bytes.Repeat([]byte{0xaa}, len(tr.object))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := tr.begin(t, 2)
			_, err := w.Write(tc.body)
			if err == nil {
				_, err = w.Finish()
			}
			if !errors.Is(err, ErrUploadInvalid) {
				t.Fatalf("invalid item returned %v", err)
			}
			if err := w.Abort(); err != nil {
				t.Fatal(err)
			}
			tr.expectMask(t, 0)
			if _, err := os.Lstat(filepath.Join(tr.dir, w.name)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid item left a linked slot", err)
			}
		})
	}
}

func TestPutUncertainMarkerRequiresReopen(t *testing.T) {
	tr := reservePutTrial(t)
	w := tr.begin(t, 2)
	if _, err := w.Write(tr.object); err != nil {
		t.Fatal(err)
	}
	// Inject a storage failure after file and directory sync, just before
	// SQLite could mark the slot. The peer must never see a successful PUT.
	if err := tr.store.db.exec("CREATE TRIGGER reject_stage BEFORE UPDATE OF mask ON upload_v1 WHEN NEW.mask!=OLD.mask BEGIN SELECT RAISE(ABORT,'synthetic write failure'); END"); err != nil {
		t.Fatal(err)
	}
	if state, err := w.Finish(); err == nil || state.Code != 0 {
		t.Fatal("failed staged marker acknowledged", state, err)
	}
	if _, err := tr.store.UploadStatus(tr.id, tr.fp, time.Now().Unix()); err == nil {
		t.Fatal("unreconciled store reported a state")
	}
	if err := tr.store.db.exec("DROP TRIGGER reject_stage"); err != nil {
		t.Fatal(err)
	}
	tr.reopen(t)
	tr.expectMask(t, 2)
}
