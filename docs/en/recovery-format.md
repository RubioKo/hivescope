<!-- docs-pair: recovery-format; revision: 4 -->

[English](recovery-format.md) | [Español](../es/recovery-format.md) | [Index](README.md)

# Recovering one file on a replacement computer

**Proposed format v1 for the desktop LAN trial.** Ana's original computer is off.
She has her recovery package, its passphrase and access to the provider that
stored her encrypted file. The package lets her new computer identify that
provider, prove its right to read and decrypt the result locally. The provider
can confirm or deny each read independently. Go tests cover local unlock
and a catalog request to a provider authenticated over loopback TCP. The
read authorization and file transfer below describe the planned CLI and
provider. [First recovery trial](readiness.md)

## What Ana keeps

An explicit export produces one 255-byte binary file for one folder and one
agreement. The only cleartext is the 33-byte header. Read from left to right.
Numbers are unsigned, big-endian. Byte counts are exact.

| Part | Contents, in order | Bytes |
| --- | --- | ---: |
| Header | ASCII `HSRP`, version byte `01`, random salt (16), random GCM nonce (12) | 33 |
| Body | AES-256-GCM ciphertext of the payload below | 206 |
| Tag | Full GCM authentication tag | 16 |

The passphrase is its exact UTF-8 encoding, preserving spaces and Unicode code
points. Use Argon2id version 0x13 with 65,536 KiB, three passes, four lanes, the
header salt, empty optional secret and associated data, and a 32-byte output.
Use that output once as the AES key and authenticate the entire header as GCM
associated data. Every export, including a retry, uses a fresh cryptographically
random salt and nonce. An importer checks the size, magic and version before
deriving a key. A bad tag or wrong passphrase ends recovery locally, before
parsing the payload, contacting a provider or writing a destination file.
A copied export permits offline password guessing. The owner should use a
long, unique passphrase and keep it separately. [Argon2id profile](https://www.rfc-editor.org/rfc/rfc9106#section-7.4) · [AES-GCM parameters](https://www.rfc-editor.org/rfc/rfc5116#section-5.2)

| Payload offset | Field | Purpose |
| ---: | --- | --- |
| 0 | `folder_id` (16) | Opaque folder identifier. |
| 16 | `agreement_id` (16) | Opaque storage agreement identifier. |
| 32 | `generation` (8) | Positive generation of the owner's registered recovery grant. |
| 40 | `provider_key` (32) | Pinned raw Ed25519 public key of the provider's libp2p identity. |
| 72 | `ipv4` (4), `tcp_port` (2) | Direct LAN endpoint in network byte order, port 1–65535. |
| 78 | `recovery_seed` (32) | Ed25519 seed for proving possession of the registered read grant. |
| 110 | `folder_read_key` (32) | Folder read secret for opening the encrypted catalog and recovering its per-version keys. |
| 142 | `catalog_verify_key` (32) | Pinned Ed25519 public key of the authorized catalog signer. |
| 174 | `catalog_locator` (32) | Opaque identifier of the encrypted catalog to request first. |

The owner keeps the folder administration key separately. The replacement
generates its own device key. Before requesting a challenge it authenticates
the provider over libp2p and compares the remote identity key with the pinned
bytes. The IPv4 endpoint is a connection hint. A changed or unreachable
endpoint cannot substitute a different identity. This first profile uses
Ed25519 peer identities and direct TCP/IPv4. Reject generation zero, port
zero or malformed keys before connecting. The file's version key is sealed
inside the signed catalog. [File catalog and block verification](file-catalog.md) · [Peer authentication](https://github.com/libp2p/specs/blob/master/tls/tls.md)

After authenticating the payload, validate both pinned Ed25519 public keys.
Decode each as a point on Edwards25519 and require its canonical encoding to
match the original 32 bytes. Reject a point whose product by the cofactor
eight is the identity. The recovery seed and symmetric folder read key are
each 32 opaque bytes. The Go reader uses `SetBytes`, `Bytes` and
`MultByCofactor` for the public-key checks. [Ed25519 point decoding](https://www.rfc-editor.org/rfc/rfc8032#section-5.1.3) · [Go point operations](https://pkg.go.dev/filippo.io/edwards25519@v1.2.0#Point)

## What the replacement signs

For each read, sign the concatenation below with pure Ed25519. Quoted text
means exact ASCII. Hex pairs and key material represent raw bytes. The number
in parentheses gives the field size in bytes. No separators, padding, text
encodings of keys or length prefixes are added. The provider verifies the
64-byte signature against the active recovery key it has on record.
[Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-5.1)

| Order | Signed fields |
| --- | --- |
| 1 | ASCII `HiveScope/recovery-read/v1`, one zero byte, then the operation (1): `01` reads a catalog and `02` reads an object. |
| 2 | Provider public key (32), then replacement device public key (32), both from the authenticated connection. |
| 3 | Folder ID (16), agreement ID (16), then grant generation (8). |
| 4 | Requested locator (32), then the provider's fresh random challenge (32). |

The replacement builds the signed bytes locally from its intended read and
the two authenticated identities. It accepts only the challenge for that
pending read. The provider binds the challenge to this connection and these exact fields,
expires it 60 seconds after issuance on its own monotonic clock and consumes
it atomically on the first response. Retries and reconnects require a fresh
challenge. At the read decision it checks the signature, current durable
grant, scope, item membership and agreement's read window. Serialize this
decision with revocation so no new read can use the old grant after revocation
is acknowledged. Revocation cannot recall bytes already delivered. After
restart, discard challenges, reload the highest persisted generation and deny
reads if that state is unavailable. A denial returns zero catalog or object
bytes and leaves an existing destination intact. [Agreement and grace period](storage.md)

## Two examples with the same export

Only public test material is used here. In the notation below, `a1 × 16`
means sixteen bytes of value 0xa1. The recovery, provider and replacement
keys are respectively TEST 1, TEST 2 and TEST 3 in RFC 8032 §7.1. Use the
public key from TEST 1024 for catalog verification. These keys are reserved
for examples. [Published Ed25519 fixtures](https://www.rfc-editor.org/rfc/rfc8032#section-7.1)

The export has folder `a1 × 16`, agreement `b2 × 16`, generation `7`, endpoint
`192.0.2.10:4001`, folder read key `e5 × 32` and catalog locator `c3 × 32`.
The object locator in the trial catalog is `d4 × 32`. For this example only,
the passphrase is `HiveScope fixture 1`, the salt is the ascending bytes
`00` through `0f`, and the nonce is `10` through `1b`. Repeated challenge
bytes below make the examples reproducible.

| Provider state | Catalog read | Object read | Expected outcome |
| --- | --- | --- | --- |
| **A. Active at generation 7.** Original computer off, clean replacement, provider reachable and agreement readable. | Sign `01` for `c3 × 32` with challenge `11 × 32`. | After authenticating the catalog, sign `02` for `d4 × 32` with a new challenge `22 × 32`. | Both reads succeed. After local verification and decryption, the output equals the independent sample `48 69 76 65 0a` byte for byte. |
| **B. Revoked at generation 8.** The provider has acknowledged the owner's signed revocation and restarted. The replacement still holds the generation 7 export. | Try the old grant with a fresh challenge `33 × 32`. | Independently try the known object locator with a fresh challenge `44 × 32`. | Even correctly signed requests for generation 7 yield zero bytes of either item. An existing destination containing `6b 65 65 70 0a` retains those exact bytes. |

The signature identifies the recovery key. The provider's stored authorization
determines whether it may still read. If Ana loses the provider or her
passphrase, this one-provider trial cannot restore the file. An old export and
a single provider also cannot prove that a catalog is the latest version.

## Byte-level reference values

The example package has SHA-256
`943561b2a66368c9f780b44fab56aa89a1d46c57b20a7d3b73b39dc090d548b4`
and ends with tag `954c1dd1d641d7aab3ecdf664d56ef7d`.

Each bracket identifies the repeated challenge byte. The first line is the
SHA-256 of the 196 signed bytes, followed by the 64-byte signature split over
two lines. Sign the 196 bytes directly. The hash is a check on the encoding.
All four signatures verify mathematically, including the two that the provider
must refuse in case B. The values were checked using the published test keys.
The provider and recovery outcomes above still require a working prototype.

```text
[11]
051413d2b0f8963776b0757fa880d42dbb719b00ec1cb1e59c26aee6471eec43
fdf3cebfcb16055487868992ba1a08ac944f10bfefba8214f7ce539051d98025
5625bfcd48e01b8919bf78a080478dddafa69f8d0727a6592dd920470436b60a

[22]
81ed1365abcf3e1199a232837f940022f5ad3b1b866bf688199ae385a1e456d4
60c3faf2269e13e4c1dd5f3b2c4869d82213e4fc88cad0db08aab237780b689a
e33e4534314dbe532c9d47e1fc7405f247d7ac313ef3eb755bd18535ddc06806

[33]
5092bbf2018d71746727ae0b1936d4153ca93fb3e790cba191b40397b2083326
93ef07ba8fcb1d785d4904c41a8bae9cb59d7c932be1e402ae8e4e11271b8d7d
08a8923305a86132df74305abe70cba7cf7fed87a79507f2932fcea8d2ce9f06

[44]
76a16f0b3aa8a8934557d0a9c698e4cee327f932d0b2475e2efa62159c0dc403
ce65e6e3d879f9a2e1e384193c5767474a0f974afeae75b522a18dae04dc1c99
25677b6f04d6e6428c8149f5ebb63b0b074edc2f446d01e9be184b001e3bfe07
```
