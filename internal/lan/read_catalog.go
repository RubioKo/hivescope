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
	if pkg == nil {
		return nil, errors.New("missing unlocked recovery package")
	}
	var snapshot *storageformat.Snapshot
	err := withAuthorizedRead(ctx, pkg, device, catalogRequest(pkg), func(r io.Reader) error {
		sealed, err := receiveCatalog(r)
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

// Network progress renews the idle timeout. The enclosing connection still
// enforces the overall exchange deadline, including on a trickling response.
type idleReadStream struct{ network.MuxedStream }

func (s idleReadStream) Read(p []byte) (int, error) {
	if err := s.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return 0, err
	}
	return s.MuxedStream.Read(p)
}

func withAuthorizedRead(ctx context.Context, pkg *storageformat.RecoveryPackage, device crypto.PrivKey,
	request []byte, receive func(io.Reader) error) error {
	return withReadChallenge(ctx, pkg, device, request, func(challenge [32]byte, stream network.MuxedStream) error {
		// The handshake proved possession of device and matched the provider
		// pin. Only the nonce comes from the peer; all other fields are local.
		raw, err := device.GetPublic().Raw()
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return errors.New("invalid authenticated replacement identity")
		}
		key := ed25519.NewKeyFromSeed(pkg.RecoverySeed[:])
		defer clear(key)
		proof := ed25519.Sign(key, readTranscript(pkg.ProviderKey, [32]byte(raw), request, challenge))
		if err := writeFrame(stream, proofOp, proof); err != nil {
			return fmt.Errorf("send read proof: %w", err)
		}
		if err := stream.CloseWrite(); err != nil {
			return err
		}
		return receive(idleReadStream{stream})
	})
}

// request is the exact READ_BEGIN payload: K followed by scope, generation
// and locator. The v1 transcript inserts both authenticated keys after K.
func readTranscript(provider, replacement [32]byte, request []byte, challenge [32]byte) []byte {
	return slices.Concat([]byte("HiveScope/recovery-read/v1\x00"), request[:1],
		provider[:], replacement[:], request[1:], challenge[:])
}

func readSealedSize(r io.Reader) (uint64, error) {
	op, sizeField, err := readFrame(r, 8)
	if err != nil {
		return 0, err
	}
	if op == errorOp {
		if err := expectEOF(r); err != nil {
			return 0, err
		}
		return 0, providerError(sizeField)
	}
	if op != readSizeOp || len(sizeField) != 8 {
		return 0, errors.New("expected READ_SIZE")
	}
	return binary.BigEndian.Uint64(sizeField), nil
}

func receiveCatalog(r io.Reader) ([]byte, error) {
	size, err := readSealedSize(r)
	if err != nil {
		return nil, err
	}
	if size < catalogMinSize || size > catalogMaxSize {
		return nil, errors.New("catalog size outside v1 limits")
	}
	// At most 497 bytes. Never return a partial catalog on error.
	sealed, err := io.ReadAll(&sealedItemReader{wire: r, remaining: size})
	if err != nil {
		return nil, err
	}
	return sealed, nil
}
