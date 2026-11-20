// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

func TestChallengeFrame(t *testing.T) {
	var nonce [32]byte
	for i := range nonce {
		nonce[i] = byte(i)
	}
	frame := append([]byte{0, 0, 0, 36, 8, 0x25, 0x12, 32}, nonce[:]...)
	src := bytes.NewReader(append(bytes.Clone(frame), 0xaa))
	got, err := readChallenge(iotest.OneByteReader(src))
	if err != nil || got != nonce || src.Len() != 1 {
		t.Fatalf("fragmented challenge: error=%v, remaining bytes=%d", err, src.Len())
	}

	wrongOp := bytes.Clone(frame)
	wrongOp[5] = 0x21 // DATA is not a challenge.
	wrongType := bytes.Clone(frame)
	wrongType[6] = 0x10
	wrongLength := bytes.Clone(frame)
	wrongLength[7]--
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"incomplete prefix", frame[:3]},
		{"truncated challenge", frame[:len(frame)-1]},
		{"out of order operation", wrongOp},
		{"wrong payload field type", wrongType},
		{"wrong payload length", wrongLength},
		{"provider busy", []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, 4}},
		{"short challenge", []byte{0, 0, 0, 5, 8, 0x25, 0x12, 1, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readChallenge(bytes.NewReader(tc.data))
			if err == nil || got != [32]byte{} {
				t.Fatal("invalid or denied reply yielded a usable challenge")
			}
		})
	}
	oversized := bytes.NewReader([]byte{0xff, 0xff, 0xff, 0xff, 0xaa})
	if _, err := readChallenge(oversized); err == nil || oversized.Len() != 1 {
		t.Fatal("oversized declaration was not rejected before reading its body")
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestIncompleteCatalogRequest(t *testing.T) {
	if err := writeCatalogRead(shortWriter{}, &storageformat.RecoveryPackage{}); err != io.ErrShortWrite {
		t.Fatalf("truncated catalog request returned %v", err)
	}
}
