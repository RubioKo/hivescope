// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"
	"testing/iotest"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

func TestCatalogReadTranscriptVector(t *testing.T) {
	// Public peer identities and transcript digest from recovery-format.md.
	peerKey := func(s string) [32]byte {
		t.Helper()
		b, err := hex.DecodeString(s)
		if err != nil || len(b) != 32 {
			t.Fatal("invalid fixture key")
		}
		return [32]byte(b)
	}
	pkg := storageformat.RecoveryPackage{Generation: 7}
	pkg.Catalog.FolderID = [16]byte(bytes.Repeat([]byte{0xa1}, 16))
	pkg.Catalog.AgreementID = [16]byte(bytes.Repeat([]byte{0xb2}, 16))
	pkg.Catalog.CatalogLocator = [32]byte(bytes.Repeat([]byte{0xc3}, 32))
	provider := peerKey("3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c")
	device := peerKey("fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025")
	challenge := [32]byte(bytes.Repeat([]byte{0x11}, 32))
	transcript := readTranscript(provider, device, catalogRequest(&pkg), challenge)
	want := peerKey("051413d2b0f8963776b0757fa880d42dbb719b00ec1cb1e59c26aee6471eec43")
	if len(transcript) != 196 || sha256.Sum256(transcript) != want {
		t.Fatal("catalog read differs from the published signing transcript")
	}
}

func TestReceiveCatalogBoundaries(t *testing.T) {
	frame := func(op byte, payload []byte) []byte {
		t.Helper()
		var buf bytes.Buffer
		if err := writeFrame(&buf, op, payload); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	declared := func(n uint64) []byte { return frame(readSizeOp, binary.BigEndian.AppendUint64(nil, n)) }
	sealed := make([]byte, catalogMinSize)
	data := frame(dataOp, sealed)
	end := frame(endOp, nil)
	denied := frame(errorOp, []byte{2})
	packet := func(parts ...[]byte) []byte { return bytes.Join(parts, nil) }
	for _, size := range []int{catalogMinSize, catalogMaxSize} {
		sealed := bytes.Repeat([]byte{0xab}, size)
		r := bytes.NewReader(packet(declared(uint64(size)), frame(dataOp, sealed), end))
		got, err := receiveCatalog(iotest.OneByteReader(r))
		if err != nil || !bytes.Equal(got, sealed) {
			t.Fatalf("fragmented %d-byte catalog: %v", size, err)
		}
	}
	for name, wire := range map[string][]byte{
		"too small":         declared(catalogMinSize - 1),
		"too large":         declared(catalogMaxSize + 1),
		"short DATA":        packet(declared(catalogMinSize+1), data, end),
		"missing END":       packet(declared(catalogMinSize), data),
		"trailing frame":    packet(declared(catalogMinSize), data, end, end),
		"denial with data":  packet(denied, data),
		"error during read": packet(declared(catalogMinSize), denied),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := receiveCatalog(bytes.NewReader(wire))
			if got != nil || err == nil {
				t.Fatal("incomplete or invalid response exposed catalog bytes")
			}
		})
	}
	if got, err := receiveCatalog(bytes.NewReader(denied)); got != nil || !errors.Is(err, ErrReadDenied) {
		t.Fatalf("authenticated denial: bytes=%d, err=%v", len(got), err)
	}
}

func TestNoncanonicalStorageFrame(t *testing.T) {
	for name, msg := range map[string][]byte{
		"overlong op":       {8, 0xa1, 0},
		"explicit empty":    {8, endOp, 0x12, 0},
		"overlong length":   {8, dataOp, 0x12, 0x81, 0, 'x'},
		"duplicate payload": {8, dataOp, 0x12, 1, 'x', 0x12, 1, 'y'},
		"truncated varint":  {8, dataOp, 0x12, 0x80},
	} {
		t.Run(name, func(t *testing.T) {
			packet := append(binary.BigEndian.AppendUint32(nil, uint32(len(msg))), msg...)
			if _, _, err := readFrame(bytes.NewReader(packet), catalogMaxSize); err == nil {
				t.Fatal("accepted noncanonical frame")
			}
		})
	}
}
