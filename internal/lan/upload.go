// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net/netip"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

const (
	reserveOp          = 0x01
	statusOp           = 0x02
	stateOp            = 0x23
	uploadManifestSize = 276
	uploadDomain       = "HiveScope/upload/v1\x00"
)

var (
	ErrUploadDenied      = grantstore.ErrUploadDenied
	ErrUploadUnavailable = errors.New("provider cannot determine the upload state")
)

// UploadItem describes the exact sealed object or signed catalog. Digest is
// SHA-256 of all the bytes to be uploaded at the opaque locator.
type UploadItem struct {
	Locator, Digest [32]byte
	Size            uint64
}

// NewUploadManifest encodes a single reservation for both sealed items. The
// uploader supplies a fresh random request ID and saves this manifest with its
// source bytes before making the first request. A retry uses the saved bytes.
func NewUploadManifest(provider, uploader [32]byte, folder, agreement, request [16]byte,
	object, catalog UploadItem) ([uploadManifestSize]byte, error) {
	var m [uploadManifestSize]byte
	copy(m[:20], uploadDomain)
	copy(m[20:52], provider[:])
	copy(m[52:84], uploader[:])
	copy(m[84:100], folder[:])
	copy(m[100:116], agreement[:])
	copy(m[116:132], request[:])
	copy(m[132:164], object.Locator[:])
	binary.BigEndian.PutUint64(m[164:172], object.Size)
	copy(m[172:204], object.Digest[:])
	copy(m[204:236], catalog.Locator[:])
	binary.BigEndian.PutUint64(m[236:244], catalog.Size)
	copy(m[244:276], catalog.Digest[:])
	if err := validateUploadManifest(m); err != nil {
		return [uploadManifestSize]byte{}, err
	}
	return m, nil
}

// validateUploadManifest checks the v1 wire fields independent of the local
// enrollment. The provider must also match both identities and the agreement.
func validateUploadManifest(m [uploadManifestSize]byte) error {
	if !storageformat.ValidEd25519Key(m[20:52]) || !storageformat.ValidEd25519Key(m[52:84]) ||
		[32]byte(m[20:52]) == [32]byte(m[52:84]) {
		return errors.New("upload manifest requires distinct valid provider and uploader keys")
	}
	_, _, err := grantstore.UploadItemSizes(m)
	return err
}

func uploadReference(m [uploadManifestSize]byte) [80]byte {
	var ref [80]byte
	copy(ref[:48], m[84:132]) // folder, agreement, request
	digest := sha256.Sum256(m[:])
	copy(ref[48:], digest[:])
	return ref
}

// SubmitReservation asks the pinned provider to reserve both items described
// by the uploader's saved manifest. If the reply is lost, keep the same
// manifest and query ReservationStatus before retrying. Success with a pending
// state reserves capacity, without confirming that either item is stored.
func SubmitReservation(ctx context.Context, endpoint netip.AddrPort, pin [32]byte, device crypto.PrivKey,
	manifest [uploadManifestSize]byte) (grantstore.UploadState, error) {
	return requestReservation(ctx, endpoint, pin, device, manifest, reserveOp)
}

// ReservationStatus asks for the state of a saved upload without allocating
// capacity. An unknown state permits retrying the original manifest and ID.
func ReservationStatus(ctx context.Context, endpoint netip.AddrPort, pin [32]byte, device crypto.PrivKey,
	manifest [uploadManifestSize]byte) (grantstore.UploadState, error) {
	return requestReservation(ctx, endpoint, pin, device, manifest, statusOp)
}

func requestReservation(ctx context.Context, endpoint netip.AddrPort, pin [32]byte, device crypto.PrivKey,
	m [uploadManifestSize]byte, op byte) (grantstore.UploadState, error) {
	var state grantstore.UploadState
	if err := checkUploadIdentity(pin, device, m); err != nil {
		return state, err
	}
	payload := m[:]
	if op == statusOp {
		ref := uploadReference(m)
		payload = ref[:]
	}
	err := withStorageStream(ctx, endpoint, pin, device, func(stream network.MuxedStream) error {
		if err := writeFrame(stream, op, payload); err != nil {
			return err
		}
		if err := stream.CloseWrite(); err != nil {
			return err
		}
		var err error
		state, err = readReservationState(stream, op)
		return err
	})
	if err != nil {
		return grantstore.UploadState{}, err
	}
	return state, nil
}

func checkUploadIdentity(pin [32]byte, device crypto.PrivKey, m [uploadManifestSize]byte) error {
	if err := validateUploadManifest(m); err != nil {
		return err
	}
	if [32]byte(m[20:52]) != pin {
		return ErrProviderIdentity
	}
	if device == nil || device.Type() != crypto.Ed25519 {
		return ErrUploadDenied
	}
	uploader, err := device.GetPublic().Raw()
	if err != nil || len(uploader) != ed25519.PublicKeySize || [32]byte(uploader) != [32]byte(m[52:84]) {
		return ErrUploadDenied
	}
	return nil
}

// This admission client accepts states that require no signed receipt. Future
// committed and unavailable states must be verified by the receipt path.
func readReservationState(r io.Reader, requestOp byte) (grantstore.UploadState, error) {
	var state grantstore.UploadState
	op, payload, err := readFrame(r, 10)
	if err != nil {
		return state, err
	}
	if err := expectEOF(r); err != nil {
		return state, err
	}
	return decodeUploadState(requestOp, op, payload)
}

func decodeUploadState(requestOp, op byte, payload []byte) (grantstore.UploadState, error) {
	var state grantstore.UploadState
	if op == errorOp {
		if len(payload) == 1 {
			switch payload[0] {
			case 1:
				return state, grantstore.ErrUploadInvalid
			case 2:
				return state, ErrUploadDenied
			case 3:
				return state, grantstore.ErrUploadConflict
			case 4:
				return state, grantstore.ErrUploadBusy
			case 5:
				return state, grantstore.ErrUploadCapacity
			case 6:
				return state, grantstore.ErrUploadPaused
			case 7:
				return state, ErrUploadUnavailable
			}
		}
		return state, providerError(payload)
	}
	if op != stateOp || len(payload) == 0 {
		return state, errors.New("expected a reservation state")
	}
	switch payload[0] {
	case 0:
		if (requestOp == statusOp || requestOp == putOp) && len(payload) == 1 {
			return state, nil
		}
	case 1:
		if len(payload) == 10 && payload[1] <= 3 {
			// Absolute time belongs to the provider. The client's wall clock
			// cannot decide whether a reservation has expired.
			deadline := binary.BigEndian.Uint64(payload[2:])
			if deadline > 0 && deadline <= math.MaxInt64 {
				return grantstore.UploadState{Code: 1, Mask: payload[1], Deadline: int64(deadline)}, nil
			}
		}
	case 3, 4:
		if len(payload) == 1 {
			return grantstore.UploadState{Code: payload[0]}, nil
		}
	}
	return state, errors.New("unexpected or malformed reservation state")
}
