// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

const (
	catalogMinSize = 243
	catalogMaxSize = 497
)

// ReadCatalog uses the unlocked package to authorize one catalog read on a
// pinned connection. Only a complete, locally verified catalog yields a snapshot.
func ReadCatalog(ctx context.Context, pkg *storageformat.RecoveryPackage, device crypto.PrivKey) (*storageformat.Snapshot, error) {
	var snapshot *storageformat.Snapshot
	err := WithCatalogChallenge(ctx, pkg, device, func(challenge [32]byte, stream network.MuxedStream) error {
		// The handshake proved possession of device and matched the provider
		// pin. Only the nonce comes from the peer; all other fields are local.
		raw, err := device.GetPublic().Raw()
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return errors.New("invalid authenticated replacement identity")
		}
		key := ed25519.NewKeyFromSeed(pkg.RecoverySeed[:])
		defer clear(key)
		proof := ed25519.Sign(key, readTranscript(pkg.ProviderKey, [32]byte(raw), catalogRequest(pkg), challenge))
		if err := writeFrame(stream, proofOp, proof); err != nil {
			return fmt.Errorf("send catalog proof: %w", err)
		}
		if err := stream.CloseWrite(); err != nil {
			return err
		}
		if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}
		sealed, err := receiveCatalog(stream)
		if err != nil {
			return fmt.Errorf("receive catalog: %w", err)
		}
		snapshot, err = storageformat.OpenCatalog(bytes.NewReader(sealed), pkg.Catalog)
		return err
	})
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// request is the exact READ_BEGIN payload: K followed by scope, generation
// and locator. The v1 transcript inserts both authenticated keys after K.
func readTranscript(provider, replacement [32]byte, request []byte, challenge [32]byte) []byte {
	return slices.Concat([]byte("HiveScope/recovery-read/v1\x00"), request[:1],
		provider[:], replacement[:], request[1:], challenge[:])
}

func receiveCatalog(r io.Reader) ([]byte, error) {
	op, sizeField, err := readFrame(r, 8)
	if err != nil {
		return nil, err
	}
	if op == errorOp {
		if err := expectEOF(r); err != nil {
			return nil, err
		}
		return nil, providerError(sizeField)
	}
	if op != readSizeOp || len(sizeField) != 8 {
		return nil, errors.New("expected catalog READ_SIZE")
	}
	size := binary.BigEndian.Uint64(sizeField)
	if size < catalogMinSize || size > catalogMaxSize {
		return nil, errors.New("catalog size outside v1 limits")
	}
	// A complete v1 catalog fits in a single DATA frame.
	op, sealed, err := readFrame(r, int(size))
	if err != nil {
		return nil, err
	}
	if op != dataOp || len(sealed) != int(size) {
		return nil, errors.New("catalog DATA differs from declared size")
	}
	op, _, err = readFrame(r, 0)
	if err != nil {
		return nil, err
	}
	if op != endOp {
		return nil, errors.New("expected catalog END")
	}
	if err := expectEOF(r); err != nil {
		return nil, err
	}
	return sealed, nil
}
