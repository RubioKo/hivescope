<!-- docs-pair: architecture; revision: 4 -->

[English](architecture.md) | [Español](../es/architecture.md) | [Index](README.md)

# Architecture and interfaces

The proposed desktop node has a background service and a CLI. Go is the first
language to evaluate. The repository currently provides design documents.

## Components

| Component | Responsibility |
| --- | --- |
| Identity and authorization | Authenticate each device, check the host's resource profile and check separate owner-issued folder grants. |
| Discovery and transport | Find candidates in replaceable community directories and establish authenticated peer connections. |
| Data protection and storage | Encrypt versions and catalogs, reserve physical bytes, persist immutable objects and verify receipts. |
| Maintenance and synchronization | Check reachability, renew agreements, repair ciphertext, reconcile signed versions and preserve conflicting edits. |
| Local resource control | Enforce the host's schedule, disk and transfer quotas, compute limits, concurrency and pause. |
| Private session | In a later milestone, verify the remote execution environment, release session keys and run the requester's software. |

A participant can request resources without contributing hardware. Each host
decides which services to offer in a persistent local profile. Machines can
negotiate a storage agreement without the people exchanging contact details.
Only the folder owner can grant devices the ability to read its files.

## Storage data flow

1. An authorized device takes a stable local file version and encrypts it in
   blocks. In milestone 1 a single LAN provider stores the complete encrypted
   object and an authenticated catalog. Automatic public discovery follows in milestone 2.
2. The host reserves capacity and, after durably storing and checking the bytes,
   acknowledges the write. Interrupted writes and lost replies must be recoverable
   using the same request ID. An unconfirmed upload remains pending.
3. Milestone 3 adds redundant fragments, signed encrypted metadata and replicated
   catalogs. Each provider accepts a limited storage agreement. Readers verify
   metadata and all required blocks before publishing the local file.
4. An authorized device or a replacement with a valid recovery grant retrieves
   and verifies the data, decrypts locally and atomically publishes the recovered
   version. Bounded streaming buffers keep memory use independent of file size.

## Network and discovery

The proposed go-libp2p transport uses QUIC and TCP, authenticated channels,
AutoNAT, DCUtR and Circuit Relay v2. A milestone 2 node publishes a signed,
expiring service advertisement in a replaceable directory. A requester can try
another directory when the first is unavailable. Advertisements contain service
and contact information. File names, folder catalogs and session contents stay
out of the directory. The local node authenticates the candidate directly.

The host checks the schedule, real free space on the offered volume, outstanding
commitments and local limits before reserving capacity. Two requests competing
for the final slot can obtain at most one reservation. Expiry frees an unused
reservation. A pause stops new reservations and preserves existing storage
retention obligations. When capacity is unavailable, the requester can wait or
cancel. [Resource offers](resource-offers.md) · [Storage policy](storage.md)

Use a direct path when available. Configurable relays forward encrypted traffic
under transfer budgets. Failure of both paths leaves work pending with a visible
connectivity reason. Measure directory renewals and synchronized reconnects,
relay saturation and repair traffic separately from desktop resource use.

## Persistence and interfaces

SQLite holds each node's index, reservations, agreements and resumable queue.
Objects occupy a separate configured directory. Confirming a write requires the
file and its index entry to survive restart. The redundancy milestone replicates
encrypted catalogs so a replacement can recover without the original database.

The CLI and later GUI share operations for resource profiles, requests, pause,
status, folder authorization and recovery. P2P protocols are versioned, with
Protobuf messages and bounded streaming frames. Reject unknown versions and
oversized requests before reserving resources. The following contracts describe
the behavior to specify and test before choosing a wire encoding.

