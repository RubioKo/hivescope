// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"io"
	"os"
)

const intentName = "restore-v1"

// The per-restore directory is private to the caller, including its ACL.
// Keep it distinct from the selected file's parent so comparison staging
// and the immutable intent cannot occupy the destination name.
func openRestoreState(path string, destDir *os.File) (*publicationDirectory, error) {
	if path == "" {
		return nil, errors.New("missing private restore state directory")
	}
	dest, err := destDir.Stat()
	if err != nil {
		return nil, err
	}
	state, err := openPublicationDirectory(path)
	if err != nil {
		return nil, err
	}
	info, err := state.native.Stat()
	if err != nil {
		return nil, errors.Join(err, state.close())
	}
	if os.SameFile(dest, info) {
		return nil, errors.Join(errors.New("restore state requires a separate directory"), state.close())
	}
	return state, nil
}

// The verified signature binds the full catalog and its signing context.
// A domain-separated HKDF output lets the caller check the original path
// after unlocking, without persisting the path or a reusable encryption key.
func (s *Snapshot) intentBinding(destination string) ([]byte, error) {
	return hkdf.Key(sha256.New, s.versionKey[:], s.catalogSignature[:],
		"HiveScope/local-publication/v1\x00"+destination, sha256.Size)
}

func (s *Snapshot) recordIntent(stateDir string, destDir *os.File, destination string) (err error) {
	state, err := openRestoreState(stateDir, destDir)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, state.close()) }()
	binding, err := s.intentBinding(destination)
	if err != nil {
		return err
	}
	file, err := state.root.OpenFile(intentName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = file.Write(binding); err == nil {
		err = file.Sync()
	}
	if err = errors.Join(err, file.Close()); err == nil {
		err = syncPublishDir(state.native)
	}
	if err != nil {
		return errors.Join(err, state.root.Remove(intentName))
	}
	return nil
}

func (s *Snapshot) checkIntent(state *publicationDirectory, destination string) error {
	want, err := s.intentBinding(destination)
	if err != nil {
		return err
	}
	file, err := openComparedFile(state.native, intentName)
	if err != nil {
		return err
	}
	got, readErr := io.ReadAll(io.LimitReader(file, sha256.Size+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return err
	}
	if !hmac.Equal(want, got) {
		return errors.New("restore record does not match the signed catalog and destination")
	}
	return nil
}
