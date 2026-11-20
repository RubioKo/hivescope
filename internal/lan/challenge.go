// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

// Both Protobuf fields and all payload lengths in this exchange use
// one-byte varints. The outer frame has a four-byte big-endian length.
const (
	opField      = 0x08
	payloadField = 0x12
	readBegin    = 0x07
	challengeOp  = 0x25
	errorOp      = 0x7f
	catalogKind  = 0x01
)

func writeCatalogRead(w io.Writer, pkg *storageformat.RecoveryPackage) error {
	const payloadSize = 1 + 16 + 16 + 8 + 32
	frame := binary.BigEndian.AppendUint32(nil, 4+payloadSize)
	frame = append(frame, opField, readBegin, payloadField, payloadSize, catalogKind)
	frame = append(frame, pkg.Catalog.FolderID[:]...)
	frame = append(frame, pkg.Catalog.AgreementID[:]...)
	frame = binary.BigEndian.AppendUint64(frame, pkg.Generation)
	frame = append(frame, pkg.Catalog.CatalogLocator[:]...)
	n, err := w.Write(frame)
	if err == nil && n != len(frame) {
		return io.ErrShortWrite
	}
	return err
}

func readChallenge(r io.Reader) ([32]byte, error) {
	var challenge [32]byte
	var length [4]byte
	if _, err := io.ReadFull(r, length[:]); err != nil {
		return challenge, err
	}
	const challengeSize = uint32(4 + len(challenge))
	const errorSize = 4 + 1
	size := binary.BigEndian.Uint32(length[:])
	if size != challengeSize && size != errorSize {
		return challenge, errors.New("invalid challenge response length")
	}
	var message [challengeSize]byte
	if _, err := io.ReadFull(r, message[:size]); err != nil {
		return challenge, err
	}
	if message[0] != opField || message[2] != payloadField || int(message[3]) != int(size)-4 {
		return challenge, errors.New("invalid challenge response fields")
	}
	if size == errorSize && message[1] == errorOp && message[4] >= 1 && message[4] <= 7 {
		return challenge, fmt.Errorf("provider returned storage error code 0x%02x", message[4])
	}
	if size != challengeSize || message[1] != challengeOp {
		return challenge, errors.New("expected CHALLENGE for the pending catalog read")
	}
	copy(challenge[:], message[4:])
	return challenge, nil
}
