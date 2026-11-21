// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// v1 uses two ordered Protobuf fields, with field 2 omitted for an empty
// payload. Catalog callers further restrict each read to the expected size.
const maxDataPayload = 65536

func writeFrame(w io.Writer, op byte, payload []byte) error {
	if op == 0 || op >= 128 || len(payload) > maxDataPayload {
		return errors.New("invalid storage frame")
	}
	frame := append(frameHeader(op, len(payload)), payload...)
	n, err := w.Write(frame)
	if err == nil && n != len(frame) {
		return io.ErrShortWrite
	}
	return err
}

// The header also lets a PUT sender stream a bounded DATA payload from disk.
func frameHeader(op byte, payloadSize int) []byte {
	frame := []byte{0, 0, 0, 0, opField, op}
	if payloadSize != 0 {
		frame = append(frame, payloadField)
		frame = binary.AppendUvarint(frame, uint64(payloadSize))
	}
	binary.BigEndian.PutUint32(frame, uint32(len(frame)-4+payloadSize))
	return frame
}

var errMalformedFrame = errors.New("malformed storage frame")

func readFrame(r io.Reader, maxPayload int) (byte, []byte, error) {
	if maxPayload < 0 || maxPayload > maxDataPayload {
		return 0, nil, errors.New("invalid frame bound")
	}
	var prefix [4]byte
	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return 0, nil, err
	}
	size := binary.BigEndian.Uint32(prefix[:])
	maxSize := 2 // operation field, no payload
	if maxPayload > 0 {
		maxSize = 3 + len(binary.AppendUvarint(nil, uint64(maxPayload))) + maxPayload
	}
	if size < 2 || size > uint32(maxSize) {
		return 0, nil, fmt.Errorf("%w: length outside expected range", errMalformedFrame)
	}
	body := make([]byte, size)
	if _, err := io.ReadFull(r, body); err != nil {
		return 0, nil, err
	}
	if body[0] != opField || body[1] == 0 || body[1] >= 128 {
		return 0, nil, fmt.Errorf("%w: invalid operation field", errMalformedFrame)
	}
	if len(body) == 2 {
		return body[1], nil, nil
	}
	if body[2] != payloadField {
		return 0, nil, fmt.Errorf("%w: invalid payload field", errMalformedFrame)
	}
	length, n := binary.Uvarint(body[3:])
	if n <= 0 || length == 0 || length > uint64(maxPayload) ||
		n != len(binary.AppendUvarint(nil, length)) || length != uint64(len(body)-3-n) {
		return 0, nil, fmt.Errorf("%w: invalid payload length", errMalformedFrame)
	}
	return body[1], body[3+n:], nil
}

func expectEOF(r io.Reader) error {
	var extra [1]byte
	_, err := io.ReadFull(r, extra[:])
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("unexpected data after final storage frame")
}
