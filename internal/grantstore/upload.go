// SPDX-License-Identifier: AGPL-3.0-only

package grantstore

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// UploadLimits applies to one dedicated trial host, uploader and agreement.
// A zero limit admits no new requests.
type UploadLimits struct {
	HostBytes, AgreementBytes, RequesterBytes, FreeMargin uint64
	MaxRequests                                           int
	Paused                                                bool
}

type UploadState struct {
	Code, Mask byte
	Deadline   int64
	Receipt    [132]byte
}

var (
	ErrUploadConflict         = errors.New("conflicting upload request or locator")
	ErrUploadCapacity         = errors.New("upload reservation exceeds capacity")
	ErrUploadPaused           = errors.New("new upload reservations are paused")
	ErrUploadDenied           = errors.New("upload scope or agreement denies admission")
	ErrPublicationUnavailable = errors.New("committed upload pair is unavailable")
)

// UploadItemSizes validates the sealed lengths, encoding domain and two
// distinct opaque locators in a v1 manifest.
func UploadItemSizes(m [276]byte) (uint64, uint64, error) {
	object := binary.BigEndian.Uint64(m[164:172])
	catalog := binary.BigEndian.Uint64(m[236:244])
	if string(m[:20]) != "HiveScope/upload/v1\x00" || object < 16 || object > 67_109_120 ||
		catalog < 243 || catalog > 497 || bytes.Equal(m[132:164], m[204:236]) {
		return 0, 0, errors.New("invalid reservation manifest")
	}
	return object, catalog, nil
}
