// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// PutWriter owns one unnamed slot file for one live reservation. All mutable
// fields and all allocations are protected by the store lock; network waits
// occur outside it. The writer becomes unusable after Abort or Finish.
type PutWriter struct {
	store       *Store
	id          [16]byte
	fingerprint [32]byte
	kind        byte
	name        string
	size        uint64
	digest      [sha256.Size]byte
	deadline    int64
	started     time.Time
	limits      UploadLimits
	file        *os.File
	written     uint64
	hash        hash.Hash
}

// BeginPut acquires the request's single writer only when the requested item
// still needs data. A nil writer and nil error return the durable state.
func (s *Store) BeginPut(id [16]byte, fingerprint [32]byte, kind byte, limits UploadLimits, now time.Time) (*PutWriter, UploadState, error) {
	if kind != 1 && kind != 2 {
		return nil, UploadState{}, ErrUploadInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var w *PutWriter
	state, err := s.uploadTransactionLocked(now.Unix(), nil, func() (UploadState, error) {
		state, err := s.lookupUpload(id, fingerprint, nil)
		if err != nil || state.Code != 1 || state.Mask&kind != 0 {
			return state, err
		}
		if s.writers[id] != nil {
			return UploadState{}, ErrUploadBusy
		}
		if err := s.checkPutCapacity(limits); err != nil {
			return UploadState{}, err
		}
		where := fmt.Sprintf(" FROM upload_v1 WHERE id=X'%x'", id)
		slot, err := s.uploadNumber("SELECT slot" + where)
		if err != nil {
			return UploadState{}, err
		}
		m, err := s.db.scalar("SELECT manifest" + where)
		if err != nil || len(m) != 276 || sha256.Sum256(m) != fingerprint {
			return UploadState{}, errors.Join(errors.New("reservation manifest changed"), err)
		}
		offset := 164 // sealed object size followed by its SHA-256
		if kind == 1 {
			offset = 236 // complete signed catalog
		}
		fd, err := unix.Openat(int(s.dir.Fd()), ".", unix.O_TMPFILE|unix.O_RDWR|unix.O_CLOEXEC, 0600)
		if err != nil {
			return UploadState{}, putIOError(err)
		}
		name := uploadSlotName(slot, kind)
		w = &PutWriter{
			store: s, id: id, fingerprint: fingerprint, kind: kind, name: name,
			size: binary.BigEndian.Uint64(m[offset:]), digest: [32]byte(m[offset+8 : offset+40]),
			started: now, deadline: state.Deadline, limits: limits,
			file: os.NewFile(uintptr(fd), name), hash: sha256.New(),
		}
		if s.writers == nil {
			s.writers = make(map[[16]byte]*PutWriter)
		}
		s.writers[id] = w
		return state, nil
	})
	if err != nil && w != nil {
		return nil, UploadState{}, errors.Join(err, w.abortLocked())
	}
	return w, state, err
}

func (w *PutWriter) Size() uint64 { return w.size }

func putIOError(err error) error {
	if errors.Is(err, unix.ENOSPC) || errors.Is(err, unix.EDQUOT) {
		return errors.Join(ErrUploadCapacity, err)
	}
	return err
}

// Validate the wall clock against monotonic time elapsed since admission.
// A retry may use another stream after the trusted clock has been verified.
func (w *PutWriter) checkClock(now time.Time) error {
	drift := now.Round(0).Sub(w.started.Round(0)) - now.Sub(w.started)
	if drift > 2*time.Second || drift < -2*time.Second {
		return errors.New("clock changed during upload")
	}
	return nil
}

func (w *PutWriter) Write(data []byte) (int, error) {
	s := w.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	if len(data) == 0 || len(data) > 65536 || uint64(len(data)) > w.size-w.written {
		return 0, errors.Join(ErrUploadInvalid, w.abortLocked())
	}
	now := time.Now()
	if err := w.checkClock(now); err != nil {
		return 0, errors.Join(err, w.abortLocked())
	}
	if now.Unix() >= w.deadline {
		// Persist expiry before any future decision can reuse this space.
		_, err := s.uploadTransactionLocked(now.Unix(), nil, func() (UploadState, error) {
			return UploadState{}, nil
		})
		if err == nil {
			err = ErrUploadDenied
		}
		return 0, errors.Join(err, w.abortLocked())
	}
	if err := s.uploadReady(now.Unix()); err != nil {
		return 0, errors.Join(err, w.abortLocked())
	}
	if err := s.checkPutCapacity(w.limits); err != nil {
		return 0, errors.Join(err, w.abortLocked())
	}
	n, err := w.file.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		return n, errors.Join(putIOError(err), w.abortLocked())
	}
	w.hash.Write(data)
	w.written += uint64(n)
	return n, nil
}

