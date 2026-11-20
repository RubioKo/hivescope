// SPDX-License-Identifier: AGPL-3.0-only

package conformance

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

func TestFiveByteUploadReceipt(t *testing.T) {
	// Public test vectors from RFC 8032 section 7.1: TEST 2 (provider)
	// and TEST SHA(abc) (uploader), as pinned in upload-confirmation.md.
	providerSeed := mustHex(t, "4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb")
	providerKey := ed25519.PublicKey(mustHex(t, readFixtureProviderPublic))
	uploaderKey := mustHex(t, "ec172b93ad5e563bf4932c70e1245034c35467ef2efd4d64ebf819683467e2bf")
	object, catalog := fiveByteSnapshot(t)
	objectDigest := sha256.Sum256(object)
	catalogDigest := sha256.Sum256(catalog)

	manifest := []byte("HiveScope/upload/v1\x00")
	manifest = append(manifest, providerKey...)
	manifest = append(manifest, uploaderKey...)
	manifest = append(manifest, repeated(0xa1, 16)...) // folder
	manifest = append(manifest, repeated(0xb2, 16)...) // agreement
	manifest = append(manifest, repeated(0x15, 16)...) // request
	manifest = append(manifest, repeated(0xd4, 32)...) // object locator
	objectSizeOffset := len(manifest)
	manifest = binary.BigEndian.AppendUint64(manifest, uint64(len(object)))
	manifest = append(manifest, objectDigest[:]...)
	manifest = append(manifest, repeated(0xc3, 32)...) // catalog locator
	manifest = binary.BigEndian.AppendUint64(manifest, uint64(len(catalog)))
	manifest = append(manifest, catalogDigest[:]...)
	if len(manifest) != 276 {
		t.Fatalf("manifest length = %d, want 276", len(manifest))
	}
	requireDigest(t, manifest, "5cef2907ccccffa568c547206a2a4324294367cb9d54695ba33671d48a149e03")

	message := fixtureReceiptMessage(manifest)
	if len(message) != 68 {
		t.Fatalf("signed receipt message length = %d, want 68", len(message))
	}
	signature := ed25519.Sign(ed25519.NewKeyFromSeed(providerSeed), message)
	publishedSignature := mustHex(t, "d391894a4b1772edefcca619d2ffad9cc9d67e223a3f0f8f643bfc1a4db85377"+
		"8fc23139c2702f0713d2d5d9635ea5222935924ecfac6d115465a0b228510405")
	if !bytes.Equal(signature, publishedSignature) {
		t.Fatalf("receipt signature = %x, want %x", signature, publishedSignature)
	}
	if !ed25519.Verify(providerKey, message, publishedSignature) {
		t.Fatal("published receipt signature failed verification with the pinned provider key")
	}
	receipt := append(message, signature...)
	if len(receipt) != 132 {
		t.Fatalf("signed receipt length = %d, want 132", len(receipt))
	}
	requireDigest(t, receipt, "ff68ed2f38d36637e790a353415e3d69c92cdf6eabcfa5639250ae48b2175e2d")

	// Keep the original sealed data and signature. Change only the manifest's
	// claimed object length from 21 to 22 and recompute its receipt message.
	binary.BigEndian.PutUint64(manifest[objectSizeOffset:], uint64(len(object))+1)
	if ed25519.Verify(providerKey, fixtureReceiptMessage(manifest), publishedSignature) {
		t.Fatal("receipt signature accepted a manifest with a changed object length")
	}
}

func fixtureReceiptMessage(manifest []byte) []byte {
	fingerprint := sha256.Sum256(manifest)
	message := append([]byte("HiveScope/stored/v1\x00"), fingerprint[:]...)
	message = binary.BigEndian.AppendUint64(message, 1700000000) // commit time
	return binary.BigEndian.AppendUint64(message, 1702592000)    // agreement expiry
}
