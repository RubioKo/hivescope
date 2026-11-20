// SPDX-License-Identifier: AGPL-3.0-only

//go:build !windows && (!linux || android)

package storageformat

import (
	"errors"
	"os"
)

func validatePublishName(string) error { return nil }

func openPublishDir(string) (*os.File, error) {
	return nil, errors.ErrUnsupported
}

func publishStaged(*os.File, string, string, bool) error {
	return errors.ErrUnsupported
}

func syncPublishDir(*os.File) error {
	return errors.ErrUnsupported
}
