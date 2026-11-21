// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

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
		if err := p.serveStream(stream, [32]byte(replacement), negotiationDeadline); err != nil {
			return err
		}
		negotiationDeadline = time.Now().Add(10 * time.Second)
	}
}

func (p *ReadProvider) serveStream(stream network.MuxedStream, replacement [32]byte, negotiationDeadline time.Time) error {
	defer stream.Close()
	if err := stream.SetDeadline(negotiationDeadline); err != nil {
		return err
	}
	protocols := ms.NewMultistreamMuxer[string]()
	protocols.AddHandler(storageProtocol, nil)
	if _, _, err := protocols.Negotiate(stream); err != nil {
		return err
	}
	if err := stream.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	op, request, err := readFrame(stream, grantRecordSize)
	if err != nil {
		stream.Reset()
		return err
	}
	if op == grantUpdateOp {
		return p.serveGrantUpdate(stream, request)
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

func (p *ReadProvider) authorizedItem(request []byte, nonce, replacement [32]byte, proof []byte, issuedGeneration uint64, deadline time.Time) ([]byte, byte) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.offline {
		return nil, 7
	}
	now := time.Now()
	a := p.agreement
	if !now.Before(deadline) || !a.readable(now) || p.recovery == [32]byte{} || p.generation != issuedGeneration ||
		p.generation != binary.BigEndian.Uint64(request[33:41]) || a.FolderID != [16]byte(request[1:17]) ||
		a.AgreementID != [16]byte(request[17:33]) ||
		!ed25519.Verify(p.recovery[:], readTranscript(a.ProviderKey, replacement, request, nonce), proof) {
		return nil, 2
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
	return item.Sealed, 0
}

func (p *ReadProvider) releaseRead(send func(byte, []byte) error, request []byte, nonce, replacement [32]byte, proof []byte, issuedGeneration uint64, deadline time.Time) error {
	sealed, code := p.authorizedItem(request, nonce, replacement, proof, issuedGeneration, deadline)
	if code != 0 {
		return send(errorOp, []byte{code})
	}
	// A grant change can run between any two frames, including after PROOF
	// but before READ_SIZE. A pending writer has priority over new read locks.
	sendCurrent := func(op byte, payload []byte) error {
		p.mu.RLock()
		defer p.mu.RUnlock()
		if p.offline {
			return ErrGrantUnavailable
		}
		if p.generation != issuedGeneration || !p.agreement.readable(time.Now()) {
			return ErrReadDenied
		}
		return send(op, payload)
	}
	err := sendItem(sendCurrent, sealed)
	if err == ErrReadDenied {
		return send(errorOp, []byte{2})
	}
	if err == ErrGrantUnavailable {
		return send(errorOp, []byte{7})
	}
	return err
}

func sendItem(send func(byte, []byte) error, sealed []byte) error {
	if err := send(readSizeOp, binary.BigEndian.AppendUint64(nil, uint64(len(sealed)))); err != nil {
		return err
	}
	for len(sealed) > 0 {
		n := min(maxDataPayload, len(sealed))
		if err := send(dataOp, sealed[:n]); err != nil {
			return err
		}
		sealed = sealed[n:]
	}
	return send(endOp, nil)
}
