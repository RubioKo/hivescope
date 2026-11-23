// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package conformance

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"net"
	"net/netip"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/sys/unix"
)

func TestLostReservationReplyKeepsOneCharge(t *testing.T) {
	// Public RFC 8032 section 7.1 test identities used by the receipt fixture.
	identity := func(seed string) crypto.PrivKey {
		t.Helper()
		key, err := crypto.UnmarshalEd25519PrivateKey(ed25519.NewKeyFromSeed(mustHex(t, seed)))
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	host := identity("4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb")
	uploader := identity("833fe62409237b9d62ec77587520911e9a759cec1d19755b7da901b96dca3d42")
	authority, public, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authorityKey, err := public.Raw()
	if err != nil {
		t.Fatal(err)
	}
	f := hiveFixture()
	scope := f.context(t)
	manifest := fiveByteReservationManifest(t)
	now := time.Now().Truncate(time.Second)
	agreement := lan.CatalogAgreement{
		ProviderKey: [32]byte(manifest[20:52]), AuthorityKey: [32]byte(authorityKey),
		FolderID: scope.FolderID, AgreementID: scope.AgreementID,
		Start: now.Add(-time.Hour), Expiry: now.Add(time.Hour),
	}
	uploaderKey := [32]byte(manifest[52:84])
	limits := grantstore.UploadLimits{
		HostBytes: 64 << 20, AgreementBytes: 64 << 20, RequesterBytes: 64 << 20,
		FreeMargin: 1 << 20, MaxRequests: 1,
	}
	dir, storeID := filepath.Join(t.TempDir(), "reservations"), [32]byte{0x31}
	provider, err := lan.CreateUploadProvider(dir, storeID, agreement, uploaderKey, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	trial := reservationTrial{listener, host, agreement.ProviderKey, provider}

	state, err := trial.exchange(t, lan.ReservationStatus, uploader, manifest, false)
	if err != nil || state != (grantstore.UploadState{}) {
		t.Fatalf("initial STATUS: state=%+v err=%v", state, err)
	}
	if rows, bytes, err := provider.UploadUsage(); err != nil || rows != 0 || bytes != 0 {
		t.Fatalf("unknown STATUS allocated space: rows=%d bytes=%d err=%v", rows, bytes, err)
	}
	before := time.Now().Unix()
	state, err = trial.exchange(t, lan.SubmitReservation, uploader, manifest, true)
	if err == nil || state != (grantstore.UploadState{}) {
		t.Fatalf("client accepted a lost reply: state=%+v err=%v", state, err)
	}
	after := time.Now().Unix()
	var volume unix.Statfs_t
	if err := unix.Statfs(dir, &volume); err != nil {
		t.Fatal(err)
	}
	rows, charge, err := provider.UploadUsage()
	if err != nil || rows != 1 || charge < uint64(2*volume.Bsize) {
		t.Fatalf("durable charge after the lost reply: rows=%d bytes=%d err=%v", rows, charge, err)
	}
	first, err := trial.exchange(t, lan.ReservationStatus, uploader, manifest, false)
	if err != nil || first.Code != 1 || first.Mask != 0 || first.Deadline < before+900 || first.Deadline > after+900 {
		t.Fatalf("first admission deadline and stage mask: state=%+v err=%v", first, err)
	}
	if err := provider.Close(); err != nil {
		t.Fatal(err)
	}
	limits.Paused = true
	reopened, err := lan.OpenUploadProvider(dir, storeID, agreement, uploaderKey, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	trial.provider = reopened
	// Ensure that issuing a fresh 15-minute deadline on retry would change it.
	time.Sleep(time.Until(time.Unix(first.Deadline-900+1, 0)))
	for _, operation := range []reservationCall{lan.ReservationStatus, lan.SubmitReservation, lan.SubmitReservation} {
		got, err := trial.exchange(t, operation, uploader, manifest, false)
		if err != nil || got != first {
			t.Fatalf("exact retry after reopen: got=%+v original=%+v err=%v", got, first, err)
		}
	}
	changed := manifest
	changed[172] ^= 1 // different object digest, same scoped ID
	for _, operation := range []reservationCall{lan.SubmitReservation, lan.ReservationStatus} {
		got, err := trial.exchange(t, operation, uploader, changed, false)
		if !errors.Is(err, grantstore.ErrUploadConflict) || got != (grantstore.UploadState{}) {
			t.Fatalf("conflict disclosed or replaced state: got=%+v err=%v", got, err)
		}
	}
	// The folder authority is a valid identity but is not the enrolled uploader.
	otherManifest := manifest
	copy(otherManifest[52:84], authorityKey)
	for _, operation := range []reservationCall{lan.SubmitReservation, lan.ReservationStatus} {
		got, err := trial.exchange(t, operation, authority, otherManifest, false)
		if !errors.Is(err, lan.ErrUploadDenied) || got != (grantstore.UploadState{}) {
			t.Fatalf("unenrolled TLS peer received state: got=%+v err=%v", got, err)
		}
	}
	got, err := trial.exchange(t, lan.ReservationStatus, uploader, manifest, false)
	if err != nil || got != first {
		t.Fatalf("original reservation after conflicting retries: got=%+v err=%v", got, err)
	}
	if count, charged, err := reopened.UploadUsage(); err != nil || count != 1 || charged != charge {
		t.Fatalf("retry changed the charge: rows=%d bytes=%d original=%d err=%v", count, charged, charge, err)
	}
	t.Logf("lost RESERVE reply: 1 retained ID, %d committed capacity bytes before and after retry, original deadline, 0 staged items", charge)
}

func fiveByteReservationManifest(t *testing.T) [276]byte {
	t.Helper()
	object, catalog := fiveByteSnapshot(t)
	m := hiveFixture().reservationManifest(t, object, catalog)
	requireDigest(t, m[:], "5cef2907ccccffa568c547206a2a4324294367cb9d54695ba33671d48a149e03")
	return m
}

func (f snapshotFixture) reservationManifest(t *testing.T, object, catalog []byte) [276]byte {
	t.Helper()
	scope := f.context(t)
	objectDigest, catalogDigest := sha256.Sum256(object), sha256.Sum256(catalog)
	m, err := lan.NewUploadManifest([32]byte(mustHex(t, readFixtureProviderPublic)),
		[32]byte(mustHex(t, "ec172b93ad5e563bf4932c70e1245034c35467ef2efd4d64ebf819683467e2bf")),
		scope.FolderID, scope.AgreementID, [16]byte(repeated(0x15, 16)),
		lan.UploadItem{Locator: [32]byte(repeated(f.objectLocator, 32)), Digest: objectDigest, Size: uint64(len(object))},
		lan.UploadItem{Locator: scope.CatalogLocator, Digest: catalogDigest, Size: uint64(len(catalog))})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type reservationCall func(context.Context, netip.AddrPort, [32]byte, crypto.PrivKey, [276]byte) (grantstore.UploadState, error)

type reservationTrial struct {
	listener *net.TCPListener
	host     crypto.PrivKey
	pin      [32]byte
	provider *lan.ReadProvider
}

func (r reservationTrial) exchange(t *testing.T, operation reservationCall, device crypto.PrivKey, m [276]byte, loseReply bool) (grantstore.UploadState, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if err := r.listener.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	var dropped atomic.Bool
	served := make(chan error, 1)
	go func() {
		conn, err := r.listener.Accept()
		if err == nil {
			if loseReply {
				conn = dropAdmittedReply{conn, r.provider, &dropped}
			}
			err = r.provider.ServeRead(ctx, conn, r.host)
		}
		served <- err
	}()
	address := r.listener.Addr().(*net.TCPAddr).AddrPort()
	endpoint := netip.AddrPortFrom(address.Addr().Unmap(), address.Port())
	state, clientErr := operation(ctx, endpoint, r.pin, device, m)
	r.listener.SetDeadline(time.Now())
	serverErr := <-served
	if loseReply && !dropped.Load() {
		t.Fatalf("connection did not reach the post-admission cut: client=%v provider=%v", clientErr, serverErr)
	}
	if !loseReply && serverErr != nil {
		t.Fatalf("provider exchange: %v", serverErr)
	}
	return state, clientErr
}

// Each TLS write passes normally until the real SQLite store reports a
// committed ID. At that point no further socket bytes reach the uploader.
type dropAdmittedReply struct {
	net.Conn
	provider *lan.ReadProvider
	dropped  *atomic.Bool
}

func (c dropAdmittedReply) Write(b []byte) (int, error) {
	count, _, err := c.provider.UploadUsage()
	if err != nil {
		return 0, err
	}
	if count > 0 {
		c.dropped.Store(true)
		c.Conn.Close()
		return 0, net.ErrClosed
	}
	return c.Conn.Write(b)
}
