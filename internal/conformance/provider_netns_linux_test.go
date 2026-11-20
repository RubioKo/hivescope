// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android

package conformance

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// scripts/test-provider-netns.sh runs each side as a separate process in
// its own Linux network namespace. The script switches the requester's route.
func TestCatalogPinAcrossNetworkHosts(t *testing.T) {
	role := os.Getenv("HIVESCOPE_NETNS_ROLE")
	if role == "" {
		t.Skip("run bash scripts/test-provider-netns.sh for the isolated network trial")
	}
	host, pin := os.Getenv("HIVESCOPE_NETNS_HOST"), os.Getenv("HIVESCOPE_NETNS_PIN")
	if (host != "A" && host != "B") || (pin != "A" && pin != "B") {
		t.Fatal("expected synthetic hosts A or B")
	}
	keyFromSeed := func(seed []byte) crypto.PrivKey {
		t.Helper()
		key, err := crypto.UnmarshalEd25519PrivateKey(ed25519.NewKeyFromSeed(seed))
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	device := keyFromSeed(repeated(0x91, 32))
	alternate := keyFromSeed(repeated(0x92, 32))
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	switch role {
	case "provider":
		provider := alternate
		if host == "A" {
			// Public RFC 8032 section 7.1 TEST 2, the provider in HSRP's
			// published five-byte example. This seed is only a test fixture.
			provider = keyFromSeed(mustHex(t, "4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb"))
		}
		deviceID, err := peer.IDFromPrivateKey(device)
		if err != nil {
			t.Fatal(err)
		}
		listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 4001})
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		deadline, _ := ctx.Deadline()
		if err := listener.SetDeadline(deadline); err != nil {
			t.Fatal(err)
		}
		fmt.Println("HiveScope namespace provider ready")
		var challenge [32]byte
		rand.Read(challenge[:])
		var got catalogObservation
		err = serveCatalogChallenge(ctx, listener, provider, deviceID, fixtureStorageProtocol, challenge, &got)
		want := catalogObservation{tlsAttempts: 1}
		if pin == host {
			want.authenticated, want.storageStreams, want.applicationBytes, want.catalogRequests = true, 1, 81, 1
		}
		if got != want || (err == nil) != (pin == host) || ctx.Err() != nil {
			t.Fatalf("host %s, package pin %s: got %+v, want %+v, error %v", host, pin, got, want, err)
		}
		t.Logf("host %s, package pin %s: TLS attempts %d, application bytes %d, catalog requests %d",
			host, pin, got.tlsAttempts, got.applicationBytes, got.catalogRequests)

	case "requester":
		const passphrase = "HiveScope fixture 1"
		var export []byte
		if pin == "A" {
			export = recoveryFixture(t, passphrase, 0).seal()
			requireDigest(t, export, "943561b2a66368c9f780b44fab56aa89a1d46c57b20a7d3b73b39dc090d548b4")
		} else {
			f := recoveryFixture(t, passphrase, 42)
			public, err := alternate.GetPublic().Raw()
			if err != nil {
				t.Fatal(err)
			}
			copy(f.payload[40:72], public)
			export = f.seal()
		}
		var callbacks int
		err := storageformat.WithRecoveryPackage(bytes.NewReader(export), []byte(passphrase), func(p *storageformat.RecoveryPackage) error {
			return lan.WithCatalogChallenge(ctx, p, device, func([32]byte, network.MuxedStream) error {
				callbacks++
				return nil
			})
		})
		if pin == host {
			if err != nil || callbacks != 1 {
				t.Fatalf("expected one challenge from host %s, got %d: %v", host, callbacks, err)
			}
		} else if !errors.Is(err, lan.ErrProviderIdentity) || callbacks != 0 {
			t.Fatalf("expected identity rejection before challenge from host %s, got %d: %v", host, callbacks, err)
		}
	default:
		t.Fatal("unknown namespace test role")
	}
}
