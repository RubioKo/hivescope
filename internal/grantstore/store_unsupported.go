// SPDX-License-Identifier: AGPL-3.0-only

//go:build !linux || android || !cgo

package grantstore

type Store struct{}

func open(string, [32]byte, bool) (*Store, error) { return nil, ErrUnsupported }
func (*Store) Record() ([]byte, error)            { return nil, ErrUnsupported }
func (*Store) Replace([]byte, []byte) error       { return ErrUnsupported }
func (*Store) Close() error                       { return nil }
