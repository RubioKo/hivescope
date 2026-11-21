// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/RubioKo/hivescope/internal/storageformat"
)

const (
	opField       = 0x08
	payloadField  = 0x12
	grantUpdateOp = 0x06
	readBegin     = 0x07
	proofOp       = 0x08
	dataOp        = 0x21
	endOp         = 0x22
	grantAckOp    = 0x24
	challengeOp   = 0x25
	readSizeOp    = 0x26
	errorOp       = 0x7f
	catalogKind   = 0x01
	objectKind    = 0x02
)

// ErrReadDenied is the authenticated provider's refusal to authorize the read.
var ErrReadDenied = errors.New("provider denied the read")

func catalogRequest(pkg *storageformat.RecoveryPackage) []byte {
	return readRequest(pkg, catalogKind, pkg.Catalog.CatalogLocator)
}

func readRequest(pkg *storageformat.RecoveryPackage, kind byte, locator [32]byte) []byte {
	request := append([]byte{kind}, pkg.Catalog.FolderID[:]...)
	request = append(request, pkg.Catalog.AgreementID[:]...)
	request = binary.BigEndian.AppendUint64(request, pkg.Generation)
	return append(request, locator[:]...)
}

func readChallenge(r io.Reader) ([32]byte, error) {
	op, payload, err := readFrame(r, 32)
	if err != nil {
		return [32]byte{}, err
	}
	if op == errorOp {
		return [32]byte{}, providerError(payload)
	}
	if op != challengeOp || len(payload) != 32 {
		return [32]byte{}, errors.New("expected CHALLENGE for the pending read")
	}
	return [32]byte(payload), nil
}

func providerError(payload []byte) error {
	if len(payload) != 1 || payload[0] < 1 || payload[0] > 7 {
		return errors.New("malformed provider error")
	}
	if payload[0] == 2 {
		return ErrReadDenied
	}
	return fmt.Errorf("provider returned storage error code 0x%02x", payload[0])
}
