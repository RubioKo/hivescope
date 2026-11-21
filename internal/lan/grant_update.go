// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"net/netip"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

// SubmitGrantUpdate sends the owner's saved signed record on a pinned
// connection. A transport failure leaves confirmation pending. Retrying
// the same record is safe, including after the provider restarts.
func SubmitGrantUpdate(ctx context.Context, endpoint netip.AddrPort, pin [32]byte, device crypto.PrivKey,
	signed []byte) ([32]byte, error) {
	if len(signed) != grantRecordSize || [32]byte(signed[28:60]) != pin {
		return [32]byte{}, errors.New("grant record does not identify the expected provider")
	}
	record := [grantRecordSize]byte(signed)
	digest := sha256.Sum256(record[:])
	err := withStorageStream(ctx, endpoint, pin, device, func(stream network.MuxedStream) error {
		if err := writeFrame(stream, grantUpdateOp, record[:]); err != nil {
			return err
		}
		if err := stream.CloseWrite(); err != nil {
			return err
		}
		return verifyGrantAck(stream, digest)
	})
	if err != nil {
		return [32]byte{}, err
	}
	return digest, nil
}

func verifyGrantAck(r io.Reader, digest [32]byte) error {
	op, payload, err := readFrame(r, sha256.Size)
	if err != nil {
		return err
	}
	if err := expectEOF(r); err != nil {
		return err
	}
	if op == errorOp {
		if len(payload) == 1 {
			switch payload[0] {
			case 2:
				return ErrGrantDenied
			case 3:
				return ErrGrantConflict
			case 7:
				return ErrGrantUnavailable
			}
		}
		return providerError(payload)
	}
	if op != grantAckOp || len(payload) != sha256.Size || [32]byte(payload) != digest {
		return errors.New("provider did not acknowledge the exact signed grant")
	}
	return nil
}

func (p *ReadProvider) serveGrantUpdate(stream io.ReadWriter, record []byte) error {
	if len(record) != grantRecordSize || expectEOF(stream) != nil {
		return writeFrame(stream, errorOp, []byte{1})
	}
	err := p.applyGrant(record, true)
	if err != nil {
		switch {
		case errors.Is(err, ErrGrantDenied):
			return writeFrame(stream, errorOp, []byte{2})
		case errors.Is(err, ErrGrantConflict):
			return writeFrame(stream, errorOp, []byte{3})
		case errors.Is(err, errInvalidGrant):
			return writeFrame(stream, errorOp, []byte{1})
		default:
			return writeFrame(stream, errorOp, []byte{7})
		}
	}
	digest := sha256.Sum256(record)
	return writeFrame(stream, grantAckOp, digest[:])
}
