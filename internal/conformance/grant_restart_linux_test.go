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
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

// Every process receives the same open listener. Only SQLite can carry the
// accepted grant across SIGKILL. The sealed files and identity are independent
// synthetic fixtures provisioned afresh for each process.
func TestAcknowledgedRevocationSurvivesRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	endpoint := listener.Addr().(*net.TCPAddr).AddrPort()
	phrase := rand.Text()
	bundle := recoveryFixture(t, phrase, 41)
	ip := endpoint.Addr().As4()
	copy(bundle.payload[72:76], ip[:])
	binary.BigEndian.PutUint16(bundle.payload[76:78], endpoint.Port())
	err = storageformat.WithRecoveryPackage(bytes.NewReader(bundle.seal()), []byte(phrase), func(pkg *storageformat.RecoveryPackage) error {
		runGrantRestart(t, ctx, listener, pkg)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runGrantRestart(t *testing.T, ctx context.Context, listener *net.TCPListener, pkg *storageformat.RecoveryPackage) {
	t.Helper()
	root := t.TempDir()
	authorityKey, authority, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	device, public, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := public.Raw()
	if err != nil {
		t.Fatal(err)
	}
	trial := loopbackRead{pkg: pkg, device: device, deviceKey: [32]byte(raw),
		authority: authority, recovery: ed25519.NewKeyFromSeed(pkg.RecoverySeed[:])}
	active := trial.signedGrant(7, 1, trial.recovery.Public().(ed25519.PublicKey))
	revoked := trial.signedGrant(8, 2, make([]byte, 32))
	if err := os.WriteFile(filepath.Join(root, "authority.pub"), authorityKey, 0600); err != nil {
		t.Fatal(err)
	}
	enrollment := filepath.Join(root, "initial.grant")
	if err := os.WriteFile(enrollment, active, 0600); err != nil {
		t.Fatal(err)
	}
	confirm := func(record []byte) {
		t.Helper()
		got, err := lan.SubmitGrantUpdate(ctx, pkg.Endpoint, pkg.ProviderKey, device, record)
		if err != nil || got != sha256.Sum256(record) {
			t.Fatalf("expected ACK matching the complete signed grant, err=%v", err)
		}
	}
	stop := startGrantProcess(t, ctx, root, "create", listener)
	confirm(active)
	if err := os.Remove(enrollment); err != nil {
		t.Fatal(err)
	}
	stop()

	// Positive control on the reopen path, with no enrollment grant on disk.
	stop = startGrantProcess(t, ctx, root, "open", listener)
	snapshot, err := lan.ReadCatalog(ctx, pkg, device)
	if err != nil {
		t.Fatal(err)
	}
	privateDir := t.TempDir()
	destination := filepath.Join(t.TempDir(), "hive.txt")
	recoverToDestination := func() error {
		file, err := lan.RecoverFile(ctx, pkg, device, privateDir)
		if err != nil {
			return err
		}
		plain, readErr := io.ReadAll(file)
		if err := errors.Join(readErr, file.Close()); err != nil {
			return err
		}
		return os.WriteFile(destination, plain, 0600)
	}
	if err := recoverToDestination(); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(destination); err != nil || !bytes.Equal(got, hiveFixture().plaintext) {
		t.Fatalf("active grant after restart did not recover the five-byte fixture: %v", err)
	}
	if err := os.WriteFile(destination, []byte("keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	confirm(revoked)
	stop() // No graceful shutdown or application-level flush after the ACK.

	stop = startGrantProcess(t, ctx, root, "open", listener)
	defer stop()
	// Probe both locators before any administrative retry: resending the
	// revocation here could conceal a provider that reopened with stale state.
	denied := []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, 2}
	var catalogNonce [32]byte
	err = lan.WithCatalogChallenge(ctx, pkg, device, func(n [32]byte, s network.MuxedStream) error {
		catalogNonce = n
		return checkProofReply(s, denied, trial.signRead(1, pkg.Catalog.CatalogLocator, n))
	})
	if err != nil {
		t.Fatalf("expected DENIED and zero catalog bytes: %v", err)
	}
	err = lan.WithObjectChallenge(ctx, pkg, device, snapshot, func(n [32]byte, s network.MuxedStream) error {
		if n == catalogNonce {
			return errors.New("object reused the catalog challenge")
		}
		return checkProofReply(s, denied, trial.signRead(2, snapshot.ObjectLocator(), n))
	})
	if err != nil {
		t.Fatalf("expected DENIED and zero object bytes: %v", err)
	}
	if _, err := lan.SubmitGrantUpdate(ctx, pkg.Endpoint, pkg.ProviderKey, device, active); !errors.Is(err, lan.ErrGrantConflict) {
		t.Fatalf("superseded generation 7 update: %v", err)
	}
	confirm(revoked) // Identical retry after restart returns the same digest.
	if err := recoverToDestination(); !errors.Is(err, lan.ErrReadDenied) {
		t.Fatalf("old recovery package: %v", err)
	}
	if got, err := os.ReadFile(destination); err != nil || string(got) != "keep\n" {
		t.Fatalf("revoked recovery changed the existing destination: %v", err)
	}
	expectEmptyDir(t, privateDir)
	t.Log("active restart: 5 verified bytes; revoked restart: DENIED, 0 catalog bytes, 0 object bytes; destination preserved")
}

func startGrantProcess(t *testing.T, ctx context.Context, dir, mode string, listener *net.TCPListener) func() {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	socket, err := listener.File()
	if err != nil {
		t.Fatal(err)
	}
	defer socket.Close()
	ready, signal, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer ready.Close()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestGrantProviderProcess$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HIVESCOPE_TEST_GRANT_PROCESS="+mode)
	cmd.ExtraFiles = []*os.File{socket, signal}
	var log bytes.Buffer
	cmd.Stdout, cmd.Stderr = &log, &log
	err = cmd.Start()
	signal.Close()
	if err != nil {
		t.Fatal(err)
	}
	stop := func() {
		t.Helper()
		if cmd.ProcessState != nil {
			return
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if cmd.ProcessState == nil || cmd.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
			t.Errorf("provider exited unexpectedly: %s", log.Bytes())
		}
	}
	t.Cleanup(stop)
	var result [1]byte
	if _, err := io.ReadFull(ready, result[:]); err != nil || result[0] != 1 {
		stop()
		t.Fatalf("provider did not open its grant store: %v\n%s", err, log.Bytes())
	}
	return stop
}

func TestGrantProviderProcess(t *testing.T) {
	mode := os.Getenv("HIVESCOPE_TEST_GRANT_PROCESS")
	if mode == "" {
		t.Skip("subprocess for the persistent grant trial")
	}
	authority, err := os.ReadFile("authority.pub")
	if err != nil || len(authority) != ed25519.PublicKeySize {
		t.Fatal("invalid synthetic authority enrollment")
	}
	// RFC 8032 TEST 2 is the provider pinned in the recovery fixture.
	seed := mustHex(t, "4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb")
	host, err := crypto.UnmarshalEd25519PrivateKey(ed25519.NewKeyFromSeed(seed))
	if err != nil {
		t.Fatal(err)
	}
	f := hiveFixture()
	scope := f.context(t)
	agreement := lan.CatalogAgreement{ProviderKey: [32]byte(mustHex(t, readFixtureProviderPublic)), AuthorityKey: [32]byte(authority),
		FolderID: scope.FolderID, AgreementID: scope.AgreementID, Start: time.Unix(1_600_000_000, 0), Expiry: time.Unix(4_000_000_000, 0)}
	object, catalog := fiveByteSnapshot(t)
	cat := lan.CommittedCatalog{Locator: scope.CatalogLocator, Digest: sha256.Sum256(catalog), Sealed: catalog}
	obj := lan.CommittedObject{Locator: [32]byte(repeated(f.objectLocator, 32)), Digest: sha256.Sum256(object), Sealed: object}
	storeID := [32]byte{0x42}
	var provider *lan.ReadProvider
	switch mode {
	case "create":
		initial, err := os.ReadFile("initial.grant")
		if err != nil {
			t.Fatal(err)
		}
		provider, err = lan.CreatePersistentReadProvider("grants", storeID, agreement, cat, obj, initial)
		if err != nil {
			t.Fatal(err)
		}
	case "open":
		provider, err = lan.OpenPersistentReadProvider("grants", storeID, agreement, cat, obj)
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("unknown grant trial mode")
	}
	defer provider.Close()
	socket, signal := os.NewFile(3, "listener"), os.NewFile(4, "ready")
	listener, err := net.FileListener(socket)
	socket.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if _, err := signal.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	signal.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatal(err)
		}
		if err := provider.ServeRead(t.Context(), conn, host); err != nil {
			t.Fatalf("serve grant trial: %v", err)
		}
	}
}
