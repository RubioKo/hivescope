// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testManifest(seed byte) (m [276]byte) {
	copy(m[:20], "HiveScope/upload/v1\x00")
	m[20], m[52], m[84], m[100] = 1, 2, 3, 4
	m[116], m[132], m[204] = seed, 2*seed, 2*seed+1
	binary.BigEndian.PutUint64(m[164:172], 21)
	binary.BigEndian.PutUint64(m[236:244], 250)
	return m
}

func uploadLimits() UploadLimits {
	return UploadLimits{HostBytes: 16 << 20, AgreementBytes: 16 << 20, RequesterBytes: 16 << 20, MaxRequests: uploadCeiling}
}

func TestUploadAdmissionSurvivesReopen(t *testing.T) {
	dir, binding, s := newStore(t)
	m := testManifest(1)
	scope := [96]byte(m[20:116])
	if err := s.InitUploads(scope); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	state, err := s.ReserveUpload(m, uploadLimits(), now, now+1200)
	if err != nil || state != (UploadState{Code: 1, Deadline: now + 900}) {
		t.Fatalf("first admission: %+v %v", state, err)
	}
	_, charge, err := s.UploadUsage()
	if err != nil || charge <= 271 {
		t.Fatalf("missing physical charge: %d %v", charge, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.InitUploads(scope); err == nil {
		t.Fatal("existing ledger was reinitialized")
	}
	if err := s.CheckUploads(scope); err != nil {
		t.Fatal(err)
	}
	l := UploadLimits{Paused: true} // all caps reduced to zero
	got, err := s.ReserveUpload(m, l, now, now+10)
	if err != nil || got != state {
		t.Fatalf("retry with a paused, full host altered the outcome: %+v %v", got, err)
	}
	id := [16]byte(m[116:132])
	if got, err := s.UploadStatus(id, sha256.Sum256(m[:]), now); err != nil || got != state {
		t.Fatalf("restarted status: %+v %v", got, err)
	}
	if got, err := s.UploadStatus([16]byte{0xff}, [32]byte{}, now); err != nil || got.Code != 0 {
		t.Fatal("unknown status allocated space")
	}
	other := m
	other[172] ^= 1
	if _, err := s.ReserveUpload(other, l, now, now+900); !errors.Is(err, ErrUploadConflict) {
		t.Fatal("changed manifest did not conflict", err)
	}
	if _, err := s.UploadStatus(id, sha256.Sum256(other[:]), now); !errors.Is(err, ErrUploadConflict) {
		t.Fatal("wrong fingerprint disclosed status", err)
	}
	fresh := testManifest(2)
	if _, err := s.ReserveUpload(fresh, l, now, now+900); !errors.Is(err, ErrUploadPaused) {
		t.Fatal("paused provider accepted a new ID", err)
	}
	fresh[132] = m[204]
	if _, err := s.ReserveUpload(fresh, uploadLimits(), now, now+900); !errors.Is(err, ErrUploadConflict) {
		t.Fatal("accepted a cross-role locator collision", err)
	}
	if count, again, err := s.UploadUsage(); err != nil || count != 1 || again != charge {
		t.Fatalf("retries and status changed usage: %d %d %v", count, again, err)
	}
}

func TestUploadRaceForLastPhysicalCapacity(t *testing.T) {
	_, _, s := newStore(t)
	m := testManifest(1)
	if err := s.InitUploads([96]byte(m[20:116])); err != nil {
		t.Fatal(err)
	}
	volume, err := s.uploadVolume()
	if err != nil {
		t.Fatal(err)
	}
	page, err := s.uploadNumber("PRAGMA page_size")
	if err != nil {
		t.Fatal(err)
	}
	policy := uploadLimits()
	charge := volume.commitment(21, 250)
	policy.HostBytes = volume.metadataAllowance(page) + charge
	if volume.unit >= 250 && charge != 4*volume.unit {
		t.Fatalf("incorrect pair allocation: %d with block %d", charge, volume.unit)
	}
	now := time.Now().Unix()
	start := make(chan struct{})
	results := make(chan error, 2)
	for seed := byte(1); seed <= 2; seed++ {
		go func() {
			<-start
			_, err := s.ReserveUpload(testManifest(seed), policy, now, now+120)
			results <- err
		}()
	}
	close(start)
	first, second := <-results, <-results
	if !((first == nil && errors.Is(second, ErrUploadCapacity)) || (second == nil && errors.Is(first, ErrUploadCapacity))) {
		t.Fatalf("last physical slot results: %v %v", first, second)
	}
	if count, got, err := s.UploadUsage(); err != nil || count != 1 || got != charge {
		t.Fatalf("last slot usage: %d %d %v", count, got, err)
	}
}

func TestUploadExpiryAndLimits(t *testing.T) {
	_, _, s := newStore(t)
	m := testManifest(1)
	if err := s.InitUploads([96]byte(m[20:116])); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	for _, field := range []string{"host", "agreement", "requester", "margin", "retained IDs"} {
		l := uploadLimits()
		switch field {
		case "host":
			l.HostBytes = 0
		case "agreement":
			l.AgreementBytes = 0
		case "requester":
			l.RequesterBytes = 0
		case "margin":
			l.FreeMargin = math.MaxUint64
		case "retained IDs":
			l.MaxRequests = 0
		}
		if _, err := s.ReserveUpload(m, l, now, now+1); !errors.Is(err, ErrUploadCapacity) {
			t.Fatalf("%s did not reject admission: %v", field, err)
		}
	}
	malformed := m
	copy(malformed[204:236], m[132:164])
	if _, err := s.ReserveUpload(malformed, uploadLimits(), now, now+1); err == nil {
		t.Fatal("accepted repeated locator")
	}
	wrongScope := m
	wrongScope[52]++
	if _, err := s.ReserveUpload(wrongScope, uploadLimits(), now, now+1); !errors.Is(err, ErrUploadDenied) {
		t.Fatal("accepted wrong uploader")
	}
	state, err := s.ReserveUpload(m, uploadLimits(), now, now+1)
	if err != nil || state.Deadline != now+1 {
		t.Fatal("failed to clip deadline to agreement expiry", err)
	}
	id := [16]byte(m[116:132])
	if got, err := s.UploadStatus(id, sha256.Sum256(m[:]), now+1); err != nil || got != (UploadState{Code: 3}) {
		t.Fatal("exact deadline did not expire the pending slots", err)
	}
	if got, err := s.ReserveUpload(m, uploadLimits(), now+1, now+1200); err != nil || got.Code != 3 {
		t.Fatal("identical retry resurrected expired slots", err)
	}
	if _, err := s.UploadStatus(id, sha256.Sum256(m[:]), now); err == nil {
		t.Fatal("status accepted time behind persisted high water")
	}
	if count, charged, err := s.UploadUsage(); err != nil || count != 1 || charged != 0 {
		t.Fatal("expired unstaged slots failed to release space or forgot the terminal ID", count, charged, err)
	}
	l := uploadLimits()
	l.MaxRequests = 1
	if _, err := s.ReserveUpload(testManifest(2), l, now+1, now+100); !errors.Is(err, ErrUploadCapacity) {
		t.Fatal("terminal ID escaped the retention ceiling", err)
	}
}

func TestUploadReopenRequiresKnownLedger(t *testing.T) {
	scope := [96]byte{1}
	for _, kind := range []string{"missing table", "inconsistent charge", "wrong profile"} {
		t.Run(kind, func(t *testing.T) {
			dir, binding, s := newStore(t)
			m := testManifest(1)
			testScope := [96]byte(m[20:116])
			if kind != "missing table" {
				if err := s.InitUploads(testScope); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "inconsistent charge" {
				now := time.Now().Unix()
				if _, err := s.ReserveUpload(m, uploadLimits(), now, now+900); err != nil {
					t.Fatal(err)
				}
				if err := s.db.exec("UPDATE upload_v1 SET charge=1"); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "wrong profile" {
				testScope = scope
			}
			if err := s.Close(); err != nil {
				t.Fatal(err)
			}
			s, err := Open(dir, binding)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			if err := s.CheckUploads(testScope); err == nil {
				t.Fatal("enabled inconsistent ledger")
			}
			if err := s.InitUploads(testScope); err == nil {
				t.Fatal("recreated existing ledger")
			}
		})
	}
}

func TestUploadUnexpectedEntryBlocksAdmission(t *testing.T) {
	dir, _, s := newStore(t)
	m := testManifest(1)
	if err := s.InitUploads([96]byte(m[20:116])); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "unknown"), []byte{1}, 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if _, err := s.ReserveUpload(m, uploadLimits(), now, now+900); err == nil {
		t.Fatal("allocated new space alongside unknown files")
	}
}

func TestUploadFailedWriteRequiresReopen(t *testing.T) {
	dir, binding, s := newStore(t)
	m := testManifest(1)
	scope := [96]byte(m[20:116])
	if err := s.InitUploads(scope); err != nil {
		t.Fatal(err)
	}
	if err := s.Replace(nil, recordAt(7)); err == nil {
		t.Fatal("read grant installed in an upload enrollment")
	}
	if err := s.db.exec("PRAGMA query_only=ON"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if state, err := s.ReserveUpload(m, uploadLimits(), now, now+900); err == nil || state.Code != 0 {
		t.Fatalf("failed transaction reported admission: %+v %v", state, err)
	}
	if _, err := s.UploadStatus([16]byte(m[116:132]), sha256.Sum256(m[:]), now); err == nil {
		t.Fatal("unreconciled failure reported a definitive outcome")
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
	if n, charge, err := reopened.UploadUsage(); err != nil || n != 0 || charge != 0 {
		t.Fatal("failed transaction left a partial commitment", n, charge, err)
	}
	if state, err := reopened.ReserveUpload(m, uploadLimits(), now, now+900); err != nil || state.Code != 1 {
		t.Fatal("could not retry the original manifest after reopening", err)
	}
}

func TestUploadHardLedgerBoundFitsSQLiteBudget(t *testing.T) {
	_, _, s := newStore(t)
	m := testManifest(1)
	if err := s.InitUploads([96]byte(m[20:116])); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	limits := uploadLimits()
	limits.MaxRequests = 4096
	for i := 1; i <= uploadCeiling; i++ {
		if _, err := s.ReserveUpload(testManifest(byte(i)), limits, now, now+900); err != nil {
			t.Fatalf("retained ID %d did not fit the trial index: %v", i, err)
		}
	}
	if _, err := s.ReserveUpload(testManifest(uploadCeiling+1), limits, now, now+900); !errors.Is(err, ErrUploadCapacity) {
		t.Fatal("hard retained-ID ceiling was not enforced", err)
	}
}
