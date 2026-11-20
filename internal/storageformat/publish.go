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

// ErrPublicationUncertain means the save could not confirm the directory
// flush after rename, or a later check could not establish the outcome.
var ErrPublicationUncertain = errors.New("publication uncertain")

// SaveObject verifies src, then atomically publishes the complete plaintext
// at the caller's destination in an existing private local directory.
// With replace false an existing name is never overwritten, even in a race.
// With replace true the caller explicitly authorizes replacement of that name.
// A successful return requires both the staged file and the directory to sync.
func (s *Snapshot) SaveObject(src io.Reader, destination string, replace bool) error {
	return s.saveObject(src, destination, replace, "", publishStaged, syncPublishDir)
}

// SaveRecoverableObject also records the selected path and snapshot as an
// opaque binding before the rename. The caller supplies a separate, existing
// private state directory dedicated to this attempt and retains it for reopen.
func (s *Snapshot) SaveRecoverableObject(src io.Reader, destination string, replace bool, stateDir string) error {
	if stateDir == "" {
		return errors.New("missing private restore state directory")
	}
	return s.saveObject(src, destination, replace, stateDir, publishStaged, syncPublishDir)
}

func publicationPath(destination string) (string, string, error) {
	parent, leaf := filepath.Split(destination)
	if leaf == "." || !filepath.IsLocal(leaf) {
		return "", "", fmt.Errorf("invalid destination name: %w", os.ErrInvalid)
	}
	if err := validatePublishName(leaf); err != nil {
		return "", "", err
	}
	parent, err := filepath.Abs(parent)
	return parent, leaf, err
}

type publicationDirectory struct {
	native *os.File
	root   *os.Root
}

func (d *publicationDirectory) close() error {
	return errors.Join(d.root.Close(), d.native.Close())
}

func openPublicationDirectory(parent string) (*publicationDirectory, error) {
	dir, err := openPublishDir(parent)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, errors.Join(err, dir.Close())
	}
	d := &publicationDirectory{dir, root}
	// Both handles must address the same directory even during a path change.
	dirInfo, err := dir.Stat()
	if err != nil {
		return nil, errors.Join(err, d.close())
	}
	rootInfo, err := root.Stat(".")
	if err != nil {
		return nil, errors.Join(err, d.close())
	}
	if !os.SameFile(dirInfo, rootInfo) {
		return nil, errors.Join(errors.New("directory changed while opening it"), d.close())
	}
	return d, nil
}

func (s *Snapshot) saveObject(src io.Reader, destination string, replace bool, stateDir string,
	publish func(*os.File, string, string, bool) error, syncDir func(*os.File) error) error {
	parent, leaf, err := publicationPath(destination)
	if err != nil {
		return err
	}
	dir, err := openPublicationDirectory(parent)
	if err != nil {
		return err
	}
	defer dir.close()

	// Use a DOS 8.3-compatible staging name so Windows cannot create an
	// extra short-name alias that happens to be the chosen destination.
	random := rand.Text()
	stage := random[:8] + "." + random[8:11]
	if strings.EqualFold(stage, leaf) {
		return errors.New("temporary name conflicts with chosen destination")
	}
	file, err := s.verifyToTemp(src, dir.root, stage)
	if err != nil {
		return err
	}
	// Close even when the sync fails. No rename is attempted on either error.
	syncErr := file.Sync()
	if err := errors.Join(syncErr, file.Close()); err != nil {
		return errors.Join(err, dir.root.Remove(stage))
	}
	if stateDir != "" {
		if err := s.recordIntent(stateDir, dir.native, filepath.Join(parent, leaf)); err != nil {
			return errors.Join(err, dir.root.Remove(stage))
		}
	}
	if err := publish(dir.native, stage, leaf, replace); err != nil {
		return errors.Join(err, dir.root.Remove(stage))
	}
	// The temporary name is gone. Never try to roll back or unlink the
	// destination if the following durability barrier fails.
	if err := syncDir(dir.native); err != nil {
		return fmt.Errorf("%w: sync destination directory: %w", ErrPublicationUncertain, err)
	}
	return nil
}
