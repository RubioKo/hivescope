// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package conformance

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
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

func TestInterruptedCatalogPutKeepsStagedObject(t *testing.T) {
	// The same public RFC 8032 identities appear in the five-byte manifest.
	fixtureKey := func(seed string) crypto.PrivKey {
		t.Helper()
		key, err := crypto.UnmarshalEd25519PrivateKey(ed25519.NewKeyFromSeed(mustHex(t, seed)))
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	host := fixtureKey("4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb")
	uploader := fixtureKey("833fe62409237b9d62ec77587520911e9a759cec1d19755b7da901b96dca3d42")
	authority, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	m := fiveByteReservationManifest(t)
	object, catalog := fiveByteSnapshot(t)
	if len(object) != 21 || len(catalog) != 250 {
		t.Fatal("incorrect sealed fixture sizes")
	}
	now := time.Now().Truncate(time.Second)
	agreement := lan.CatalogAgreement{
		ProviderKey: [32]byte(m[20:52]), AuthorityKey: [32]byte(authority),
		FolderID: [16]byte(m[84:100]), AgreementID: [16]byte(m[100:116]),
		Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour),
	}
	limits := grantstore.UploadLimits{HostBytes: 64 << 20, AgreementBytes: 64 << 20, RequesterBytes: 64 << 20, FreeMargin: 1 << 20, MaxRequests: 1}
	dir, storeID, writer := filepath.Join(t.TempDir(), "provider"), [32]byte{0x59}, [32]byte(m[52:84])
	p, err := lan.CreateUploadProvider(dir, storeID, agreement, writer, limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	trial := reservationTrial{listener, host, agreement.ProviderKey, p}
	initial, err := trial.exchange(t, lan.SubmitReservation, uploader, m, false)
	if err != nil || initial.Code != 1 || initial.Mask != 0 || initial.Deadline <= now.Unix() {
		t.Fatalf("initial reservation: %+v, %v", initial, err)
	}
	count, charge, err := p.UploadUsage()
	if err != nil || count != 1 || charge == 0 {
		t.Fatalf("initial usage: %d requests, %d bytes, %v", count, charge, err)
	}
	assertState := func(action reservationCall, mask byte) {
		t.Helper()
		want := initial
		want.Mask = mask
		if state, err := trial.exchange(t, action, uploader, m, false); err != nil || state != want {
			t.Fatalf("expected %+v, got %+v, %v", want, state, err)
		}
	}
	put := func(kind byte, source io.Reader) reservationCall {
		return func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte) (grantstore.UploadState, error) {
			return lan.PutUploadItem(ctx, addr, pin, key, m, kind, source)
		}
	}
	assertState(put(2, bytes.NewReader(object)), 2)
	objectPath := filepath.Join(dir, "item-1-2")
	catalogPath := filepath.Join(dir, "item-1-1")
	original := checkStagedBytes(t, objectPath, object)

	_, err = trial.exchange(t, func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte) (grantstore.UploadState, error) {
		return grantstore.UploadState{}, interruptCatalogFrame(ctx, addr, pin, key, m, catalog[:125])
	}, uploader, m, false)
	if err != nil {
		t.Fatal(err)
	}
	assertState(lan.ReservationStatus, 2)
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := lan.OpenUploadProvider(dir, storeID, agreement, writer, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	trial.provider = reopened
	assertState(lan.ReservationStatus, 2)
	if _, err := os.Lstat(catalogPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("interrupted catalog left a named slot: %v", err)
	}
	// Even an explicit retry of the object must return STATE before reading
	// its source. Only the catalog needs a new DATA frame.
	duplicate := &countedPutSource{}
	assertState(put(2, duplicate), 2)
	if duplicate.calls != 0 {
		t.Fatal("provider requested a second transfer of the staged object")
	}
	source := bytes.NewReader(catalog)
	assertState(put(1, source), 3)
	if source.Len() != 0 {
		t.Fatal("catalog retry did not read the complete source")
	}
	assertState(lan.ReservationStatus, 3)
	checkStagedBytes(t, catalogPath, catalog)
	final := checkStagedBytes(t, objectPath, object)
	if !os.SameFile(original, final) || !original.ModTime().Equal(final.ModTime()) {
		t.Fatal("catalog retry replaced or rewrote the staged object")
	}
	if n, used, err := reopened.UploadUsage(); err != nil || n != 1 || used != charge {
		t.Fatalf("reservation changed: count=%d charge=%d original=%d err=%v", n, used, charge, err)
	}
	// Even the enrolled uploader with both known locators cannot read staged
	// data. Exercise the actual read protocol after the last staged marker.
	f := hiveFixture()
	catalogContext := f.context(t)
	snapshot, err := storageformat.OpenCatalog(bytes.NewReader(catalog), catalogContext)
	if err != nil {
		t.Fatal(err)
	}
	_, recovery, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().(*net.TCPAddr).AddrPort()
	read := loopbackRead{listener: listener, host: host, device: uploader, deviceKey: writer, provider: reopened, recovery: recovery,
		pkg: &storageformat.RecoveryPackage{Catalog: catalogContext, Generation: 1, ProviderKey: agreement.ProviderKey,
			Endpoint: netip.AddrPortFrom(address.Addr().Unmap(), address.Port())}}
	read.session(t, "reads of both staged slots", 2, func(ctx context.Context) error {
		assertOldReadsDenied(t, ctx, &read, snapshot)
		return nil
	})
	t.Logf("catalog interrupted at 125/250 payload bytes; staged masks 02 after reopen, 03 after catalog retry; 0 object bytes retransmitted, 1 reservation, %d capacity bytes", charge)
}

