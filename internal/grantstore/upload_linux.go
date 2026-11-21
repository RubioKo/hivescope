// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

// This lower trial ceiling leaves room for every retained manifest and index
// entry within the 32-page SQLite budget, including terminal IDs.
const uploadCeiling = 64

// Each retained row owns two slots, identified locally by its slot number and
// role. No payload file exists before PUT. Both locators live in the manifest,
// so their ownership and charge become visible in the same SQLite commit.
const uploadTable = `CREATE TABLE upload_v1(
	slot INTEGER PRIMARY KEY,
	id BLOB NOT NULL UNIQUE CHECK(typeof(id)='blob' AND id=substr(manifest,117,16)),
	manifest BLOB NOT NULL CHECK(typeof(manifest)='blob' AND length(manifest)=276),
	fingerprint BLOB NOT NULL CHECK(typeof(fingerprint)='blob' AND length(fingerprint)=32),
	deadline INTEGER NOT NULL CHECK(typeof(deadline)='integer' AND deadline>0),
	charge INTEGER NOT NULL CHECK(typeof(charge)='integer' AND charge>0),
	expired INTEGER NOT NULL DEFAULT 0 CHECK(expired IN (0,1))
)`

// Version 1 reserved the same two slots without writing payloads. Preserve
// its exact rows, original deadlines and charges when enabling staging.
const stagingSchema = `ALTER TABLE upload_v1 ADD COLUMN mask INTEGER NOT NULL DEFAULT 0 CHECK(mask BETWEEN 0 AND 3);
	ALTER TABLE upload_v1 ADD COLUMN cleanup INTEGER NOT NULL DEFAULT 0 CHECK(cleanup IN (0,1) AND (cleanup=0 OR expired=1));
	PRAGMA user_version=2;`

// InitUploads is an explicit provisioning step on a new dedicated store. A
// missing upload table on an existing store must never look like no uploads.
func (s *Store) InitUploads(scope [96]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.fresh || s.db == nil || s.fault != nil || len(s.record) != 0 || s.pending != nil {
		return errors.New("upload enrollment requires a new empty store")
	}
	s.fresh = false
	s.fault = s.db.exec(fmt.Sprintf("BEGIN IMMEDIATE; CREATE TABLE upload_profile(id INTEGER PRIMARY KEY CHECK(id=1), scope BLOB NOT NULL CHECK(length(scope)=96)); INSERT INTO upload_profile VALUES(1,X'%x'); %s; %s COMMIT", scope, uploadTable, stagingSchema))
	s.uploadScope = scope
	s.uploads = s.fault == nil
	return s.fault
}

// CheckUploads reconciles the retained manifests, their accounting and live
// locator ownership before allowing service. The bounded scan never creates
// tables or rows on the reopen path.
func (s *Store) CheckUploads(scope [96]byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil || s.fault != nil || len(s.record) != 0 || s.pending != nil {
		return errors.New("cannot reopen upload store")
	}
	if err := s.checkBacking(time.Now().Unix()); err != nil {
		return err
	}
	s.uploadScope = scope
	s.fault = s.db.expect(fmt.Sprintf("SELECT count(*) FROM upload_profile WHERE id=1 AND scope=X'%x'", scope), "1")
	if s.fault == nil {
		s.fault = s.checkUploadRows()
	}
	if s.fault == nil {
		s.fault = s.reopenUploadSlots()
	}
	s.uploads = s.fault == nil
	return s.fault
}

