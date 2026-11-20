// SPDX-License-Identifier: AGPL-3.0-only

package conformance

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

func TestCatalogFramingAndTrust(t *testing.T) {
	f := hiveFixture()
	_, catalog := fiveByteSnapshot(t)
	ctx := f.context(t)
	badMagic := bytes.Clone(catalog)
	badMagic[0] ^= 1
	huge := bytes.Clone(catalog[:40])
	binary.BigEndian.PutUint32(huge[36:], 0xffffffff)
	otherAgreement := ctx
	otherAgreement.AgreementID[0] ^= 1
	otherReadKey := ctx
	otherReadKey.FolderReadKey[0] ^= 1

	for _, tc := range []struct {
		name    string
		catalog []byte
		context storageformat.CatalogContext
	}{
		{"unknown format", badMagic, ctx},
		{"oversized entry declaration", huge, ctx},
		{"truncated signature", catalog[:len(catalog)-1], ctx},
		{"trailing catalog byte", append(bytes.Clone(catalog), 0), ctx},
		{"different agreement", catalog, otherAgreement},
		{"wrong folder read key", catalog, otherReadKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, err := storageformat.OpenCatalog(bytes.NewReader(tc.catalog), tc.context)
			if err == nil || snapshot != nil {
				t.Fatal("invalid catalog returned success or exposed a snapshot")
			}
		})
	}
}

func TestAuthenticatedCatalogBounds(t *testing.T) {
	// Authenticate each entry with a distinct catalog salt. This reaches
	// the field parser independently of header and signature checks.
	for i, tc := range []struct {
		name     string
		display  string
		nameSize uint16
		fileSize uint64
		valid    bool
	}{
		{"single-byte name", "x", 1, 0, true},
		{"max name and file", strings.Repeat("é", 127) + "x", 255, 64 << 20, true},
		{"name is display text", "../sample", 9, 5, true},
		{"mismatched name length", "é", 1, 5, false},
		{"zero name length", "x", 0, 5, false},
		{"invalid UTF-8", "\xff", 1, 5, false},
		{"over file ceiling", "x", 1, (64 << 20) + 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := hiveFixture()
			f.name = tc.display
			f.catalogSalt = 0xb0 + byte(i)
			entry := f.entry(nil)
			binary.BigEndian.PutUint64(entry[80:88], tc.fileSize)
			binary.BigEndian.PutUint16(entry[88:90], tc.nameSize)
			snapshot, err := storageformat.OpenCatalog(bytes.NewReader(f.sealCatalog(t, entry)), f.context(t))
			if !tc.valid {
				if err == nil || snapshot != nil {
					t.Fatal("malformed signed entry returned success or exposed a snapshot")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if snapshot.Name() != tc.display || snapshot.Size() != tc.fileSize {
				t.Fatal("valid signed entry returned different metadata")
			}
		})
	}
}
