// SPDX-License-Identifier: AGPL-3.0-only

package storageformat

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

const blockSize = 4 << 20

// SealedObjectSize includes a 16-byte AES-GCM tag for every authenticated
// block, including the single block of an empty file.
func (s *Snapshot) SealedObjectSize() uint64 {
	blocks := max(1, (s.size+blockSize-1)/blockSize)
	return s.size + 16*blocks
}

func newGCM(key [32]byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// OpenObject verifies exactly one complete sealed object from src. The caller
// supplies an existing, private staging directory under its exclusive control
// (including its ACL on Windows). Only a fully verified file is returned.
// The caller must close the reader to remove the temporary plaintext file.
func (s *Snapshot) OpenObject(src io.Reader, privateDir string) (io.ReadCloser, error) {
	if privateDir == "" {
		return nil, errors.New("missing private staging directory")
	}
	root, err := os.OpenRoot(privateDir)
	if err != nil {
		return nil, err
	}
	name := "hivescope-" + rand.Text()
	file, err := s.verifyToTemp(src, root, name)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	staged := &stagedReader{file: file, root: root, name: name}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Join(err, staged.Close())
	}
	return staged, nil
}

// verifyToTemp creates an owned temporary file and returns it still open only
// after the complete ciphertext and the signed catalog agree.
func (s *Snapshot) verifyToTemp(src io.Reader, root *os.Root, name string) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	if err := s.decryptObject(src, file); err != nil {
		return nil, errors.Join(err, file.Close(), root.Remove(name))
	}
	return file, nil
}

// decryptObject writes authenticated blocks into unpublished staging only.
func (s *Snapshot) decryptObject(src io.Reader, staged *os.File) error {
	gcm, err := newGCM(s.versionKey)
	if err != nil {
		return err
	}
	prefix := append([]byte("HiveScope/block/v1\x00"), s.folderID[:]...)
	prefix = append(prefix, s.versionID[:]...)
	prefix = binary.BigEndian.AppendUint64(prefix, s.size)
	hash := sha256.New()
	remaining := s.size
	buffer := make([]byte, int(min(remaining, blockSize))+gcm.Overhead())
	defer clear(buffer)
	// Even an empty file has one authenticated record.
	for index := uint32(0); ; index++ {
		n := min(remaining, blockSize)
		record := buffer[:int(n)+gcm.Overhead()]
		if _, err := io.ReadFull(src, record); err != nil {
			return fmt.Errorf("read sealed block %d: %w", index, err)
		}
		hash.Write(record) // Open reuses the ciphertext buffer for plaintext.
		var nonce [12]byte
		binary.BigEndian.PutUint32(nonce[8:], index)
		aad := binary.BigEndian.AppendUint32(prefix, index)
		aad = binary.BigEndian.AppendUint32(aad, uint32(n))
		plain, err := gcm.Open(record[:0], nonce[:], record, aad)
		if err != nil {
			return fmt.Errorf("authenticate block %d: %w", index, err)
		}
		if _, err := staged.Write(plain); err != nil {
			return fmt.Errorf("stage block %d: %w", index, err)
		}
		remaining -= n
		if remaining == 0 {
			break
		}
	}
	var extra [1]byte
	if _, err := io.ReadFull(src, extra[:]); err == nil {
		return errors.New("trailing bytes in sealed object")
	} else if err != io.EOF {
		return fmt.Errorf("read end of sealed object: %w", err)
	}
	if !bytes.Equal(hash.Sum(nil), s.objectDigest[:]) {
		return errors.New("sealed object digest differs from signed catalog")
	}
	return nil
}

// Keep the writable file private. The handle exposes only Read and Close.
type stagedReader struct {
	file *os.File
	root *os.Root
	name string
	once sync.Once
	err  error
}

func (r *stagedReader) Read(p []byte) (int, error) { return r.file.Read(p) }

func (r *stagedReader) Close() error {
	r.once.Do(func() {
		closeErr := r.file.Close()
		removeErr := r.root.Remove(r.name)
		r.err = errors.Join(closeErr, removeErr, r.root.Close())
	})
	return r.err
}
