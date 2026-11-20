// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func openComparedFile(dir *os.File, leaf string) (*os.File, error) {
	// FlushFileBuffers needs write access. FILE_OPEN never creates or
	// truncates the file. Deny writers and renames during the comparison.
	file, err := openComparisonHandle(dir, leaf, windows.FILE_GENERIC_READ|windows.FILE_GENERIC_WRITE, windows.FILE_SHARE_READ)
	if errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
		// An already-open reader or a read-only file may still permit a
		// comparison. A matching file must pass the same Sync barrier.
		return openComparisonHandle(dir, leaf, windows.FILE_GENERIC_READ, windows.FILE_SHARE_READ)
	}
	return file, err
}

func openComparisonHandle(dir *os.File, leaf string, access, share uint32) (*os.File, error) {
	handle, err := openExistingFileAt(dir, leaf, access, share)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(handle), leaf)
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	attrs := info.Sys().(*syscall.Win32FileAttributeData).FileAttributes
	if !info.Mode().IsRegular() || attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return nil, errors.Join(fmt.Errorf("destination must be a regular file: %w", fs.ErrInvalid), file.Close())
	}
	return file, nil
}

func comparedFileUnchanged(dir *os.File, leaf string, before os.FileInfo) (bool, error) {
	file, err := openComparisonHandle(dir, leaf, windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE)
	if err != nil {
		return false, err
	}
	now, statErr := file.Stat()
	if err := errors.Join(statErr, file.Close()); err != nil {
		return false, err
	}
	return os.SameFile(before, now) && before.Size() == now.Size() && before.ModTime().Equal(now.ModTime()), nil
}
