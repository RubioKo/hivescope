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
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/iotest"

	"github.com/RubioKo/hivescope/internal/storageformat"
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

type snapshotFixture struct {
	name      string
	plaintext []byte

	catalogLocator, objectLocator      byte
	versionID, versionKey, catalogSalt byte
}

const fixtureBlockSize = 4 << 20

func hiveFixture() snapshotFixture {
	return snapshotFixture{"hive.txt", []byte("Hive\n"), 0xc3, 0xd4, 0x71, 0x81, 0x91}
}

func emptyFixture() snapshotFixture {
	return snapshotFixture{"empty.bin", nil, 0xc4, 0xd5, 0x72, 0x82, 0x92}
}

func edgeFixture() snapshotFixture {
	return snapshotFixture{"edge.bin", append(repeated(0x61, fixtureBlockSize), 0x62), 0xc5, 0xd6, 0x73, 0x83, 0x93}
}

func (f snapshotFixture) context(t *testing.T) storageformat.CatalogContext {
	t.Helper()
	return storageformat.CatalogContext{
		FolderID:         [16]byte(repeated(0xa1, 16)),
		AgreementID:      [16]byte(repeated(0xb2, 16)),
		CatalogLocator:   [32]byte(repeated(f.catalogLocator, 32)),
		FolderReadKey:    [32]byte(repeated(0xe5, 32)),
		CatalogVerifyKey: [32]byte(mustHex(t, readFixtureCatalogPublic)),
	}
}

// Fixtures seal their own bytes, independently of the reader. The vectors in
// docs/en/file-catalog.md pin the result of this test-only encoder.
func (f snapshotFixture) sealObject(t *testing.T) []byte {
	t.Helper()
	size := len(f.plaintext)
	gcm := fixtureGCM(t, repeated(f.versionKey, 32))
	var object []byte
	for index := 0; ; index++ {
		start := index * fixtureBlockSize
		end := min(start+fixtureBlockSize, size)
		chunk := f.plaintext[start:end]
		var nonce [12]byte
		binary.BigEndian.PutUint32(nonce[8:], uint32(index))
		aad := slices.Concat([]byte("HiveScope/block/v1\x00"), repeated(0xa1, 16), repeated(f.versionID, 16))
		aad = binary.BigEndian.AppendUint64(aad, uint64(size))
		aad = binary.BigEndian.AppendUint32(aad, uint32(index))
		aad = binary.BigEndian.AppendUint32(aad, uint32(len(chunk)))
		object = gcm.Seal(object, nonce[:], chunk, aad)
		if end == size {
			return object
		}
	}
}

func (f snapshotFixture) entry(object []byte) []byte {
	entry := slices.Concat(repeated(f.versionID, 16), repeated(f.objectLocator, 32), repeated(f.versionKey, 32))
	entry = binary.BigEndian.AppendUint64(entry, uint64(len(f.plaintext)))
	entry = binary.BigEndian.AppendUint16(entry, uint16(len(f.name)))
	entry = append(entry, f.name...)
	digest := sha256.Sum256(object)
	return append(entry, digest[:]...)
}

func (f snapshotFixture) sealCatalog(t *testing.T, entry []byte) []byte {
	t.Helper()
	salt := repeated(f.catalogSalt, 32)
	header := append([]byte("HSC1"), salt...)
	header = binary.BigEndian.AppendUint32(header, uint32(len(entry)))
	context := slices.Concat(repeated(0xa1, 16), repeated(0xb2, 16), repeated(f.catalogLocator, 32))
	aad := slices.Concat([]byte("HiveScope/catalog/v1\x00"), context, header)
	sealed := fixtureCatalogGCM(t, salt).Seal(nil, make([]byte, 12), entry, aad)
	unsigned := slices.Concat(header, sealed)
	transcript := slices.Concat([]byte("HiveScope/catalog-sign/v1\x00"), context, unsigned)
	// Public RFC 8032 §7.1 TEST 1024 signing seed.
	seed := mustHex(t, "f5e5767cf153319517630f226876b86c8160cc583bc013744c6bf255f5cc0ee5")
	return append(unsigned, ed25519.Sign(ed25519.NewKeyFromSeed(seed), transcript)...)
}

func fiveByteSnapshot(t *testing.T) (object, catalog []byte) {
	t.Helper()
	f := hiveFixture()
	object = f.sealObject(t)
	return object, f.sealCatalog(t, f.entry(object))
}

func TestPublishedSnapshots(t *testing.T) {
	for _, tc := range []struct {
		f                     snapshotFixture
		objectBytes           int
		objectSHA, catalogSHA string
	}{
		{hiveFixture(), 21,
			"695479c3a0c4494d2aaa784c4786fd9bf5fc1838114df1c4d72cc50466574398",
			"96ec886154982747c3560a392c77ab56d33125215d42c95aad695954ca7fa1cf"},
		{emptyFixture(), 16,
			"a543d20ea0e7176c0a2a0883c09f85a8d5df42e683334d48a349cd94679fbcbb",
			"5a4bec97a22e1043f605df722f7b8f80142ffaa66642a4bcb5e03007cef72f40"},
		{edgeFixture(), 4_194_337,
			"6d1e257b5e9f01c9bb0cc90ce845c857ab4fe173d45dc2b0b46b13520f3b018b",
			"aa5b005c0b98ffa622e41d4b6a773c81073ea6d2db9b3309dc03beb0d98a450b"},
	} {
		t.Run(tc.f.name, func(t *testing.T) {
			f := tc.f
			object := f.sealObject(t)
			catalog := f.sealCatalog(t, f.entry(object))
			if len(object) != tc.objectBytes {
				t.Fatalf("sealed size = %d, want %d", len(object), tc.objectBytes)
			}
			requireDigest(t, object, tc.objectSHA)
			requireDigest(t, catalog, tc.catalogSHA)
			if f.name == "hive.txt" && !bytes.Equal(object, mustHex(t, "bac6d517956355ed4498a00c916ef93307ce8c69f8")) {
				t.Fatal("five-byte example differs from the published ciphertext")
			}
			checkRestored(t, f, object, catalog)
		})
	}
}