// Finish is called only after the transport has verified END and the request's
// orderly end. DATA alone, including every byte of a valid item, cannot stage it.
func (w *PutWriter) Finish() (UploadState, error) {
	s := w.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if w.file == nil {
		return UploadState{}, os.ErrClosed
	}
	if w.written != w.size || !bytes.Equal(w.hash.Sum(nil), w.digest[:]) {
		return UploadState{}, errors.Join(ErrUploadInvalid, w.abortLocked())
	}
	now := time.Now()
	if err := w.checkClock(now); err != nil {
		return UploadState{}, errors.Join(err, w.abortLocked())
	}
	if err := s.uploadReady(now.Unix()); err != nil {
		return UploadState{}, errors.Join(err, w.abortLocked())
	}
	if now.Unix() >= w.deadline {
		state, err := s.uploadTransactionLocked(now.Unix(), nil, func() (UploadState, error) {
			return s.lookupUpload(w.id, w.fingerprint, nil)
		})
		return state, errors.Join(err, w.abortLocked())
	}
	if err := s.checkPutCapacity(w.limits); err != nil {
		return UploadState{}, errors.Join(err, w.abortLocked())
	}
	if err := w.file.Sync(); err != nil {
		return UploadState{}, errors.Join(putIOError(err), w.abortLocked())
	}
	// fsync can materialize delayed allocations or run past the deadline.
	// Sample the clock and check capacity again before linking the slot.
	now = time.Now()
	if err := w.checkClock(now); err != nil {
		return UploadState{}, errors.Join(err, w.abortLocked())
	}
	state, err := s.uploadTransactionLocked(now.Unix(), nil, func() (UploadState, error) {
		state, err := s.lookupUpload(w.id, w.fingerprint, nil)
		if err != nil || state.Code != 1 {
			return state, err
		}
		if state.Mask&w.kind != 0 || s.writers[w.id] != w {
			return UploadState{}, errors.New("upload writer lost its reserved slot")
		}
		if err := s.checkPutCapacity(w.limits); err != nil {
			return UploadState{}, err
		}
		fd := int(w.file.Fd())
		if err := unix.Linkat(unix.AT_FDCWD, fmt.Sprintf("/proc/self/fd/%d", fd),
			int(s.dir.Fd()), w.name, unix.AT_SYMLINK_FOLLOW); err != nil {
			return UploadState{}, putIOError(err)
		}
		// From the link onward, a failure has an uncertain outcome. Keep the
		// named slot for reconciliation and do not acknowledge an uncertain bit.
		if err := s.dir.Sync(); err != nil {
			return UploadState{}, err
		}
		if err := s.db.exec(fmt.Sprintf("UPDATE upload_v1 SET mask=mask|%d WHERE id=X'%x' AND expired=0 AND (mask&%d)=0",
			w.kind, w.id, w.kind)); err != nil {
			return UploadState{}, err
		}
		if err := s.db.expect("SELECT changes()", "1"); err != nil {
			return UploadState{}, err
		}
		state.Mask |= w.kind
		return state, nil
	})
	err = errors.Join(err, w.abortLocked())
	if err != nil {
		return UploadState{}, err
	}
	return state, nil
}

// Abort releases the writer and its private file descriptor. Until a fully
// verified file is linked, closing the descriptor discards all partial bytes.
func (w *PutWriter) Abort() error {
	w.store.mu.Lock()
	defer w.store.mu.Unlock()
	return w.abortLocked()
}

func (w *PutWriter) abortLocked() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	delete(w.store.writers, w.id)
	if err != nil {
		w.store.fault = errors.Join(w.store.fault, err)
	}
	return err
}
