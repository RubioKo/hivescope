// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"errors"
	"io"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/libp2p/go-libp2p/core/network"
)

func (p *ReadProvider) beginPut(peer [32]byte, request []byte) (*grantstore.PutWriter, grantstore.UploadState, byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var state grantstore.UploadState
	if code := p.checkUploader(peer); code != 0 {
		return nil, state, code
	}
	if len(request) != 81 || (request[80] != catalogKind && request[80] != objectKind) {
		return nil, state, 1
	}
	now := time.Now()
	ref := [80]byte(request[:80])
	if code := p.checkUploadReference(ref, now); code != 0 {
		return nil, state, code
	}
	writer, state, err := p.upload.store.BeginPut([16]byte(ref[32:48]), [32]byte(ref[48:]), request[80], p.upload.limits, now)
	return writer, state, uploadErrorCode(err)
}

func (p *ReadProvider) servePut(stream network.MuxedStream, peer [32]byte, request []byte) error {
	writer, state, code := p.beginPut(peer, request)
	if code != 0 {
		return writeFrame(stream, errorOp, []byte{code})
	}
	body := idleReadStream{stream}
	if writer == nil {
		// Both peers wait for the other side if we demand EOF before sending
		// the early STATE. The client half-closes after seeing this reply.
		if err := writeUploadState(stream, state); err != nil {
			return err
		}
		if err := expectEOF(body); err != nil {
			stream.Reset()
			return err
		}
		return nil
	}
	defer writer.Abort()
	if err := writeFrame(stream, readyOp, nil); err != nil {
		return err
	}
	err := receivePut(body, writer, writer.Size())
	if err == nil {
		state, err = writer.Finish()
	}
	if err != nil {
		if abortErr := writer.Abort(); abortErr != nil {
			err = abortErr // uncertain local cleanup requires reconciliation
		}
		if errors.Is(err, errMalformedFrame) {
			stream.Reset()
			return err
		}
		if deadlineErr := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); deadlineErr != nil {
			return deadlineErr
		}
		return writeFrame(stream, errorOp, []byte{uploadErrorCode(err)})
	}
	if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	return writeUploadState(stream, state)
}

func receivePut(wire io.Reader, dst io.Writer, size uint64) error {
	item := sealedItemReader{wire: wire, remaining: size}
	for {
		err := item.nextFrame()
		if err == io.EOF {
			return nil // only END and a clean half-close can produce this EOF
		}
		if err != nil {
			return errors.Join(grantstore.ErrUploadInvalid, err)
		}
		if n, err := dst.Write(item.pending); err != nil {
			return err
		} else if n != len(item.pending) {
			return io.ErrShortWrite
		}
		item.pending = nil
	}
}