func checkRestored(t *testing.T, f snapshotFixture, object, catalog []byte) {
	t.Helper()
	snapshot, err := storageformat.OpenCatalog(bytes.NewReader(catalog), f.context(t))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Name() != f.name || snapshot.Size() != uint64(len(f.plaintext)) ||
		snapshot.ObjectLocator() != [32]byte(repeated(f.objectLocator, 32)) {
		t.Fatal("authenticated snapshot metadata differs from the fixture")
	}
	privateDir := t.TempDir()
	reader, err := snapshot.OpenObject(iotest.HalfReader(bytes.NewReader(object)), privateDir)
	if err != nil {
		t.Fatal(err)
	}
	got, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("reading or closing the verified file failed: %v, %v", readErr, closeErr)
	}
	if !bytes.Equal(got, f.plaintext) {
		t.Fatalf("restored content differs (got %d bytes, want %d)", len(got), len(f.plaintext))
	}
	expectEmptyDir(t, privateDir)
}

func expectEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("staging directory contains %d leftover entries", len(entries))
	}
}

func TestExactBlockSize(t *testing.T) {
	f := snapshotFixture{"block.bin", repeated(0x65, fixtureBlockSize), 0xc6, 0xd7, 0x75, 0x85, 0x95}
	object := f.sealObject(t)
	if len(object) != fixtureBlockSize+16 {
		t.Fatal("an exact 4 MiB file needs one authenticated record")
	}
	checkRestored(t, f, object, f.sealCatalog(t, f.entry(object)))
}

func TestRejectedObjects(t *testing.T) {
	edge := edgeFixture()
	object := edge.sealObject(t)
	catalog := edge.sealCatalog(t, edge.entry(object))
	changed := bytes.Clone(object)
	changed[len(changed)-1] ^= 1
	// Pin the changed bytes so only GCM authentication can reject this block.
	badTagFixture := edge
	badTagFixture.catalogSalt = 0x97
	badTagCatalog := badTagFixture.sealCatalog(t, badTagFixture.entry(changed))

	hive := hiveFixture()
	hiveObject, hiveCatalog := fiveByteSnapshot(t)
	empty := emptyFixture()
	emptyCatalog := empty.sealCatalog(t, empty.entry(empty.sealObject(t)))

	// All GCM records remain authentic. Only the digest in this separately
	// sealed and signed catalog differs from the object it describes.
	badDigestEntry := edge.entry(object)
	badDigestEntry[len(badDigestEntry)-1] ^= 1
	badDigestFixture := edge
	badDigestFixture.catalogSalt = 0x96
	badDigestCatalog := badDigestFixture.sealCatalog(t, badDigestEntry)
	for _, tc := range []struct {
		name    string
		f       snapshotFixture
		object  []byte
		catalog []byte
	}{
		{"missing empty record", empty, nil, emptyCatalog},
		{"substituted block", hive, mustHex(t, "3a5fd4b146e8da1f9141eea7225f8b00b480c6ac5e"), hiveCatalog},
		{"truncated second block", edge, object[:len(object)-1], catalog},
		{"invalid second tag", edge, changed, badTagCatalog},
		{"trailing byte", hive, append(bytes.Clone(hiveObject), 0xf0), hiveCatalog},
		{"signed digest mismatch", edge, object, badDigestCatalog},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, err := storageformat.OpenCatalog(bytes.NewReader(tc.catalog), tc.f.context(t))
			if err != nil {
				t.Fatalf("test catalog should authenticate: %v", err)
			}
			privateDir := t.TempDir()
			got, err := snapshot.OpenObject(bytes.NewReader(tc.object), privateDir)
			if got != nil {
				got.Close()
				t.Fatal("received a plaintext reader for a rejected object")
			}
			if err == nil {
				t.Error("accepted an invalid object")
			}
			expectEmptyDir(t, privateDir)
		})
	}
}

func TestAlteredCatalogSignature(t *testing.T) {
	_, catalog := fiveByteSnapshot(t)
	catalog[len(catalog)-1] ^= 1
	if got, err := storageformat.OpenCatalog(bytes.NewReader(catalog), hiveFixture().context(t)); got != nil || err == nil {
		t.Fatal("altered signature produced a snapshot or omitted the error")
	}
}

func TestIndependentStaging(t *testing.T) {
	object, catalog := fiveByteSnapshot(t)
	snapshot, err := storageformat.OpenCatalog(bytes.NewReader(catalog), hiveFixture().context(t))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	destination := filepath.Join(dir, snapshot.Name())
	if err := os.WriteFile(destination, []byte("keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := snapshot.OpenObject(bytes.NewReader(object), dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := reader.Close(); err != nil {
			t.Error(err)
		}
	})
	if other, err := snapshot.OpenObject(bytes.NewReader(object[:1]), dir); err == nil || other != nil {
		t.Fatal("truncated second request succeeded or exposed a reader")
	}
	content, err := io.ReadAll(reader)
	if err != nil || string(content) != "Hive\n" {
		t.Fatalf("verified reader after rejected request: %q, %v", content, err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(destination)
	if err != nil || string(old) != "keep\n" {
		t.Fatalf("pre-existing file changed: %q, %v", old, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected only the pre-existing file after cleanup: %v, %v", entries, err)
	}
}