func (s *Store) checkUploadRows() error {
	count, err := s.uploadNumber("SELECT count(*) FROM upload_v1")
	if err != nil || count > uploadCeiling {
		return errors.Join(errors.New("invalid or missing upload ledger"), err)
	}
	volume, err := s.uploadVolume()
	if err != nil {
		return err
	}
	live := make(map[[32]byte]bool)
	for i := uint64(0); i < count; i++ {
		data, err := s.db.scalar(fmt.Sprintf("SELECT manifest FROM upload_v1 ORDER BY slot LIMIT 1 OFFSET %d", i))
		if err != nil || len(data) != 276 {
			return errors.Join(errors.New("invalid persisted upload manifest"), err)
		}
		m := [276]byte(data)
		if [96]byte(m[20:116]) != s.uploadScope {
			return errors.New("persisted upload has a different enrollment")
		}
		obj, cat, err := UploadItemSizes(m)
		if err != nil {
			return err
		}
		charge := volume.commitment(obj, cat)
		id := [16]byte(m[116:132])
		fingerprint := sha256.Sum256(data)
		if err := s.db.expect(fmt.Sprintf("SELECT count(*) FROM upload_v1 WHERE id=X'%x' AND fingerprint=X'%x' AND charge=%d", id, fingerprint, charge), "1"); err != nil {
			return err
		}
		deadline, err := s.uploadNumber(fmt.Sprintf("SELECT CASE WHEN expired THEN 0 ELSE deadline END FROM upload_v1 WHERE id=X'%x'", id))
		if err != nil {
			return err
		}
		if deadline != 0 {
			for _, locator := range [][32]byte{[32]byte(m[132:164]), [32]byte(m[204:236])} {
				if live[locator] {
					return errors.New("two pending slots own the same locator")
				}
				live[locator] = true
			}
		}
	}
	return nil
}

func (s *Store) uploadNumber(query string) (uint64, error) {
	data, err := s.db.scalar(query)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(data), 10, 63)
}

func (s *Store) uploadReady(now int64) error {
	if s.db == nil {
		return os.ErrClosed
	}
	if s.fault != nil {
		return s.fault
	}
	if !s.uploads {
		return errors.New("upload ledger has not been reconciled")
	}
	// A backwards clock holds state and commitments until trusted time returns.
	if now < s.lastUTC {
		return errors.New("clock behind persisted upload state")
	}
	s.fault = s.checkBacking(now)
	return s.fault
}

// A successful commit precedes every result. An uncertain write disables the
// instance until Open + CheckUploads reconciles SQLite. Expected refusals are
// safe to return after persisting observed expiry and trusted time.
func (s *Store) uploadTransaction(now int64, scope *[96]byte, operation func() (UploadState, error)) (UploadState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.uploadTransactionLocked(now, scope, operation)
}

func (s *Store) uploadTransactionLocked(now int64, scope *[96]byte, operation func() (UploadState, error)) (UploadState, error) {
	if err := s.uploadReady(now); err != nil {
		return UploadState{}, err
	}
	if scope != nil && *scope != s.uploadScope {
		return UploadState{}, ErrUploadDenied
	}
	if err := s.expireUploads(now); err != nil {
		s.fault = err
		return UploadState{}, err
	}
	if err := s.db.exec("BEGIN IMMEDIATE"); err != nil {
		s.fault = err
		return UploadState{}, err
	}
	var state UploadState
	var result error
	if now > s.lastUTC {
		s.fault = s.db.exec(fmt.Sprintf("UPDATE enrollment SET last_utc=%d WHERE id=1", now))
	}
	if s.fault == nil {
		state, result = operation()
		if result != nil && !errors.Is(result, ErrUploadConflict) && !errors.Is(result, ErrUploadPaused) && !errors.Is(result, ErrUploadCapacity) && !errors.Is(result, ErrUploadDenied) && !errors.Is(result, ErrUploadBusy) && !errors.Is(result, ErrUploadInvalid) {
			s.fault = result
		}
	}
	if s.fault == nil {
		s.fault = s.db.exec("COMMIT")
	}
	if s.fault != nil {
		_ = s.db.exec("ROLLBACK")
		return UploadState{}, s.fault
	}
	s.lastUTC = now
	return state, result
}

