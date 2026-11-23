// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	manet "github.com/multiformats/go-multiaddr/net"
	ms "github.com/multiformats/go-multistream"
)

// ServeCatalog retains the original catalog provider entry point.
func (p *ReadProvider) ServeCatalog(ctx context.Context, socket net.Conn, identity crypto.PrivKey) error {
	return p.ServeRead(ctx, socket, identity)
}

// ServeRead takes ownership of an accepted TCP socket. Each negotiated
// stream gets its own request, authenticated replacement identity and nonce.
func (p *ReadProvider) ServeRead(ctx context.Context, socket net.Conn, identity crypto.PrivKey) error {
	defer socket.Close()
	if identity == nil || identity.Type() != crypto.Ed25519 {
		return errors.New("provider requires its enrolled Ed25519 identity")
	}
	local, err := identity.GetPublic().Raw()
	if err != nil || len(local) != ed25519.PublicKeySize || [32]byte(local) != p.agreement.ProviderKey {
		return errors.New("provider identity differs from its enrollment")
	}
	connCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	stop := context.AfterFunc(connCtx, func() { socket.Close() })
	defer stop()
	mux := *yamux.DefaultTransport
	mux.MaxIncomingStreams = 1 // one active read on this trial connection
	up, err := newLANUpgrader(identity, &mux)
	if err != nil {
		return err
	}
	raw, err := manet.WrapNetConn(socket)
	if err != nil {
		return err
	}
	handshake, finish := context.WithTimeout(connCtx, 10*time.Second)
	defer finish()
	negotiationDeadline, _ := handshake.Deadline()
	if err := socket.SetDeadline(negotiationDeadline); err != nil {
		return err
	}
	conn, err := up.Upgrade(handshake, nil, raw, network.DirInbound, "", &network.NullScope{})
	if err != nil {
		return err
	}
	defer conn.Close()
	finish()
	key := conn.RemotePublicKey()
	if key == nil || key.Type() != crypto.Ed25519 {
		return errors.New("replacement must authenticate an Ed25519 identity")
	}
	replacement, err := key.Raw()
	if err != nil || len(replacement) != ed25519.PublicKeySize {
		return errors.New("invalid authenticated replacement public key")
	}
	// Keep yamux alive until the peer closes. Closing the connection as soon
	// as a stream half-closes could discard its asynchronously queued reply.
	for {
		if err := socket.SetDeadline(negotiationDeadline); err != nil {
			return err
		}
		stream, err := conn.AcceptStream()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return connCtx.Err()
			}
			return err
		}
		if err := socket.SetDeadline(time.Time{}); err != nil {
			stream.Reset()
			return err
		}
		if err := p.serveStream(stream, [32]byte(replacement), negotiationDeadline, identity); err != nil {
			return err
		}
		negotiationDeadline = time.Now().Add(10 * time.Second)
	}
}

func (p *ReadProvider) serveStream(stream network.MuxedStream, replacement [32]byte, negotiationDeadline time.Time, identity crypto.PrivKey) error {
	defer stream.Close()
	if err := stream.SetDeadline(negotiationDeadline); err != nil {
		return err
	}
	protocols := ms.NewMultistreamMuxer[string]()
	protocols.AddHandler(storageProtocol, nil)
	if _, _, err := protocols.Negotiate(stream); err != nil {
		return err
	}
	firstFrameWindow := 30 * time.Second
	controlOnly := false
	if p.upload != nil {
		allowed, last := p.upload.enterStream(replacement)
		if !allowed {
			return writeFrame(stream, errorOp, []byte{4})
		}
		defer p.upload.leaveStream(replacement)
		if last {
			controlOnly = true
			firstFrameWindow = 2 * time.Second
		}
	}
	if err := stream.SetDeadline(time.Now().Add(firstFrameWindow)); err != nil {
		return err
	}
	op, request, err := readFrame(stream, uploadManifestSize)
	if err != nil {
		stream.Reset()
		return err
	}
	if controlOnly {
		switch op {
		case reserveOp, putOp, finalizeOp, readBegin:
			return writeFrame(stream, errorOp, []byte{4})
		case statusOp, grantUpdateOp:
			// A complete, supported control operation met the shorter deadline.
		default:
			stream.Reset()
			return errors.New("unrecognized operation in the control slot")
		}
		if err := stream.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}
	}
	if op == grantUpdateOp {
		return p.serveGrantUpdate(stream, request)
	}
	if op == reserveOp || op == statusOp || op == finalizeOp {
		return p.serveUpload(stream, replacement, op, request, identity.Sign)
	}
	if op == putOp {
		return p.servePut(stream, replacement, request)
	}
	if op != readBegin || len(request) != 73 || (request[0] != catalogKind && request[0] != objectKind) || binary.BigEndian.Uint64(request[33:41]) == 0 {
		return writeFrame(stream, errorOp, []byte{1})
	}
	// Only this goroutine ever sees this outstanding challenge. It reads
	// exactly one proof and checks for EOF; every exit destroys the challenge.
	var nonce [32]byte
	rand.Read(nonce[:])
	p.mu.RLock()
	issuedGeneration := p.generation
	p.mu.RUnlock()
	deadline := time.Now().Add(60 * time.Second)
	if err := writeFrame(stream, challengeOp, nonce[:]); err != nil {
		return err
	}
	if err := stream.SetReadDeadline(deadline); err != nil {
		return err
	}
	op, proof, err := readFrame(stream, ed25519.SignatureSize)
	if err != nil && time.Now().Before(deadline) && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		stream.Reset()
		return err
	}
	validProofFrame := err == nil && op == proofOp && len(proof) == ed25519.SignatureSize && expectEOF(stream) == nil
	if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	if !time.Now().Before(deadline) {
		return writeFrame(stream, errorOp, []byte{2})
	}
	if !validProofFrame {
		return writeFrame(stream, errorOp, []byte{1})
	}
	send := func(op byte, payload []byte) error {
		if err := stream.SetWriteDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return err
		}
		return writeFrame(stream, op, payload)
	}
	return p.releaseRead(send, request, nonce, replacement, proof, issuedGeneration, deadline)
}

