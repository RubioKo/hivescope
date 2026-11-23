// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCancelClosesActiveWriterAndRetainsTerminalID(t *testing.T) {
	tr := newFinalizeTrial(t)
	tr.stage(t, 2, tr.object)
	w := tr.begin(t, 1)
	wrong := tr.fp
	wrong[0] ^= 1
	if got, err := tr.store.CancelUpload(tr.id, wrong, time.Now().Unix()); !errors.Is(err, ErrUploadConflict) || got != (UploadState{}) {
		t.Fatal("wrong fingerprint", got, err)
	}
	if got, err := tr.store.CancelUpload([16]byte{0xff}, tr.fp, time.Now().Unix()); err != nil || got != (UploadState{}) {
		t.Fatal("unknown request", got, err)
	}
	if _, err := w.Write(tr.catalog[:125]); err != nil {
		t.Fatal("unrelated cancellation interrupted the writer", err)
	}
	if got, err := tr.store.CancelUpload(tr.id, tr.fp, time.Now().Unix()); err != nil || got != (UploadState{Code: 4}) {
		t.Fatal("cancel", got, err)
	}
	if n, err := w.Write(tr.catalog[125:]); !errors.Is(err, os.ErrClosed) || n != 0 {
		t.Fatal("cancelled writer accepted bytes", n, err)
	}
	if got, err := w.Finish(); !errors.Is(err, os.ErrClosed) || got != (UploadState{}) {
		t.Fatal("cancelled writer staged an item", got, err)
	}
	tr.reopen(t)
	want := UploadState{Code: 4}
	if got, err := tr.finalize(uploadLimits()); err != nil || got != want || tr.calls != 0 {
		t.Fatal("finalized a cancelled upload", got, err)
	}
	if got, err := tr.store.ReserveUpload(tr.m, uploadLimits(), time.Now().Unix(), tr.state.Deadline); err != nil || got != want {
		t.Fatal("reservation replay changed terminal state", got, err)
	}
	if next, got, err := tr.store.BeginPut(tr.id, tr.fp, 1, uploadLimits(), time.Now()); next != nil || err != nil || got != want {
		t.Fatal("PUT reopened a cancelled slot", got, err)
	}
	// Expiry cannot relabel an already cancelled ID, even after its old deadline.
	if got, err := tr.store.CancelUpload(tr.id, tr.fp, tr.state.Deadline); err != nil || got != want {
		t.Fatal("late replay changed cancellation", got, err)
	}
	if n, charge, err := tr.store.UploadUsage(); err != nil || n != 1 || charge != 0 {
		t.Fatal("terminal usage", n, charge, err)
	}
	for kind := byte(1); kind <= 2; kind++ {
		if _, err := os.Lstat(filepath.Join(tr.dir, uploadSlotName(1, kind))); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("cancelled slot remains", kind, err)
		}
	}
}

