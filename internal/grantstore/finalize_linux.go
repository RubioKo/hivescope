// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// A retained receipt is the membership record for exactly the two manifest
// roles. Health is derived from the files, independently of the receipt.
func (s *Store) publication(m [276]byte, keep byte) (UploadState, *os.File, uint64, error) {
	row := fmt.Sprintf(" FROM upload_v1 WHERE id=X'%x'", m[116:132])
	raw, err := s.db.scalar("SELECT receipt" + row)
	if err != nil || len(raw) == 0 {
		return UploadState{}, nil, 0, err
	}
	if len(raw) != 132 {
		return UploadState{}, nil, 0, errors.New("invalid saved receipt size")
	}
	receipt := [132]byte(raw)
	expiry := int64(binary.BigEndian.Uint64(raw[60:68]))
	if err := VerifyUploadReceipt(receipt, [32]byte(s.uploadScope[:32]), sha256.Sum256(m[:]), expiry); err != nil {
		return UploadState{}, nil, 0, err
	}
	committed := binary.BigEndian.Uint64(raw[52:60])
	slot, err := s.uploadNumber("SELECT slot" + row + fmt.Sprintf(" AND expired=0 AND cleanup=0 AND mask=3 AND deadline>%d AND deadline<=%d AND (SELECT last_utc FROM enrollment WHERE id=1)>=%d", committed, expiry, committed))
	if err != nil {
		return UploadState{}, nil, 0, err
	}
	state := UploadState{Code: 2, Receipt: receipt}
	var reader *os.File
	var size uint64
	for kind := byte(1); kind <= 2; kind++ {
		offset := 164
		if kind == 1 {
			offset = 236
		}
		itemSize := binary.BigEndian.Uint64(m[offset : offset+8])
		f, err := s.verifiedSlot(uploadSlotName(slot, kind), itemSize, [32]byte(m[offset+8:offset+40]))
		if err == nil && f != nil {
			if keep == kind {
				reader, size = f, itemSize
				continue
			}
			err = f.Close()
			if err == nil {
				continue
			}
		}
		if reader != nil {
			err = errors.Join(err, reader.Close())
		}
		if err != nil {
			return UploadState{}, nil, 0, err
		}
		state.Code = 5
		return state, nil, 0, nil
	}
	return state, reader, size, nil
}

// Missing or damaged content is a property of this pair. Never unlink
// committed bytes when checking their integrity or substitute a new receipt.
func (s *Store) verifiedSlot(name string, size uint64, digest [32]byte) (*os.File, error) {
	f, err := s.openOwned(name, unix.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(f, int64(size)+1))
	if err != nil || n != int64(size) || [32]byte(hash.Sum(nil)) != digest {
		return nil, errors.Join(err, f.Close())
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Join(err, f.Close())
	}
	return f, nil
}

