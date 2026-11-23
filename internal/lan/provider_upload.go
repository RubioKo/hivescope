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
	p := &ReadProvider{agreement: a, upload: &uploadAdmission{store: s, writer: uploader, limits: limits, opened: time.Now()}}
	if err := p.restoreGrant(s); err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return p, nil
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

// UploadAccounting reports the local physical charge in each stage of publication.
func (p *ReadProvider) UploadAccounting() (reserved, stored uint64, err error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.offline || p.upload == nil {
		return 0, 0, ErrUploadUnavailable
	}
	return p.upload.store.UploadAccounting()
}

func (p *ReadProvider) serveUpload(stream io.ReadWriter, peer [32]byte, op byte, request []byte, sign func([]byte) ([]byte, error)) error {
	if err := expectEOF(stream); err != nil {
		return writeFrame(stream, errorOp, []byte{1})
	}
	state, code := p.admitUpload(peer, op, request, sign)
	if code != 0 {
		return writeFrame(stream, errorOp, []byte{code})
	}
	return writeUploadState(stream, state)
}

func writeUploadState(stream io.Writer, state grantstore.UploadState) error {
	reply := []byte{state.Code}
	if state.Code == 1 {
		reply = append(reply, state.Mask)
		reply = binary.BigEndian.AppendUint64(reply, uint64(state.Deadline))
	} else if state.Code == 2 || state.Code == 5 {
		reply = append(reply, state.Receipt[:]...)
	}
	return writeFrame(stream, stateOp, reply)
}

func (p *ReadProvider) admitUpload(peer [32]byte, op byte, request []byte, sign func([]byte) ([]byte, error)) (grantstore.UploadState, byte) {
	// Serialize the clock sample with admission. A slower goroutine must not
	// submit an older time after another request advances the durable clock.
	p.mu.Lock()
	defer p.mu.Unlock()
	var state grantstore.UploadState
	if code := p.checkUploader(peer); code != 0 {
		return state, code
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
	case statusOp, finalizeOp, cancelOp:
		if len(request) != len(ref) {
			return state, 1
		}
		ref = [80]byte(request)
	default:
		return state, 1
	}
	now := time.Now()
	if code := p.checkUploadReference(ref, now); code != 0 {
		return state, code
	}
	var err error
	u := p.upload
	switch op {
	case finalizeOp:
		state, err = u.store.FinalizeUpload([16]byte(ref[32:48]), [32]byte(ref[48:]), u.limits, now, a.Expiry.Unix(), sign)
	case cancelOp:
		state, err = u.store.CancelUpload([16]byte(ref[32:48]), [32]byte(ref[48:]), now.Unix())
	case statusOp:
		state, err = u.store.UploadStatus([16]byte(ref[32:48]), [32]byte(ref[48:]), now.Unix())
	case reserveOp:
		state, err = u.store.ReserveUpload(manifest, u.limits, now.Unix(), a.Expiry.Unix())
	}
	return state, uploadErrorCode(err)
}

func (p *ReadProvider) checkUploader(peer [32]byte) byte {
	if p.upload == nil || peer != p.upload.writer {
		return 2
	}
	if p.offline {
		return 7
	}
	return 0
}

func (p *ReadProvider) checkUploadReference(ref [80]byte, now time.Time) byte {
	a := p.agreement
	if [16]byte(ref[:16]) != a.FolderID || [16]byte(ref[16:32]) != a.AgreementID {
		return 2
	}
	if !p.upload.clockCertain(now) {
		return 7
	}
	if !a.readable(now) {
		return 2
	}
	return 0
}

func (u *uploadAdmission) clockCertain(now time.Time) bool {
	drift := (now.Unix() - u.opened.Unix()) - int64(now.Sub(u.opened)/time.Second)
	return drift >= -2 && drift <= 2
}

func uploadErrorCode(err error) byte {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, grantstore.ErrUploadInvalid):
		return 1
	case errors.Is(err, grantstore.ErrUploadDenied):
		return 2
	case errors.Is(err, grantstore.ErrUploadConflict):
		return 3
	case errors.Is(err, grantstore.ErrUploadBusy):
		return 4
	case errors.Is(err, grantstore.ErrUploadCapacity):
		return 5
	case errors.Is(err, grantstore.ErrUploadPaused):
		return 6
	default:
		return 7
	}
}
