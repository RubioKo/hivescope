// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android

package storageformat

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

func openComparedFile(dir *os.File, leaf string) (*os.File, error) {
	// O_NONBLOCK prevents an unexpected FIFO from waiting for a writer.
	flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC | unix.O_NONBLOCK | unix.O_NOCTTY
	fd, err := unix.Openat(int(dir.Fd()), leaf, flags, 0)
	runtime.KeepAlive(dir)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), leaf)
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if !info.Mode().IsRegular() {
		return nil, errors.Join(fmt.Errorf("destination must be a regular file: %w", fs.ErrInvalid), file.Close())
	}
	return file, nil
}

func comparedFileUnchanged(dir *os.File, leaf string, before os.FileInfo) (bool, error) {
	// Inspect the name again. fstat on the original descriptor would still
	// succeed after an editor replaced the directory entry with another file.
	var now unix.Stat_t
	err := unix.Fstatat(int(dir.Fd()), leaf, &now, unix.AT_SYMLINK_NOFOLLOW)
	runtime.KeepAlive(dir)
	if err != nil {
		return false, err
	}
	initial := before.Sys().(*syscall.Stat_t)
	return now.Dev == initial.Dev && now.Ino == initial.Ino && now.Size == initial.Size &&
		now.Mtim.Sec == initial.Mtim.Sec && now.Mtim.Nsec == initial.Mtim.Nsec &&
		now.Ctim.Sec == initial.Ctim.Sec && now.Ctim.Nsec == initial.Ctim.Nsec, nil
}
