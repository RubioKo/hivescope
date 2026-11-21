// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
)

// With no store attached, any accidental call into SQLite fails the test.
func TestReservationRefusalsPrecedeLookupAndAllocation(t *testing.T) {
	m := wireTestManifest(t)
	host, writer := [32]byte(m[20:52]), [32]byte(m[52:84])
	p := &ReadProvider{
		agreement: CatalogAgreement{ProviderKey: host, FolderID: [16]byte(m[84:100]), AgreementID: [16]byte(m[100:116]),
			Start: time.Now().Add(-time.Hour), Expiry: time.Now().Add(time.Hour)},
		upload: &uploadAdmission{writer: writer, opened: time.Now()},
	}
	ref := uploadReference(m)
	wrongFolder, wrongAgreement, wrongHost, wrongWriter := m, ref, m, m
	wrongFolder[84] ^= 1
	wrongAgreement[16] ^= 1
	wrongHost[20] ^= 1
	wrongWriter[52] ^= 1
	for _, tc := range []struct {
		name    string
		peer    [32]byte
		op      byte
		payload []byte
		code    byte
	}{
		{"wrong TLS peer on reserve", host, reserveOp, m[:], 2},
		{"wrong TLS peer on status", host, statusOp, ref[:], 2},
		{"different provider", writer, reserveOp, wrongHost[:], 2},
		{"different manifest writer", writer, reserveOp, wrongWriter[:], 2},
		{"different folder", writer, reserveOp, wrongFolder[:], 2},
		{"different agreement", writer, statusOp, wrongAgreement[:], 2},
		{"short reserve", writer, reserveOp, m[:275], 1},
		{"short status", writer, statusOp, ref[:79], 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if state, code := p.admitUpload(tc.peer, tc.op, tc.payload); code != tc.code || state != (grantstore.UploadState{}) {
				t.Fatalf("got state %+v, error %d", state, code)
			}
		})
	}
	for _, reader := range []io.Reader{bytes.NewReader([]byte{0}), failingUploadReader{}} {
		var reply bytes.Buffer
		stream := struct {
			io.Reader
			io.Writer
		}{reader, &reply}
		if err := p.serveUpload(stream, writer, reserveOp, m[:]); err != nil {
			t.Fatal(err)
		}
		op, payload, err := readFrame(&reply, 1)
		if err != nil || op != errorOp || !bytes.Equal(payload, []byte{1}) || expectEOF(&reply) != nil {
			t.Fatalf("incomplete request produced %x %x, %v", op, payload, err)
		}
	}
	// PUT must apply the same independent enrollment before creating a writer.
	request := append(ref[:], byte(UploadCatalog))
	for _, tc := range []struct {
		peer    [32]byte
		payload []byte
		code    byte
	}{
		{host, request, 2},
		{writer, append(wrongAgreement[:], byte(UploadObject)), 2},
		{writer, ref[:], 1},
		{writer, append(ref[:], byte(3)), 1},
	} {
		w, state, code := p.beginPut(tc.peer, tc.payload)
		if w != nil || state != (grantstore.UploadState{}) || code != tc.code {
			t.Fatalf("refused PUT returned a writer or reservation: state=%+v, code=%d", state, code)
		}
	}
}

type failingUploadReader struct{}

func (failingUploadReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestUploadControlSlotAvailableAtStreamLimit(t *testing.T) {
	var u uploadAdmission
	for i := byte(0); i < 16; i++ {
		peer := [32]byte{i / 2}
		if allowed, last := u.enterStream(peer); !allowed || last != (i == 15) {
			t.Fatalf("stream %d: allowed=%v controlOnly=%v", i, allowed, last)
		}
		if i%2 == 1 {
			if allowed, _ := u.enterStream(peer); allowed {
				t.Fatal("admitted a third stream for the same identity")
			}
		}
	}
	if allowed, _ := u.enterStream([32]byte{8}); allowed {
		t.Fatal("exceeded the host stream limit")
	}
	for i := byte(0); i < 16; i++ {
		u.leaveStream([32]byte{i / 2})
	}
	if u.active != 0 || len(u.byPeer) != 0 {
		t.Fatal("closed sessions still occupy capacity")
	}
}
