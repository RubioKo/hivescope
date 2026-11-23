// SPDX-License-Identifier: AGPL-3.0-only

package conformance

import (
	"bytes"
	"crypto/cipher"
	"crypto/ed25519"
	"encoding/binary"
	"net/netip"
	"testing"

	"github.com/RubioKo/hivescope/internal/storageformat"
	"golang.org/x/crypto/argon2"
)

// All identities below are public RFC 8032 section 7.1 test vectors.
// The fixed salt, nonce, passphrase and challenges belong only to the
// synthetic example in docs/en/recovery-format.md.
const (
	readFixtureRecoverySeed   = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
	readFixtureRecoveryPublic = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
	readFixtureProviderPublic = "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c"
	readFixtureNewPeerPublic  = "fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025"
	readFixtureCatalogPublic  = "278117fc144c72340f67d0f2316e8386ceffbf2b2428c9c51fef7c597f1d426e"
)

func recoveryFixtureAEAD(t *testing.T, passphrase string, salt []byte) cipher.AEAD {
	t.Helper()
	key := argon2.IDKey([]byte(passphrase), salt, 3, 65536, 4, 32)
	return fixtureGCM(t, key)
}

type recoveryExample struct {
	header, payload []byte
	aead            cipher.AEAD
}

func recoveryFixture(t *testing.T, passphrase string, variant byte) recoveryExample {
	t.Helper()
	salt := mustHex(t, "000102030405060708090a0b0c0d0e0f")
	nonce := mustHex(t, "101112131415161718191a1b")
	// Variant zero is the published example. Give each deliberately
	// different authenticated payload its own synthetic salt and nonce.
	salt[0] ^= variant
	nonce[0] ^= variant
	pkg := &storageformat.RecoveryPackage{
		Catalog: hiveFixture().context(t), Generation: 7,
		ProviderKey:  [32]byte(mustHex(t, readFixtureProviderPublic)),
		Endpoint:     netip.MustParseAddrPort("192.0.2.10:4001"),
		RecoverySeed: [32]byte(mustHex(t, readFixtureRecoverySeed)),
	}
	return encodeRecoveryExample(t, pkg, passphrase, salt, nonce)
}

func encodeRecoveryExample(t *testing.T, pkg *storageformat.RecoveryPackage, passphrase string, salt, nonce []byte) recoveryExample {
	t.Helper()
	header := append([]byte("HSRP\x01"), salt...)
	header = append(header, nonce...)
	scope := pkg.Catalog
	payload := append([]byte{}, scope.FolderID[:]...)
	payload = append(payload, scope.AgreementID[:]...)
	payload = binary.BigEndian.AppendUint64(payload, pkg.Generation)
	payload = append(payload, pkg.ProviderKey[:]...)
	ip := pkg.Endpoint.Addr().As4()
	payload = append(payload, ip[:]...)
	payload = binary.BigEndian.AppendUint16(payload, pkg.Endpoint.Port())
	payload = append(payload, pkg.RecoverySeed[:]...)
	payload = append(payload, scope.FolderReadKey[:]...)
	payload = append(payload, scope.CatalogVerifyKey[:]...)
	payload = append(payload, scope.CatalogLocator[:]...)

	return recoveryExample{header, payload, recoveryFixtureAEAD(t, passphrase, salt)}
}

func (f recoveryExample) seal() []byte {
	return f.aead.Seal(bytes.Clone(f.header), f.header[21:33], f.payload, f.header)
}

