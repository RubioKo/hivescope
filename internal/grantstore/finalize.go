// SPDX-License-Identifier: AGPL-3.0-only

package grantstore

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
)

const receiptDomain = "HiveScope/stored/v1\x00"

// VerifyUploadReceipt authenticates the original commit for this immutable
// manifest and agreement, independently of the provider's current health.
func VerifyUploadReceipt(receipt [132]byte, provider, fingerprint [32]byte, expiry int64) error {
	committed := binary.BigEndian.Uint64(receipt[52:60])
	if string(receipt[:20]) != receiptDomain || !bytes.Equal(receipt[20:52], fingerprint[:]) ||
		expiry <= 0 || committed == 0 || committed >= uint64(expiry) ||
		binary.BigEndian.Uint64(receipt[60:68]) != uint64(expiry) ||
		!ed25519.Verify(provider[:], receipt[:68], receipt[68:]) {
		return errors.New("invalid signed upload receipt")
	}
	return nil
}
