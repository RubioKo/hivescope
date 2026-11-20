<!-- docs-pair: readiness; revision: 6 -->

[English](readiness.md) | [Español](../es/readiness.md) | [Index](README.md)

# Is HiveScope ready for its first prototype?

The [architecture](architecture.md) defines the node, roles, trust boundaries,
data flow and storage model. The [roadmap](roadmap.md) sets the milestones.
Contributors can start specifying the secure core now. Implementing each part
requires settling the corresponding contract below. The current CI checks
documentation. Reliability, interoperability and recovery require running software.

## First recovery from one storage provider

**Proposed scope for milestone 1.** Use one owner's device, one storage provider
at a known LAN address and a direct connection. A third, clean device profile represents
the owner after losing the original device. A single provider keeps a complete
encrypted object and a minimal authenticated catalog that points to it. Losing
that provider makes this demonstration unavailable. The three-provider placement
policy and the `protected` state belong to milestone 3.

1. The peers authenticate each other. The provider checks the request against its
   local resource profile and reserves capacity under a storage agreement. The client
   records the provider's identity for recovery. The test supplies its address directly.
2. The owner encrypts a synthetic file and uploads it and the catalog. The
   provider uses the reservation, persists the bytes and acknowledges the operation.
3. The owner exports a password-protected recovery package. Switch off the original
   device and start with an empty local database and a fresh device identity.
4. Using only the package, its password and the reachable provider, recover the
   file. Compare every byte with an independent test fixture. Record which
   credential authorized retrieval and which local information was required.

## How a replacement device earns its first read

**Design decision for review.** The provider accepts proof of possession of a
recovery signing key that the owner registered for one folder and one storage
agreement. Its only permission is to fetch that folder's encrypted catalog and
the encrypted objects covered by the agreement. This lets the owner recover a
copy after losing a device while keeping administrative authority separate.

| Who | What they hold and do |
| --- | --- |
| Owner | At agreement setup, both parties pin the folder authority's public key. The owner uses its private key to sign the initial recovery grant and later replacements or revocations. The owner keeps that administrative key separately from the recovery package. |
| Provider | Stores the signed grant with the opaque folder and agreement IDs, provider identity, recovery public key, read scope and generation. Persists the highest accepted generation and its active or revoked status. |
| Replacement device | Creates its own device identity. The password unlocks the package locally. It contains the recovery signing key and grant reference (including its generation), folder decryption and catalog verification material, an opaque catalog locator, and the storage provider's connection details and pinned fingerprint. |

