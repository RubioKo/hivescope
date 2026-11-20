// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/windows"
)

func validatePublishName(name string) error {
	// Check the stem separately so that device aliases such as NUL.txt
	// cannot acquire different meanings across Windows versions.
	stem, _, _ := strings.Cut(name, ".")
	stem = strings.TrimRight(stem, " ")
	if !filepath.IsLocal(name) || (stem != "" && !filepath.IsLocal(stem)) ||
		!utf8.ValidString(name) || strings.ContainsAny(name, `\/:*?"<>|`) ||
		strings.HasSuffix(name, ".") || strings.HasSuffix(name, " ") ||
		strings.IndexFunc(name, func(r rune) bool { return r < 32 }) >= 0 {
		return fmt.Errorf("invalid destination filename: %w", fs.ErrInvalid)
	}
	return nil
}

func openPublishDir(path string) (*os.File, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	volume := filepath.VolumeName(absolute)
	if len(volume) != 2 || volume[1] != ':' {
		return nil, fmt.Errorf("destination requires a local drive path: %w", errors.ErrUnsupported)
	}
	path16, err := windows.UTF16PtrFromString(`\\?\` + absolute)
	if err != nil {
		return nil, err
	}
	// A normal read-only os.Open directory handle cannot request the
	// Windows metadata flush. Permit other callers to open the same directory.
	handle, err := windows.CreateFile(path16, windows.GENERIC_WRITE|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, err
	}
	dir := os.NewFile(uintptr(handle), absolute)
	info, err := dir.Stat()
	if err != nil {
		return nil, errors.Join(err, dir.Close())
	}
	if !info.IsDir() {
		return nil, errors.Join(fmt.Errorf("destination parent is not a directory: %w", fs.ErrInvalid), dir.Close())
	}
	// MS-FSA 2.1.5.7, product note 80: only NTFS implements this flush
	// of directory metadata. Other filesystems may return success doing nothing.
	var fileSystem [32]uint16
	if err := windows.GetVolumeInformationByHandle(handle, nil, 0, nil, nil, nil, &fileSystem[0], uint32(len(fileSystem))); err != nil {
		return nil, errors.Join(err, dir.Close())
	}
	if !strings.EqualFold(windows.UTF16ToString(fileSystem[:]), "NTFS") {
		return nil, errors.Join(fmt.Errorf("directory flush requires NTFS: %w", errors.ErrUnsupported), dir.Close())
	}
	// Ask the opened handle for a volume GUID path. Mapped network shares
	// cannot supply this form, even if their drive letter looks local.
	const volumeNameGUID = 1
	var probe [1]uint16
	if _, err := windows.GetFinalPathNameByHandle(handle, &probe[0], uint32(len(probe)), volumeNameGUID); err != nil {
		return nil, errors.Join(fmt.Errorf("resolve local destination volume: %w", err), dir.Close())
	}
	return dir, nil
}

func syncPublishDir(dir *os.File) error {
	// On Windows os.File.Sync calls FlushFileBuffers on this writable handle.
	return dir.Sync()
}

func publishStaged(dir *os.File, stagedName, destName string, replace bool) error {
	// Open the existing staging entry relative to the pinned directory.
	// It was created exclusively in a private directory by this operation.
	src, err := windows.NewNTUnicodeString(stagedName)
	if err != nil {
		return err
	}
	parent := windows.Handle(dir.Fd())
	defer runtime.KeepAlive(dir)
	attr := windows.OBJECT_ATTRIBUTES{
		RootDirectory: parent,
		ObjectName:    src,
		Attributes:    windows.OBJ_CASE_INSENSITIVE,
	}
	attr.Length = uint32(unsafe.Sizeof(attr))
	var file windows.Handle
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&file, windows.DELETE|windows.SYNCHRONIZE, &attr, &status, nil, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_OPEN,
		windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if err != nil {
		return fmt.Errorf("open staged file for rename: %w", publishWindowsError(err))
	}
	defer windows.CloseHandle(file)

	// FILE_RENAME_INFORMATION. Native components are limited to 255 UTF-16
	// code units on NTFS. The fixed buffer carries the name inline and keeps
	// the handle naturally aligned on both 32-bit and 64-bit Windows.
	var rename struct {
		ReplaceIfExists uint32
		RootDirectory   windows.Handle
		FileNameLength  uint32
		FileName        [256]uint16
	}
	name, err := windows.UTF16FromString(destName)
	if err != nil {
		return err
	}
	if len(name) > len(rename.FileName) {
		return windows.ERROR_FILENAME_EXCED_RANGE
	}
	if replace {
		rename.ReplaceIfExists = 1
	}
	// A simple target name with NULL RootDirectory renames within the
	// source's own directory. The source handle already pins that directory.
	rename.FileNameLength = uint32(2 * (len(name) - 1))
	copy(rename.FileName[:], name)
	err = windows.NtSetInformationFile(file, &status, (*byte)(unsafe.Pointer(&rename)),
		uint32(unsafe.Sizeof(rename)), windows.FileRenameInformation)
	if err != nil {
		return fmt.Errorf("rename staged file: %w", publishWindowsError(err))
	}
	return nil
}

func publishWindowsError(err error) error {
	if status, ok := err.(windows.NTStatus); ok {
		// The Win32 mapping preserves errors.Is(err, fs.ErrExist) on collision.
		return status.Errno()
	}
	return err
}