func TestRecoveryPackage(t *testing.T) {
	f := recoveryFixture(t, "HiveScope fixture 1", 0)
	export := f.seal()
	if len(f.header) != 33 || len(f.payload) != 206 || len(export) != 255 {
		t.Fatalf("header/payload/export lengths = %d/%d/%d, want 33/206/255",
			len(f.header), len(f.payload), len(export))
	}
	requireDigest(t, export, "943561b2a66368c9f780b44fab56aa89a1d46c57b20a7d3b73b39dc090d548b4")
	if !bytes.HasSuffix(export, mustHex(t, "954c1dd1d641d7aab3ecdf664d56ef7d")) {
		t.Fatal("recovery package authentication tag differs from published value")
	}

	opened, err := f.aead.Open(nil, export[21:33], export[33:], export[:33])
	if err != nil || !bytes.Equal(opened, f.payload) {
		t.Fatalf("recovery package did not reproduce the fixture payload (error: %v)", err)
	}
	wrongKey := recoveryFixtureAEAD(t, "wrong fixture passphrase", export[5:21])
	if opened, err := wrongKey.Open(nil, export[21:33], export[33:], export[:33]); err == nil || len(opened) != 0 {
		t.Fatal("wrong passphrase must reject the package without plaintext")
	}
}

// The last two signatures also verify: they are valid proofs made with the
// old key. Enforcing revocation is the provider's separate M1-03 trial.
func TestRecoveryReadTranscripts(t *testing.T) {
	recoveryKey := ed25519.NewKeyFromSeed(mustHex(t, readFixtureRecoverySeed))
	verificationKey := ed25519.PublicKey(mustHex(t, readFixtureRecoveryPublic))
	provider := mustHex(t, readFixtureProviderPublic)
	replacement := mustHex(t, readFixtureNewPeerPublic)

	for _, tc := range []struct {
		name      string
		operation byte
		locator   byte
		challenge byte
		digest    string
		signature string
	}{
		{
			"active/catalog", 1, 0xc3, 0x11,
			"051413d2b0f8963776b0757fa880d42dbb719b00ec1cb1e59c26aee6471eec43",
			"fdf3cebfcb16055487868992ba1a08ac944f10bfefba8214f7ce539051d98025" +
				"5625bfcd48e01b8919bf78a080478dddafa69f8d0727a6592dd920470436b60a",
		},
		{
			"active/object", 2, 0xd4, 0x22,
			"81ed1365abcf3e1199a232837f940022f5ad3b1b866bf688199ae385a1e456d4",
			"60c3faf2269e13e4c1dd5f3b2c4869d82213e4fc88cad0db08aab237780b689a" +
				"e33e4534314dbe532c9d47e1fc7405f247d7ac313ef3eb755bd18535ddc06806",
		},
		{
			"old-grant/catalog", 1, 0xc3, 0x33,
			"5092bbf2018d71746727ae0b1936d4153ca93fb3e790cba191b40397b2083326",
			"93ef07ba8fcb1d785d4904c41a8bae9cb59d7c932be1e402ae8e4e11271b8d7d" +
				"08a8923305a86132df74305abe70cba7cf7fed87a79507f2932fcea8d2ce9f06",
		},
		{
			"old-grant/object", 2, 0xd4, 0x44,
			"76a16f0b3aa8a8934557d0a9c698e4cee327f932d0b2475e2efa62159c0dc403",
			"ce65e6e3d879f9a2e1e384193c5767474a0f974afeae75b522a18dae04dc1c99" +
				"25677b6f04d6e6428c8149f5ebb63b0b074edc2f446d01e9be184b001e3bfe07",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transcript := append([]byte("HiveScope/recovery-read/v1\x00"), tc.operation)
			transcript = append(transcript, provider...)
			transcript = append(transcript, replacement...)
			transcript = append(transcript, repeated(0xa1, 16)...)
			transcript = append(transcript, repeated(0xb2, 16)...)
			transcript = binary.BigEndian.AppendUint64(transcript, 7)
			transcript = append(transcript, repeated(tc.locator, 32)...)
			transcript = append(transcript, repeated(tc.challenge, 32)...)
			if len(transcript) != 196 {
				t.Fatalf("read transcript has %d bytes, want 196", len(transcript))
			}
			requireDigest(t, transcript, tc.digest)
			signature := ed25519.Sign(recoveryKey, transcript)
			if !bytes.Equal(signature, mustHex(t, tc.signature)) {
				t.Fatal("read signature differs from published value")
			}
			if !ed25519.Verify(verificationKey, transcript, signature) {
				t.Fatal("read signature failed verification with the pinned recovery key")
			}
		})
	}
}
