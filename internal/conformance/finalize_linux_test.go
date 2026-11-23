// SPDX-License-Identifier: AGPL-3.0-only

//go:build linux && !android && cgo

package conformance

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
)

type publicationTrial struct {
	reservationTrial
	reads     *loopbackRead
	manifest  [276]byte
	uploader  crypto.PrivKey
	agreement lan.CatalogAgreement
	limits    grantstore.UploadLimits
	storeID   [32]byte
	dir       string
}

func newPublicationTrial(t *testing.T) *publicationTrial {
	t.Helper()
	object, catalog := fiveByteSnapshot(t)
	return newSnapshotPublicationTrial(t, hiveFixture(), object, catalog)
}

func newSnapshotPublicationTrial(t *testing.T, fixture snapshotFixture, object, catalog []byte) *publicationTrial {
	t.Helper()
	// The sealed pair and its manifest use two public RFC 8032 identities.
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
	authorityKey, authority, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	m := fixture.reservationManifest(t, object, catalog)
	start := time.Now().Truncate(time.Second)
	agreement := lan.CatalogAgreement{
		ProviderKey: [32]byte(m[20:52]), AuthorityKey: [32]byte(authorityKey),
		FolderID: [16]byte(m[84:100]), AgreementID: [16]byte(m[100:116]),
		Start: start.Add(-time.Hour), Expiry: start.Add(time.Hour),
	}
	limits := grantstore.UploadLimits{HostBytes: 64 << 20, AgreementBytes: 64 << 20, RequesterBytes: 64 << 20, FreeMargin: 1 << 20, MaxRequests: 1}
	dir, storeID, writer := filepath.Join(t.TempDir(), "provider"), [32]byte{0x61}, [32]byte(m[52:84])
	provider, err := lan.CreateUploadProvider(dir, storeID, agreement, writer, limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { provider.Close() })
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	trial := reservationTrial{listener, host, agreement.ProviderKey, provider}

	// Install a valid read grant before the upload. The replacement has its
	// own TLS identity, independent of both the uploader and its recovery key.
	device, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	deviceKey, err := pub.Raw()
	if err != nil {
		t.Fatal(err)
	}
	recoveryKey, recovery, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &storageformat.RecoveryPackage{
		Catalog: fixture.context(t), Generation: 7, ProviderKey: agreement.ProviderKey,
		Endpoint: listener.Addr().(*net.TCPAddr).AddrPort(), RecoverySeed: [32]byte(recovery.Seed()),
	}
	reads := &loopbackRead{listener: listener, host: host, device: device, deviceKey: [32]byte(deviceKey), authority: authority, recovery: recovery, pkg: pkg, provider: provider}
	if err := provider.LoadGrant(reads.signedGrant(pkg.Generation, 1, recoveryKey)); err != nil {
		t.Fatal(err)
	}
	return &publicationTrial{reservationTrial: trial, reads: reads, manifest: m, uploader: uploader,
		agreement: agreement, limits: limits, storeID: storeID, dir: dir}
}

// Use fresh salt and nonce for the actual endpoint and newly enrolled recovery
// key. Deterministic header bytes belong only to the published vectors.
func (trial *publicationTrial) recoveryExport(t *testing.T, passphrase string) []byte {
	t.Helper()
	var salt [16]byte
	var nonce [12]byte
	rand.Read(salt[:])
	rand.Read(nonce[:])
	return encodeRecoveryExample(t, trial.reads.pkg, passphrase, salt[:], nonce[:]).seal()
}

func (trial *publicationTrial) reopen(t *testing.T) {
	t.Helper()
	if err := trial.provider.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := lan.OpenUploadProvider(trial.dir, trial.storeID, trial.agreement, [32]byte(trial.manifest[52:84]), trial.limits)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reopened.Close() })
	trial.provider, trial.reads.provider = reopened, reopened
}

func (trial *publicationTrial) reserve(t *testing.T) (grantstore.UploadState, uint64) {
	t.Helper()
	state, err := trial.exchange(t, lan.SubmitReservation, trial.uploader, trial.manifest, false)
	if err != nil || state.Code != 1 || state.Mask != 0 {
		t.Fatalf("RESERVE: %+v, %v", state, err)
	}
	rows, charge, err := trial.provider.UploadUsage()
	if err != nil || rows != 1 || charge == 0 {
		t.Fatalf("reservation: rows=%d charge=%d err=%v", rows, charge, err)
	}
	return state, charge
}

func (trial *publicationTrial) stage(t *testing.T, kind byte, sealed []byte, mask byte, deadline int64) {
	t.Helper()
	put := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, manifest [276]byte) (grantstore.UploadState, error) {
		return lan.PutUploadItem(ctx, addr, pin, key, manifest, kind, bytes.NewReader(sealed))
	}
	got, err := trial.exchange(t, put, trial.uploader, trial.manifest, false)
	if err != nil || got.Code != 1 || got.Mask != mask || got.Deadline != deadline {
		t.Fatalf("PUT kind %d: %+v, %v", kind, got, err)
	}
}

