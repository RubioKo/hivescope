// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/libp2p/go-libp2p/core/network"
)

type putTestStream struct {
	network.MuxedStream
	reply         bytes.Buffer
	sent          bytes.Buffer
	closed, reset bool
}

func (s *putTestStream) Read(p []byte) (int, error)       { return s.reply.Read(p) }
func (s *putTestStream) Write(p []byte) (int, error)      { return s.sent.Write(p) }
func (s *putTestStream) CloseWrite() error                { s.closed = true; return nil }
func (s *putTestStream) Reset() error                     { s.reset = true; return nil }
func (s *putTestStream) SetReadDeadline(time.Time) error  { return nil }
func (s *putTestStream) SetWriteDeadline(time.Time) error { return nil }

func TestPutRequiresCompleteDataEndAndEOF(t *testing.T) {
	sealed := bytes.Repeat([]byte{0x54}, maxDataPayload+1)
	stream := new(putTestStream)
	if err := writeFrame(&stream.reply, readyOp, nil); err != nil {
		t.Fatal(err)
	}
	want := grantstore.UploadState{Code: 1, Mask: UploadObject, Deadline: 1700000800}
	if err := writeUploadState(&stream.reply, want); err != nil {
		t.Fatal(err)
	}
	got, err := exchangePut(stream, UploadObject, uint64(len(sealed)), bytes.NewReader(sealed))
	if err != nil || got != want || !stream.closed || stream.reset {
		t.Fatalf("PUT exchange returned %+v, %v", got, err)
	}
	wire := stream.sent.Bytes()
	var dst bytes.Buffer
	if err := receivePut(bytes.NewReader(wire), &dst, uint64(len(sealed))); err != nil || !bytes.Equal(dst.Bytes(), sealed) {
		t.Fatalf("two-frame item changed: %v", err)
	}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"cut in the second frame", wire[:len(wire)-7]},
		{"missing END", wire[:len(wire)-6]},
		{"extra data after END", append(bytes.Clone(wire), 1)},
		{"END before DATA", wire[len(wire)-6:]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := receivePut(bytes.NewReader(tc.data), io.Discard, uint64(len(sealed))); !errors.Is(err, grantstore.ErrUploadInvalid) {
				t.Fatalf("incomplete item could be staged: %v", err)
			}
		})
	}
}

func TestPutEarlyStateSendsNoPayload(t *testing.T) {
	for _, state := range []grantstore.UploadState{{}, {Code: 3}, {Code: 1, Mask: UploadCatalog, Deadline: 1700000800}} {
		stream := new(putTestStream)
		if err := writeUploadState(&stream.reply, state); err != nil {
			t.Fatal(err)
		}
		source := bytes.NewReader(make([]byte, 250))
		got, err := exchangePut(stream, UploadCatalog, 250, source)
		if err != nil || got != state || !stream.closed || stream.sent.Len() != 0 || source.Len() != 250 {
			t.Fatalf("early state %+v consumed source bytes: %+v %v", state, got, err)
		}
	}
	stream := new(putTestStream)
	if err := writeUploadState(&stream.reply, grantstore.UploadState{Code: 1, Mask: UploadObject, Deadline: 1700000800}); err != nil {
		t.Fatal(err)
	}
	if _, err := exchangePut(stream, UploadCatalog, 250, bytes.NewReader(nil)); err == nil {
		t.Fatal("object bit alone acknowledged the catalog")
	}
}

func TestPutLocalSourceMustMatchDeclaredLength(t *testing.T) {
	for _, n := range []int{125, 251} {
		stream := new(putTestStream)
		if err := writeFrame(&stream.reply, readyOp, nil); err != nil {
			t.Fatal(err)
		}
		state, err := exchangePut(stream, UploadCatalog, 250, bytes.NewReader(make([]byte, n)))
		if err == nil || state.Code != 0 || !stream.reset || stream.closed {
			t.Fatalf("source length %d returned %+v, %v", n, state, err)
		}
		if stream.sent.Len() != len(frameHeader(dataOp, 250))+min(n, 250) {
			t.Fatal("unexpected bytes transmitted after source mismatch")
		}
	}
}
