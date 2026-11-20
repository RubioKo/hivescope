// SPDX-License-Identifier: AGPL-3.0-only

// Package conformance reproduces the public v1 examples without running a node.
package conformance

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"slices"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func repeated(b byte, size int) []byte {
	return bytes.Repeat([]byte{b}, size)
}

func requireDigest(t *testing.T, data []byte, wantHex string) {
	t.Helper()
	got := sha256.Sum256(data)
	if !bytes.Equal(got[:], mustHex(t, wantHex)) {
		t.Fatalf("SHA-256 = %x, want %s", got, wantHex)
	}
}

func fixtureGCM(t *testing.T, key []byte) cipher.AEAD {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	return gcm
}

func fixtureCatalogGCM(t *testing.T, salt []byte) cipher.AEAD {
	t.Helper()
	key, err := hkdf.Key(sha256.New, repeated(0xe5, 32), salt, "HiveScope/catalog-key/v1", 32)
	if err != nil {
		t.Fatal(err)
	}
	return fixtureGCM(t, key)
}

func fixtureCatalogContext() []byte {
	return slices.Concat(repeated(0xa1, 16), repeated(0xb2, 16), repeated(0xc3, 32))
}

// The published file fits in one block at index zero.
func fixtureBlockAAD(versionID []byte, size uint64) []byte {
	aad := slices.Concat([]byte("HiveScope/block/v1\x00"), repeated(0xa1, 16), versionID)
	aad = binary.BigEndian.AppendUint64(aad, size)
	aad = binary.BigEndian.AppendUint32(aad, 0)
	return binary.BigEndian.AppendUint32(aad, uint32(size))
}

// fiveByteSnapshot reproduces docs/en/file-catalog.md. These fixed inputs and
// the public RFC 8032 §7.1 TEST 1024 signing seed are only for this test.
func fiveByteSnapshot(t *testing.T) (object, catalog []byte) {
	t.Helper()
	plaintext := []byte("Hive\n")
	name := []byte("hive.txt")
	versionID := repeated(0x71, 16)
	versionKey := repeated(0x81, 32)
	size := uint64(len(plaintext))
	object = fixtureGCM(t, versionKey).Seal(nil, make([]byte, 12), plaintext, fixtureBlockAAD(versionID, size))
	digest := sha256.Sum256(object)

	entry := slices.Concat(versionID, repeated(0xd4, 32), versionKey)
	entry = binary.BigEndian.AppendUint64(entry, size)
	entry = binary.BigEndian.AppendUint16(entry, uint16(len(name)))
	entry = append(entry, name...)
	entry = append(entry, digest[:]...)

	salt := repeated(0x91, 32)
	header := append([]byte("HSC1"), salt...)
	header = binary.BigEndian.AppendUint32(header, uint32(len(entry)))
	context := fixtureCatalogContext()
	aad := slices.Concat([]byte("HiveScope/catalog/v1\x00"), context, header)
	sealed := fixtureCatalogGCM(t, salt).Seal(nil, make([]byte, 12), entry, aad)
	unsigned := slices.Concat(header, sealed)
	transcript := slices.Concat([]byte("HiveScope/catalog-sign/v1\x00"), context, unsigned)
	seed := mustHex(t, "f5e5767cf153319517630f226876b86c8160cc583bc013744c6bf255f5cc0ee5")
	return object, append(unsigned, ed25519.Sign(ed25519.NewKeyFromSeed(seed), transcript)...)
}

// Open only the published hive.txt example. The future client needs a
// general parser, multi-block handling and atomic filesystem publication.
func openFiveByteSnapshot(t *testing.T, object, catalog []byte) ([]byte, error) {
	t.Helper()
	if len(catalog) != 250 || string(catalog[:4]) != "HSC1" || binary.BigEndian.Uint32(catalog[36:40]) != 130 {
		return nil, errors.New("unexpected fixture catalog header or length")
	}
	unsigned := catalog[:len(catalog)-ed25519.SignatureSize]
	context := fixtureCatalogContext()
	transcript := slices.Concat([]byte("HiveScope/catalog-sign/v1\x00"), context, unsigned)
	publicKey := ed25519.PublicKey(mustHex(t, readFixtureCatalogPublic))
	if !ed25519.Verify(publicKey, transcript, catalog[len(unsigned):]) {
		return nil, errors.New("invalid fixture catalog signature")
	}
	header := catalog[:40]
	aad := slices.Concat([]byte("HiveScope/catalog/v1\x00"), context, header)
	entry, err := fixtureCatalogGCM(t, header[4:36]).Open(nil, make([]byte, 12), unsigned[40:], aad)
	if err != nil {
		return nil, err
	}
	versionID, locator, versionKey := entry[:16], entry[16:48], entry[48:80]
	size := binary.BigEndian.Uint64(entry[80:88])
	nameLen := binary.BigEndian.Uint16(entry[88:90])
	if nameLen != 8 || string(entry[90:98]) != "hive.txt" || !bytes.Equal(locator, repeated(0xd4, 32)) {
		return nil, errors.New("unexpected fixture name or locator")
	}
	if size != 5 || len(object) != int(size)+16 {
		return nil, errors.New("unexpected fixture object length")
	}
	digest := sha256.Sum256(object)
	if !bytes.Equal(digest[:], entry[98:]) {
		return nil, errors.New("object digest differs from the authenticated catalog")
	}
	return fixtureGCM(t, versionKey).Open(nil, make([]byte, 12), object, fixtureBlockAAD(versionID, size))
}

func TestFiveByteSnapshot(t *testing.T) {
	object, catalog := fiveByteSnapshot(t)
	if !bytes.Equal(object, mustHex(t, "bac6d517956355ed4498a00c916ef93307ce8c69f8")) {
		t.Fatalf("sealed hive.txt differs from the 21 published bytes: %x", object)
	}
	requireDigest(t, object, "695479c3a0c4494d2aaa784c4786fd9bf5fc1838114df1c4d72cc50466574398")
	requireDigest(t, catalog, "96ec886154982747c3560a392c77ab56d33125215d42c95aad695954ca7fa1cf")
	got, err := openFiveByteSnapshot(t, object, catalog)
	if err != nil || !bytes.Equal(got, []byte{0x48, 0x69, 0x76, 0x65, 0x0a}) {
		t.Fatalf("restored bytes = %x, error = %v", got, err)
	}

	t.Run("substituted block", func(t *testing.T) {
		other := mustHex(t, "3a5fd4b146e8da1f9141eea7225f8b00b480c6ac5e")
		if got, err := openFiveByteSnapshot(t, other, catalog); err == nil || len(got) != 0 {
			t.Fatal("substituted block yielded a successful read or plaintext")
		}
	})
	t.Run("altered catalog signature", func(t *testing.T) {
		altered := bytes.Clone(catalog)
		altered[len(altered)-1] ^= 1
		if got, err := openFiveByteSnapshot(t, object, altered); err == nil || len(got) != 0 {
			t.Fatal("altered catalog yielded a successful read or plaintext")
		}
	})
}
