// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android

package storageformat

import (
	"os"
	"runtime"

	"golang.org/x/sys/unix"
)

func validatePublishName(string) error { return nil }

func openPublishDir(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_DIRECTORY, 0)
}

func publishStaged(dir *os.File, stagedName, destName string, replace bool) error {
	fd := int(dir.Fd())
	defer runtime.KeepAlive(dir)
	if replace {
		return unix.Renameat(fd, stagedName, fd, destName)
	}
	// One filesystem operation enforces create-only even with another writer.
	return unix.Renameat2(fd, stagedName, fd, destName, unix.RENAME_NOREPLACE)
}

func syncPublishDir(dir *os.File) error {
	return dir.Sync()
}
