// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func putIndexFixture() (m [276]byte, object, catalog []byte) {
	m = testManifest(1)
	object, catalog = bytes.Repeat([]byte{0xd1}, 21), bytes.Repeat([]byte{0xc1}, 250)
	od, cd := sha256.Sum256(object), sha256.Sum256(catalog)
	copy(m[172:204], od[:])
	copy(m[244:276], cd[:])
	return m, object, catalog
}

func TestReopenMigratesReservedUploadInPlace(t *testing.T) {
	dir, binding, s := newStore(t)
	m, _, _ := putIndexFixture()
	scope, id, fp := [96]byte(m[20:116]), [16]byte(m[116:132]), sha256.Sum256(m[:])
	// A store created by the reservation-only implementation, including its
	// original SQLite schema and one actual charge and deadline.
	volume, err := s.uploadVolume()
	if err != nil {
		t.Fatal(err)
	}
	charge := volume.commitment(21, 250)
	deadline := time.Now().Unix() + 120
	if err := s.db.exec(fmt.Sprintf(`CREATE TABLE upload_profile(id INTEGER PRIMARY KEY CHECK(id=1), scope BLOB NOT NULL CHECK(length(scope)=96));
		INSERT INTO upload_profile VALUES(1,X'%x'); %s;
		INSERT INTO upload_v1(id,manifest,fingerprint,deadline,charge) VALUES(X'%x',X'%x',X'%x',%d,%d)`,
		scope, uploadTable, id, m, fp, deadline, charge)); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err := reopened.CheckUploads(scope); err != nil {
		t.Fatal(err)
	}
	got, err := reopened.UploadStatus(id, fp, time.Now().Unix())
	if err != nil || got != (UploadState{Code: 1, Deadline: deadline}) {
		t.Fatalf("migrated reservation %+v, %v", got, err)
	}
	if n, used, err := reopened.UploadUsage(); err != nil || n != 1 || used != charge {
		t.Fatalf("migration changed capacity: %d %d %v", n, used, err)
	}
}