1. The new device checks the provider fingerprint on an authenticated libp2p
   connection. The provider can challenge this new peer before granting any read.
   [libp2p peer authentication](https://github.com/libp2p/specs/blob/master/tls/tls.md)
2. For each catalog or object read, the provider issues a fresh, short-lived,
   single-use challenge on that connection. The device signs it with the recovery
   key, binding the recovery-read operation, both peer identities, agreement,
   folder, grant generation and requested opaque locator. The password and
   private keys stay on the device.
3. Before sending bytes the provider consumes the challenge once, checks the
   signature against its stored active key, matches the request to that connection
   and grant, and checks the requested item and the agreement's read window.
   Existing data is readable during the [agreement's grace period](storage.md).
   Missing authorization state, a stale grant or an out-of-scope request is denied.
   A successful read grants no write, renewal or device-enrollment rights.
4. The device verifies and decrypts the catalog, then requests its encrypted
   object under the same checks. It authenticates the recovered contents before
   replacing the destination. An old export and a single provider cannot
   establish that the catalog is the latest version.

The provider verifies grant records with the folder authority pinned at setup.
It accepts changes only at a higher generation for the same scope. To revoke,
the owner signs a record marking the grant revoked.
The provider acknowledges it after persisting the change and invalidating
outstanding challenges for the old grant. Restart discards outstanding challenges
and retains the revocation. A replacement grant uses a new key and requires a
new export. Anyone with the old package and password can use it until the
provider learns of revocation. Revoking one device identity alone does not revoke
the recovery key. Previously delivered bytes and keys remain with their holder.

The [proposed v1 recovery format](recovery-format.md) gives the package layout,
the exact bytes signed for each read and two examples using the same export.

## Decisions still needed for implementation

The first-read rule above settles part of decision 1. The remaining work in
each row needs a reviewed specification in English and Spanish.

| Order | Design question | Evidence needed to start that component |
| --- | --- | --- |
| 1. Identity and recovery | How are local secrets unlocked and the separate administrative key kept recoverable? | Specify custody and local unlock on each target OS. Use the scoped first-read grant and revocation cases above as constraints. |
| 2. File and catalog format | The [single-file catalog proposal](file-catalog.md) specifies the version key, encrypted blocks and retry rules. | Reference values cover a five-byte file, an empty file, a two-block file and a substituted record. A future CLI must reproduce them and demonstrate that a failed restore preserves the destination. |
| 3. Transfer protocol | The [upload contract](upload-confirmation.md) defines the request identity, signed receipt and retry outcomes. | Specify the versioned messages for reservation, transfer, status and finalization, including frame sizes, concurrency, time limits and errors. |
| 4. Disk and restart behavior | The upload contract specifies the commit point and how to reconcile the two files, index and quota. | Demonstrate durability on each supported filesystem. Interrupt the provider on both sides of commit and verify visibility, accounting and receipt replay after restart. |

Three implementation details inform these decisions. Go's AEAD interface requires
nonce uniqueness for each key. Protobuf serialization does not guarantee canonical
bytes for a signature. SQLite's transaction guarantee applies to its database
changes. HiveScope also writes fragment files outside that database.
[Go AEAD](https://pkg.go.dev/crypto/cipher#AEAD) ·
[Protobuf serialization](https://protobuf.dev/programming-guides/serialization-not-canonical/) ·
[SQLite atomic commit](https://sqlite.org/atomiccommit.html)

## Evidence to collect from the prototype

These scenarios are planned. Record the build, operating system, inputs and
observed outcome when the first implementation can run them.

| Scenario | Observable success |
| --- | --- |
| Upload an empty file and a file of 4 MiB + 1 byte | Retrieval preserves length and exact bytes across the block boundary. |
| Disconnect after the provider commits but before the client sees the reply | Repeating the operation completes with one stored object and one quota charge. |
| Interrupt between writing the file and updating the index | Restart settles or releases the reservation. Every acknowledged object is readable after restart. |
| Two concurrent uploads compete for capacity sufficient for one | At most one completes. Physical usage including temporary files stays inside the agreed limits. |
| Supply a corrupted block, an unapproved reader or a wrong recovery password | Each case reports failure and preserves any existing destination file. |
| Valid grant. With the original device offline and an empty replacement profile, use only the package, password and reachable provider. | The new identity receives the catalog and object after proving possession. The recovered file matches the independent fixture byte for byte. |
| Revoked grant. Starting from the same fixture, have the provider acknowledge the owner's revocation, restart it, and leave the original device offline. Retry from a clean profile with the old package and password. | Even a correct signature on a fresh challenge with the old key yields zero catalog or object bytes. Check both locators independently. The existing destination file stays intact. |

## What can wait

Milestone 2 adds automatic placement using replaceable directories and tests
concurrent reservations and retries between previously unknown peers. Milestone 3
tests 2+1 repair, audit costs, offline conflicts and filename portability. The
desktop alpha will measure home connectivity, relay bandwidth, disk use and battery
impact on Windows, Linux and macOS. General-purpose private computing and persistent
hosting have separate acceptance criteria in the [roadmap](roadmap.md).

**Next design step.** Define the messages exchanged during this upload. Specify
how the client reserves space, sends each item, finalizes the pair and asks for
status, with bounded frames, timeouts and an actionable response for each error.
