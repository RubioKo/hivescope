<!-- docs-pair: architecture; revision: 2 -->

[English](architecture.md) | [Español](../es/architecture.md) | [Index](README.md)

# Architecture and interfaces

All components in this document are proposed. The first implementation will use
a single Go executable with daemon and CLI modes. No Go module or executable is
included in this documentation baseline.

## Components

| Component | Responsibility |
| --- | --- |
| Identity and authorization | Per-device identities, peer approval and folder permissions. |
| P2P transport | Authenticated connections, address updates, NAT traversal and bounded relaying. |
| Synchronization | File watching, scans, signed versions, conflict handling and local materialization. |
| Data protection | Authenticated encryption, encrypted catalogs and key recovery. |
| Storage | Immutable fragments, integrity checks, reservations and physical quotas. |
| Maintenance | Reachability checks, audits, repair and agreement renewal. |
| Resource control | Disk accounting, transfer rates, schedules, concurrency and pause. |

A device may perform several roles. Folder readers, encrypted-storage providers
and public relays have separate permissions. A node's reachability or reputation
does not authorize it to decrypt content.

## Data flow

1. An authorized device observes a stable local change and creates an immutable version.
2. It splits the file into blocks, encrypts them and produces redundant fragments.
3. Approved providers reserve capacity, verify received bytes and issue receipts.
4. The device publishes signed, encrypted version metadata to its approved peers.
5. Another authorized device reconciles versions, retrieves enough verified
   fragments, decrypts and replaces its local file atomically.

Transfers are streamed with bounded buffers. Local edits remain pending until
their metadata and required fragments receive the relevant confirmations.

## Network and discovery

Use go-libp2p with QUIC and TCP, authenticated secure channels, AutoNAT, DCUtR and
Circuit Relay v2. The initial network uses exchanged peer contacts and LAN discovery.
Only approved peers can negotiate storage or access folder operations.
Signed, expiring address records allow known peers to exchange address updates.

Public relay contacts are configurable and replaceable. Relays forward encrypted
traffic, enforce transfer budgets and hold no account or credit authority.
Direct connections are preferred; a relay may still be needed under restrictive
NAT. Failure of both paths leaves work pending with an explicit connectivity state.
There is no public DHT or global file index in the initial implementation.

## Persistence and interfaces

SQLite stores each node's local index, agreements and resumable work queue.
Fragments live in an independently configured storage directory. This database
is not a ledger shared by the network. Encrypted catalogs and version records are
replicated so recovery does not depend on one device's SQLite database.

The CLI will cover identity, peer approval, folder authorization, agreements,
status, pause and recovery. Future GUI clients will use the same daemon operations.
P2P messages will use versioned protocols, Protobuf and bounded streaming frames.
Unknown protocol versions and oversized messages must fail before resource allocation.

The initial design contracts are:

| Contract | Required meaning |
| --- | --- |
| `DeviceIdentity` | Peer identifier and public verification key; never a recovery secret. |
| `FolderAuthorization` | Owner-signed device membership, role and authorization epoch. |
| `StorageAgreement` | Parties, physical quota, validity, transfer limits and permitted operations. |
| `FileVersion` | Authorized writer, causal version, encrypted file metadata and block references. |
| `ShardReceipt` | Provider-signed acknowledgment of fragment ID, digest, size and agreement. |

These are semantic boundaries, not stable APIs or finalized wire schemas. Each
implementation milestone must specify its exact encoding, limits, replay handling
and compatibility rules before adding protocol code.

## Ownership and recovery

Each folder has a separate owner-controlled authority and encryption keys.
Each device has its own identity; copying one identity to several devices is
not the enrollment process. Enrollment verifies a fingerprint and transfers
folder secrets over an authenticated channel. Revoking a device starts a new
authorization epoch and rotates keys for future versions; it cannot retract
plaintext or keys already learned by that device.

In milestone 1, the encrypted export lets a fresh device identity recover a
folder through a scoped read grant. The owner keeps the administrative signing
key separately. The [first-read decision](readiness.md) describes what the
provider checks and how revocation affects an old export. Persistent device
enrollment requires separate authorization. There is no central password reset.
Local secret protection and the export format still require specification and
review in the secure-core milestone.

## Later AI service

Reuse identity, transport, resource limits and agreement concepts while isolating
the compute runtime from storage. Do not permit arbitrary peer-supplied programs.
Start with approved model runtimes and complete models on capable providers.
Compute quotas, output integrity and query privacy need their own design review.

## References

- [libp2p hole punching and relay behavior](https://github.com/libp2p/specs/blob/master/connections/hole-punching.md)
- [Tahoe-LAFS architecture](https://tahoe-lafs.readthedocs.io/en/stable/architecture.html)
- [Petals collaborative inference](https://github.com/bigscience-workshop/petals)
