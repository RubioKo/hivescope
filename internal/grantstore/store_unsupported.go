// SPDX-License-Identifier: AGPL-3.0-only

//go:build !linux || android || !cgo

package grantstore

import "time"

type Store struct{}

func open(string, [32]byte, bool) (*Store, error) { return nil, ErrUnsupported }
func (*Store) Record() ([]byte, error)            { return nil, ErrUnsupported }
func (*Store) Replace([]byte, []byte) error       { return ErrUnsupported }
func (*Store) PendingRevocation() ([]byte, error) { return nil, ErrUnsupported }
func (*Store) CompleteRevocation([]byte) error    { return ErrUnsupported }
func (*Store) Close() error                       { return nil }

func (*Store) InitUploads([96]byte) error  { return ErrUnsupported }
func (*Store) CheckUploads([96]byte) error { return ErrUnsupported }
func (*Store) ReserveUpload([276]byte, UploadLimits, int64, int64) (UploadState, error) {
	return UploadState{}, ErrUnsupported
}
func (*Store) UploadStatus([16]byte, [32]byte, int64) (UploadState, error) {
	return UploadState{}, ErrUnsupported
}
func (*Store) UploadUsage() (int, uint64, error) { return 0, 0, ErrUnsupported }

type PutWriter struct{}

func (*Store) BeginPut([16]byte, [32]byte, byte, UploadLimits, time.Time) (*PutWriter, UploadState, error) {
	return nil, UploadState{}, ErrUnsupported
}
func (*PutWriter) Size() uint64                 { return 0 }
func (*PutWriter) Write([]byte) (int, error)    { return 0, ErrUnsupported }
func (*PutWriter) Finish() (UploadState, error) { return UploadState{}, ErrUnsupported }
func (*PutWriter) Abort() error                 { return nil }
