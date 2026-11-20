<!-- docs-pair: file-catalog; revision: 3 -->

[English](file-catalog.md) | [Español](../es/file-catalog.md) | [Index](README.md)

# How Ana recovers the right file

**Proposed single-file format for the LAN trial.** Ana opens her recovery
package on a replacement computer. It points to an encrypted catalog.
After verifying the catalog's signature, her computer can open it to learn
which object to request, which key opens that version and which bytes to
expect. The provider stores the catalog and object under opaque locators.
It can observe their sizes and the reads. The file name and version key stay
inside the encrypted catalog. [Recovery package and read authorization](recovery-format.md)

## The catalog at rest

Version 1 describes one immutable file of 0 to 64 MiB. This proposed ceiling
bounds the first trial. At export time the owner pins the location of this
catalog. A new snapshot gets fresh random locators and needs a new export to
point to it. The previous export still refers to its original snapshot. Authentication
establishes which snapshot was signed. A replacement with only an old export
and one provider cannot establish that this is the newest snapshot.

The definitions below concatenate raw bytes in the stated order. A quoted
label is ASCII with no terminator unless a zero byte is shown separately.
All integers are unsigned, big-endian. All tags are 16 bytes.

| On disk, in order | Encoding |
| --- | --- |
| Header | Four ASCII bytes `HSC1`, a fresh 32-byte random salt, then `u32(entry_bytes)`. The header is 40 bytes. |
| Encrypted entry | Exactly `entry_bytes` of AES-256-GCM ciphertext followed by its tag. |
| Signature | 64 bytes of pure Ed25519 from the catalog signer whose public key Ana pinned in the package. |

