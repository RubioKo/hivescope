<!-- docs-pair: readiness; revision: 1 -->

[English](readiness.md) | [Español](../es/readiness.md) | [Index](README.md)

# Is HiveScope ready for its first prototype?

The [architecture](architecture.md) defines the node, roles, trust boundaries,
data flow and storage model. The [roadmap](roadmap.md) sets the milestones.
Contributors can start specifying the secure core now. Implementing each part
requires settling the corresponding contract below. The current CI checks
documentation. Reliability, interoperability and recovery require running software.

## The first manual round trip

**Proposed scope for milestone 1.** Use one owner's device, one approved storage
provider and a direct LAN connection. A third, clean device profile represents
the owner after losing the original device. A single provider keeps a complete
encrypted object and a minimal authenticated catalog that points to it. Losing
that provider makes this demonstration unavailable. The three-provider placement
policy and the `protected` state belong to milestone 2.

1. The owner verifies the provider's fingerprint and both accept an agreement.
2. The owner encrypts a synthetic file and uploads it and the catalog. The
   provider reserves capacity, persists the bytes and acknowledges the operation.
3. The owner exports a password-protected recovery package. Switch off the original
   device and start with an empty local database and a fresh device identity.
4. Using only the package, its password and the reachable provider, recover the
   file. Compare every byte with an independent test fixture. Record which
   credential authorized retrieval and which local information was required.

Restoring the folder keys solves decryption. The provider must also recognize
the replacement device's authority to read under the agreement. This is the first
design question to resolve because its answer affects enrollment, the agreement,
the catalog lookup and the export format.

## Four decisions to settle

Each row is open. Contributors can propose the listed artifact in English and
Spanish. The maintainer records the chosen behavior, rationale, accepted example
and failure example in a reviewed PR before that component is implemented.

| Order | Design question | Evidence needed to start that component |
| --- | --- | --- |
| 1. Identity and recovery | Who holds the folder authority? What can each device or provider do? How does a recovered device prove access, discover a catalog and handle a stale export or revoked credentials? | A role/operation table and a recovery walkthrough naming the signer, verifier, required secrets, local unlock method on each target OS and limits of freshness. Include a denied operation. |
| 2. File and catalog format | How are version keys generated, wrapped and recovered? Which bytes bind a block to its version, position and length? What happens to the nonce and ciphertext after a restart? | A versioned byte layout for the encrypted object, catalog and recovery package, plus synthetic input/output vectors for an empty file and a multi-block file. Include a mismatched block and a wrong password. |
| 3. Transfer protocol | How do peers reserve, send, resume, acknowledge and retrieve an object? What precisely gets signed? How does a peer distinguish a retry from a new request? | A versioned message table with required fields, signed bytes, frame sizes, concurrency and time limits, error responses, retry rules and agreement validity during transfer. Include an acknowledged upload whose reply is lost. |
| 4. Disk and restart behavior | At which point may a provider acknowledge durable storage? How do two reservations share the last free bytes? How does restart reconcile the fragment directory with SQLite? | A state table covering reservation, temporary file, verified bytes, durable index and acknowledgment. For each interruption, document what survives, what is charged and how orphaned bytes or reservations are reclaimed. |

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
| Restore on a clean device with the original offline | The recovered file matches the fixture. The operator can identify the agreement, catalog and recovery material used. |

## What can wait

Milestone 2 will determine the audit cost, repair thresholds under real churn and
behavior when providers fail under the 2 + 1 profile. Milestone 3 will test
offline conflicts and filename portability. The closed alpha will measure home
network connectivity, relay bandwidth, disk use and battery impact across Windows,
Linux and macOS. Keeping the direct LAN experiment small leaves those platform
targets in place. The later AI service has its own compute and privacy design.

**Next contribution.** Trace a replacement device's first read in decision 1,
including the provider's reason for accepting or denying it. Once agreed, use
that example as the input for the file format and transfer specifications.
