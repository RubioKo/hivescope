// SPDX-License-Identifier: AGPL-3.0-only

package lan

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"net/netip"
	"testing"

	"github.com/RubioKo/hivescope/internal/grantstore"
	"github.com/libp2p/go-libp2p/core/crypto"
)

func wireTestManifest(t *testing.T) [276]byte {
	t.Helper()
	// The published five-byte fixture supplies both sealed digests and its
	// RFC 8032 public identities. The status reference uses the same manifest.
	field := func(s string) [32]byte {
		t.Helper()
		raw, err := hex.DecodeString(s)
		if err != nil || len(raw) != 32 {
			t.Fatal("invalid fixture field")
		}
		return [32]byte(raw)
	}
	provider := field("3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c")
	uploader := field("ec172b93ad5e563bf4932c70e1245034c35467ef2efd4d64ebf819683467e2bf")
	object := UploadItem{Locator: [32]byte(bytes.Repeat([]byte{0xd4}, 32)), Size: 21,
		Digest: field("695479c3a0c4494d2aaa784c4786fd9bf5fc1838114df1c4d72cc50466574398")}
	catalog := UploadItem{Locator: [32]byte(bytes.Repeat([]byte{0xc3}, 32)), Size: 250,
		Digest: field("96ec886154982747c3560a392c77ab56d33125215d42c95aad695954ca7fa1cf")}
	id := func(value byte) [16]byte { return [16]byte(bytes.Repeat([]byte{value}, 16)) }
	m, err := NewUploadManifest(provider, uploader, id(0xa1), id(0xb2), id(0x15), object, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(m[:]) != field("5cef2907ccccffa568c547206a2a4324294367cb9d54695ba33671d48a149e03") {
		t.Fatal("constructed manifest differs from the published fingerprint")
	}
	return m
}

func TestUploadManifestReferenceAndBounds(t *testing.T) {
	m := wireTestManifest(t)
	if err := validateUploadManifest(m); err != nil {
		t.Fatal(err)
	}
	ref := uploadReference(m)
	if !bytes.Equal(ref[:48], m[84:132]) || [32]byte(ref[48:]) != sha256.Sum256(m[:]) {
		t.Fatal("status reference differs from the manifest scope or fingerprint")
	}
	for _, tc := range []struct {
		objectSize, catalogSize uint64
		valid                   bool
	}{
		{16, 243, true}, {67_109_120, 497, true},
		{15, 250, false}, {67_109_121, 250, false}, {21, 242, false}, {21, 498, false},
	} {
		input := m
		binary.BigEndian.PutUint64(input[164:172], tc.objectSize)
		binary.BigEndian.PutUint64(input[236:244], tc.catalogSize)
		if (validateUploadManifest(input) == nil) != tc.valid {
			t.Errorf("unexpected result for lengths %d, %d", tc.objectSize, tc.catalogSize)
		}
	}
	wrongDomain := m
	wrongDomain[19] = 'x'
	collision := m
	copy(collision[204:236], m[132:164])
	sameKey := m
	copy(sameKey[52:84], m[20:52])
	badKey := m
	clear(badKey[52:84])
	for _, bad := range [][276]byte{wrongDomain, collision, sameKey, badKey} {
		if validateUploadManifest(bad) == nil {
			t.Fatal("accepted an invalid upload manifest")
		}
	}
}

func TestReservationRejectsMismatchedLocalIdentityBeforeDial(t *testing.T) {
	device, public, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	uploader, err := public.Raw()
	if err != nil {
		t.Fatal(err)
	}
	m := wireTestManifest(t)
	wrongUploader := [32]byte(m[52:84])
	pin, _ := grantTestKey(t)
	copy(m[52:84], uploader)
	// An invalid endpoint would fail inside withStorageStream. Both identity
	// checks must finish before any upload request can be sent there.
	for _, action := range []func([276]byte) error{
		func(m [276]byte) error {
			_, err := SubmitReservation(t.Context(), netip.AddrPort{}, pin, device, m)
			return err
		},
		func(m [276]byte) error {
			_, err := ReservationStatus(t.Context(), netip.AddrPort{}, pin, device, m)
			return err
		},
		func(m [276]byte) error {
			_, err := CancelUpload(t.Context(), netip.AddrPort{}, pin, device, m, 1702592000)
			return err
		},
	} {
		if err := action(m); !errors.Is(err, ErrProviderIdentity) {
			t.Fatalf("wrong manifest provider returned %v", err)
		}
		other := m
		copy(other[20:52], pin[:])
		copy(other[52:84], wrongUploader[:])
		if err := action(other); !errors.Is(err, ErrUploadDenied) {
			t.Fatalf("wrong manifest uploader returned %v", err)
		}
	}
}

func TestReservationStateRequiresCanonicalReplyAndEOF(t *testing.T) {
	frame := func(op byte, payload []byte) []byte {
		t.Helper()
		var wire bytes.Buffer
		if err := writeFrame(&wire, op, payload); err != nil {
			t.Fatal(err)
		}
		return wire.Bytes()
	}
	for _, mask := range []byte{0, 1, 2, 3} {
		pending := binary.BigEndian.AppendUint64([]byte{1, mask}, 1_700_000_000)
		got, err := readReservationState(bytes.NewReader(frame(stateOp, pending)), reserveOp, receiptExpectation{})
		if err != nil || got != (grantstore.UploadState{Code: 1, Mask: mask, Deadline: 1_700_000_000}) {
			t.Fatalf("pending state with mask %d: %+v, %v", mask, got, err)
		}
	}
	for _, state := range []byte{0, 3, 4} {
		got, err := readReservationState(bytes.NewReader(frame(stateOp, []byte{state})), statusOp, receiptExpectation{})
		if err != nil || got != (grantstore.UploadState{Code: state}) {
			t.Fatalf("state %d: %+v, %v", state, got, err)
		}
	}
	pending := frame(stateOp, binary.BigEndian.AppendUint64([]byte{1, 0}, 1))
	for _, code := range []byte{0, 3, 4} {
		got, err := readReservationState(bytes.NewReader(frame(stateOp, []byte{code})), cancelOp, receiptExpectation{})
		if err != nil || got != (grantstore.UploadState{Code: code}) {
			t.Fatalf("cancel returned %+v, %v for state %d", got, err, code)
		}
	}
	for _, bad := range [][]byte{pending, frame(stateOp, []byte{4, 0}), frame(stateOp, []byte{2}), frame(stateOp, []byte{5})} {
		if got, err := readReservationState(bytes.NewReader(bad), cancelOp, receiptExpectation{}); err == nil || got != (grantstore.UploadState{}) {
			t.Fatal("accepted a pending or malformed cancellation reply")
		}
	}
	for name, wire := range map[string][]byte{
		"missing": nil, "truncated": pending[:len(pending)-1], "extra frame": bytes.Repeat(pending, 2),
		"wrong op": frame(grantAckOp, []byte{1}), "empty": frame(stateOp, nil),
		"unknown after reserve": frame(stateOp, []byte{0}), "unsupported state": frame(stateOp, []byte{7}),
		"missing receipt": frame(stateOp, []byte{2}), "expired with extra byte": frame(stateOp, []byte{3, 0}),
		"mask outside two items": frame(stateOp, binary.BigEndian.AppendUint64([]byte{1, 4}, 1)),
		"zero deadline":          frame(stateOp, binary.BigEndian.AppendUint64([]byte{1, 0}, 0)),
		"incomplete pending":     frame(stateOp, []byte{1, 0}),
		"timestamp overflow":     frame(stateOp, binary.BigEndian.AppendUint64([]byte{1, 0}, math.MaxUint64)),
		"malformed error":        frame(errorOp, []byte{3, 0}),
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := readReservationState(bytes.NewReader(wire), reserveOp, receiptExpectation{}); err == nil || got != (grantstore.UploadState{}) {
				t.Fatal("unverified provider reply returned a state")
			}
		})
	}
	for code, want := range map[byte]error{
		2: ErrUploadDenied, 3: grantstore.ErrUploadConflict, 5: grantstore.ErrUploadCapacity,
		6: grantstore.ErrUploadPaused, 7: ErrUploadUnavailable,
	} {
		wire := frame(errorOp, []byte{code})
		if got, err := readReservationState(bytes.NewReader(wire), reserveOp, receiptExpectation{}); !errors.Is(err, want) || got != (grantstore.UploadState{}) {
			t.Fatalf("error %d: %+v, %v", code, got, err)
		}
	}
}

