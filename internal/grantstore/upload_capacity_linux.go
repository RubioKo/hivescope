// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"golang.org/x/sys/unix"
)

type uploadVolume struct {
	unit, available, freeInodes uint64
}

func (s *Store) uploadVolume() (uploadVolume, error) {
	var fs unix.Statfs_t
	if err := unix.Fstatfs(int(s.dir.Fd()), &fs); err != nil {
		return uploadVolume{}, err
	}
	// f_bavail uses fragment units. Round physical commitments up to the
	// larger block size. Reject unknown geometry on an unqualified volume.
	if fs.Frsize <= 0 || fs.Bsize < fs.Frsize || fs.Bsize > 1<<20 || fs.Bavail > fs.Blocks {
		return uploadVolume{}, errors.New("unsupported upload volume geometry")
	}
	fragment := uint64(fs.Frsize)
	available := uint64(math.MaxUint64)
	if fs.Bavail <= math.MaxUint64/fragment {
		available = fs.Bavail * fragment
	}
	return uploadVolume{uint64(fs.Bsize), available, fs.Ffree}, nil
}

func (v uploadVolume) rounded(bytes uint64) uint64 {
	return (bytes + v.unit - 1) / v.unit * v.unit
}

func (v uploadVolume) commitment(object, catalog uint64) uint64 {
	return v.slotCommitment(object) + v.slotCommitment(catalog)
}

func (v uploadVolume) slotCommitment(size uint64) uint64 {
	// Separate blocks for object and catalog. Allow at least one extra block
	// per slot and one per 16 data blocks for extent and directory metadata.
	payload := v.rounded(size)
	return payload + ((payload/v.unit+15)/16)*v.unit
}

func (v uploadVolume) metadataAllowance(page uint64) uint64 {
	// The private database caps its index at 32 pages. Budget a rollback
	// copy of every page with its 8-byte record overhead and a 64 KiB journal
	// header sector per page, plus directory/anchor and cleanup space.
	return v.rounded(32*page) + v.rounded(32*(page+8+65536)+65536) + 8*v.unit
}

type uploadAllocation struct {
	metadata, items, itemInodes uint64
}

func allocatedBytes(st unix.Stat_t) (uint64, error) {
	if st.Blocks < 0 || uint64(st.Blocks) > math.MaxUint64/512 {
		return 0, errors.New("invalid volume allocation")
	}
	return uint64(st.Blocks) * 512, nil // st_blocks always uses 512-byte units
}

func (a *uploadAllocation) addItem(st unix.Stat_t, slotBudget uint64) error {
	size, err := allocatedBytes(st)
	if err != nil {
		return err
	}
	if size > slotBudget {
		return ErrUploadCapacity
	}
	a.items += size // at most 128 item slots of bounded size
	a.itemInodes++
	return nil
}

// Count linked slots and each writer's unnamed file once. statfs already
// excludes these blocks and inodes from its available totals.
func (s *Store) uploadAllocations(volume uploadVolume) (uploadAllocation, error) {
	var used uploadAllocation
	fd, err := unix.Openat(int(s.dir.Fd()), ".", unix.O_DIRECTORY|unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return used, err
	}
	dir := os.NewFile(uintptr(fd), "upload store metadata")
	defer dir.Close()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return used, err
	}
	used.metadata, err = allocatedBytes(stat)
	if err != nil {
		return used, err
	}
	entries, err := dir.ReadDir(2*uploadCeiling + 4)
	if err != nil && !errors.Is(err, io.EOF) {
		return used, err
	}
	for _, entry := range entries {
		name := entry.Name()
		st, err := s.ownedFile(name)
		if err != nil {
			return used, err
		}
		if slot, kind, ok := parseUploadSlot(name); ok {
			m, err := s.db.scalar(fmt.Sprintf("SELECT manifest FROM upload_v1 WHERE slot=%d AND expired=0", slot))
			if err != nil || len(m) != 276 {
				return used, errors.Join(errors.New("item without a live owner"), err)
			}
			object, catalog, err := UploadItemSizes([276]byte(m))
			if err != nil {
				return used, err
			}
			size := object
			if kind == 1 {
				size = catalog
			}
			if err := used.addItem(st, volume.slotCommitment(size)); err != nil {
				return used, err
			}
			continue
		}
		if name != indexName && name != anchorName && name != indexName+"-journal" {
			return used, errors.New("unexpected file in reservation store")
		}
		size, err := allocatedBytes(st)
		if err != nil || size > math.MaxUint64-used.metadata {
			return used, errors.Join(errors.New("invalid allocated metadata"), err)
		}
		used.metadata += size
	}
	for _, w := range s.writers {
		if err := unix.Fstat(int(w.file.Fd()), &stat); err != nil {
			return used, err
		}
		if stat.Nlink == 0 {
			if err := used.addItem(stat, volume.slotCommitment(w.Size())); err != nil {
				return used, err
			}
		}
	}
	return used, nil
}

func (s *Store) admitUpload(object, catalog uint64, limits UploadLimits) (uint64, error) {
	count, _, err := s.uploadUsage()
	if err != nil {
		return 0, err
	}
	if count >= min(limits.MaxRequests, uploadCeiling) {
		return 0, ErrUploadCapacity
	}
	volume, err := s.uploadVolume()
	if err != nil {
		return 0, err
	}
	charge := volume.commitment(object, catalog)
	return charge, s.checkUploadCapacity(limits, volume, charge, 2)
}

func (s *Store) checkPutCapacity(limits UploadLimits) error {
	volume, err := s.uploadVolume()
	if err != nil {
		return err
	}
	return s.checkUploadCapacity(limits, volume, 0, 0)
}

func (s *Store) checkUploadCapacity(limits UploadLimits, volume uploadVolume, newCharge, newInodes uint64) error {
	page, err := s.uploadNumber("PRAGMA page_size")
	if err != nil || page < 512 || page > 65536 {
		return errors.Join(errors.New("unsupported SQLite page size"), err)
	}
	fixed := volume.metadataAllowance(page)
	used, err := s.uploadAllocations(volume)
	if err != nil {
		return err
	}
	_, charged, err := s.uploadUsage()
	if err != nil {
		return err
	}
	capBytes := min(limits.HostBytes, limits.AgreementBytes, limits.RequesterBytes)
	if used.metadata > fixed || used.items > charged || fixed > capBytes || charged > capBytes-fixed || newCharge > capBytes-fixed-charged {
		return ErrUploadCapacity
	}
	required := fixed - used.metadata + (charged - used.items) + newCharge
	if required > volume.available || limits.FreeMargin > volume.available-required {
		return ErrUploadCapacity
	}
	live, err := s.uploadNumber("SELECT count(*) FROM upload_v1 WHERE expired=0")
	if err != nil {
		return err
	}
	if used.itemInodes > 2*live || volume.freeInodes < 2*live-used.itemInodes+newInodes+2 {
		return ErrUploadCapacity
	}
	return nil
}
