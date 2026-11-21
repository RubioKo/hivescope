// SPDX-License-Identifier: AGPL-3.0-only

//go:build !linux || android || !cgo

package grantstore

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