func TestLostFinalizeReplyReturnsOriginalReceipt(t *testing.T) {
	trial := newPublicationTrial(t)
	m, uploader, reads := trial.manifest, trial.uploader, trial.reads
	object, catalog := fiveByteSnapshot(t)
	expiry := trial.agreement.Expiry.Unix()
	receiptDir := t.TempDir()
	confirm := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, manifest [276]byte) (grantstore.UploadState, error) {
		return lan.ConfirmUpload(ctx, addr, pin, key, manifest, expiry, receiptDir)
	}
	status := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, manifest [276]byte) (grantstore.UploadState, error) {
		return lan.UploadStatus(ctx, addr, pin, key, manifest, expiry)
	}
	state, charge := trial.reserve(t)
	trial.stage(t, 2, object, 2, state.Deadline)
	partial, err := trial.exchange(t, confirm, uploader, m, false)
	if err != nil || partial.Code != 1 || partial.Mask != 2 || partial.Deadline != state.Deadline || partial.Receipt != [132]byte{} {
		t.Fatalf("FINALIZE with only the object: %+v, %v", partial, err)
	}
	expectEmptyDir(t, receiptDir)
	trial.stage(t, 1, catalog, 3, state.Deadline)
	objectPath, catalogPath := filepath.Join(trial.dir, "item-1-2"), filepath.Join(trial.dir, "item-1-1")
	stagedObject, stagedCatalog := checkStagedBytes(t, objectPath, object), checkStagedBytes(t, catalogPath, catalog)
	snapshot, err := storageformat.OpenCatalog(bytes.NewReader(catalog), reads.pkg.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	reads.session(t, "staged pair is private even with a valid grant", 2, func(ctx context.Context) error {
		assertOldReadsDenied(t, ctx, reads, snapshot)
		return nil
	})
	reserved, stored, err := trial.provider.UploadAccounting()
	if err != nil || reserved != charge || stored != 0 {
		t.Fatalf("staged pair: reserved=%d stored=%d err=%v", reserved, stored, err)
	}
	before := time.Now().Unix()
	firstSignature := loseFinalizeReply(t, trial.reservationTrial, confirm, uploader, m, charge)
	after := time.Now().Unix()
	expectEmptyDir(t, receiptDir)
	original, err := trial.exchange(t, status, uploader, m, false)
	if err != nil || original.Code != 2 {
		t.Fatalf("STATUS after the lost reply: %+v, %v", original, err)
	}
	receipt := original.Receipt
	if receipt != firstSignature {
		t.Fatal("STATUS did not return the receipt signed during the interrupted FINALIZE")
	}
	fingerprint := sha256.Sum256(m[:])
	commitTime := int64(binary.BigEndian.Uint64(receipt[52:60]))
	if !bytes.Equal(receipt[:20], []byte("HiveScope/stored/v1\x00")) || !bytes.Equal(receipt[20:52], fingerprint[:]) ||
		commitTime < before || commitTime > after || int64(binary.BigEndian.Uint64(receipt[60:68])) != expiry ||
		!ed25519.Verify(trial.pin[:], receipt[:68], receipt[68:]) {
		t.Fatal("receipt did not authenticate the original manifest, commit time and agreement expiry")
	}
	trial.reopen(t)
	// A newly signed receipt using the current time must now be different.
	time.Sleep(time.Until(time.Unix(commitTime+1, 0)))
	badPath := filepath.Join(receiptDir, "missing-directory")
	withoutLocalStorage := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, manifest [276]byte) (grantstore.UploadState, error) {
		return lan.ConfirmUpload(ctx, addr, pin, key, manifest, expiry, badPath)
	}
	failed, err := trial.exchange(t, withoutLocalStorage, uploader, m, false)
	if !errors.Is(err, os.ErrNotExist) || failed != (grantstore.UploadState{}) {
		t.Fatalf("confirmed without saving a receipt: %+v, %v", failed, err)
	}
	expectEmptyDir(t, receiptDir)
	terms := lan.UploadTerms{Expiry: expiry}
	source := &countedPutSource{}
	retryPut := func(ctx context.Context, addr netip.AddrPort, pin [32]byte, key crypto.PrivKey, manifest [276]byte) (grantstore.UploadState, error) {
		return terms.Put(ctx, addr, pin, key, manifest, 1, source)
	}
	for _, operation := range []reservationCall{status, confirm, terms.Reserve, retryPut, confirm, status} {
		got, err := trial.exchange(t, operation, uploader, m, false)
		if err != nil || got != original {
			t.Fatalf("retry changed committed state: %+v, %v", got, err)
		}
	}
	if source.calls != 0 {
		t.Fatal("a committed PUT retry requested payload bytes")
	}
	receiptPath := filepath.Join(receiptDir, fmt.Sprintf("%x.receipt", fingerprint))
	saved, err := os.ReadFile(receiptPath)
	if err != nil || !bytes.Equal(saved, receipt[:]) {
		t.Fatalf("local confirmation lacks the exact signed receipt: %v", err)
	}
	changed := m
	changed[172] ^= 1
	conflict, err := trial.exchange(t, confirm, uploader, changed, false)
	if !errors.Is(err, grantstore.ErrUploadConflict) || conflict != (grantstore.UploadState{}) {
		t.Fatalf("changed manifest reused the committed ID: %+v, %v", conflict, err)
	}
	rows, total, err := trial.provider.UploadUsage()
	if err != nil || rows != 1 || total != charge {
		t.Fatalf("retried upload: rows=%d total=%d original charge=%d err=%v", rows, total, charge, err)
	}
	reserved, stored, err = trial.provider.UploadAccounting()
	if err != nil || reserved != 0 || stored != charge {
		t.Fatalf("retried pair: reserved=%d stored=%d err=%v", reserved, stored, err)
	}
	storedObject, storedCatalog := checkStagedBytes(t, objectPath, object), checkStagedBytes(t, catalogPath, catalog)
	if !os.SameFile(stagedObject, storedObject) || !os.SameFile(stagedCatalog, storedCatalog) ||
		!stagedObject.ModTime().Equal(storedObject.ModTime()) || !stagedCatalog.ModTime().Equal(storedCatalog.ModTime()) {
		t.Fatal("publication or retry replaced or rewrote a staged file")
	}
	checkRecoveredFile(t, reads, []byte("Hive\n"))
	t.Logf("lost FINALIZE reply: original 132-byte receipt, 1 committed upload, 271 sealed bytes, 1 quota conversion (%d bytes), 5 verified plaintext bytes", charge)
}

