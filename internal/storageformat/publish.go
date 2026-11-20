// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrPublicationUncertain means the namespace switch succeeded but syncing
// its directory failed. The verified file may already be at the destination.
var ErrPublicationUncertain = errors.New("publication uncertain")

// SaveObject verifies src, then atomically publishes the complete plaintext
// at the caller's destination in an existing private local directory.
// With replace false an existing name is never overwritten, even in a race.
// With replace true the caller explicitly authorizes replacement of that name.
// A successful return requires both the staged file and the directory to sync.
func (s *Snapshot) SaveObject(src io.Reader, destination string, replace bool) error {
	return s.saveObject(src, destination, replace, publishStaged, syncPublishDir)
}

func (s *Snapshot) saveObject(src io.Reader, destination string, replace bool,
	publish func(*os.File, string, string, bool) error, syncDir func(*os.File) error) error {
	parent, leaf := filepath.Split(destination)
	if leaf == "." || !filepath.IsLocal(leaf) {
		return fmt.Errorf("invalid destination name: %w", os.ErrInvalid)
	}
	if err := validatePublishName(leaf); err != nil {
		return err
	}
	parent, err := filepath.Abs(parent)
	if err != nil {
		return err
	}
	dir, err := openPublishDir(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer root.Close()
	// The native rename and os.Root operations must address the same
	// directory even if the caller's path changes while it is being opened.
	dirInfo, err := dir.Stat()
	if err != nil {
		return err
	}
	rootInfo, err := root.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(dirInfo, rootInfo) {
		return errors.New("destination directory changed while opening it")
	}

	// Use a DOS 8.3-compatible staging name so Windows cannot create an
	// extra short-name alias that happens to be the chosen destination.
	random := rand.Text()
	stage := random[:8] + "." + random[8:11]
	if strings.EqualFold(stage, leaf) {
		return errors.New("temporary name conflicts with chosen destination")
	}
	file, err := s.verifyToTemp(src, root, stage)
	if err != nil {
		return err
	}
	// Close even when the sync fails. No rename is attempted on either error.
	syncErr := file.Sync()
	if err := errors.Join(syncErr, file.Close()); err != nil {
		return errors.Join(err, root.Remove(stage))
	}
	if err := publish(dir, stage, leaf, replace); err != nil {
		return errors.Join(err, root.Remove(stage))
	}
	// The temporary name is gone. Never try to roll back or unlink the
	// destination if the following durability barrier fails.
	if err := syncDir(dir); err != nil {
		return fmt.Errorf("%w: sync destination directory: %w", ErrPublicationUncertain, err)
	}
	return nil
}
