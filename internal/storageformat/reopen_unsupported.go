// SPDX-License-Identifier: AGPL-3.0-only

//go:build !windows && (!linux || android)

package storageformat

import (
	"errors"
	"os"
)

func openComparedFile(*os.File, string) (*os.File, error) {
	return nil, errors.ErrUnsupported
}

func comparedFileUnchanged(*os.File, string, os.FileInfo) (bool, error) {
	return false, errors.ErrUnsupported
}