type readableItem struct {
	io.ReadCloser
	size uint64
}

func (p *ReadProvider) authorizedItem(request []byte, nonce, replacement [32]byte, proof []byte, issuedGeneration uint64, deadline time.Time) (*readableItem, byte) {
	// Order the clock sample and the durable lookup with grant and upload updates.
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.offline {
		return nil, 7
	}
	now := time.Now()
	a := p.agreement
	if p.pendingRevocation || !now.Before(deadline) || p.recovery == [32]byte{} || p.generation != issuedGeneration ||
		p.generation != binary.BigEndian.Uint64(request[33:41]) || a.FolderID != [16]byte(request[1:17]) ||
		a.AgreementID != [16]byte(request[17:33]) ||
		!ed25519.Verify(p.recovery[:], readTranscript(a.ProviderKey, replacement, request, nonce), proof) {
		return nil, 2
	}
	if p.upload != nil && !p.upload.clockCertain(now) {
		return nil, 7
	}
	if !a.readable(now) {
		return nil, 2
	}
	if p.upload != nil {
		file, size, err := p.upload.store.OpenPublishedItem(request[0], [32]byte(request[41:73]), now.Unix())
		if errors.Is(err, grantstore.ErrUploadDenied) {
			return nil, 2
		}
		if err != nil {
			return nil, 7
		}
		return &readableItem{ReadCloser: file, size: size}, 0
	}
	var item *CommittedItem
	switch request[0] {
	case catalogKind:
		item = &p.catalog
	case objectKind:
		item = p.object
	}
	if item == nil || item.Locator != [32]byte(request[41:73]) {
		return nil, 2
	}
	// Locally detected damage makes the committed pair unavailable. Reveal
	// its condition only after authenticating the read and checking membership.
	if p.catalog.Sealed == nil || (p.object != nil && p.object.Sealed == nil) {
		return nil, 7
	}
	return &readableItem{ReadCloser: io.NopCloser(bytes.NewReader(item.Sealed)), size: uint64(len(item.Sealed))}, 0
}

func (p *ReadProvider) releaseRead(send func(byte, []byte) error, request []byte, nonce, replacement [32]byte, proof []byte, issuedGeneration uint64, deadline time.Time) error {
	item, code := p.authorizedItem(request, nonce, replacement, proof, issuedGeneration, deadline)
	if code != 0 {
		return send(errorOp, []byte{code})
	}
	defer item.Close()
	// A grant change can run between any two frames, including after PROOF
	// but before READ_SIZE. A pending writer has priority over new read locks.
	sendCurrent := func(op byte, payload []byte) error {
		p.mu.RLock()
		defer p.mu.RUnlock()
		now := time.Now()
		if p.offline || (p.upload != nil && !p.upload.clockCertain(now)) {
			return ErrGrantUnavailable
		}
		if p.pendingRevocation || p.generation != issuedGeneration || !p.agreement.readable(now) {
			return ErrReadDenied
		}
		return send(op, payload)
	}
	err := sendItem(sendCurrent, item)
	if err == ErrReadDenied {
		return send(errorOp, []byte{2})
	}
	if err == ErrGrantUnavailable {
		return send(errorOp, []byte{7})
	}
	return err
}

func sendItem(send func(byte, []byte) error, item *readableItem) error {
	if err := send(readSizeOp, binary.BigEndian.AppendUint64(nil, item.size)); err != nil {
		return err
	}
	buf := make([]byte, min(uint64(maxDataPayload), item.size))
	for remaining := item.size; remaining > 0; {
		frame := buf[:min(uint64(len(buf)), remaining)]
		if _, err := io.ReadFull(item, frame); err != nil {
			return err
		}
		if err := send(dataOp, frame); err != nil {
			return err
		}
		remaining -= uint64(len(frame))
	}
	return send(endOp, nil)
}