func (s *Store) lookupUpload(id [16]byte, fingerprint [32]byte, exact []byte) (UploadState, error) {
	manifest, err := s.db.scalar(fmt.Sprintf("SELECT coalesce((SELECT manifest FROM upload_v1 WHERE id=X'%x'),X'')", id))
	if err != nil || len(manifest) == 0 {
		return UploadState{}, err
	}
	if len(manifest) != 276 {
		return UploadState{}, errors.New("unreadable saved upload manifest")
	}
	if sha256.Sum256(manifest) != fingerprint || exact != nil && !bytes.Equal(exact, manifest) {
		return UploadState{}, ErrUploadConflict
	}
	deadline, err := s.uploadNumber(fmt.Sprintf("SELECT CASE WHEN expired THEN 0 ELSE deadline END FROM upload_v1 WHERE id=X'%x'", id))
	if err != nil {
		return UploadState{}, err
	}
	if deadline == 0 {
		return UploadState{Code: 3}, nil
	}
	mask, err := s.uploadNumber(fmt.Sprintf("SELECT mask FROM upload_v1 WHERE id=X'%x'", id))
	if err != nil || mask > 3 {
		return UploadState{}, errors.Join(errors.New("invalid staged item mask"), err)
	}
	return UploadState{Code: 1, Mask: byte(mask), Deadline: int64(deadline)}, nil
}

func (s *Store) ReserveUpload(m [276]byte, limits UploadLimits, now, expiry int64) (UploadState, error) {
	obj, cat, err := UploadItemSizes(m)
	if err != nil || now <= 0 || now > math.MaxInt64-900 {
		return UploadState{}, errors.Join(errors.New("invalid upload admission"), err)
	}
	id := [16]byte(m[116:132])
	fingerprint := sha256.Sum256(m[:])
	scope := [96]byte(m[20:116])
	return s.uploadTransaction(now, &scope, func() (UploadState, error) {
		prior, err := s.lookupUpload(id, fingerprint, m[:])
		if err != nil || prior.Code != 0 {
			return prior, err
		}
		if expiry <= now {
			return UploadState{}, ErrUploadDenied
		}
		if limits.Paused {
			return UploadState{}, ErrUploadPaused
		}
		locators := fmt.Sprintf("IN (X'%x',X'%x')", m[132:164], m[204:236])
		collisions, err := s.uploadNumber("SELECT count(*) FROM upload_v1 WHERE expired=0 AND (substr(manifest,133,32) " + locators + " OR substr(manifest,205,32) " + locators + ")")
		if err != nil {
			return UploadState{}, err
		}
		if collisions != 0 {
			return UploadState{}, ErrUploadConflict
		}
		charge, err := s.admitUpload(obj, cat, limits)
		if err != nil {
			return UploadState{}, err
		}
		deadline := min(now+900, expiry)
		if err := s.db.exec(fmt.Sprintf("INSERT INTO upload_v1(id,manifest,fingerprint,deadline,charge) VALUES(X'%x',X'%x',X'%x',%d,%d)", id, m, fingerprint, deadline, charge)); err != nil {
			return UploadState{}, err
		}
		return UploadState{Code: 1, Deadline: deadline}, nil
	})
}

func (s *Store) UploadStatus(id [16]byte, fingerprint [32]byte, now int64) (UploadState, error) {
	return s.uploadTransaction(now, nil, func() (UploadState, error) {
		return s.lookupUpload(id, fingerprint, nil)
	})
}

// UploadUsage reports retained request IDs and bytes committed to pending
// payload slots. The fixed index and journal allowance is budgeted separately.
func (s *Store) UploadUsage() (int, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.uploadReady(s.lastUTC); err != nil {
		return 0, 0, err
	}
	return s.uploadUsage()
}

func (s *Store) uploadUsage() (int, uint64, error) {
	count, err := s.uploadNumber("SELECT count(*) FROM upload_v1")
	if err != nil || count > uploadCeiling {
		return 0, 0, errors.Join(errors.New("invalid retained upload count"), err)
	}
	charged, err := s.uploadNumber("SELECT coalesce(sum(charge),0) FROM upload_v1 WHERE expired=0 OR cleanup=1")
	return int(count), charged, err
}
