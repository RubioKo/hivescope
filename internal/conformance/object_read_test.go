// SPDX-License-Identifier: AGPL-3.0-only

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
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

type loopbackRead struct {
	listener            *net.TCPListener
	host, device        crypto.PrivKey
	deviceKey           [32]byte
	authority, recovery ed25519.PrivateKey
	pkg                 *storageformat.RecoveryPackage
	provider            *lan.ReadProvider
}

func newLoopbackRead(t *testing.T, f snapshotFixture, object, catalog []byte) *loopbackRead {
	t.Helper()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	address := listener.Addr().(*net.TCPAddr).AddrPort()
	newPeer := func() (crypto.PrivKey, [32]byte) {
		t.Helper()
		key, pub, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := pub.Raw()
		if err != nil {
			t.Fatal(err)
		}
		return key, [32]byte(raw)
	}
	host, hostKey := newPeer()
	device, deviceKey := newPeer()
	authorityKey, authority, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	recoveryKey, recovery, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &storageformat.RecoveryPackage{
		Catalog: f.context(t), Generation: 7, ProviderKey: hostKey,
		Endpoint: netip.AddrPortFrom(address.Addr().Unmap(), address.Port()), RecoverySeed: [32]byte(recovery.Seed()),
	}
	agreement := lan.CatalogAgreement{ProviderKey: hostKey, AuthorityKey: [32]byte(authorityKey),
		FolderID: pkg.Catalog.FolderID, AgreementID: pkg.Catalog.AgreementID,
		Start: time.Now().Add(-time.Hour), Expiry: time.Now().Add(time.Hour)}
	p, err := lan.NewReadProvider(agreement, lan.CommittedCatalog{
		Locator: pkg.Catalog.CatalogLocator, Digest: sha256.Sum256(catalog), Sealed: catalog,
	}, lan.CommittedObject{
		Locator: [32]byte(repeated(f.objectLocator, 32)), Digest: sha256.Sum256(object), Sealed: object,
	})
	if err != nil {
		t.Fatal(err)
	}
	trial := &loopbackRead{listener: listener, host: host, device: device, deviceKey: deviceKey,
		pkg: pkg, provider: p, authority: authority, recovery: recovery}
	if err := p.LoadGrant(trial.signedGrant(7, 1, recoveryKey)); err != nil {
		t.Fatal(err)
	}
	return trial
}

func (r *loopbackRead) signedGrant(generation uint64, state byte, key []byte) []byte {
	record := append([]byte("HiveScope/recovery-grant/v1\x00"), r.pkg.ProviderKey[:]...)
	record = append(record, r.pkg.Catalog.FolderID[:]...)
	record = append(record, r.pkg.Catalog.AgreementID[:]...)
	record = binary.BigEndian.AppendUint64(record, generation)
	record = append(record, state, 1) // scope: read this agreement
	record = append(record, key...)
	return append(record, ed25519.Sign(r.authority, record)...)
}

func (r *loopbackRead) signRead(kind byte, locator, challenge [32]byte) []byte {
	record := append([]byte("HiveScope/recovery-read/v1\x00"), kind)
	record = append(record, r.pkg.ProviderKey[:]...)
	record = append(record, r.deviceKey[:]...)
	record = append(record, r.pkg.Catalog.FolderID[:]...)
	record = append(record, r.pkg.Catalog.AgreementID[:]...)
	record = binary.BigEndian.AppendUint64(record, r.pkg.Generation)
	record = append(record, locator[:]...)
	record = append(record, challenge[:]...)
	return ed25519.Sign(r.recovery, record)
}

// Every item gets a fresh connection to the same enrolled provider.
func (r *loopbackRead) session(t *testing.T, name string, connections int, client func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	deadline, _ := ctx.Deadline()
	if err := r.listener.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	served := make(chan error, 1)
	go func() {
		for range connections {
			socket, err := r.listener.Accept()
			if err == nil {
				err = r.provider.ServeRead(ctx, socket, r.host)
			}
			if err != nil {
				served <- err
				return
			}
		}
		served <- nil
	}()
	clientErr := client(ctx)
	if clientErr != nil {
		cancel()
	}
	// The client has returned: an unaccepted connection is a missing
	// exchange, including when testing an expected error on the second read.
	r.listener.SetDeadline(time.Now())
	serverErr := <-served
	if clientErr != nil || serverErr != nil {
		t.Fatalf("%s: requester=%v, provider=%v", name, clientErr, serverErr)
	}
}

