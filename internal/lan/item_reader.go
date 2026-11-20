// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"errors"
	"io"
)

// sealedItemReader exposes only DATA payloads to the format verifier. EOF
// means END followed by an orderly stream close, even if the exact number
// of advertised bytes has already arrived.
type sealedItemReader struct {
	wire      io.Reader
	remaining uint64 // sealed bytes not yet framed
	pending   []byte // at most one DATA frame
	terminal  error
}

func (r *sealedItemReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.terminal != nil {
		return 0, r.terminal
	}
	if len(r.pending) == 0 {
		if err := r.nextFrame(); err != nil {
			r.terminal = err
			return 0, err
		}
	}
	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

func (r *sealedItemReader) nextFrame() error {
	want := int(min(r.remaining, maxDataPayload))
	op, payload, err := readFrame(r.wire, max(want, 1)) // allow a terminal error even after the last DATA
	if err == io.EOF {
		return io.ErrUnexpectedEOF // stream closed before DATA or END
	}
	if err != nil {
		return err
	}
	if op == errorOp {
		if err := expectEOF(r.wire); err != nil {
			return err
		}
		return providerError(payload)
	}
	if want == 0 {
		if op != endOp || len(payload) != 0 {
			return errors.New("expected END after final DATA")
		}
		if err := expectEOF(r.wire); err != nil {
			return err
		}
		return io.EOF
	}
	if op != dataOp || len(payload) != want {
		return errors.New("DATA differs from expected v1 frame length")
	}
	r.remaining -= uint64(want)
	r.pending = payload
	return nil
}
