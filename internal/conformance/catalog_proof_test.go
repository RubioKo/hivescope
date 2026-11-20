// SPDX-License-Identifier: AGPL-3.0-only

package conformance

import (
	"context"
	"errors"
	"testing"

	"github.com/RubioKo/hivescope/internal/lan"
	"github.com/libp2p/go-libp2p/core/network"
)

func TestCatalogProofAcceptedOnce(t *testing.T) {
	object, catalog := fiveByteSnapshot(t)
	trial := newLoopbackRead(t, hiveFixture(), object, catalog)
	pkg, device := trial.pkg, trial.device
	session := func(name string, client func(context.Context) error) {
		t.Helper()
		trial.session(t, name, 1, client)
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
		return trial.signRead(1, pkg.Catalog.CatalogLocator, challenge)
	}
	probe := func(name string, proofs func([32]byte) ([][]byte, error), want []byte) {
		t.Helper()
		session(name, func(ctx context.Context) error {
			return lan.WithCatalogChallenge(ctx, pkg, device, func(challenge [32]byte, stream network.MuxedStream) error {
				signatures, err := proofs(challenge)
				if err != nil {
					return err
				}
				return checkProofReply(stream, want, signatures...)
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
		err := trial.provider.LoadGrant(trial.signedGrant(8, 2, make([]byte, 32)))
		return [][]byte{sign(challenge)}, err
	}, denied)
	t.Logf("current grant: %d catalog bytes; replay: DENIED, 0 catalog bytes; revoked during challenge: DENIED, 0 catalog bytes", len(catalog))
}