// Compare every byte through EOF, including the single terminal error frame
// on denial. Reading one byte beyond the expected reply detects leaked data.
func checkProofReply(stream network.MuxedStream, want []byte, proofs ...[]byte) error {
	var frames []byte
	for _, proof := range proofs {
		frames = append(frames, 0, 0, 0, 68, 8, 8, 0x12, 64)
		frames = append(frames, proof...)
	}
	if _, err := stream.Write(frames); err != nil {
		return err
	}
	if err := stream.CloseWrite(); err != nil {
		return err
	}
	got, err := io.ReadAll(io.LimitReader(stream, int64(len(want)+1)))
	if err != nil {
		return err
	}
	if !bytes.Equal(got, want) {
		return fmt.Errorf("unexpected reply through EOF: received %d bytes, expected %d", len(got), len(want))
	}
	return nil
}

func TestObjectRequiresIndependentProof(t *testing.T) {
	object, catalog := fiveByteSnapshot(t)
	trial := newLoopbackRead(t, hiveFixture(), object, catalog)
	// Fixed wire encodings for the 250-byte catalog and 21-byte sealed object.
	sizeFrame := []byte{0, 0, 0, 12, 8, 0x26, 0x12, 8, 0, 0, 0, 0, 0, 0, 0}
	endFrame := []byte{0, 0, 0, 2, 8, 0x22}
	wantCatalog := bytes.Join([][]byte{sizeFrame, {250}, {0, 0, 0, 255, 8, 0x21, 0x12, 0xfa, 1}, catalog, endFrame}, nil)
	wantObject := bytes.Join([][]byte{sizeFrame, {21}, {0, 0, 0, 25, 8, 0x21, 0x12, 21}, object, endFrame}, nil)
	denied := []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, 2}

	var signedCatalogProof []byte
	var catalogChallenge [32]byte
	trial.session(t, "catalog proof accepted", 1, func(ctx context.Context) error {
		return lan.WithCatalogChallenge(ctx, trial.pkg, trial.device, func(challenge [32]byte, stream network.MuxedStream) error {
			catalogChallenge = challenge
			signedCatalogProof = trial.signRead(1, trial.pkg.Catalog.CatalogLocator, challenge)
			return checkProofReply(stream, wantCatalog, signedCatalogProof)
		})
	})
	var snapshot *storageformat.Snapshot
	trial.session(t, "verify remote catalog for the object locator", 1, func(ctx context.Context) error {
		var err error
		snapshot, err = lan.ReadCatalog(ctx, trial.pkg, trial.device)
		return err
	})
	for _, tc := range []struct {
		name string
		sign func([32]byte) []byte
		want []byte
	}{
		{"replay exact accepted catalog proof", func([32]byte) []byte { return signedCatalogProof }, denied},
		{"fresh proof for wrong kind", func(n [32]byte) []byte { return trial.signRead(1, snapshot.ObjectLocator(), n) }, denied},
		{"fresh proof for wrong locator", func(n [32]byte) []byte { return trial.signRead(2, trial.pkg.Catalog.CatalogLocator, n) }, denied},
		{"fresh proof for the object", func(n [32]byte) []byte { return trial.signRead(2, snapshot.ObjectLocator(), n) }, wantObject},
	} {
		trial.session(t, tc.name, 1, func(ctx context.Context) error {
			return lan.WithObjectChallenge(ctx, trial.pkg, trial.device, snapshot, func(challenge [32]byte, stream network.MuxedStream) error {
				if challenge == catalogChallenge {
					return errors.New("provider repeated the catalog challenge")
				}
				return checkProofReply(stream, tc.want, tc.sign(challenge))
			})
		})
	}
	checkRecoveredFile(t, trial, hiveFixture().plaintext)
	t.Logf("accepted catalog proof replay: DENIED, 0 object bytes; fresh object proof: %d sealed bytes", len(object))
}

func checkRecoveredFile(t *testing.T, trial *loopbackRead, expected []byte) {
	t.Helper()
	dir := t.TempDir()
	trial.session(t, "catalog then object", 2, func(ctx context.Context) error {
		file, err := lan.RecoverFile(ctx, trial.pkg, trial.device, dir)
		if err != nil {
			return err
		}
		plain, readErr := io.ReadAll(file)
		if err := errors.Join(readErr, file.Close()); err != nil {
			return err
		}
		if !bytes.Equal(plain, expected) {
			return errors.New("recovered file differs from independent plaintext")
		}
		return nil
	})
	expectEmptyDir(t, dir)
}

