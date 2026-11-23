// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/RubioKo/hivescope/internal/storageformat"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

func objectRequest(pkg *storageformat.RecoveryPackage, snapshot *storageformat.Snapshot) ([]byte, error) {
	if pkg == nil || !snapshot.FromCatalog(pkg.Catalog) {
		return nil, errors.New("object read requires the verified catalog for this recovery package")
	}
	return readRequest(pkg, objectKind, snapshot.ObjectLocator()), nil
}

// WithObjectChallenge starts a separate read of the object named in the
// verified catalog. The continuation runs only after the authenticated
// provider has issued a fresh challenge for that object.
func WithObjectChallenge(ctx context.Context, pkg *storageformat.RecoveryPackage, device crypto.PrivKey,
	snapshot *storageformat.Snapshot, use func([32]byte, network.MuxedStream) error) error {
	request, err := objectRequest(pkg, snapshot)
	if err != nil {
		return err
	}
	return withReadChallenge(ctx, pkg, device, request, use)
}

// RecoverFile reads the signed catalog, then authenticates a separate read
// for its object. The caller supplies an existing private staging directory
// and must close the verified plaintext reader to remove the staged file.
func RecoverFile(ctx context.Context, pkg *storageformat.RecoveryPackage, device crypto.PrivKey, privateDir string) (io.ReadCloser, error) {
	if privateDir == "" {
		return nil, errors.New("missing private staging directory")
	}
	var verified io.ReadCloser
	err := withRecoveryObject(ctx, pkg, device, func(snapshot *storageformat.Snapshot, sealed io.Reader) error {
		var err error
		verified, err = snapshot.OpenObject(sealed, privateDir)
		return err
	})
	if err != nil {
		return nil, err
	}
	return verified, nil
}

// RecoverFileTo verifies the remote snapshot and atomically publishes it at
// the selected destination. The caller supplies a private destination directory
// and a separate private state directory dedicated to this attempt. The latter
// retains the restore intent if the final directory sync is uncertain.
func RecoverFileTo(ctx context.Context, pkg *storageformat.RecoveryPackage, device crypto.PrivKey,
	destination string, replace bool, stateDir string) error {
	if destination == "" || stateDir == "" {
		return errors.New("recovery requires a destination and a private restore state directory")
	}
	return withRecoveryObject(ctx, pkg, device, func(snapshot *storageformat.Snapshot, sealed io.Reader) error {
		return snapshot.SaveRecoverableObject(sealed, destination, replace, stateDir)
	})
}

func withRecoveryObject(ctx context.Context, pkg *storageformat.RecoveryPackage, device crypto.PrivKey,
	receive func(*storageformat.Snapshot, io.Reader) error) error {
	snapshot, err := ReadCatalog(ctx, pkg, device)
	if err != nil {
		return err
	}
	request, err := objectRequest(pkg, snapshot)
	if err != nil {
		return err
	}
	err = withAuthorizedRead(ctx, pkg, device, request, func(r io.Reader) error {
		size, err := readSealedSize(r)
		if err != nil {
			return err
		}
		if size != snapshot.SealedObjectSize() {
			return errors.New("declared object size differs from the verified catalog")
		}
		return receive(snapshot, &sealedItemReader{wire: r, remaining: size})
	})
	if err != nil {
		return fmt.Errorf("recover object: %w", err)
	}
	return nil
}
