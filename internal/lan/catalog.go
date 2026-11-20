// SPDX-License-Identifier: AGPL-3.0-only

// Package lan reads a HiveScope recovery catalog on a direct, authenticated LAN connection.
package lan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/sec"
	"github.com/libp2p/go-libp2p/core/transport"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	manet "github.com/multiformats/go-multiaddr/net"
	multistream "github.com/multiformats/go-multistream"
)

const storageProtocol = "/hivescope/storage/1.0.0"

// ErrProviderIdentity means the endpoint proved a different peer identity.
var ErrProviderIdentity = errors.New("provider identity differs from the recovery package")

// WithCatalogChallenge starts a catalog read inside WithRecoveryPackage's
// owner-side continuation. device is the replacement's own Ed25519 identity.
// Only the pinned provider receives READ_BEGIN. On a well-formed CHALLENGE,
// use can continue the exchange on that very stream before this call returns.
// Receiving a challenge does not authorize a read or confirm a recovery.
func WithCatalogChallenge(ctx context.Context, pkg *storageformat.RecoveryPackage, device crypto.PrivKey,
	use func([32]byte, network.MuxedStream) error) (err error) {
	if pkg == nil || use == nil || device == nil || device.Type() != crypto.Ed25519 {
		return errors.New("catalog read requires an unlocked package, an Ed25519 device identity and a continuation")
	}
	if !pkg.Endpoint.Addr().Is4() || pkg.Endpoint.Port() == 0 || pkg.Generation == 0 {
		return errors.New("invalid endpoint or grant generation")
	}
	exchange, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	// Closing TCP to interrupt an outstanding operation can surface as a
	// socket error. Preserve the caller's cancellation as a cause as well.
	defer func() {
		if cause := exchange.Err(); err != nil && cause != nil {
			err = errors.Join(err, cause)
		}
	}()
	handshake, endHandshake := context.WithTimeout(exchange, 10*time.Second)
	defer endHandshake()
	var dialer net.Dialer
	socket, err := dialer.DialContext(handshake, "tcp4", pkg.Endpoint.String())
	if err != nil {
		return fmt.Errorf("connect to provider: %w", err)
	}
	defer socket.Close()
	// TLS negotiation takes a context. Closing this dedicated socket also
	// interrupts multistream and application I/O when the caller cancels.
	stop := context.AfterFunc(exchange, func() { socket.Close() })
	defer stop()
	conn, err := authenticateProvider(handshake, socket, device, pkg.ProviderKey)
	if err != nil {
		return err
	}
	defer conn.Close()
	stream, err := conn.OpenStream(handshake)
	if err != nil {
		return fmt.Errorf("open storage stream: %w", err)
	}
	defer stream.Close()
	handshakeDeadline, _ := handshake.Deadline()
	if err := stream.SetDeadline(handshakeDeadline); err != nil {
		return err
	}
	if err := multistream.SelectProtoOrFail(storageProtocol, stream); err != nil {
		return fmt.Errorf("negotiate storage protocol: %w", err)
	}
	if err := handshake.Err(); err != nil {
		return err
	}
	endHandshake()
	if err := stream.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	if err := writeCatalogRead(stream, pkg); err != nil {
		return fmt.Errorf("send catalog READ_BEGIN: %w", err)
	}
	if err := stream.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}
	challenge, err := readChallenge(stream)
	if err != nil {
		return fmt.Errorf("receive catalog challenge: %w", err)
	}
	if err := stream.SetDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return err
	}
	if err := exchange.Err(); err != nil {
		return err
	}
	return use(challenge, stream)
}

func authenticateProvider(ctx context.Context, socket net.Conn, device crypto.PrivKey, pin [32]byte) (transport.CapableConn, error) {
	expectedKey, err := crypto.UnmarshalEd25519PublicKey(pin[:])
	if err != nil {
		return nil, err
	}
	expectedPeer, err := peer.IDFromPublicKey(expectedKey)
	if err != nil {
		return nil, err
	}
	mux := *yamux.DefaultTransport
	mux.MaxIncomingStreams = 0 // Only the requester opens streams on this connection.
	upgrade, err := newLANUpgrader(device, &mux)
	if err != nil {
		return nil, err
	}
	raw, err := manet.WrapNetConn(socket)
	if err != nil {
		return nil, err
	}
	// TLS verifies possession of the libp2p identity against the expected
	// peer ID during the handshake. Never learn that ID from the endpoint.
	conn, err := upgrade.Upgrade(ctx, nil, raw, network.DirOutbound, expectedPeer, &network.NullScope{})
	if err != nil {
		var mismatch sec.ErrPeerIDMismatch
		if errors.As(err, &mismatch) {
			return nil, ErrProviderIdentity
		}
		return nil, fmt.Errorf("authenticate provider connection: %w", err)
	}
	// Compare the identity key (not the rotating TLS certificate's key)
	// on the same connection from which the storage stream will be opened.
	if !expectedKey.Equals(conn.RemotePublicKey()) {
		conn.Close()
		return nil, ErrProviderIdentity
	}
	return conn, nil
}

func newLANUpgrader(identity crypto.PrivKey, mux *yamux.Transport) (transport.Upgrader, error) {
	muxers := []upgrader.StreamMuxer{{ID: yamux.ID, Muxer: mux}}
	tls, err := libp2ptls.New(libp2ptls.ID, identity, muxers)
	if err != nil {
		return nil, err
	}
	return upgrader.New([]sec.SecureTransport{tls}, muxers, nil, nil, nil)
}
