// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

func TestCancelPendingNegotiation(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if err := listener.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	device, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, provider, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pin, err := provider.Raw()
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan error, 1)
	go func() {
		socket, err := listener.Accept()
		if err != nil {
			accepted <- err
			return
		}
		defer socket.Close()
		if err := socket.SetDeadline(deadline); err != nil {
			accepted <- err
			return
		}
		// Cancel only after the client started negotiating. Leave the peer
		// silent to make the cancellation responsible for ending the read.
		var b [1]byte
		_, err = io.ReadFull(socket, b[:])
		cancel()
		if err == nil {
			_, err = io.Copy(io.Discard, socket)
		}
		accepted <- err
	}()
	pkg := storageformat.RecoveryPackage{
		Endpoint:    listener.Addr().(*net.TCPAddr).AddrPort(),
		Generation:  1,
		ProviderKey: [32]byte(pin),
	}
	err = WithCatalogChallenge(ctx, &pkg, device, func([32]byte, network.MuxedStream) error {
		t.Fatal("a cancelled handshake reached the catalog continuation")
		return nil
	})
	listener.Close()
	if peerErr := <-accepted; !errors.Is(err, context.Canceled) || peerErr != nil {
		t.Fatalf("cancellation: client=%v peer=%v", err, peerErr)
	}
}