| Contract | Required meaning |
| --- | --- |
| `DeviceIdentity` | Peer identifier and public verification key. Each recovery secret is held separately. |
| `FolderAuthorization` | Owner-signed device membership, role and authorization epoch. |
| Service advertisement | Signed and expiring capability and contact information for discovery. |
| Resource reservation | Unique request scoped to the requester and service, reserved quantity, expiry and status across retries. |
| `StorageAgreement` | Parties, physical quota, validity, transfer limits and permitted operations agreed by the two nodes. |
| `FileVersion` | Authorized writer, causal version, encrypted metadata and block references. |
| `ShardReceipt` | Provider-signed acknowledgment of fragment ID, digest, size and agreement. |

## Ownership and recovery

Each folder has separate owner-controlled authority and encryption keys. Each
device has its own identity. Enrollment verifies its fingerprint and transfers
the appropriate folder secrets over an authenticated channel. Revoking a device
starts a new authorization epoch and rotates keys for future versions. Content
already read by that device may remain in its possession.

The milestone 1 design under review gives a replacement device a narrow grant
to retrieve encrypted objects and the catalog. The encrypted recovery package
contains the recovery signing key, decryption material and provider locator
and fingerprint. The owner keeps the administrative signing key separately.
The provider verifies a fresh proof of possession and the current grant. After
acknowledging revocation it must still refuse the old grant after a restart.
The package format and local secret protection need review before implementation.
Enrolling the replacement as a permanent reader requires separate authorization.
[First-read grant, revocation and recovery tests](readiness.md)

## Private compute and persistent services

The requester chooses the software inside a private execution environment. The
host supplies a bounded amount of CPU, RAM, bandwidth and, later, GPU capacity.
Milestone 5 will evaluate a confidential VM. The requester must verify fresh
evidence bound to the guest's session key before releasing any secrets to it.
If the job requires a GPU, verification must also cover the accelerator and its
data path. An incompatible host leaves the request waiting with keys held by
the requester. Session access is independent of folder decryption authority.
[Security model and proposed tests](security.md)

A persistent service needs durable state and exclusive authority to write it.
An authorized standby must verify the last committed save and acquire exclusive
write authority before accepting traffic. A returning host must observe the
handoff before resuming writes. A private session on a new host requires fresh
verification there. Record downtime, reconnection latency and progress lost.

## Technology evaluation

Benchmark the Go node on a four-core PC with 8 GB of RAM. Measure idle and peak
resident memory, idle CPU, recovery throughput and time to honor pause. Test
direct and relayed transfers under recorded conditions. If Go misses an agreed
resource budget, profile the cause and compare Rust with the same hardware,
file, encryption, block size and concurrency. Verify confidential-session
hosting separately on compatible hardware.

OpenClaw's capability reconciliation, placement selector and handling of repeat
requests inform the design. Its TypeScript SDK could later expose HiveScope to
OpenClaw as an optional storage provider. REA's staged-file handling and Windows
process limits provide test cases for recovery and enforceable host quotas.
Record upstream revision, provenance, license and applicable third-party notices
for any code reused. [OpenClaw selector](https://github.com/openclaw/openclaw/blob/3b63f9ea1f552930f8d20215a1cb89a95a261fd0/src/gateway/worker-environments/device-placement-selector.ts#L74-L110) · [OpenClaw storage SDK](https://github.com/openclaw/openclaw/blob/3b63f9ea1f552930f8d20215a1cb89a95a261fd0/docs/plugins/sdk-overview/capabilities.md#L59-L72) · [REA staged files](https://github.com/morluto/rea/blob/8bb01b514ac766c0c3ecc8f362bd38186eaec9c2/src/filesystem/OwnedStagedOutput.ts) · [REA process controls](https://github.com/morluto/rea/blob/8bb01b514ac766c0c3ecc8f362bd38186eaec9c2/native/windows/README.md)

Further reading on transport and recovery. [libp2p hole punching](https://github.com/libp2p/specs/blob/master/connections/hole-punching.md) · [Tahoe-LAFS architecture](https://tahoe-lafs.readthedocs.io/en/stable/architecture.html)
