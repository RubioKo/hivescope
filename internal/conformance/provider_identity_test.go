// SPDX-License-Identifier: AGPL-3.0-only

package conformance

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/sec"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
	"github.com/libp2p/go-libp2p/p2p/net/upgrader"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	manet "github.com/multiformats/go-multiaddr/net"
	ms "github.com/multiformats/go-multistream"
)

// Count the provider's TLS handshakes even if the client rejects its
// identity before a multiplexed connection is established.
type observedTLS struct {
	sec.SecureTransport
	inbound int
}

func (s *observedTLS) SecureInbound(ctx context.Context, raw net.Conn, id peer.ID) (sec.SecureConn, error) {
	s.inbound++
	return s.SecureTransport.SecureInbound(ctx, raw, id)
}

type catalogObservation struct {
	tlsAttempts, storageStreams, applicationBytes, catalogRequests int
	authenticated                                                  bool
}

const fixtureStorageProtocol = "/hivescope/storage/1.0.0"

func serveCatalogChallenge(ctx context.Context, listener *net.TCPListener, provider crypto.PrivKey, device peer.ID,
	supported string, challenge [32]byte, observation *catalogObservation) error {
	muxers := []upgrader.StreamMuxer{{ID: yamux.ID, Muxer: yamux.DefaultTransport}}
	secure, err := libp2ptls.New(libp2ptls.ID, provider, muxers)
	if err != nil {
		return err
	}
	probe := &observedTLS{SecureTransport: secure}
	defer func() { observation.tlsAttempts = probe.inbound }()
	up, err := upgrader.New([]sec.SecureTransport{probe}, muxers, nil, nil, nil)
	if err != nil {
		return err
	}
	raw, err := listener.AcceptTCP()
	if err != nil {
		return err
	}
	defer raw.Close()
	stop := context.AfterFunc(ctx, func() { raw.Close() })
	defer stop()
	conn, err := manet.WrapNetConn(raw)
	if err != nil {
		return err
	}
	accepted, err := up.Upgrade(ctx, nil, conn, network.DirInbound, device, &network.NullScope{})
	if err != nil {
		return err
	}
	defer accepted.Close()
	observation.authenticated = true
	stream, err := accepted.AcceptStream()
	if err != nil {
		return err
	}
	defer stream.Close()
	protocols := ms.NewMultistreamMuxer[string]()
	protocols.AddHandler(supported, nil)
	// V1 negotiation sends two protocol names, each with a one-byte length
	// and a newline. Watch the very next byte too: a pipelined application
	// write must be detected even if this provider declines the protocol.
	controlBytes := int64(2 + len(ms.ProtocolID) + 2 + len(fixtureStorageProtocol))
	input := &io.LimitedReader{R: stream, N: controlBytes + 1}
	counted := struct {
		io.Reader
		io.WriteCloser
	}{input, stream}
	if _, _, err := protocols.Negotiate(counted); err != nil {
		if input.N == 0 {
			observation.applicationBytes++ // first byte beyond the v1 offer
		}
		return err
	}
	observation.storageStreams++

	// An independent encoding of the documented READ_BEGIN for the catalog.
	want := []byte{0, 0, 0, 77, 0x08, 0x07, 0x12, 73, 0x01}
	want = append(want, repeated(0xa1, 16)...)
	want = append(want, repeated(0xb2, 16)...)
	want = binary.BigEndian.AppendUint64(want, 7)
	want = append(want, repeated(0xc3, 32)...)
	got := make([]byte, len(want))
	n, err := io.ReadFull(stream, got)
	observation.applicationBytes += n
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return errors.New("incorrect catalog request frame")
	}
	observation.catalogRequests++
	reply := append([]byte{0, 0, 0, 36, 0x08, 0x25, 0x12, 32}, challenge[:]...)
	if _, err := stream.Write(reply); err != nil {
		return err
	}
	// Keep the session open until the client finishes its callback and
	// closes the stream. This test stops before submitting a recovery proof.
	n, err = stream.Read(got[:1])
	observation.applicationBytes += n
	if n != 0 || err == nil {
		return errors.New("unexpected additional application bytes after challenge")
	}
	return ctx.Err()
}

