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
	"time"

	"golang.org/x/sys/unix"
)

const (
	intentName = "grant.pending"
	intentV1   = "HSRI\x01"
	// magic, scope digest, old generation, old record digest, new signed record
	intentSize = len(intentV1) + sha256.Size + 8 + sha256.Size + recordSize
)

type revocationIntent struct {
	previousGeneration uint64
	previousDigest     [sha256.Size]byte
	target             [recordSize]byte
}

func generation(record []byte) uint64 {
	if len(record) == 0 {
		return 0 // explicit empty enrollment
	}
	return binary.BigEndian.Uint64(record[92:100])
}

func (s *Store) publishIntent(previous, next []byte) error {
	intent := revocationIntent{generation(previous), sha256.Sum256(previous), [recordSize]byte(next)}
	scope := sha256.Sum256(s.scope)
	value := append([]byte(intentV1), scope[:]...)
	value = binary.BigEndian.AppendUint64(value, intent.previousGeneration)
	value = append(value, intent.previousDigest[:]...)
	value = append(value, next...)

	// O_TMPFILE leaves no partial named marker if preparation is interrupted.
	// The procfs form of linkat works for the unprivileged Linux service user.
	dirfd := int(s.dir.Fd())
	fd, err := unix.Openat(dirfd, ".", unix.O_TMPFILE|unix.O_WRONLY|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(fd), intentName)
	if _, err := f.Write(value); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := f.Sync(); err != nil {
		return errors.Join(err, f.Close())
	}
	err = unix.Linkat(unix.AT_FDCWD, fmt.Sprintf("/proc/self/fd/%d", fd), dirfd, intentName, unix.AT_SYMLINK_FOLLOW)
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	if err := s.dir.Sync(); err != nil {
		return err
	}
	s.pending = &intent
	return nil
}

func (s *Store) readIntent() (*revocationIntent, error) {
	value, err := s.readOwned(intentName, intentSize)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(value) != intentSize || string(value[:len(intentV1)]) != intentV1 {
		return nil, errors.New("invalid revocation intent format")
	}
	fields := value[len(intentV1):]
	if [sha256.Size]byte(fields[:32]) != sha256.Sum256(s.scope) {
		return nil, errors.New("revocation intent belongs to a different store")
	}
	intent := &revocationIntent{binary.BigEndian.Uint64(fields[32:40]), [32]byte(fields[40:72]), [recordSize]byte(fields[72:])}
	target := intent.target
	// The provider also authenticates the target with the enrolled authority.
	if target[100] != 2 || target[101] != 1 || [32]byte(target[102:134]) != [32]byte{} ||
		generation(target[:]) <= intent.previousGeneration {
		return nil, errors.New("invalid pending revocation or generation")
	}
	return intent, nil
}

// Across interruption the SQLite row and checkpoint may describe old/old,
// new/old or new/new. The independent checkpoint must never move backwards.
func (s *Store) checkPending(now int64) error {
	if err := s.checkRow(now); err != nil {
		return err
	}
	marker, err := s.readIntent()
	if err != nil {
		return err
	}
	if marker == nil || *marker != *s.pending {
		return errors.New("revocation intent changed during recovery")
	}
	anchor, err := s.readOwned(anchorName, sha256.Size)
	if err != nil {
		return err
	}
	oldRow := sha256.Sum256(s.record) == marker.previousDigest && generation(s.record) == marker.previousGeneration
	newRow := bytes.Equal(s.record, marker.target[:])
	newDigest := sha256.Sum256(marker.target[:])
	oldAnchor := bytes.Equal(anchor, marker.previousDigest[:])
	newAnchor := bytes.Equal(anchor, newDigest[:])
	if !(oldRow && oldAnchor || newRow && (oldAnchor || newAnchor)) {
		return errors.New("grant row or checkpoint disagrees with the pending transition")
	}
	stage, err := s.readOwned(stageName, sha256.Size)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !newRow || !oldAnchor || !bytes.HasPrefix(newDigest[:], stage) {
		return errors.New("unexpected staged checkpoint for pending revocation")
	}
	return nil
}

// PendingRevocation returns the sole signed update eligible for retry.
// Callers verify its authority and deny every read until the retry succeeds.
func (s *Store) PendingRevocation() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fault != nil {
		return nil, s.fault
	}
	if s.db == nil {
		return nil, os.ErrClosed
	}
	if s.pending == nil {
		return nil, nil
	}
	if err := s.checkPending(time.Now().Unix()); err != nil {
		s.fault = err
		return nil, err
	}
	return bytes.Clone(s.pending.target[:]), nil
}

// CompleteRevocation resumes only after the caller verifies the retransmitted
// signed record and its agreement. A rejected retry leaves the intent in place.
func (s *Store) CompleteRevocation(signed []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fault != nil {
		return s.fault
	}
	if s.db == nil {
		return os.ErrClosed
	}
	if s.pending == nil || !bytes.Equal(signed, s.pending.target[:]) {
		return errors.New("retry differs from the pending revocation")
	}
	s.fault = s.resumeRevocation()
	return s.fault
}

func (s *Store) resumeRevocation() error {
	now := time.Now().Unix()
	if err := s.checkPending(now); err != nil {
		return err
	}
	// The previous process might have stopped after linkat but before sync.
	if err := s.dir.Sync(); err != nil {
		return err
	}
	target := s.pending.target[:]
	if !bytes.Equal(s.record, target) {
		if err := s.advanceRow(target, now); err != nil {
			return err
		}
	}
	digest := sha256.Sum256(target)
	anchor, err := s.readOwned(anchorName, sha256.Size)
	if err != nil {
		return err
	}
	if bytes.Equal(anchor, s.pending.previousDigest[:]) {
		if err := s.resumeAnchor(digest); err != nil {
			return err
		}
	} else if !bytes.Equal(anchor, digest[:]) {
		return errors.New("checkpoint changed during revocation retry")
	}
	return s.finishIntent()
}

func (s *Store) resumeAnchor(digest [sha256.Size]byte) error {
	f, err := s.openOwned(stageName, unix.O_RDWR)
	if errors.Is(err, os.ErrNotExist) {
		return s.writeAnchor(s.pending.target[:])
	}
	if err != nil {
		return err
	}
	prefix, err := io.ReadAll(io.LimitReader(f, sha256.Size+1))
	if err != nil || !bytes.HasPrefix(digest[:], prefix) {
		return errors.Join(errors.New("unrecognized staged grant checkpoint"), err, f.Close())
	}
	// Only append missing bytes at the offset just verified; never truncate
	// or unlink a staging file of unknown contents or ownership.
	return s.publishAnchor(f, digest[len(prefix):])
}

func (s *Store) finishIntent() error {
	if s.pending == nil || !bytes.Equal(s.record, s.pending.target[:]) {
		return errors.New("revocation has not reached the grant index")
	}
	if err := s.checkPending(time.Now().Unix()); err != nil {
		return err
	}
	if err := s.checkAnchor(); err != nil {
		return err
	}
	// Even a matching checkpoint may come from a rename immediately before
	// the previous process died. Make its name durable before retiring intent.
	if err := s.dir.Sync(); err != nil {
		return err
	}
	if err := unix.Unlinkat(int(s.dir.Fd()), intentName, 0); err != nil {
		return err
	}
	if err := s.dir.Sync(); err != nil {
		return err
	}
	s.pending = nil
	return nil
}
