// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package conformance

import (
	"bytes"
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func TestCancelAndFinalizeKeepOneOutcome(t *testing.T) {
	for _, tc := range []struct {
		name         string
		publishFirst bool
	}{{"cancel first with lost reply", false}, {"finalize first", true}} {
		t.Run(tc.name, func(t *testing.T) {
			trial := newPublicationTrial(t)
			m, uploader, reads := trial.manifest, trial.uploader, trial.reads
			expiry := trial.agreement.Expiry.Unix()
			cancel := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte) (grantstore.UploadState, error) {
				return lan.CancelUpload(ctx, addr, pin, key, m, expiry)
			}
			finalize := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte) (grantstore.UploadState, error) {
				return lan.FinalizeUpload(ctx, addr, pin, key, m, expiry)
			}
			status := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte) (grantstore.UploadState, error) {
				return lan.UploadStatus(ctx, addr, pin, key, m, expiry)
			}
			empty := grantstore.UploadState{}
			if got, err := trial.exchange(t, cancel, uploader, m, false); err != nil || got != empty {
				t.Fatalf("CANCEL for an unknown request: %+v, %v", got, err)
			}
			if n, charge, err := trial.provider.UploadUsage(); err != nil || n != 0 || charge != 0 {
				t.Fatalf("CANCEL allocated an unknown request: n=%d charge=%d err=%v", n, charge, err)
			}
			pending, charge := trial.reserve(t)
			object, catalog := fiveByteSnapshot(t)
			trial.stage(t, 2, object, 2, pending.Deadline)
			trial.stage(t, 1, catalog, 3, pending.Deadline)
			checkStagedBytes(t, filepath.Join(trial.dir, "item-1-1"), catalog)
			checkStagedBytes(t, filepath.Join(trial.dir, "item-1-2"), object)

			// The replacement holds a valid read grant. Its own authenticated
			// uploader identity must not be able to cancel or discover the upload.
			other := m
			copy(other[52:84], reads.deviceKey[:])
			if got, err := trial.exchange(t, cancel, reads.device, other, false); !errors.Is(err, lan.ErrUploadDenied) || got != empty {
				t.Fatalf("unenrolled uploader: %+v, %v", got, err)
			}
			changed := m
			changed[172] ^= 1 // same scoped ID, different object digest
			if got, err := trial.exchange(t, cancel, uploader, changed, false); !errors.Is(err, grantstore.ErrUploadConflict) || got != empty {
				t.Fatalf("CANCEL with changed manifest: %+v, %v", got, err)
			}
			afterRefusal, err := trial.exchange(t, status, uploader, m, false)
			if err != nil || afterRefusal != (grantstore.UploadState{Code: 1, Mask: 3, Deadline: pending.Deadline}) {
				t.Fatalf("refused cancellation changed pending state: %+v, %v", afterRefusal, err)
			}

			want := grantstore.UploadState{Code: 4}
			var wantStored uint64
			loser := finalize
			if tc.publishFirst {
				want, err = trial.exchange(t, finalize, uploader, m, false)
				if err != nil || want.Code != 2 || want.Receipt == [132]byte{} {
					t.Fatalf("original FINALIZE: %+v, %v", want, err)
				}
				wantStored = charge
				loser = cancel
			} else {
				loseTerminalReply(t, trial.reservationTrial, cancel, uploader, m, 0)
			}
			if got, err := trial.exchange(t, loser, uploader, m, false); err != nil || got != want {
				t.Fatalf("second operation changed the original outcome: got %+v, want %+v, err=%v", got, want, err)
			}
			trial.reopen(t)
			for _, op := range []reservationCall{cancel, finalize, status, (lan.UploadTerms{Expiry: expiry}).Reserve, cancel} {
				if got, err := trial.exchange(t, op, uploader, m, false); err != nil || got != want {
					t.Fatalf("request did not retain the original outcome: got %+v, want %+v, err=%v", got, want, err)
				}
			}
			if n, charged, err := trial.provider.UploadUsage(); err != nil || n != 1 || charged != wantStored {
				t.Fatalf("terminal ID or charge changed: rows=%d charge=%d want=%d err=%v", n, charged, wantStored, err)
			}
			if reserved, stored, err := trial.provider.UploadAccounting(); err != nil || reserved != 0 || stored != wantStored {
				t.Fatalf("after retries: reserved=%d stored=%d wantStored=%d err=%v", reserved, stored, wantStored, err)
			}
			if tc.publishFirst {
				checkRecoveredFile(t, reads, []byte("Hive\n"))
				t.Logf("FINALIZE won: identical 132-byte receipt, %d stored bytes charged once, 5 recovered bytes", wantStored)
				return
			}
			for _, slot := range []string{"item-1-1", "item-1-2"} {
				if _, err := os.Lstat(filepath.Join(trial.dir, slot)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("cancelled slot %s remains: %v", slot, err)
				}
			}
			snapshot, err := storageformat.OpenCatalog(bytes.NewReader(catalog), reads.pkg.Catalog)
			if err != nil {
				t.Fatal(err)
			}
			reads.session(t, "read grant after cancelled upload", 2, func(ctx context.Context) error {
				assertOldReadsDenied(t, ctx, reads, snapshot)
				return nil
			})
			t.Log("CANCEL won: lost reply, restart, 1 retained ID, 0 charged bytes, 0 catalog bytes, 0 object bytes")
		})
	}
}