// The trial holds one live reservation with a positive charge before the
// request. Drop outgoing TLS data once that charge is durably converted or
// released. Negotiation and submission use the production transport unchanged.
type dropTerminalReply struct {
	net.Conn
	provider   *lan.ReadProvider
	dropped    *atomic.Bool
	wantStored uint64
}

func (c dropTerminalReply) Write(b []byte) (int, error) {
	reserved, stored, err := c.provider.UploadAccounting()
	if err != nil {
		return 0, err
	}
	if reserved == 0 && stored == c.wantStored {
		c.dropped.Store(true)
		c.Conn.Close()
		return 0, net.ErrClosed
	}
	return c.Conn.Write(b)
}

// Capture the first receipt at the real provider signer, independently of its
// database and protocol replies. TLS signatures pass through unchanged.
type receiptWitness struct {
	crypto.PrivKey
	signed chan [132]byte
}

func (w receiptWitness) Sign(message []byte) ([]byte, error) {
	signature, err := w.PrivKey.Sign(message)
	if err == nil && len(message) == 68 && bytes.HasPrefix(message, []byte("HiveScope/stored/v1\x00")) {
		var receipt [132]byte
		copy(receipt[:68], message)
		copy(receipt[68:], signature)
		select {
		case w.signed <- receipt:
		default:
			return nil, errors.New("interrupted FINALIZE signed more than one receipt")
		}
	}
	return signature, err
}

func loseFinalizeReply(t *testing.T, trial reservationTrial, confirm reservationCall, uploader crypto.PrivKey, m [276]byte, charge uint64) [132]byte {
	t.Helper()
	host := receiptWitness{trial.host, make(chan [132]byte, 1)}
	trial.host = host
	loseTerminalReply(t, trial, confirm, uploader, m, charge)
	select {
	case signed := <-host.signed:
		return signed
	default:
		t.Fatal("missing provider signature before the connection was closed")
		return [132]byte{}
	}
}

func loseTerminalReply(t *testing.T, trial reservationTrial, operation reservationCall, uploader crypto.PrivKey, m [276]byte, wantStored uint64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if err := trial.listener.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	var dropped atomic.Bool
	served := make(chan error, 1)
	go func() {
		conn, err := trial.listener.Accept()
		if err == nil {
			err = trial.provider.ServeRead(ctx, dropTerminalReply{conn, trial.provider, &dropped, wantStored}, trial.host)
		}
		served <- err
	}()
	endpoint := trial.listener.Addr().(*net.TCPAddr).AddrPort()
	state, err := operation(ctx, endpoint, trial.pin, uploader, m)
	serverErr := <-served
	if err == nil || state != (grantstore.UploadState{}) || !dropped.Load() {
		t.Fatalf("expected an unconfirmed client after the durable transition: state=%+v client=%v server=%v dropped=%t", state, err, serverErr, dropped.Load())
	}
}
