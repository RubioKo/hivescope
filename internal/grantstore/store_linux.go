// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	indexName  = "grants.sqlite"
	anchorName = "grant.checkpoint"
	stageName  = "checkpoint.next"
	recordSize = 198
)

type Store struct {
	mu          sync.Mutex
	dir         *os.File
	rootStat    unix.Stat_t
	indexIno    uint64
	db          *index
	scope       []byte
	record      []byte
	pending     *revocationIntent
	lastUTC     int64
	fault       error
	fresh       bool // only a new, unused enrollment may initialize upload tables
	uploads     bool
	uploadScope [96]byte // exact provider, uploader, folder and agreement
	writers     map[[16]byte]*PutWriter
}

func open(dir string, binding [32]byte, create bool) (_ *Store, err error) {
	if dir == "" || binding == [32]byte{} {
		return nil, errors.New("missing grant store path or enrollment binding")
	}
	path, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if create {
		// Even a partially initialized directory belongs to the earlier
		// provisioning attempt and cannot be silently reused.
		if err := os.Mkdir(path, 0700); err != nil {
			return nil, err
		}
		parent, err := os.Open(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		if err := errors.Join(parent.Sync(), parent.Close()); err != nil {
			return nil, err
		}
	}
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	s := &Store{dir: os.NewFile(uintptr(fd), path), fresh: create}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	if err := unix.Fstat(fd, &s.rootStat); err != nil {
		return nil, err
	}
	if s.rootStat.Uid != uint32(os.Geteuid()) || s.rootStat.Mode&0077 != 0 {
		return nil, errors.New("grant store requires a private service-owned directory")
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return nil, fmt.Errorf("acquire grant store lock: %w", err)
	}
	// A bounded upload index can own two named slots per retained request.
	entries, err := s.dir.ReadDir(2*uploadCeiling + 6)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	var linkedSlots []uint64
	for _, entry := range entries {
		name := entry.Name()
		slot, _, item := parseUploadSlot(name)
		if create || (!item && name != indexName && name != indexName+"-journal" && name != anchorName &&
			name != intentName && name != stageName) {
			return nil, errors.New("unexpected entry in private grant store")
		}
		if item {
			linkedSlots = append(linkedSlots, slot)
		}
		if _, err := s.ownedFile(name); err != nil {
			return nil, err
		}
	}
	if create {
		dbFile, err := s.newFile(indexName)
		if err != nil {
			return nil, err
		}
		if err := dbFile.Close(); err != nil {
			return nil, err
		}
	}
	stat, err := s.ownedFile(indexName)
	if err != nil {
		return nil, err
	}
	s.indexIno = stat.Ino
	if err := s.checkLocation(); err != nil {
		return nil, err
	}
	s.db, err = openIndex(filepath.Join(path, indexName), create)
	if err != nil {
		return nil, err
	}
	// Bind this index to the enrolled profile and the provisioned directory
	// on its volume. Replacing or copying the directory requires reconciliation.
	scope := binary.BigEndian.AppendUint64(binding[:], uint64(s.rootStat.Dev))
	scope = binary.BigEndian.AppendUint64(scope, s.rootStat.Ino)
	s.scope = scope
	now := time.Now().Unix()
	if create {
		err = s.db.exec(fmt.Sprintf(`BEGIN IMMEDIATE;
			PRAGMA user_version=1;
			CREATE TABLE enrollment(id INTEGER PRIMARY KEY CHECK(id=1),
				scope BLOB NOT NULL, record BLOB NOT NULL, last_utc INTEGER NOT NULL);
			INSERT INTO enrollment VALUES(1,X'%x',X'',%d); COMMIT;`, scope, now))
		if err != nil {
			return nil, err
		}
		if err := s.writeAnchor(nil); err != nil {
			return nil, err
		}
	}
	version, err := s.db.scalar("PRAGMA user_version")
	if err != nil {
		return nil, err
	}
	switch string(version) {
	case "1":
		if len(linkedSlots) != 0 {
			return nil, errors.New("unexpected item in a grant or reservation-only store")
		}
	case "2", "3", "4":
		if err := s.db.expect("SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('upload_profile','upload_v1')", "2"); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported store schema")
	}
	for _, slot := range linkedSlots {
		if err := s.db.expect(fmt.Sprintf("SELECT count(*) FROM upload_v1 WHERE slot=%d AND (expired=0 OR cleanup=1)", slot), "1"); err != nil {
			return nil, err
		}
	}
	if err := s.db.expect("PRAGMA integrity_check(1)", "ok"); err != nil {
		return nil, err
	}
	if err := s.db.expect("SELECT count(*) FROM enrollment", "1"); err != nil {
		return nil, err
	}
	s.record, err = s.db.scalar(fmt.Sprintf(`SELECT record FROM enrollment WHERE id=1 AND scope=X'%x'
		AND typeof(record)='blob' AND typeof(last_utc)='integer'`, scope))
	if err != nil {
		return nil, err
	}
	if len(s.record) != 0 && (len(s.record) != recordSize || binary.BigEndian.Uint64(s.record[92:100]) == 0) {
		return nil, errors.New("invalid persisted grant length or generation")
	}
	if len(s.record) == 0 {
		s.record = nil
	}
	utc, err := s.db.scalar("SELECT last_utc FROM enrollment WHERE id=1")
	if err != nil {
		return nil, err
	}
	s.lastUTC, err = strconv.ParseInt(string(utc), 10, 64)
	if err != nil || s.lastUTC <= 0 || now < s.lastUTC {
		return nil, errors.New("grant store clock requires reconciliation")
	}
	s.pending, err = s.readIntent()
	if err != nil {
		return nil, err
	}
	if s.pending != nil {
		if err := s.checkPending(now); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.ownedFile(stageName); !errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("unexpected checkpoint staging without a revocation intent")
		}
		if err := s.checkAnchor(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) ownedFile(name string) (unix.Stat_t, error) {
	var st unix.Stat_t
	if err := unix.Fstatat(int(s.dir.Fd()), name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return st, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG || st.Dev != s.rootStat.Dev || st.Nlink != 1 ||
		st.Uid != uint32(os.Geteuid()) || st.Mode&0077 != 0 {
		return st, errors.New("unexpected ownership, type, links or volume for grant store entry")
	}
	return st, nil
}

func (s *Store) newFile(name string) (*os.File, error) {
	fd, err := unix.Openat(int(s.dir.Fd()), name, unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_WRONLY|unix.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

func (s *Store) checkLocation() error {
	var current unix.Stat_t
	if err := unix.Lstat(s.dir.Name(), &current); err != nil {
		return err
	}
	if current.Dev != s.rootStat.Dev || current.Ino != s.rootStat.Ino ||
		current.Uid != uint32(os.Geteuid()) || current.Mode&0077 != 0 {
		return errors.New("grant store directory moved")
	}
	st, err := s.ownedFile(indexName)
	if err != nil {
		return err
	}
	if st.Ino != s.indexIno {
		return errors.New("grant index replaced while open")
	}
	return nil
}

func (s *Store) openOwned(name string, flags int) (*os.File, error) {
	entry, err := s.ownedFile(name)
	if err != nil {
		return nil, err
	}
	fd, err := unix.Openat(int(s.dir.Fd()), name, flags|unix.O_NOFOLLOW|unix.O_CLOEXEC|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), name)
	var opened unix.Stat_t
	if err := unix.Fstat(fd, &opened); err != nil || opened.Dev != entry.Dev || opened.Ino != entry.Ino {
		return nil, errors.Join(errors.New("grant store entry changed while opening"), err, f.Close())
	}
	return f, nil
}

func (s *Store) readOwned(name string, maxBytes int64) ([]byte, error) {
	f, err := s.openOwned(name, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, maxBytes+1))
}

func (s *Store) checkAnchor() error {
	value, err := s.readOwned(anchorName, sha256.Size)
	if err != nil {
		return err
	}
	want := sha256.Sum256(s.record)
	if !bytes.Equal(value, want[:]) {
		return errors.New("grant checkpoint differs from SQLite state; reconciliation required")
	}
	return nil
}

func (s *Store) writeAnchor(record []byte) error {
	f, err := s.newFile(stageName)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(record)
	return s.publishAnchor(f, digest[:])
}

// f is either newly created or points immediately after a verified prefix.
func (s *Store) publishAnchor(f *os.File, remainder []byte) error {
	if _, err := f.Write(remainder); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := errors.Join(f.Sync(), f.Close()); err != nil {
		return err
	}
	fd := int(s.dir.Fd())
	if err := unix.Renameat(fd, stageName, fd, anchorName); err != nil {
		return err
	}
	return s.dir.Sync()
}

func (s *Store) checkBacking(now int64) error {
	if err := s.checkRow(now); err != nil {
		return err
	}
	if _, err := s.ownedFile(intentName); !errors.Is(err, os.ErrNotExist) {
		return errors.New("unexpected revocation intent during normal grant use")
	}
	return s.checkAnchor()
}

func (s *Store) checkRow(now int64) error {
	if now < s.lastUTC {
		return errors.New("clock behind persisted grant state")
	}
	if err := s.checkLocation(); err != nil {
		return err
	}
	return s.compareRow(s.record, s.lastUTC)
}

// All interpolated fields are either fixed-length binary encoded as hex
// or a decimal Unix second. No peer-supplied SQL or path enters the query.
func (s *Store) compareRow(record []byte, lastUTC int64) error {
	return s.db.expect(fmt.Sprintf(`SELECT count(*) FROM enrollment
		WHERE id=1 AND scope=X'%x' AND record=X'%x' AND last_utc=%d`,
		s.scope, record, lastUTC), "1")
}

func (s *Store) Record() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fault != nil {
		return nil, s.fault
	}
	if s.db == nil {
		return nil, os.ErrClosed
	}
	if s.pending != nil {
		return nil, ErrPendingRevocation
	}
	if err := s.checkBacking(time.Now().Unix()); err != nil {
		s.fault = err
		return nil, err
	}
	// A process could have removed a completed intent just before its final
	// directory sync. An identical retry must not acknowledge that gap.
	if err := s.dir.Sync(); err != nil {
		s.fault = err
		return nil, err
	}
	return bytes.Clone(s.record), nil
}

// Replace atomically advances the exact record the caller validated. The
// signed bytes include the unsigned big-endian generation and grant state.
func (s *Store) Replace(previous, next []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fault != nil {
		return s.fault
	}
	if s.db == nil {
		return os.ErrClosed
	}
	if s.pending != nil {
		return ErrPendingRevocation
	}
	if !bytes.Equal(previous, s.record) || len(next) != recordSize {
		return errors.New("grant replacement does not match current state")
	}
	generation := binary.BigEndian.Uint64(next[92:100])
	if generation == 0 || (len(previous) != 0 && generation <= binary.BigEndian.Uint64(previous[92:100])) {
		return errors.New("grant replacement must advance the generation")
	}
	// A commit or subsequent checkpoint flush may have reached disk even
	// when it reports an error. Stop serving until reopened and reconciled.
	now := time.Now().Unix()
	s.fault = s.checkBacking(now)
	if s.fault == nil && next[100] == 2 {
		s.fault = s.publishIntent(previous, next)
	}
	if s.fault == nil {
		s.fault = s.advanceRow(next, now)
	}
	if s.fault == nil {
		s.fault = s.writeAnchor(next)
	}
	if s.fault == nil && s.pending != nil {
		s.fault = s.finishIntent()
	}
	return s.fault
}

func (s *Store) advanceRow(next []byte, now int64) error {
	if err := s.db.exec(fmt.Sprintf("UPDATE enrollment SET record=X'%x',last_utc=%d WHERE id=1 AND record=X'%x'",
		next, now, s.record)); err != nil {
		return err
	}
	if err := s.db.expect("SELECT changes()", "1"); err != nil {
		return err
	}
	if err := s.compareRow(next, now); err != nil {
		return err
	}
	s.record = bytes.Clone(next)
	s.lastUTC = now
	return nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	for _, w := range s.writers {
		err = errors.Join(err, w.abortLocked())
	}
	if s.db != nil {
		err = errors.Join(err, s.db.close())
		s.db = nil
	}
	if s.dir != nil {
		err = errors.Join(err, s.dir.Close())
		s.dir = nil
	}
	return err
}
