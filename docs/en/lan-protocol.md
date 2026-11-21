<!-- docs-pair: lan-protocol; revision: 5 -->

[English](lan-protocol.md) | [Español](../es/lan-protocol.md) | [Index](README.md)

# The first storage exchange on a LAN

This is the v1 design for one encrypted file on one provider. The original
device uploads the object and its catalog under one reservation. A replacement
reads each item under a separate proof. The reservation, receipt and recovery
proof have the same meaning across reconnects. In the Go loopback trial, the
provider starts with the sealed catalog and object already provisioned. The client
verifies the catalog before requesting the object under a fresh proof. The Linux
provider with a SQLite grant store acknowledges signed grant updates and reloads
the latest grant after a process restart. Uploads remain a later step.
[Recovery trial](readiness.md)

## Connection and framing

The trial starts with an agreement provisioned locally on both sides and a
direct IPv4/TCP endpoint. Use libp2p TLS 1.3 with Ed25519 peer identities and
yamux. Compare the authenticated provider's raw identity key with the pin.
Negotiate the exact stream protocol `/hivescope/storage/1.0.0`. A failed pin
or unsupported version ends the attempt before sending application data.
One bidirectional stream carries one exchange. Separate streams may share
the authenticated connection. [libp2p TLS](https://github.com/libp2p/specs/blob/master/tls/tls.md) · [yamux](https://github.com/libp2p/specs/blob/master/yamux/README.md)

For a catalog read, the replacement dials the address in the unlocked package
using its own device identity. It gives libp2p the peer ID derived from the
pinned Ed25519 key and checks the authenticated remote key on that connection.
Only then does it open the storage stream and send a catalog `READ_BEGIN`.
Another machine at the same address can receive a TCP connection attempt.
Presenting a different key ends the handshake before the catalog locator is
sent. A matching provider replies on that stream with a 32-byte `CHALLENGE`.
The subsequent `PROOF` is what the provider must check to authorize a read.

Each frame starts with a four-byte unsigned big-endian length of the
following Protobuf message (2–65,542 bytes). Check it before allocation.
Field 1 is the required operation number (VARINT). Field 2 is its payload
(LEN), omitted when empty. Send each field once, in numerical order, using
the shortest varint encodings. This v1 profile rejects all other fields,
duplicates, wrong wire types, explicit empty payloads and trailing bytes.
The frame boundary is independent of TCP reads. Payloads are at most 4,096
bytes except for `DATA`. Split each item into consecutive 65,536-byte
payloads and one shorter final payload when necessary. Every `DATA` carries
at least one byte, including for the sealed empty-file fixture. Signatures and
fingerprints cover the raw records in the linked formats. [Protobuf encoding](https://protobuf.dev/programming-guides/encoding/)

The tables give hexadecimal operation numbers. Inside each payload,
concatenate the listed raw fields in order. Integers are unsigned big-endian.
`U` is folder ID (16), agreement ID (16), request ID (16) and manifest
fingerprint (32). The authenticated uploader and provider further scope
this reference. `K` is one byte, `01` for catalog or `02` for object.
The sealed object is 16–67,109,120 bytes and the signed catalog is 243–497
bytes. The client also enforces the exact lengths derived from its own
manifest or verified catalog. [Sealed item formats](file-catalog.md)

## Upload and commit

| First request | Payload | Provider response and effect |
| --- | --- | --- |
| `01 RESERVE` | Complete 276-byte manifest | Verify its domain, identities, scope, sizes and distinct locators. Return `STATE` after durable admission or the existing state for an identical retry. A changed manifest under the same scoped ID yields `CONFLICT`. |
| `02 STATUS` | `U` | Return `STATE` without allocating space. Check uploader authorization before disclosing any state. |
| `03 PUT` | `U, K` | For a missing item under a live reservation, take its request's writer lock and send `20 READY` (empty). The uploader sends consecutive `21 DATA` frames from byte zero and one `22 END` (empty). Return `STATE` with the staged bit after length, digest and durability checks. An already staged item or terminal request returns `STATE` immediately. |
| `04 FINALIZE` | `U` | Finalize under the request lock when both staged items, authorization and deadlines are valid. A pending request missing an item returns its current `STATE`. A committed request returns its original receipt. |
| `05 CANCEL` | `U` | Serialize with the writer and commit. Return the resulting `STATE`. If commit won the race, the stored pair and receipt stand. |

For `02`–`05`, a known ID with the wrong fingerprint yields `CONFLICT` for
the authorized uploader. An unknown ID returns unknown state. Retrying a
submission reuses the original ciphertext, manifest and ID. A disconnection
during `PUT` discards partial progress on that item. After reconnecting,
ask for `STATUS` before resending. A missing reply to `FINALIZE` leaves
confirmation pending until the client retrieves and verifies the original
receipt. [Upload manifest and receipt](upload-confirmation.md) · [Provider persistence](provider-persistence.md)

Admit at most 256 retained request IDs per agreement and 4,096 per provider
in this trial. Count pending, committed, unavailable, cancelled and expired
requests until agreement closure after grace. Apply any lower host limits.
At the ceiling an extra ID gets `CAPACITY`. An existing ID can still obtain
its state. Never discard an unexpired receipt or terminal ID to make room.

`23 STATE` starts with one state byte. Its remaining payload is exact.

| State byte | Remaining fields | Client-visible result |
| --- | --- | --- |
| `00` | Empty | Unknown request. |
| `01` | Staged mask (1), absolute reservation deadline (8, UTC Unix seconds) | Pending. Mask bit 0 denotes the catalog, bit 1 the object. All other bits are zero. Mask 3 permits finalization. |
| `02` | Historical signed receipt (132) | Committed. Verify and durably save the receipt before marking the copy confirmed. |
| `03`, `04` | Empty | Expired or cancelled, respectively. A new submission needs a new ID. |
| `05` | Historical signed receipt (132) | Previously committed, currently unavailable. Show that the copy is unavailable. |

## Recovery and administrative update

| First request | Payload | Exchange on the same stream |
| --- | --- | --- |
| `07 READ_BEGIN` | `K`, folder ID (16), agreement ID (16), generation (8), locator (32) | For a well-formed request within resource limits, return `25 CHALLENGE` with 32 fresh random bytes. This step discloses no membership. The replacement sends `08 PROOF` with only the 64-byte signature over the specified read. |
| `06 GRANT_UPDATE` | Complete signed administration record (198) | An authenticated peer may relay the record. Check its owner's signature, scope and generation. Return `24 GRANT_ACK` with the SHA-256 (32) of the exact record after durable acceptance and invalidation of the old authorization. An identical current record returns the same acknowledgment. |

Each stream has at most one challenge. Bind it to that stream, connection,
both peer identities and every requested field. Consume it atomically on the
first proof, including an invalid one. After 60 seconds on the provider's
monotonic clock, discard it. On disconnect or restart discard all affected
challenges. Serialize read admission with grant changes. A stale grant,
unknown membership, missing authorization state, bad signature or closed read
window returns `DENIED` and zero item bytes. An authorized read of a known
damaged item returns `UNAVAILABLE`. [Signed read and fixtures](recovery-format.md) · [Identity and signed grants](identity-custody.md)

On a superseding update, stop queuing data for affected active reads and
invalidate their outstanding challenges before acknowledging the update.
Bytes already handed to the transport can still arrive afterward.

On an authorized read the provider sends `26 READ_SIZE` with the sealed
length (8), consecutive `DATA` frames totalling exactly that length, then
`END` and an orderly end of stream. A truncated stream, extra frame, wrong
length or failed integrity check aborts the restore. Verify the signed
catalog before issuing the separate object read. Publish the local file only
after all format and integrity checks pass.

## Limits, failures and retries

The requester half-closes its write side after its final request frame
(`END` for a streaming upload, `PROOF` for a read). It waits for `READY`
or `CHALLENGE` before sending the next part. An early `STATE` for a staged
or committed `PUT` is final. The uploader sends no data, half-closes and
reads the provider's end of stream. If the requester half-closes before a
required `END` or `PROOF`, return `INVALID` when possible, discard any
partial item or challenge and leave any reservation available until its
deadline. A premature `FINALIZE` returns the pending mask and deadline,
with both locators still private. The provider checks the request's orderly
end before changing the recorded request or grant state or sending read bytes. It closes its
write side after the final reply. An incomplete exchange never proves success.

Allow up to 16 active application streams per provider, two per peer across
connections and one writer per request. At most 15 slots may carry
`PUT` or `READ_BEGIN`. When only the reserved slot remains, grant a new
stream two seconds to present its complete first frame (at most 281 bytes
after its length prefix). Accept only `STATUS`, `CANCEL` or `GRANT_UPDATE`
there. Return `BUSY` for other recognized operations and reset unclassified
or oversized frames. The slot limits competition from data transfers.
Control requests still share it and remain subject to the per-peer limit.
Apply host transfer limits in both directions and read at most one pending data frame
per stream ahead of disk or network progress. Stop a negotiation after 10
seconds, a stream after 30 seconds idle or 5 minutes total, whichever deadline
arrives first. The wait for a recovery proof uses the 60-second challenge
limit instead of the idle limit. Measure elapsed time monotonically. The
provider persists a reservation deadline at most 15 minutes after admission,
bounded by the agreement expiry. Retries never extend it. These are trial
ceilings and deadlines, pending measurement. The provider uses trusted UTC
for absolute lifetimes. Uncertain clock or restart reconciliation returns
`UNAVAILABLE` for decisions that depend on time. [Expiry and clock handling](provider-persistence.md)

`7f ERROR` has exactly one code byte. It terminates the stream. Reset an
unparseable or oversized frame as soon as the violation is detected. Use
only these codes on the wire. Local diagnostics can add context. An authentic
administration record with an older generation or divergent bytes at the
current generation yields `CONFLICT`. An invalid authority yields `DENIED`.

| Code | Meaning and action |
| --- | --- |
| `01 INVALID` | Invalid fields, ordering, length or digest. Correct the request or compare the sealed source with the saved manifest. |
| `02 DENIED` | Authority, proof or read window failed. Stop and check the local grant, pin or agreement. Never disclose locator existence to an unauthorized reader. |
| `03 CONFLICT` | Incompatible reuse of an ID, locator or grant generation. Preserve the accepted state. Consult the saved manifest or authority record. |
| `04 BUSY` | A lock, rate or stream limit is active. Retry after backoff. |
| `05 CAPACITY`, `06 PAUSED` | A capacity limit, including a reduced host quota, blocks additional allocations and prevents partial publication. A pause blocks new reservations and preserves existing deadlines. Wait for changed conditions or cancel locally. |
| `07 UNAVAILABLE` | Reconciliation, storage failure or damaged data prevents service. Query status after recovery. Keep the local source and any previous destination. |

After an ambiguous timeout, retry on a fresh stream, starting with `STATUS`
for an upload. Wait 1, 2 and 4 seconds respectively plus 0–1 second of random
jitter before the three automatic retries, then show pending with manual retry
or cancellation. A local wait or timeout never confirms cancellation at the
provider. Stale recovery proofs always require a fresh challenge.

## Reading the five-byte example on the wire

For `hive.txt`, a single `DATA` carrying its 21 sealed object bytes begins
with hex `00 00 00 19 08 21 12 15`. The empty `END` frame is
`00 00 00 02 08 22`. After that object is staged, `STATE` is pending with
mask `02`. Staging the 250-byte catalog changes the mask to `03`. Dropping
the final commit reply and asking again must return state `02` with the
same 132-byte receipt. With a valid generation 7 package, the intended read
lengths are 250 then 21. After acknowledged revocation at generation 8,
independent attempts for both known locators must yield `02 DENIED` with
zero item bytes. These are expected interoperability results for the future
prototype. [Reference bytes](recovery-format.md) · [Receipt fingerprint](upload-confirmation.md)