func TestSignedUploadStateBindsProviderManifestAndAgreedExpiry(t *testing.T) {
	m := wireTestManifest(t)
	want := receiptExpectation{[32]byte(m[20:52]), sha256.Sum256(m[:]), 1702592000}
	body := append([]byte("HiveScope/stored/v1\x00"), want.fingerprint[:]...)
	body = binary.BigEndian.AppendUint64(body, 1700000000)
	body = binary.BigEndian.AppendUint64(body, uint64(want.expiry))
	sig, err := hex.DecodeString("d391894a4b1772edefcca619d2ffad9cc9d67e223a3f0f8f643bfc1a4db853778fc23139c2702f0713d2d5d9635ea5222935924ecfac6d115465a0b228510405")
	if err != nil {
		t.Fatal(err)
	}
	receipt := [132]byte(append(body, sig...))
	for _, code := range []byte{2, 5} {
		var wire bytes.Buffer
		state := grantstore.UploadState{Code: code, Receipt: receipt}
		if err := writeUploadState(&wire, state); err != nil {
			t.Fatal(err)
		}
		b := wire.Bytes()
		for _, op := range []byte{finalizeOp, cancelOp} {
			got, err := readReservationState(bytes.NewReader(b), op, want)
			if err != nil || got != state {
				t.Fatalf("request %x state %d: %+v, %v", op, code, got, err)
			}
		}
		wrongKey, otherManifest, otherTerms := want, want, want
		wrongKey.provider = [32]byte(m[52:84])
		otherManifest.fingerprint[20] ^= 1
		otherTerms.expiry++
		for _, expected := range []receiptExpectation{wrongKey, otherManifest, otherTerms, {}} {
			if got, err := readReservationState(bytes.NewReader(b), statusOp, expected); err == nil || got != (grantstore.UploadState{}) {
				t.Fatal("returned a receipt without verifying all local terms")
			}
		}
		damaged := bytes.Clone(b)
		damaged[len(damaged)-1] ^= 1
		for _, bad := range [][]byte{damaged, b[:len(b)-1], append(bytes.Clone(b), 0)} {
			if got, err := readReservationState(bytes.NewReader(bad), finalizeOp, want); err == nil || got != (grantstore.UploadState{}) {
				t.Fatal("accepted an invalid receipt or trailing data")
			}
		}
	}
}