func TestCatalogReadPinsProviderAtSameAddress(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().(*net.TCPAddr).AddrPort()
	newIdentity := func() crypto.PrivKey {
		t.Helper()
		key, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	owner, providerA, providerB := newIdentity(), newIdentity(), newIdentity()
	device, err := peer.IDFromPrivateKey(owner)
	if err != nil {
		t.Fatal(err)
	}
	phrase := rand.Text()
	sealFor := func(provider crypto.PrivKey, variant byte) []byte {
		t.Helper()
		pin, err := provider.GetPublic().Raw()
		if err != nil {
			t.Fatal(err)
		}
		f := recoveryFixture(t, phrase, variant)
		copy(f.payload[40:72], pin)
		v4 := address.Addr().As4()
		copy(f.payload[72:76], v4[:])
		binary.BigEndian.PutUint16(f.payload[76:78], address.Port())
		return f.seal()
	}
	exportA := sealFor(providerA, 30)
	exportB := sealFor(providerB, 31)
	for _, tc := range []struct {
		name     string
		export   []byte
		provider crypto.PrivKey
		protocol string
		wantErr  error
	}{
		{"pinned provider A", exportA, providerA, fixtureStorageProtocol, nil},
		{"same package and address now served by B", exportA, providerB, fixtureStorageProtocol, lan.ErrProviderIdentity},
		{"provider B with its own pin", exportB, providerB, fixtureStorageProtocol, nil},
		{"pinned provider A offers only v2", exportA, providerA, "/hivescope/storage/2.0.0", ms.ErrNotSupported[string]{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The listener fixes the address, including the TCP port. Each
			// accepted connection belongs to a fresh provider transport.
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			deadline, _ := ctx.Deadline()
			if err := listener.SetDeadline(deadline); err != nil {
				t.Fatal(err)
			}
			var challenge [32]byte
			rand.Read(challenge[:])
			var observed catalogObservation
			served := make(chan error, 1)
			go func() {
				served <- serveCatalogChallenge(ctx, listener, tc.provider, device, tc.protocol, challenge, &observed)
			}()
			callbacks := 0
			err := storageformat.WithRecoveryPackage(bytes.NewReader(tc.export), []byte(phrase), func(p *storageformat.RecoveryPackage) error {
				if p.Endpoint != netip.AddrPortFrom(address.Addr().Unmap(), address.Port()) {
					return errors.New("recovery package names a different endpoint")
				}
				return lan.WithCatalogChallenge(ctx, p, owner, func(got [32]byte, _ network.MuxedStream) error {
					callbacks++
					if got != challenge {
						return errors.New("wrong challenge")
					}
					return nil
				})
			})
			if err != nil {
				// A client-side failure before TCP connect must wake Accept.
				listener.SetDeadline(time.Now())
			}
			serverErr := <-served
			if observed.tlsAttempts != 1 || ctx.Err() != nil {
				t.Fatalf("expected one live TLS attempt at the selected address: %+v, client=%v, provider=%v", observed, err, serverErr)
			}
			if tc.wantErr == nil {
				if err != nil || serverErr != nil || observed.storageStreams != 1 || observed.applicationBytes != 81 ||
					observed.catalogRequests != 1 || callbacks != 1 {
					t.Fatalf("pinned provider: %+v, callbacks=%d, client=%v, provider=%v", observed, callbacks, err, serverErr)
				}
				return
			}
			wantAuthenticated := !errors.Is(tc.wantErr, lan.ErrProviderIdentity)
			if !errors.Is(err, tc.wantErr) || observed.authenticated != wantAuthenticated || serverErr == nil || observed.storageStreams != 0 ||
				observed.applicationBytes != 0 || observed.catalogRequests != 0 || callbacks != 0 {
				t.Fatalf("rejected exchange: %+v, callbacks=%d, client=%v, provider=%v", observed, callbacks, err, serverErr)
			}
		})
	}
}
