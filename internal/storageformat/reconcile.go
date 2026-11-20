// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrDestinationDiffers means a stable, existing destination differs from
// the independently verified snapshot. Leave it intact and ask for a new path.
var ErrDestinationDiffers = errors.New("destination differs from verified snapshot; choose a new destination")

// ReopenPublication checks a SaveRecoverableObject attempt after unlock.
// A nil result confirms the original path contains the exact, flushed bytes.
// Failed verification, I/O, or a concurrent change leaves the outcome uncertain.
// This operation never creates or replaces the selected file.
func (s *Snapshot) ReopenPublication(src io.Reader, destination, stateDir string) error {
	return s.reopenPublication(src, destination, stateDir, (*os.File).Sync, syncPublishDir)
}

func (s *Snapshot) reopenPublication(src io.Reader, destination, stateDir string,
	syncFile, syncDir func(*os.File) error) error {
	match, err := s.comparePublication(src, destination, stateDir, syncFile, syncDir)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPublicationUncertain, err)
	}
	if !match {
		return ErrDestinationDiffers
	}
	return nil
}

func (s *Snapshot) comparePublication(src io.Reader, destination, stateDir string,
	syncFile, syncDir func(*os.File) error) (match bool, err error) {
	if src == nil {
		return false, errors.New("sealed object unavailable")
	}
	parent, leaf, err := publicationPath(destination)
	if err != nil {
		return false, err
	}
	dir, err := openPublishDir(parent)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, dir.Close()) }()
	state, err := openRestoreState(stateDir, dir)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, state.close()) }()
	if err := s.checkIntent(state, filepath.Join(parent, leaf)); err != nil {
		return false, err
	}
	stage := "compare-" + rand.Text()
	verified, err := s.verifyToTemp(src, state.root, stage)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, verified.Close(), state.root.Remove(stage)) }()
	current, err := openComparedFile(dir, leaf)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, current.Close()) }()
	before, err := current.Stat()
	if err != nil {
		return false, err
	}
	match, err = equalContents(io.NewSectionReader(verified, 0, int64(s.size)), current)
	if err != nil {
		return false, err
	}
	if match {
		if err := syncFile(current); err != nil {
			return false, fmt.Errorf("sync compared file: %w", err)
		}
		if err := syncDir(dir); err != nil {
			return false, fmt.Errorf("sync destination directory: %w", err)
		}
	}
	// A flush on a descriptor can succeed even after its name or parent
	// moved. Check the selected path again while both handles are still open.
	pinned, err := dir.Stat()
	if err != nil {
		return false, err
	}
	named, err := os.Stat(parent)
	if err != nil {
		return false, err
	}
	stable, err := comparedFileUnchanged(dir, leaf, before)
	if err != nil {
		return false, err
	}
	if !os.SameFile(pinned, named) || !stable {
		return false, errors.New("destination changed during comparison")
	}
	return match, nil
}

// Compare the complete authenticated length, then check the destination's
// EOF. An empty file and a matching prefix with a trailing byte differ.
func equalContents(verified *io.SectionReader, current io.Reader) (bool, error) {
	var a, b [32 << 10]byte
	for left := verified.Size(); left > 0; {
		n := min(left, int64(len(a)))
		if _, err := io.ReadFull(verified, a[:n]); err != nil {
			return false, err
		}
		if _, err := io.ReadFull(current, b[:n]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return false, nil
			}
			return false, err
		}
		if !bytes.Equal(a[:n], b[:n]) {
			return false, nil
		}
		left -= n
	}
	_, err := io.ReadFull(current, b[:1])
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
