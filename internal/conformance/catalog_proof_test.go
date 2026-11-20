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
	"testing"
	"time"

	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

func TestCatalogProofAcceptedOnce(t *testing.T) {
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
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
		Catalog: hiveFixture().context(t), Generation: 7, ProviderKey: hostKey,
		Endpoint: netip.AddrPortFrom(address.Addr().Unmap(), address.Port()), RecoverySeed: [32]byte(recovery.Seed()),
	}
	_, catalog := fiveByteSnapshot(t)
	agreement := lan.CatalogAgreement{ProviderKey: hostKey, AuthorityKey: [32]byte(authorityKey),
		FolderID: pkg.Catalog.FolderID, AgreementID: pkg.Catalog.AgreementID,
		Start: time.Now().Add(-time.Hour), Expiry: time.Now().Add(time.Hour)}
	provider, err := lan.NewCatalogProvider(agreement, lan.CommittedCatalog{
		Locator: pkg.Catalog.CatalogLocator, Digest: sha256.Sum256(catalog), Sealed: catalog,
	})
	if err != nil {
		t.Fatal(err)
	}
	signedGrant := func(generation uint64, state byte, key []byte) []byte {
		record := append([]byte("HiveScope/recovery-grant/v1\x00"), hostKey[:]...)
		record = append(record, agreement.FolderID[:]...)
		record = append(record, agreement.AgreementID[:]...)
		record = binary.BigEndian.AppendUint64(record, generation)
		record = append(record, state, 1) // scope: read this agreement
		record = append(record, key...)
		return append(record, ed25519.Sign(authority, record)...)
	}
	if err := provider.LoadGrant(signedGrant(7, 1, recoveryKey)); err != nil {
		t.Fatal(err)
	}

	// Each invocation accepts one new TCP connection on the same listener,
	// with the same provider state and the same authenticated device key.
	session := func(name string, client func(context.Context) error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		deadline, _ := ctx.Deadline()
		if err := listener.SetDeadline(deadline); err != nil {
			t.Fatal(err)
		}
		served := make(chan error, 1)
		go func() {
			socket, err := listener.Accept()
			if err == nil {
				err = provider.ServeCatalog(ctx, socket, host)
			}
			served <- err
		}()
		clientErr := client(ctx)
		if clientErr != nil {
			cancel()
			listener.SetDeadline(time.Now())
		}
		serverErr := <-served
		if clientErr != nil || serverErr != nil {
			t.Fatalf("%s: requester=%v, provider=%v", name, clientErr, serverErr)
		}
	}
	// Independent wire oracle. It pins READ_SIZE, one 250-byte DATA, END
	// and EOF on success. A rejection must consist solely of ERROR and EOF.
	wantCatalog := []byte{0, 0, 0, 12, 8, 0x26, 0x12, 8, 0, 0, 0, 0, 0, 0, 0, 250}
	wantCatalog = append(wantCatalog, 0, 0, 0, 255, 8, 0x21, 0x12, 0xfa, 1)
	wantCatalog = append(wantCatalog, catalog...)
	wantCatalog = append(wantCatalog, 0, 0, 0, 2, 8, 0x22)
	denied := []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, 2}
	invalid := []byte{0, 0, 0, 5, 8, 0x7f, 0x12, 1, 1}
	sign := func(challenge [32]byte) []byte {
		record := append([]byte("HiveScope/recovery-read/v1\x00\x01"), hostKey[:]...)
		record = append(record, deviceKey[:]...)
		record = append(record, pkg.Catalog.FolderID[:]...)
		record = append(record, pkg.Catalog.AgreementID[:]...)
		record = binary.BigEndian.AppendUint64(record, pkg.Generation)
		record = append(record, pkg.Catalog.CatalogLocator[:]...)
		record = append(record, challenge[:]...)
		return ed25519.Sign(recovery, record)
	}
	probe := func(name string, proofs func([32]byte) ([][]byte, error), want []byte) {
		t.Helper()
		session(name, func(ctx context.Context) error {
			return lan.WithCatalogChallenge(ctx, pkg, device, func(challenge [32]byte, stream network.MuxedStream) error {
				signatures, err := proofs(challenge)
				if err != nil {
					return err
				}
				var frames []byte
				for _, signature := range signatures {
					frames = append(frames, 0, 0, 0, 68, 8, 8, 0x12, 64)
					frames = append(frames, signature...)
				}
				if _, err := stream.Write(frames); err != nil {
					return err
				}
				if err := stream.CloseWrite(); err != nil {
					return err
				}
				got, err := io.ReadAll(io.LimitReader(stream, int64(len(wantCatalog)+1)))
				if err != nil {
					return err
				}
				if !bytes.Equal(got, want) {
					return fmt.Errorf("unexpected reply through EOF: received %d wire bytes, expected %d", len(got), len(want))
				}
				return nil
			})
		})
	}
	var acceptedChallenge [32]byte
	var acceptedProof []byte
	probe("accepted proof", func(challenge [32]byte) ([][]byte, error) {
		acceptedChallenge, acceptedProof = challenge, sign(challenge)
		return [][]byte{acceptedProof}, nil
	}, wantCatalog)
	probe("same proof on a fresh connection", func(challenge [32]byte) ([][]byte, error) {
		if challenge == acceptedChallenge {
			return nil, errors.New("provider repeated its challenge")
		}
		return [][]byte{acceptedProof}, nil
	}, denied)
	probe("two proofs on one stream", func(challenge [32]byte) ([][]byte, error) {
		proof := sign(challenge)
		return [][]byte{proof, proof}, nil
	}, invalid)
	probe("invalid proof followed by a valid proof on one stream", func(challenge [32]byte) ([][]byte, error) {
		return [][]byte{acceptedProof, sign(challenge)}, nil
	}, invalid)
	session("fresh proof after replay rejection", func(ctx context.Context) error {
		snapshot, err := lan.ReadCatalog(ctx, pkg, device)
		if err != nil {
			return err
		}
		if snapshot == nil || snapshot.Name() != "hive.txt" || snapshot.Size() != 5 || snapshot.ObjectLocator() != [32]byte(repeated(0xd4, 32)) {
			return errors.New("unexpected authenticated snapshot")
		}
		return nil
	})
	probe("grant revoked while challenge was outstanding", func(challenge [32]byte) ([][]byte, error) {
		err := provider.LoadGrant(signedGrant(8, 2, make([]byte, 32)))
		return [][]byte{sign(challenge)}, err
	}, denied)
	t.Logf("current grant: %d catalog bytes; replay: DENIED, 0 catalog bytes; revoked during challenge: DENIED, 0 catalog bytes", len(catalog))
}
