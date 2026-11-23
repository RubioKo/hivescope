// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type finalizeTrial struct {
	*putTrial
	key   ed25519.PrivateKey
	calls int
}

func newFinalizeTrial(t *testing.T) *finalizeTrial {
	t.Helper()
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &finalizeTrial{putTrial: reservePutTrialForProvider(t, public), key: key}
}

func (tr *finalizeTrial) sign(message []byte) ([]byte, error) {
	tr.calls++
	return ed25519.Sign(tr.key, message), nil
}

func (tr *finalizeTrial) stage(t *testing.T, kind byte, data []byte) {
	t.Helper()
	w := tr.begin(t, kind)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Finish(); err != nil {
		t.Fatal(err)
	}
}

func (tr *finalizeTrial) finalize(limits UploadLimits) (UploadState, error) {
	return tr.store.FinalizeUpload(tr.id, tr.fp, limits, time.Now(), tr.state.Deadline, tr.sign)
}

func TestFinalizeRequiresBothVerifiedFilesAndQuota(t *testing.T) {
	tr := newFinalizeTrial(t)
	tr.stage(t, 2, tr.object)
	if got, err := tr.finalize(uploadLimits()); err != nil || got.Code != 1 || got.Mask != 2 || tr.calls != 0 {
		t.Fatal("premature finalize", got, err)
	}
	if f, _, err := tr.store.OpenPublishedItem(2, [32]byte(tr.m[132:164]), time.Now().Unix()); f != nil || !errors.Is(err, ErrUploadDenied) {
		t.Fatal("pending object has membership", err)
	}
	tr.stage(t, 1, tr.catalog)
	if got, err := tr.finalize(UploadLimits{}); !errors.Is(err, ErrUploadCapacity) || got.Code != 0 || tr.calls != 0 {
		t.Fatal("capacity recheck", got, err)
	}
	if err := os.WriteFile(filepath.Join(tr.dir, uploadSlotName(1, 1)), bytes.Repeat([]byte{0}, len(tr.catalog)), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := tr.finalize(uploadLimits()); err != nil || got.Code != 1 || got.Mask != 2 || tr.calls != 0 {
		t.Fatal("damaged staged catalog", got, err)
	}
	tr.stage(t, 1, tr.catalog)
	if got, err := tr.finalize(uploadLimits()); err != nil || got.Code != 2 || tr.calls != 1 {
		t.Fatal("replacement failed to finalize", got, err)
	}
}

func TestConcurrentFinalizeAndMissingCommittedMember(t *testing.T) {
	tr := newFinalizeTrial(t)
	tr.stage(t, 2, tr.object)
	tr.stage(t, 1, tr.catalog)
	charge, stored, err := tr.store.UploadAccounting()
	if err != nil || charge == 0 || stored != 0 {
		t.Fatal("pending accounting", charge, stored, err)
	}
	type outcome struct {
		state UploadState
		err   error
	}
	results := make(chan outcome, 2)
	for range 2 {
		go func() { got, err := tr.finalize(uploadLimits()); results <- outcome{got, err} }()
	}
	first, second := <-results, <-results
	if first.err != nil || second.err != nil || first.state.Code != 2 || first.state != second.state || tr.calls != 1 {
		t.Fatal("duplicate finalization", first, second, tr.calls)
	}
	if err := VerifyUploadReceipt(first.state.Receipt, [32]byte(tr.m[20:52]), tr.fp, tr.state.Deadline); err != nil {
		t.Fatal(err)
	}
	checkAccount := func() {
		t.Helper()
		if held, used, err := tr.store.UploadAccounting(); err != nil || held != 0 || used != charge {
			t.Fatal("published accounting", held, used, err)
		}
	}
	checkAccount()
	tr.reopen(t)
	clock := time.Now().Unix()
	reader, size, err := tr.store.OpenPublishedItem(1, [32]byte(tr.m[204:236]), clock)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err := errors.Join(err, reader.Close()); err != nil || size != uint64(len(data)) || !bytes.Equal(data, tr.catalog) {
		t.Fatal("verified catalog bytes", size, err)
	}
	objectPath := filepath.Join(tr.dir, uploadSlotName(1, 2))
	if err := os.Remove(objectPath); err != nil {
		t.Fatal(err)
	}
	tr.reopen(t)
	if f, _, err := tr.store.OpenPublishedItem(1, [32]byte(tr.m[204:236]), time.Now().Unix()); f != nil || !errors.Is(err, ErrPublicationUnavailable) {
		t.Fatal("healthy sibling exposed despite missing object", err)
	}
	if got, err := tr.finalize(UploadLimits{}); err != nil || got.Code != 5 || got.Receipt != first.state.Receipt || tr.calls != 1 {
		t.Fatal("lost historical receipt after damage", got, err)
	}
	checkAccount()
	if err := os.WriteFile(objectPath, tr.object, 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := tr.store.UploadStatus(tr.id, tr.fp, tr.state.Deadline+1); err != nil || got != first.state {
		t.Fatal("restored pair after reservation deadline", got, err)
	}
	checkAccount()
}

func TestFinalizeStorageFailureKeepsPendingPair(t *testing.T) {
	tr := newFinalizeTrial(t)
	tr.stage(t, 1, tr.catalog)
	tr.stage(t, 2, tr.object)
	if err := tr.store.db.exec("CREATE TRIGGER reject_publication BEFORE UPDATE OF receipt ON upload_v1 BEGIN SELECT RAISE(ABORT,'synthetic receipt failure'); END"); err != nil {
		t.Fatal(err)
	}
	if got, err := tr.finalize(uploadLimits()); err == nil || got != (UploadState{}) {
		t.Fatal("acknowledged failed publication", got, err)
	}
	if err := tr.store.db.exec("DROP TRIGGER reject_publication"); err != nil {
		t.Fatal(err)
	}
	tr.reopen(t)
	tr.expectMask(t, 3)
	if held, stored, err := tr.store.UploadAccounting(); err != nil || held == 0 || stored != 0 {
		t.Fatal("failed receipt changed charge", held, stored, err)
	}
	if got, err := tr.finalize(uploadLimits()); err != nil || got.Code != 2 {
		t.Fatal("retry after failed publication", got, err)
	}
}

func TestFinalizeRejectsWrongSigningKeyAndTamperedReceipt(t *testing.T) {
	tr := newFinalizeTrial(t)
	tr.stage(t, 1, tr.catalog)
	tr.stage(t, 2, tr.object)
	_, other, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	wrongSigner := func(message []byte) ([]byte, error) { return ed25519.Sign(other, message), nil }
	got, err := tr.store.FinalizeUpload(tr.id, tr.fp, uploadLimits(), time.Now(), tr.state.Deadline, wrongSigner)
	if err == nil || got != (UploadState{}) {
		t.Fatal("accepted wrong receipt signer", got, err)
	}
	tr.reopen(t)
	tr.expectMask(t, 3)
	got, err = tr.finalize(uploadLimits())
	if err != nil || got.Code != 2 {
		t.Fatal(got, err)
	}
	bad := got.Receipt
	bad[len(bad)-1] ^= 1
	if err := tr.store.db.exec(fmt.Sprintf("UPDATE upload_v1 SET receipt=X'%x'", bad)); err != nil {
		t.Fatal(err)
	}
	if err := tr.store.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(tr.dir, tr.binding)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CheckUploads([96]byte(tr.m[20:116])); err == nil {
		t.Fatal("reopened with invalid historical receipt")
	}
	if got, err := s.UploadStatus(tr.id, tr.fp, time.Now().Unix()); err == nil || got != (UploadState{}) {
		t.Fatal("served unreconciled publication", got, err)
	}
	if err := s.db.expect(fmt.Sprintf("SELECT count(*) FROM upload_v1 WHERE receipt=X'%x' AND expired=0 AND mask=3", bad), "1"); err != nil {
		t.Fatal("reconciliation destroyed historical state", err)
	}
}