func TestCancelCompetesWithPublicationInsideSigner(t *testing.T) {
	tr := newFinalizeTrial(t)
	tr.stage(t, 1, tr.catalog)
	tr.stage(t, 2, tr.object)
	held, _, err := tr.store.UploadAccounting()
	if err != nil || held == 0 {
		t.Fatal("initial reservation", held, err)
	}
	// Once FINALIZE enters its signer it owns the request and has not yet
	// written the receipt. Start a concurrent CANCEL in this window.
	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	signing, cancelling := make(chan struct{}), make(chan struct{})
	type result struct {
		state UploadState
		err   error
	}
	done := make(chan result, 1)
	// Sample a later, still eligible provider time for the losing request.
	clock := tr.state.Deadline - 1
	go func() {
		select {
		case <-signing:
		case <-ctx.Done():
			done <- result{err: ctx.Err()}
			return
		}
		close(cancelling)
		state, err := tr.store.CancelUpload(tr.id, tr.fp, clock)
		done <- result{state, err}
	}()
	sign := func(body []byte) ([]byte, error) {
		close(signing)
		select {
		case <-cancelling:
			return tr.sign(body)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	published, err := tr.store.FinalizeUpload(tr.id, tr.fp, uploadLimits(), time.Now(), tr.state.Deadline, sign)
	if err != nil || published.Code != 2 {
		t.Fatal("publication failed", published, err)
	}
	select {
	case cancel := <-done:
		if cancel.err != nil || cancel.state != published {
			t.Fatal("racing cancel did not return original receipt", cancel.state, cancel.err)
		}
	case <-ctx.Done():
		t.Fatal("cancel did not finish")
	}
	if err := VerifyUploadReceipt(published.Receipt, [32]byte(tr.m[20:52]), tr.fp, tr.state.Deadline); err != nil {
		t.Fatal(err)
	}
	// Replacements can also encounter a previously committed, damaged pair.
	// A late CANCEL reports its true health and retains its historical charge.
	if err := os.Remove(filepath.Join(tr.dir, uploadSlotName(1, 2))); err != nil {
		t.Fatal(err)
	}
	missing := UploadState{Code: 5, Receipt: published.Receipt}
	if got, err := tr.store.CancelUpload(tr.id, tr.fp, clock); err != nil || got != missing {
		t.Fatal("cancel changed the unavailable pair", got, err)
	}
	if reserved, stored, err := tr.store.UploadAccounting(); err != nil || reserved != 0 || stored != held {
		t.Fatal("cancel changed the published charge", reserved, stored, err)
	}
}

func TestCancelRacesFinalizeFromSameStart(t *testing.T) {
	for i := range 4 {
		t.Run(fmt.Sprintf("reservation-%d", i+1), func(t *testing.T) {
			tr := newFinalizeTrial(t)
			tr.stage(t, 1, tr.catalog)
			tr.stage(t, 2, tr.object)
			charge, _, err := tr.store.UploadAccounting()
			if err != nil || charge == 0 {
				t.Fatal("initial reservation", charge, err)
			}
			now := time.Now().Truncate(time.Second)
			type result struct {
				state UploadState
				err   error
			}
			start, results := make(chan struct{}), make(chan result, 2)
			var ready sync.WaitGroup
			ready.Add(2)
			compete := func(op func() (UploadState, error)) {
				ready.Done()
				<-start
				state, err := op()
				results <- result{state, err}
			}
			go compete(func() (UploadState, error) {
				return tr.store.FinalizeUpload(tr.id, tr.fp, uploadLimits(), now, tr.state.Deadline, tr.sign)
			})
			go compete(func() (UploadState, error) {
				return tr.store.CancelUpload(tr.id, tr.fp, now.Unix())
			})
			ready.Wait()
			close(start)
			first, second := <-results, <-results
			if first.err != nil || second.err != nil || first.state != second.state {
				t.Fatal("competing requests returned different outcomes", first, second)
			}
			var wantStored uint64
			switch first.state.Code {
			case 2:
				wantStored = charge
				if err := VerifyUploadReceipt(first.state.Receipt, [32]byte(tr.m[20:52]), tr.fp, tr.state.Deadline); err != nil || tr.calls != 1 {
					t.Fatal("invalid publication outcome", err, tr.calls)
				}
			case 4:
				if first.state != (UploadState{Code: 4}) || tr.calls != 0 {
					t.Fatal("cancellation returned or signed a receipt")
				}
			default:
				t.Fatal("unexpected race outcome", first.state)
			}
			if reserved, stored, err := tr.store.UploadAccounting(); err != nil || reserved != 0 || stored != wantStored {
				t.Fatal("accounting diverges from the single outcome", reserved, stored, err)
			}
		})
	}
}

func TestCancelCleanupFailureKeepsChargeUntilReopen(t *testing.T) {
	for _, fault := range []struct {
		name, triggerCondition string
		cleanupStarted         bool
	}{
		{"record cancellation", "OLD.cancelled=0 AND NEW.cancelled=1", false},
		{"settle after unlink", "OLD.cleanup=1 AND NEW.cleanup=0", true},
	} {
		t.Run(fault.name, func(t *testing.T) {
			tr := newFinalizeTrial(t)
			tr.stage(t, 1, tr.catalog)
			tr.stage(t, 2, tr.object)
			held, _, err := tr.store.UploadAccounting()
			if err != nil || held == 0 {
				t.Fatal("initial reservation", held, err)
			}
			trigger := "CREATE TRIGGER interrupt_cancel BEFORE UPDATE ON upload_v1 WHEN " + fault.triggerCondition +
				" BEGIN SELECT RAISE(ABORT,'synthetic cancellation failure'); END"
			if err := tr.store.db.exec(trigger); err != nil {
				t.Fatal(err)
			}
			if got, err := tr.store.CancelUpload(tr.id, tr.fp, time.Now().Unix()); err == nil || got != (UploadState{}) {
				t.Fatal("acknowledged uncertain cancellation", got, err)
			}
			marker := 0
			if fault.cleanupStarted {
				marker = 1
			}
			if err := tr.store.db.expect(fmt.Sprintf("SELECT count(*) FROM upload_v1 WHERE cancelled=%d AND cleanup=%d AND charge=%d AND length(receipt)=0", marker, marker, held), "1"); err != nil {
				t.Fatal("lost the original charge or cleanup state", err)
			}
			for kind := byte(1); kind <= 2; kind++ {
				_, err := os.Lstat(filepath.Join(tr.dir, uploadSlotName(1, kind)))
				if fault.cleanupStarted && !errors.Is(err, os.ErrNotExist) || !fault.cleanupStarted && err != nil {
					t.Fatal("slot changed out of order", kind, err)
				}
			}
			if err := tr.store.db.exec("DROP TRIGGER interrupt_cancel"); err != nil {
				t.Fatal(err)
			}
			tr.reopen(t)
			if fault.cleanupStarted {
				if got, err := tr.finalize(uploadLimits()); err != nil || got != (UploadState{Code: 4}) || tr.calls != 0 {
					t.Fatal("cleanup lost cancellation", got, err)
				}
			} else {
				tr.expectMask(t, 3)
			}
			if got, err := tr.store.CancelUpload(tr.id, tr.fp, time.Now().Unix()); err != nil || got != (UploadState{Code: 4}) {
				t.Fatal("retry after failure", got, err)
			}
			if n, charged, err := tr.store.UploadUsage(); err != nil || n != 1 || charged != 0 {
				t.Fatal("failed to settle original charge", n, charged, err)
			}
		})
	}
}

func TestCancelMigrationPreservesVersionThreeOutcomes(t *testing.T) {
	tr := newFinalizeTrial(t)
	tr.stage(t, 1, tr.catalog)
	tr.stage(t, 2, tr.object)
	published, err := tr.finalize(uploadLimits())
	if err != nil || published.Code != 2 {
		t.Fatal("create historical fixture", published, err)
	}
	_, charge, err := tr.store.UploadAccounting()
	if err != nil || charge == 0 {
		t.Fatal("fixture charge", charge, err)
	}
	// Rebuild the ledger with the actual v3 DDL and one genuine publication.
	// Two additional rows represent a pending admission and a settled expiry.
	const fields = "slot,id,manifest,fingerprint,deadline,charge,expired,mask,cleanup,receipt"
	if err := tr.store.db.exec("BEGIN IMMEDIATE; ALTER TABLE upload_v1 RENAME TO fixture; " + uploadTable + "; " +
		stagingSchema + publicationSchema + " INSERT INTO upload_v1(" + fields + ") SELECT " + fields + " FROM fixture; DROP TABLE fixture; COMMIT"); err != nil {
		t.Fatal(err)
	}
	var refs [2][276]byte
	for i := range refs {
		m := testManifest(byte(i + 2))
		copy(m[20:116], tr.m[20:116])
		refs[i] = m
		deadline := tr.state.Deadline
		if i == 1 {
			deadline = time.Now().Unix() - 1
		}
		if err := tr.store.db.exec(fmt.Sprintf("INSERT INTO upload_v1(id,manifest,fingerprint,deadline,charge,expired) VALUES(X'%x',X'%x',X'%x',%d,%d,%d)",
			m[116:132], m, sha256.Sum256(m[:]), deadline, charge, i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tr.store.db.expect("PRAGMA user_version", "3"); err != nil {
		t.Fatal(err)
	}
	tr.reopen(t)
	now := time.Now().Unix()
	if reserved, stored, err := tr.store.UploadAccounting(); err != nil || reserved != charge || stored != charge {
		t.Fatal("migration changed charges", reserved, stored, err)
	}
	if got, err := tr.store.CancelUpload(tr.id, tr.fp, now); err != nil || got != published {
		t.Fatal("migration lost signed publication", got, err)
	}
	for i, m := range refs {
		id, fp := [16]byte(m[116:132]), sha256.Sum256(m[:])
		want, afterCancel := tr.state, UploadState{Code: 4}
		if i == 1 {
			want = UploadState{Code: 3}
			afterCancel = want
		}
		if got, err := tr.store.UploadStatus(id, fp, now); err != nil || got != want {
			t.Fatal("migration changed existing reservation", i, got, err)
		}
		if got, err := tr.store.CancelUpload(id, fp, now); err != nil || got != afterCancel {
			t.Fatal("cancel after migration", i, got, err)
		}
	}
}
