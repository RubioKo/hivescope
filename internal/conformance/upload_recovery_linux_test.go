// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package conformance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func TestUploadAndRecoverBoundaryFiles(t *testing.T) {
	for _, tc := range []struct {
		fixture                 snapshotFixture
		plainBytes, sealedBytes int
	}{
		{emptyFixture(), 0, 16},
		{edgeFixture(), 4_194_305, 4_194_337},
	} {
		t.Run(tc.fixture.name, func(t *testing.T) {
			// Keep the observer's reference separate from the uploader's source.
			want := tc.fixture.plaintext
			ownerDir := t.TempDir()
			source := filepath.Join(ownerDir, tc.fixture.name)
			if err := os.WriteFile(source, want, 0600); err != nil {
				t.Fatal(err)
			}
			f := tc.fixture
			var err error
			f.plaintext, err = os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			object := f.sealObject(t)
			catalog := f.sealCatalog(t, f.entry(object))
			if len(f.plaintext) != tc.plainBytes || len(object) != tc.sealedBytes {
				t.Fatalf("unexpected boundary fixture: %d plaintext bytes, %d sealed bytes", len(f.plaintext), len(object))
			}
			trial := newSnapshotPublicationTrial(t, f, object, catalog)
			if n, charged, err := trial.provider.UploadUsage(); err != nil || n != 0 || charged != 0 {
				t.Fatalf("expected an empty provider: rows=%d charge=%d err=%v", n, charged, err)
			}
			pending, charge := trial.reserve(t)
			trial.stage(t, lan.UploadObject, object, lan.UploadObject, pending.Deadline)
			trial.stage(t, lan.UploadCatalog, catalog, lan.UploadObject|lan.UploadCatalog, pending.Deadline)

			expiry := trial.agreement.Expiry.Unix()
			confirm := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte) (grantstore.UploadState, error) {
				return lan.ConfirmUpload(ctx, addr, pin, key, m, expiry, ownerDir)
			}
			committed, err := trial.exchange(t, confirm, trial.uploader, trial.manifest, false)
			if err != nil || committed.Code != 2 {
				t.Fatalf("confirm: state=%d err=%v", committed.Code, err)
			}
			fingerprint := sha256.Sum256(trial.manifest[:])
			if err := grantstore.VerifyUploadReceipt(committed.Receipt, trial.pin, fingerprint, expiry); err != nil {
				t.Fatal(err)
			}
			receiptPath := filepath.Join(ownerDir, fmt.Sprintf("%x.receipt", fingerprint))
			requireFileBytes(t, receiptPath, committed.Receipt[:])

			const passphrase = "HiveScope boundary trial"
			backup := filepath.Join(t.TempDir(), "snapshot.hsrp")
			if err := os.WriteFile(backup, trial.recoveryExport(t, passphrase), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(source); err != nil {
				t.Fatal(err)
			}
			trial.reopen(t)
			status := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte) (grantstore.UploadState, error) {
				return lan.UploadStatus(ctx, addr, pin, key, m, expiry)
			}
			if got, err := trial.exchange(t, status, trial.uploader, trial.manifest, false); err != nil || got != committed {
				t.Fatalf("reopened provider did not return the original committed receipt: state=%d err=%v", got.Code, err)
			}
			if reserved, stored, err := trial.provider.UploadAccounting(); err != nil || reserved != 0 || stored != charge {
				t.Fatalf("reopened provider: reserved=%d stored=%d original=%d err=%v", reserved, stored, charge, err)
			}

			destination := filepath.Join(t.TempDir(), "selected.bin")
			stateDir := t.TempDir()
			export, err := os.ReadFile(backup)
			if err != nil {
				t.Fatal(err)
			}
			trial.reads.session(t, "restore the uploaded file from the encrypted export", 2, func(ctx context.Context) error {
				return storageformat.WithRecoveryPackage(bytes.NewReader(export), []byte(passphrase), func(pkg *storageformat.RecoveryPackage) error {
					return lan.RecoverFileTo(ctx, pkg, trial.reads.device, destination, false, stateDir)
				})
			})
			requireFileBytes(t, destination, want)
			t.Logf("confirmed and reopened: %d sealed object bytes, %d signed catalog bytes, original 132-byte receipt, %d restored plaintext bytes", len(object), len(catalog), len(want))
		})
	}
}
