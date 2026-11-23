// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

// SaveVerifiedReceipt persists an already authenticated receipt in the caller's
// existing private directory. The manifest fingerprint names an immutable record.
func SaveVerifiedReceipt(path string, fingerprint [32]byte, receipt [132]byte) (err error) {
	if path == "" {
		return errors.New("missing private receipt directory")
	}
	dir, err := openPublicationDirectory(path)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, dir.close()) }()
	name := hex.EncodeToString(fingerprint[:]) + ".receipt"
	if err := checkSavedReceipt(dir, name, receipt); !errors.Is(err, os.ErrNotExist) {
		return err
	}

	random := rand.Text()
	stage := random[:8] + "." + random[8:11]
	f, err := dir.root.OpenFile(stage, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(receipt[:]); err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return errors.Join(err, dir.root.Remove(stage))
	}
	if err := publishStaged(dir.native, stage, name, false); err != nil {
		cleanupErr := dir.root.Remove(stage)
		if errors.Is(err, os.ErrExist) {
			return errors.Join(checkSavedReceipt(dir, name, receipt), cleanupErr)
		}
		return errors.Join(err, cleanupErr)
	}
	return syncPublishDir(dir.native)
}

func checkSavedReceipt(dir *publicationDirectory, name string, receipt [132]byte) (err error) {
	f, err := openComparedFile(dir.native, name)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	got, err := io.ReadAll(io.LimitReader(f, int64(len(receipt)+1)))
	if err != nil {
		return err
	}
	if !bytes.Equal(got, receipt[:]) {
		return errors.New("saved receipt differs from the verified receipt")
	}
	if err := f.Sync(); err != nil {
		return err
	}
	return syncPublishDir(dir.native)
}
