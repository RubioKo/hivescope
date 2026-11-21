// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"context"
	"errors"
	"io"
	"net/netip"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

const (
	putOp   = 0x03
	readyOp = 0x20

	UploadCatalog = catalogKind
	UploadObject  = objectKind
)

// PutUploadItem stages one complete sealed item from offset zero. An early
// STATE leaves source unread. After an interrupted transfer, query STATUS with
// the saved manifest and reopen the source for each missing item.
func PutUploadItem(ctx context.Context, endpoint netip.AddrPort, pin [32]byte, device crypto.PrivKey,
	manifest [uploadManifestSize]byte, kind byte, source io.Reader) (grantstore.UploadState, error) {
	var state grantstore.UploadState
	if err := checkUploadIdentity(pin, device, manifest); err != nil {
		return state, err
	}
	if (kind != catalogKind && kind != objectKind) || source == nil {
		return state, grantstore.ErrUploadInvalid
	}
	objectSize, catalogSize, err := grantstore.UploadItemSizes(manifest)
	if err != nil {
		return state, err
	}
	size := objectSize
	if kind == catalogKind {
		size = catalogSize
	}
	ref := uploadReference(manifest)
	request := append(ref[:], kind)
	err = withStorageStream(ctx, endpoint, pin, device, func(stream network.MuxedStream) error {
		if err := writeFrame(stream, putOp, request); err != nil {
			return err
		}
		var err error
		state, err = exchangePut(stream, kind, size, source)
		return err
	})
	if err != nil {
		return grantstore.UploadState{}, err
	}
	return state, nil
}

func exchangePut(stream network.MuxedStream, kind byte, size uint64, source io.Reader) (grantstore.UploadState, error) {
	var state grantstore.UploadState
	op, payload, err := readFrame(stream, 10)
	if err != nil {
		return state, err
	}
	if op == readyOp {
		if len(payload) != 0 {
			return state, grantstore.ErrUploadInvalid
		}
		if err := sendPutData(stream, source, size); err != nil {
			stream.Reset()
			return state, err
		}
	}
	if err := stream.CloseWrite(); err != nil {
		return state, err
	}
	reply := idleReadStream{stream}
	if op == readyOp {
		state, err = readReservationState(reply, putOp)
		if err == nil && state.Code == 0 {
			return grantstore.UploadState{}, errors.New("provider lost the reservation during PUT")
		}
	} else {
		if err := expectEOF(reply); err != nil {
			return state, err
		}
		state, err = decodeUploadState(putOp, op, payload)
	}
	if err == nil && state.Code == 1 && state.Mask&kind == 0 {
		return grantstore.UploadState{}, errors.New("provider did not stage the requested item")
	}
	return state, err
}

func sendPutData(stream network.MuxedStream, source io.Reader, remaining uint64) error {
	// A frame header declares the full bounded payload even when a local
	// source fails midway. Without END and EOF the provider discards it.
	buf := make([]byte, min(remaining, maxDataPayload))
	for remaining > 0 {
		want := int64(min(remaining, maxDataPayload))
		if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}
		header := frameHeader(dataOp, int(want))
		if n, err := stream.Write(header); err != nil {
			return err
		} else if n != len(header) {
			return io.ErrShortWrite
		}
		n, err := io.CopyBuffer(stream, io.LimitReader(source, want), buf)
		if err != nil {
			return err
		}
		if n != want {
			return io.ErrUnexpectedEOF
		}
		remaining -= uint64(want)
	}
	if err := expectEOF(source); err != nil {
		return err
	}
	return writeFrame(stream, endOp, nil)
}