func TestReconcileOnlyAffectedPutSlot(t *testing.T) {
	for _, scenario := range []string{"missing catalog", "changed catalog", "unmarked complete pair"} {
		t.Run(scenario, func(t *testing.T) {
			dir, binding, s := newStore(t)
			m, object, catalog := putIndexFixture()
			scope, id, fp := [96]byte(m[20:116]), [16]byte(m[116:132]), sha256.Sum256(m[:])
			if err := s.InitUploads(scope); err != nil {
				t.Fatal(err)
			}
			now := time.Now().Unix()
			pending, err := s.ReserveUpload(m, uploadLimits(), now, now+300)
			if err != nil {
				t.Fatal(err)
			}
			objPath := filepath.Join(dir, uploadSlotName(1, 2))
			catPath := filepath.Join(dir, uploadSlotName(1, 1))
			if err := os.WriteFile(objPath, object, 0600); err != nil {
				t.Fatal(err)
			}
			wantMask := byte(2)
			switch scenario {
			case "unmarked complete pair":
				if err := os.WriteFile(catPath, catalog, 0600); err != nil {
					t.Fatal(err)
				}
				wantMask = 3
			default:
				if err := s.db.exec("UPDATE upload_v1 SET mask=3"); err != nil {
					t.Fatal(err)
				}
				if scenario == "changed catalog" {
					catalog[0] ^= 1
					if err := os.WriteFile(catPath, catalog, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err = Open(dir, binding)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.CheckUploads(scope); err != nil {
				t.Fatal(err)
			}
			pending.Mask = wantMask
			if got, err := s.UploadStatus(id, fp, time.Now().Unix()); err != nil || got != pending {
				t.Fatalf("reconciled state %+v %v", got, err)
			}
			if got, err := os.ReadFile(objPath); err != nil || !bytes.Equal(got, object) {
				t.Fatalf("object changed: %v", err)
			}
			if wantMask == 2 {
				if _, err := os.Lstat(catPath); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("damaged catalog still occupies its slot", err)
				}
			}
		})
	}
}

func TestPutExpirySettlesLinkedAndUnnamedSpace(t *testing.T) {
	dir, _, s := newStore(t)
	m, object, catalog := putIndexFixture()
	scope, id, fp := [96]byte(m[20:116]), [16]byte(m[116:132]), sha256.Sum256(m[:])
	if err := s.InitUploads(scope); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	state, err := s.ReserveUpload(m, uploadLimits(), now.Unix(), now.Unix()+300)
	if err != nil {
		t.Fatal(err)
	}
	obj, _, err := s.BeginPut(id, fp, 2, uploadLimits(), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := obj.Write(object); err != nil {
		t.Fatal(err)
	}
	if _, err := obj.Finish(); err != nil {
		t.Fatal(err)
	}
	cat, _, err := s.BeginPut(id, fp, 1, uploadLimits(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cat.Write(catalog[:125]); err != nil {
		t.Fatal(err)
	}
	// No double counting of blocks already allocated inside this reservation.
	vol, err := s.uploadVolume()
	if err != nil {
		t.Fatal(err)
	}
	used, err := s.uploadAllocations(vol)
	if err != nil || used.itemInodes != 2 || used.items == 0 {
		t.Fatal("unexpected allocated slots", used, err)
	}
	page, err := s.uploadNumber("PRAGMA page_size")
	if err != nil {
		t.Fatal(err)
	}
	_, charge, err := s.UploadUsage()
	if err != nil {
		t.Fatal(err)
	}
	vol.freeInodes = 2 // journal and cleanup only, both slot inodes already exist
	vol.available = vol.metadataAllowance(page) - used.metadata + charge - used.items
	if err := s.checkUploadCapacity(uploadLimits(), vol, 0, 0); err != nil {
		t.Fatalf("previously allocated blocks counted twice: %v", err)
	}
	vol.available--
	if err := s.checkUploadCapacity(uploadLimits(), vol, 0, 0); !errors.Is(err, ErrUploadCapacity) {
		t.Fatal("admitted over real capacity", err)
	}
	if got, err := s.UploadStatus(id, fp, state.Deadline); err != nil || got != (UploadState{Code: 3}) {
		t.Fatalf("expiry %+v %v", got, err)
	}
	if _, err := cat.Write(catalog[125:]); !errors.Is(err, os.ErrClosed) {
		t.Fatal("expired writer continued", err)
	}
	if _, err := os.Lstat(filepath.Join(dir, uploadSlotName(1, 2))); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expired object still linked", err)
	}
	if n, used, err := s.UploadUsage(); err != nil || n != 1 || used != 0 {
		t.Fatal("cleanup did not settle capacity", n, used, err)
	}
}

func TestInterruptedExpiryRetainsChargeUntilDurableCleanup(t *testing.T) {
	dir, binding, s := newStore(t)
	m, object, _ := putIndexFixture()
	scope, id, fp := [96]byte(m[20:116]), [16]byte(m[116:132]), sha256.Sum256(m[:])
	if err := s.InitUploads(scope); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if _, err := s.ReserveUpload(m, uploadLimits(), now, now+300); err != nil {
		t.Fatal(err)
	}
	_, originalCharge, err := s.UploadUsage()
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a failure unlinking the first slot. The provider must preserve
	// the foreign entry, commit expiry first and keep the charge.
	catPath := filepath.Join(dir, uploadSlotName(1, 1))
	objPath := filepath.Join(dir, uploadSlotName(1, 2))
	if err := os.WriteFile(objPath, object, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(objPath, catPath); err != nil {
		t.Fatal(err)
	}
	if err := s.db.exec(fmt.Sprintf("UPDATE upload_v1 SET deadline=%d", now)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UploadStatus(id, fp, now); err == nil {
		t.Fatal("unsafe cleanup was reported as complete")
	}
	if err := s.db.expect(fmt.Sprintf("SELECT count(*) FROM upload_v1 WHERE expired=1 AND cleanup=1 AND charge=%d", originalCharge), "1"); err != nil {
		t.Fatal("terminal state and charge missing", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Readlink(catPath); err != nil {
		t.Fatal("unknown link was removed", err)
	}
	// The operator clears the offending link. Reopen can finish the durable
	// cleanup of the owned object and settle the original expired ID once.
	if err := os.Remove(catPath); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.CheckUploads(scope); err != nil {
		t.Fatal(err)
	}
	if n, charge, err := s.UploadUsage(); err != nil || n != 1 || charge != 0 {
		t.Fatal("cleanup charge after reopen", n, charge, err)
	}
	if _, err := os.Stat(objPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("expired object still occupies the slot", err)
	}
}