// FinalizeUpload rechecks the two staged files and publishes their membership
// and the signed receipt together. A retry only reads the original decision.
func (s *Store) FinalizeUpload(id [16]byte, fingerprint [32]byte, limits UploadLimits, now time.Time, expiry int64, signer func([]byte) ([]byte, error)) (UploadState, error) {
	start := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	current := func() (int64, error) {
		sample := time.Now()
		return now.Add(sample.Round(0).Sub(start.Round(0))).Unix(), checkUploadClock(start, sample)
	}
	began, err := current()
	if err != nil {
		return UploadState{}, err
	}
	var completed, expiredAt int64
	state, err := s.uploadTransactionLocked(began, nil, func() (UploadState, error) {
		state, err := s.lookupUpload(id, fingerprint, nil)
		if err != nil || state.Code != 1 {
			return state, err
		}
		if began >= expiry {
			return UploadState{}, ErrUploadDenied
		}
		if state.Mask != 3 {
			return state, nil
		}
		if s.writers[id] != nil {
			return UploadState{}, ErrUploadBusy
		}
		row := fmt.Sprintf(" FROM upload_v1 WHERE id=X'%x'", id)
		raw, err := s.db.scalar("SELECT manifest" + row)
		if err != nil || len(raw) != 276 || sha256.Sum256(raw) != fingerprint {
			return UploadState{}, errors.Join(errors.New("publication manifest changed"), err)
		}
		slot, err := s.uploadNumber("SELECT slot" + row)
		if err != nil {
			return UploadState{}, err
		}
		mask, err := s.reconcileStagedPair(slot, [276]byte(raw))
		if err != nil {
			return UploadState{}, err
		}
		if mask != 3 {
			state.Mask = mask
			return state, nil
		}
		if err := s.checkPutCapacity(limits); err != nil {
			return UploadState{}, err
		}
		committed, err := current()
		if err != nil || committed < began {
			return UploadState{}, errors.Join(errors.New("uncertain publication time"), err)
		}
		if committed >= state.Deadline || committed >= expiry {
			expiredAt = committed
			return UploadState{}, ErrUploadDenied
		}
		var receipt [132]byte
		copy(receipt[:20], receiptDomain)
		copy(receipt[20:52], fingerprint[:])
		binary.BigEndian.PutUint64(receipt[52:60], uint64(committed))
		binary.BigEndian.PutUint64(receipt[60:68], uint64(expiry))
		if signer == nil || expiry < state.Deadline {
			return UploadState{}, ErrUploadDenied
		}
		signature, err := signer(receipt[:68])
		if err != nil || len(signature) != ed25519.SignatureSize {
			return UploadState{}, errors.Join(errors.New("provider could not sign receipt"), err)
		}
		copy(receipt[68:], signature)
		if err := VerifyUploadReceipt(receipt, [32]byte(s.uploadScope[:32]), fingerprint, expiry); err != nil {
			return UploadState{}, err
		}
		finished, err := current()
		if err != nil || finished < committed {
			return UploadState{}, errors.Join(errors.New("uncertain publication time"), err)
		}
		if finished >= state.Deadline || finished >= expiry {
			expiredAt = finished
			return UploadState{}, ErrUploadDenied
		}
		if err := s.db.exec(fmt.Sprintf("UPDATE upload_v1 SET receipt=X'%x' WHERE slot=%d AND expired=0 AND cleanup=0 AND mask=3 AND length(receipt)=0", receipt, slot)); err != nil {
			return UploadState{}, err
		}
		if err := s.db.expect("SELECT changes()", "1"); err != nil {
			return UploadState{}, err
		}
		if err := s.db.exec(fmt.Sprintf("UPDATE enrollment SET last_utc=%d WHERE id=1", finished)); err != nil {
			return UploadState{}, err
		}
		completed = finished
		return UploadState{Code: 2, Receipt: receipt}, nil
	})
	if errors.Is(err, ErrUploadDenied) && expiredAt != 0 {
		return s.uploadTransactionLocked(expiredAt, nil, func() (UploadState, error) {
			current, err := s.lookupUpload(id, fingerprint, nil)
			if err == nil && current.Code == 1 {
				return UploadState{}, ErrUploadDenied
			}
			return current, err
		})
	}
	if err == nil && completed != 0 {
		s.lastUTC = completed
	}
	return state, err
}

// OpenPublishedItem returns the verified descriptor for the requested role.
// Callers authenticate and authorize each read before opening the item.
func (s *Store) OpenPublishedItem(kind byte, locator [32]byte, now int64) (*os.File, uint64, error) {
	offset := 205
	if kind == 2 {
		offset = 133
	} else if kind != 1 {
		return nil, 0, ErrUploadDenied
	}
	var reader *os.File
	var size uint64
	state, err := s.uploadTransaction(now, nil, func() (UploadState, error) {
		raw, err := s.db.scalar(fmt.Sprintf("SELECT coalesce((SELECT manifest FROM upload_v1 WHERE expired=0 AND length(receipt)=132 AND substr(manifest,%d,32)=X'%x'),X'')", offset, locator))
		if err != nil {
			return UploadState{}, err
		}
		if len(raw) == 0 {
			return UploadState{}, ErrUploadDenied
		}
		if len(raw) != 276 {
			return UploadState{}, errors.New("invalid published manifest")
		}
		var state UploadState
		state, reader, size, err = s.publication([276]byte(raw), kind)
		return state, err
	})
	if err != nil {
		if reader != nil {
			err = errors.Join(err, reader.Close())
		}
		return nil, 0, err
	}
	if state.Code != 2 {
		return nil, 0, ErrPublicationUnavailable
	}
	return reader, size, nil
}

// UploadAccounting splits the single physical charge between reservations
// (including outstanding cleanup) and committed pairs, healthy or damaged.
func (s *Store) UploadAccounting() (uint64, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.uploadReady(s.lastUTC); err != nil {
		return 0, 0, err
	}
	reserved, err := s.uploadNumber("SELECT coalesce(sum(charge),0) FROM upload_v1 WHERE length(receipt)=0 AND (expired=0 OR cleanup=1)")
	if err != nil {
		return 0, 0, err
	}
	stored, err := s.uploadNumber("SELECT coalesce(sum(charge),0) FROM upload_v1 WHERE length(receipt)=132")
	return reserved, stored, err
}