Use the folder ID (16 bytes), agreement ID (16) and catalog locator (32)
from the unlocked package, in that order, as `context`. Derive a 32-byte
catalog key with HKDF-SHA-256, using `folder_read_key` as input key material,
the header salt, and the exact info string `HiveScope/catalog-key/v1`. Use a
12-byte zero nonce **once per derived key**. Each new seal requires a fresh
cryptographically random salt. A retry resends the original sealed bytes.
[HKDF](https://www.rfc-editor.org/rfc/rfc5869#section-2) · [AES-GCM](https://www.rfc-editor.org/rfc/rfc5116#section-5.2)

GCM's associated data is `HiveScope/catalog/v1`, one zero byte, `context` and
the complete header. The signer signs `HiveScope/catalog-sign/v1`, one zero
byte, `context`, the complete header, ciphertext and tag. Check the magic,
the declared entry length (123–377 bytes) and the exact total (120 plus
entry length) before allocation. Reject truncated or trailing bytes.
Verify the signature with `catalog_verify_key` and authenticate the GCM tag
before interpreting any entry field or requesting an object. The catalog
signing key is separate from the client's recovery signing key.
[Ed25519](https://www.rfc-editor.org/rfc/rfc8032#section-5.1)

| Entry field, in order | Bytes | What Ana does with it |
| --- | ---: | --- |
| Version ID | 16 | Identifies this immutable file version. |
| Object locator | 32 | Requests this exact opaque object under the same agreement. |
| Version key | 32 | Opens the blocks. Sealing this key inside the encrypted, signed entry protects it under the folder read key. |
| File length | 8 | Bounds recovery and determines block boundaries. Reject values above the trial ceiling. |
| Name length, display name | 2 + 1–255 | Count UTF-8 bytes. Require valid UTF-8 and an exact fit with the remaining digest. Ana chooses the local destination independently. Treat the name as display text and escape control characters when displaying it. |
| Object digest | 32 | SHA-256 of every encrypted block and tag in sequence. |

## Reading and checking the object

1. The writer takes a stable snapshot, generates a fresh random version ID and
   256-bit version key, then seals consecutive blocks of at most 4 MiB. Every
   block except the last is exactly 4 MiB. An empty file has one authenticated
   zero-length block. An exact multiple of 4 MiB has no extra empty block.
2. Number blocks from zero. For block `i`, the GCM nonce is eight zero bytes
   followed by `u32(i)`. Its associated data is `HiveScope/block/v1`, one zero
   byte, folder ID, version ID, `u64(file_bytes)`, `u32(i)` and
   `u32(block_plaintext_bytes)`. The object is the ordered concatenation of
   each block's ciphertext and tag. After an interruption, resend the stored
   ciphertext. If the sealed bytes cannot be recovered, restart sealing with
   a new version ID, key and object locator. Never encrypt a changed block
   with a key and index already used.
3. Ana requests the catalog and then the object with separate authorized
   reads. From the signed file length, the client expects
   `file_bytes + 16 × block_count` object bytes. It streams the encrypted
   records into SHA-256, authenticates each block with its expected index and
   size and stages verified plaintext in an exclusively created, owner-only
   temporary file on the destination filesystem. A short read, extra byte,
   bad tag or digest mismatch fails the recovery. Only after the final length,
   tags and digest match does it atomically publish the chosen destination.
   Write or verification failure before publication leaves the old destination
   intact. Clean up only this operation's owned temporary file.

For the reference restore, Ana selects a destination in a private local
directory. Require an explicit replacement choice if a file already exists.
Flush the verified temporary file before the atomic rename on the same
filesystem, then flush the parent directory before reporting recovery complete.
If that last flush fails, report **publication uncertain**. A complete verified
file may already be visible. On reopening, recover the same authenticated
snapshot to a separate private temporary file and compare the destination
byte for byte. If they match, flush the destination and parent before confirming.
If recovery is unavailable, keep the outcome uncertain. If the bytes differ,
leave the destination intact and ask for a new one. Record an
interrupted restore separately from a failed authentication. [Provider and filesystem assumptions](provider-persistence.md)

For a restore that can be checked after restart, the caller supplies a private
state directory dedicated to that restore. Before publication, save an opaque
record that binds the exact signed catalog to the chosen absolute destination.
The record contains neither the path nor any keys or file contents. On reopening,
the caller supplies that directory, the original recovery context and the same
destination. Check the binding before comparing the file. Keep the record for
repeated checks until the caller discards it. Use a fresh state directory for
each attempt, separate from the destination's parent.

The local file named `restore-v1` holds 32 bytes from HKDF-SHA-256. Its input
key is the authenticated version key, its salt is the verified catalog signature
and its `info` is `HiveScope/local-publication/v1`, one zero byte and the absolute
destination path in the platform's Go string representation. The client creates
this record exclusively and flushes the file and directory before the rename.
Its presence identifies the intended restore. Confirmation also requires a
fresh comparison and both destination flushes. [HKDF](https://www.rfc-editor.org/rfc/rfc5869)

## Three reproducible catalog entries

The first row continues Ana's generation 7 recovery example. For all rows
use the same folder, agreement, folder read key and catalog signing test key
as in the [recovery fixtures](recovery-format.md). Use the public TEST 1024
fixture from RFC 8032 §7.1 to sign. Each hex pair in the last two columns
means that byte repeated to fill the corresponding field, in the order
shown. These fixed values are only for comparing implementations.

| Display name | Plaintext bytes | Catalog / object locators | Version ID / key / catalog salt |
| --- | --- | --- | --- |
| `hive.txt` | `48 69 76 65 0a` | `c3 / d4` | `71 / 81 / 91` |
| `empty.bin` | Zero bytes | `c4 / d5` | `72 / 82 / 92` |
| `edge.bin` | 4,194,304 bytes of hex `61`, then one byte `62` | `c5 / d6` | `73 / 83 / 93` |

The encrypted object for `hive.txt` is exactly
`bac6d517956355ed4498a00c916ef93307ce8c69f8`. The entry that points to it
is 130 bytes. Its signed catalog is 250 bytes. The following reference values
were checked by verifying each signature and decrypting every block with
standard cryptographic libraries. The LAN recovery remains a prototype test.

| File | Blocks | Object bytes | SHA-256 of object | SHA-256 of signed catalog |
| --- | ---: | ---: | --- | --- |
| `hive.txt` | 1 | 21 | `695479c3a0c4494d2aaa784c4786fd9bf5fc1838114df1c4d72cc50466574398` | `96ec886154982747c3560a392c77ab56d33125215d42c95aad695954ca7fa1cf` |
| `empty.bin` | 1 | 16 | `a543d20ea0e7176c0a2a0883c09f85a8d5df42e683334d48a349cd94679fbcbb` | `5a4bec97a22e1043f605df722f7b8f80142ffaa66642a4bcb5e03007cef72f40` |
| `edge.bin` | 2 | 4,194,337 | `6d1e257b5e9f01c9bb0cc90ce845c857ab4fe173d45dc2b0b46b13520f3b018b` | `aa5b005c0b98ffa622e41d4b6a773c81073ea6d2db9b3309dc03beb0d98a450b` |

**Substituted block.** Keep the authentic catalog for `hive.txt`. Replace its
only encrypted record with a valid record for the same five plaintext bytes,
sealed under version ID `74 × 16` and key `84 × 32`. The replacement bytes
are `3a5fd4b146e8da1f9141eea7225f8b00b480c6ac5e`. The length still matches.
The expected key and associated data fail to open it, and its SHA-256 differs
from the signed digest. A destination containing `6b 65 65 70 0a` must retain
those five bytes. A revoked reader is denied at the provider before any
catalog or object bytes are returned. A valid catalog cannot override that
decision. [Revocation example](recovery-format.md)
