// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package grantstore

import (
	"errors"
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
	// Separate blocks for object and catalog. Allow at least one extra block
	// per slot and one per 16 data blocks for extent and directory metadata.
	slot := func(size uint64) uint64 {
		payload := v.rounded(size)
		return payload + ((payload/v.unit+15)/16)*v.unit
	}
	return slot(object) + slot(catalog)
}

func (v uploadVolume) metadataAllowance(page uint64) uint64 {
	// The private database caps its index at 32 pages. Budget a rollback
	// copy of every page with its 8-byte record overhead and a 64 KiB journal
	// header sector per page, plus directory/anchor and cleanup space.
	return v.rounded(32*page) + v.rounded(32*(page+8+65536)+65536) + 8*v.unit
}

// Account allocated metadata against the fixed allowance and fail closed on
// unexpected entries. Opening "." gives this scan its own directory cursor.
func (s *Store) allocatedMetadata() (uint64, error) {
	fd, err := unix.Openat(int(s.dir.Fd()), ".", unix.O_DIRECTORY|unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return 0, err
	}
	dir := os.NewFile(uintptr(fd), "upload store metadata")
	defer dir.Close()
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return 0, err
	}
	blocks := stat.Blocks
	entries, err := dir.ReadDir(4)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name != indexName && name != anchorName && name != indexName+"-journal" {
			return 0, errors.New("unexpected file in reservation store")
		}
		st, err := s.ownedFile(name)
		if err != nil {
			return 0, err
		}
		if st.Blocks < 0 || blocks < 0 || st.Blocks > math.MaxInt64-blocks {
			return 0, errors.New("invalid allocated volume blocks")
		}
		blocks += st.Blocks
	}
	if blocks < 0 || uint64(blocks) > math.MaxUint64/512 {
		return 0, errors.New("invalid volume allocation")
	}
	return uint64(blocks) * 512, nil // Linux st_blocks uses 512-byte units
}

func (s *Store) admitUpload(object, catalog uint64, limits UploadLimits) (uint64, error) {
	count, charged, err := s.uploadUsage()
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
	page, err := s.uploadNumber("PRAGMA page_size")
	if err != nil || page < 512 || page > 65536 {
		return 0, errors.Join(errors.New("unsupported SQLite page size"), err)
	}
	fixed := volume.metadataAllowance(page)
	allocated, err := s.allocatedMetadata()
	if err != nil {
		return 0, err
	}
	charge := volume.commitment(object, catalog)
	capBytes := min(limits.HostBytes, limits.AgreementBytes, limits.RequesterBytes)
	if allocated > fixed || fixed > capBytes || charged > capBytes-fixed || charge > capBytes-fixed-charged {
		return 0, ErrUploadCapacity
	}
	required := fixed - allocated + charged + charge
	if required > volume.available || limits.FreeMargin > volume.available-required {
		return 0, ErrUploadCapacity
	}
	live, err := s.uploadNumber("SELECT count(*) FROM upload_v1 WHERE expired=0")
	if err != nil {
		return 0, err
	}
	if volume.freeInodes < 2*(live+1)+2 { // both future slots, journal, and cleanup
		return 0, ErrUploadCapacity
	}
	return charge, nil
}
