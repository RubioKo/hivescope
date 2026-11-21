// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/RubioKo/hivescope/internal/storageformat"
)

type uploadAdmission struct {
	store  *grantstore.Store
	writer [32]byte
	limits grantstore.UploadLimits
	opened time.Time // wall and monotonic readings from the trusted trial clock

	streamMu sync.Mutex
	byPeer   map[[32]byte]int
	active   int
}

// Leave the final slot for short control requests such as STATUS, even if
// other authenticated connections are waiting for their first frame.
func (u *uploadAdmission) enterStream(peer [32]byte) (allowed, controlOnly bool) {
	u.streamMu.Lock()
	defer u.streamMu.Unlock()
	if u.active == 16 || u.byPeer[peer] >= 2 {
		return false, false
	}
	if u.byPeer == nil {
		u.byPeer = make(map[[32]byte]int)
	}
	u.active++
	u.byPeer[peer]++
	return true, u.active == 16
}

func (u *uploadAdmission) leaveStream(peer [32]byte) {
	u.streamMu.Lock()
	defer u.streamMu.Unlock()
	u.active--
	u.byPeer[peer]--
	if u.byPeer[peer] == 0 {
		delete(u.byPeer, peer)
	}
}

// CreateUploadProvider provisions an empty provider for the first upload trial.
// The enrolled uploader and store ID are local inputs independent of the peer.
func CreateUploadProvider(dir string, storeID [32]byte, agreement CatalogAgreement, uploader [32]byte, limits grantstore.UploadLimits) (*ReadProvider, error) {
	return openUploadProvider(dir, storeID, agreement, uploader, limits, true)
}

// OpenUploadProvider reconciles existing reservations before accepting requests.
// Host limits may be changed locally when reopening, including pausing new work.
func OpenUploadProvider(dir string, storeID [32]byte, agreement CatalogAgreement, uploader [32]byte, limits grantstore.UploadLimits) (*ReadProvider, error) {
	return openUploadProvider(dir, storeID, agreement, uploader, limits, false)
}

func openUploadProvider(dir string, id [32]byte, a CatalogAgreement, uploader [32]byte, limits grantstore.UploadLimits, create bool) (*ReadProvider, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}
	if !storageformat.ValidEd25519Key(uploader[:]) || uploader == a.ProviderKey || uploader == a.AuthorityKey || a.ProviderKey == a.AuthorityKey {
		return nil, errors.New("upload enrollment requires independent provider, authority and uploader keys")
	}
	enrollment, err := a.storeBinding(id)
	if err != nil {
		return nil, err
	}
	scope := append([]byte("HiveScope/local-upload-store/v1\x00"), enrollment[:]...)
	binding := sha256.Sum256(append(scope, uploader[:]...))
	var s *grantstore.Store
	if create {
		s, err = grantstore.Create(dir, binding)
	} else {
		s, err = grantstore.Open(dir, binding)
	}
	if err != nil {
		return nil, err
	}
	var manifestScope [96]byte
	copy(manifestScope[:32], a.ProviderKey[:])
	copy(manifestScope[32:64], uploader[:])
	copy(manifestScope[64:80], a.FolderID[:])
	copy(manifestScope[80:], a.AgreementID[:])
	if create {
		err = s.InitUploads(manifestScope)
	} else {
		err = s.CheckUploads(manifestScope)
	}
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return &ReadProvider{agreement: a, upload: &uploadAdmission{store: s, writer: uploader, limits: limits, opened: time.Now()}}, nil
}

// UploadUsage reports retained request IDs and outstanding physical charges.
// It is a local operator measurement and is never returned on the wire.
func (p *ReadProvider) UploadUsage() (int, uint64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.offline || p.upload == nil {
		return 0, 0, ErrUploadUnavailable
	}
	return p.upload.store.UploadUsage()
}

func (p *ReadProvider) serveUpload(stream io.ReadWriter, peer [32]byte, op byte, request []byte) error {
	if err := expectEOF(stream); err != nil {
		return writeFrame(stream, errorOp, []byte{1})
	}
	state, code := p.admitUpload(peer, op, request)
	if code != 0 {
		return writeFrame(stream, errorOp, []byte{code})
	}
	reply := []byte{state.Code}
	if state.Code == 1 {
		reply = append(reply, state.Mask)
		reply = binary.BigEndian.AppendUint64(reply, uint64(state.Deadline))
	}
	return writeFrame(stream, stateOp, reply)
}

func (p *ReadProvider) admitUpload(peer [32]byte, op byte, request []byte) (grantstore.UploadState, byte) {
	// Serialize the clock sample with admission. A slower goroutine must not
	// submit an older time after another request advances the durable clock.
	p.mu.Lock()
	defer p.mu.Unlock()
	var state grantstore.UploadState
	u := p.upload
	if u == nil || peer != u.writer {
		return state, 2
	}
	if p.offline {
		return state, 7
	}
	a := p.agreement
	var manifest [uploadManifestSize]byte
	var ref [80]byte
	switch op {
	case reserveOp:
		if len(request) != len(manifest) {
			return state, 1
		}
		manifest = [uploadManifestSize]byte(request)
		if [32]byte(manifest[20:52]) != a.ProviderKey || [32]byte(manifest[52:84]) != peer {
			return state, 2
		}
		if validateUploadManifest(manifest) != nil {
			return state, 1
		}
		ref = uploadReference(manifest)
	case statusOp:
		if len(request) != len(ref) {
			return state, 1
		}
		ref = [80]byte(request)
	default:
		return state, 1
	}
	if [16]byte(ref[:16]) != a.FolderID || [16]byte(ref[16:32]) != a.AgreementID {
		return state, 2
	}
	now := time.Now()
	drift := (now.Unix() - u.opened.Unix()) - int64(now.Sub(u.opened)/time.Second)
	if drift > 2 || drift < -2 {
		return state, 7
	}
	if !a.readable(now) {
		return state, 2
	}
	var err error
	if op == statusOp {
		state, err = u.store.UploadStatus([16]byte(ref[32:48]), [32]byte(ref[48:]), now.Unix())
	} else {
		state, err = u.store.ReserveUpload(manifest, u.limits, now.Unix(), a.Expiry.Unix())
	}
	switch {
	case err == nil:
		return state, 0
	case errors.Is(err, grantstore.ErrUploadDenied):
		return state, 2
	case errors.Is(err, grantstore.ErrUploadConflict):
		return state, 3
	case errors.Is(err, grantstore.ErrUploadCapacity):
		return state, 5
	case errors.Is(err, grantstore.ErrUploadPaused):
		return state, 6
	default:
		return state, 7
	}
}