func checkStagedBytes(t *testing.T, path string, want []byte) os.FileInfo {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != int64(len(want)) {
		t.Fatalf("invalid staged file for %d bytes: %v", len(want), err)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("unexpected staged contents: %v", err)
	}
	return info
}

type countedPutSource struct{ calls int }

func (s *countedPutSource) Read([]byte) (int, error) {
	s.calls++
	return 0, errors.New("staged item source must never be read")
}

// Send an ordinary authenticated PUT and cut one DATA frame mid-payload.
// The ordered half-close delivers the prefix before EOF and keeps the return
// direction open, allowing the test to observe the provider's rejection.
func interruptCatalogFrame(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, m [276]byte, prefix []byte) error {
	muxers := []upgrader.StreamMuxer{{ID: yamux.ID, Muxer: yamux.DefaultTransport}}
	tls, err := libp2ptls.New(libp2ptls.ID, key, muxers)
	if err != nil {
		return err
	}
	up, err := upgrader.New([]sec.SecureTransport{tls}, muxers, nil, nil, nil)
	if err != nil {
		return err
	}
	remote, err := crypto.UnmarshalEd25519PublicKey(pin[:])
	if err != nil {
		return err
	}
	id, err := peer.IDFromPublicKey(remote)
	if err != nil {
		return err
	}
	socket, err := (&net.Dialer{}).DialContext(ctx, "tcp4", addr.String())
	if err != nil {
		return err
	}
	defer socket.Close()
	stop := context.AfterFunc(ctx, func() { socket.Close() })
	defer stop()
	raw, err := manet.WrapNetConn(socket)
	if err != nil {
		return err
	}
	conn, err := up.Upgrade(ctx, nil, raw, network.DirOutbound, id, &network.NullScope{})
	if err != nil {
		return err
	}
	defer conn.Close()
	if !remote.Equals(conn.RemotePublicKey()) {
		return errors.New("wrong provider for interrupted PUT")
	}
	stream, err := conn.OpenStream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()
	if err := ms.SelectProtoOrFail(fixtureStorageProtocol, stream); err != nil {
		return err
	}
	// PUT payload = folder, agreement, request, manifest SHA-256, catalog kind.
	request := append([]byte{0, 0, 0, 85, 8, 3, 0x12, 81}, m[84:132]...)
	fingerprint := sha256.Sum256(m[:])
	request = append(request, fingerprint[:]...)
	request = append(request, 1)
	if n, err := stream.Write(request); err != nil || n != len(request) {
		return errors.Join(io.ErrShortWrite, err)
	}
	var ready [6]byte
	if _, err := io.ReadFull(stream, ready[:]); err != nil || ready != [6]byte{0, 0, 0, 2, 8, 0x20} {
		return fmt.Errorf("expected READY, got %x, err=%v", ready, err)
	}
	// A 250-byte DATA payload has a 255-byte Protobuf body. Send its true
	// declared size and half the contents; never send the remaining bytes or END.
	partial := append([]byte{0, 0, 0, 255, 8, 0x21, 0x12, 0xfa, 0x01}, prefix...)
	if n, err := stream.Write(partial); err != nil || n != len(partial) {
		return errors.Join(io.ErrShortWrite, err)
	}
	if err := stream.CloseWrite(); err != nil {
		return err
	}
	want := []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, 1}
	got, err := io.ReadAll(io.LimitReader(stream, int64(len(want)+1)))
	if err != nil || !bytes.Equal(got, want) {
		return fmt.Errorf("expected INVALID and EOF after incomplete DATA, got %x, err=%v", got, err)
	}
	return nil
}
