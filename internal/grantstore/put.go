// SPDX-License-Identifier: AGPL-3.0-only

package grantstore

import "errors"

var (
	ErrUploadBusy    = errors.New("another transfer owns this reservation")
	ErrUploadInvalid = errors.New("invalid upload item or data")
)
