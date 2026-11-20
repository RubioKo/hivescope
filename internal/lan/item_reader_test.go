// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// Object transfers can span several DATA frames. A verifier that has
// consumed every declared byte still needs END and the peer's orderly EOF.
func TestItemReaderRequiresCompleteTransfer(t *testing.T) {
	frame := func(op byte, payload []byte) []byte {
		t.Helper()
		var b bytes.Buffer
		if err := writeFrame(&b, op, payload); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	full := frame(dataOp, make([]byte, maxDataPayload))
	last := frame(dataOp, []byte{1})
	end := frame(endOp, nil)
	denied := frame(errorOp, []byte{2})
	wire := bytes.NewReader(bytes.Join([][]byte{full, last, end}, nil))
	complete, err := io.ReadAll(&sealedItemReader{wire: wire, remaining: maxDataPayload + 1})
	if err != nil || len(complete) != maxDataPayload+1 || complete[maxDataPayload] != 1 {
		t.Fatalf("complete item: bytes=%d, err=%v", len(complete), err)
	}
	for _, tc := range []struct {
		name      string
		frames    [][]byte
		bytesRead int64
		wantErr   error
	}{
		{"short first DATA", [][]byte{last, full, end}, 0, nil},
		{"early END", [][]byte{full, end}, maxDataPayload, nil},
		{"EOF before END", [][]byte{full, last}, maxDataPayload + 1, io.ErrUnexpectedEOF},
		{"truncated END", [][]byte{full, last, end[:4]}, maxDataPayload + 1, io.ErrUnexpectedEOF},
		{"trailing frame", [][]byte{full, last, end, last}, maxDataPayload + 1, nil},
		{"denied mid-transfer", [][]byte{full, denied}, maxDataPayload, ErrReadDenied},
		{"denied before END", [][]byte{full, last, denied}, maxDataPayload + 1, ErrReadDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire := bytes.NewReader(bytes.Join(tc.frames, nil))
			r := sealedItemReader{wire: wire, remaining: maxDataPayload + 1}
			n, err := io.Copy(io.Discard, &r)
			if n != tc.bytesRead || err == nil ||
				(tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
				t.Fatalf("received %d bytes, err=%v", n, err)
			}
		})
	}
}