func TestRecoverFileOverLoopback(t *testing.T) {
	for _, f := range []snapshotFixture{emptyFixture(), edgeFixture()} {
		t.Run(f.name, func(t *testing.T) {
			object := f.sealObject(t)
			trial := newLoopbackRead(t, f, object, f.sealCatalog(t, f.entry(object)))
			checkRecoveredFile(t, trial, f.plaintext)
			t.Logf("verified %d plaintext bytes from %d sealed bytes", len(f.plaintext), len(object))
		})
	}
}

func TestCorruptRemoteObjectDiscardsStaging(t *testing.T) {
	f := edgeFixture()
	object := f.sealObject(t)
	catalog := f.sealCatalog(t, f.entry(object))
	// The provider's local index agrees with the damaged bytes. The signed
	// catalog and final block tag must still protect the replacement.
	object[len(object)-1] ^= 1
	trial := newLoopbackRead(t, f, object, catalog)
	checkRejectedObject(t, trial)
}

func TestRemoteSavePreservesDestinationUntilLastBlockVerifies(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("local atomic publication currently targets Linux and Windows")
	}
	f := edgeFixture()
	object := f.sealObject(t)
	catalog := f.sealCatalog(t, f.entry(object))
	damaged := bytes.Clone(object)
	damaged[len(damaged)-1] ^= 1
	destination, stateDir := filepath.Join(t.TempDir(), "chosen.bin"), t.TempDir()
	previous := []byte("previous contents")
	if err := os.WriteFile(destination, previous, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		sealed []byte
		valid  bool
	}{
		{"damaged final block", damaged, false},
		{"verified retry", object, true},
	} {
		trial := newLoopbackRead(t, f, tc.sealed, catalog)
		trial.session(t, tc.name, 2, func(ctx context.Context) error {
			err := lan.RecoverFileTo(ctx, trial.pkg, trial.device, destination, true, stateDir)
			if tc.valid {
				return err
			}
			if err == nil {
				return errors.New("published a corrupt remote object")
			}
			return nil
		})
		if tc.valid {
			requireFileBytes(t, destination, f.plaintext)
		} else {
			requireFileBytes(t, destination, previous)
			expectEmptyDir(t, stateDir)
		}
	}
}

func checkRejectedObject(t *testing.T, trial *loopbackRead) {
	t.Helper()
	dir := t.TempDir()
	trial.session(t, "reject inconsistent object", 2, func(ctx context.Context) error {
		file, err := lan.RecoverFile(ctx, trial.pkg, trial.device, dir)
		if file != nil {
			file.Close()
			return errors.New("exposed a reader for a corrupt object")
		}
		if err == nil {
			return errors.New("accepted a corrupt object")
		}
		return nil
	})
	expectEmptyDir(t, dir)
}

func TestRemoteObjectSizeMustMatchSignedCatalog(t *testing.T) {
	f := hiveFixture()
	object, catalog := fiveByteSnapshot(t)
	for _, tc := range []struct {
		name   string
		sealed []byte
	}{
		{"missing byte", object[:len(object)-1]},
		{"extra byte", append(bytes.Clone(object), 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The provider pins the digest of the bytes it actually sends.
			// The signed catalog still requires exactly 21 sealed bytes.
			trial := newLoopbackRead(t, f, tc.sealed, catalog)
			checkRejectedObject(t, trial)
		})
	}
}

func TestObjectChallengeRejectsMismatchedCatalogLocally(t *testing.T) {
	_, catalog := fiveByteSnapshot(t)
	trusted := hiveFixture().context(t)
	snapshot, err := storageformat.OpenCatalog(bytes.NewReader(catalog), trusted)
	if err != nil {
		t.Fatal(err)
	}
	device, hostKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := hostKey.Raw()
	if err != nil {
		t.Fatal(err)
	}
	pkg := storageformat.RecoveryPackage{Catalog: trusted, Generation: 7,
		ProviderKey: [32]byte(raw), Endpoint: netip.MustParseAddrPort("192.0.2.10:4001")}
	// A cancelled context distinguishes reaching the dial from rejecting a
	// mismatched catalog locally, without opening a network connection.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	try := func() error {
		return lan.WithObjectChallenge(ctx, &pkg, device, snapshot, func([32]byte, network.MuxedStream) error {
			t.Fatal("unexpected challenge")
			return nil
		})
	}
	if err := try(); !errors.Is(err, context.Canceled) {
		t.Fatalf("matching catalog should reach cancelled dial, got %v", err)
	}
	pkg.Catalog.AgreementID[0] ^= 1
	if err := try(); err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("mismatched catalog should stop before dial, got %v", err)
	}
}
