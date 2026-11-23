<!-- docs-pair: provider-persistence; revision: 6 -->

[English](provider-persistence.md) | [Español](../es/provider-persistence.md) | [Index](README.md)

# What survives a provider restart?

In the first LAN trial, a receipt commits the provider to retaining two files
together. If Luis loses power after sending that receipt, Ana must still find
the same upload when he restarts. If his disk later loses either file, Ana
must see that the copy is unavailable. This design specifies what to store,
when to acknowledge it and how to test both outcomes.
[Upload confirmation and reference receipt](upload-confirmation.md)

## Grant storage in the Linux prototype

The operator provisions a private store with an initial signed grant. On a
change, the provider commits the new signed record to SQLite, replaces a
separate checkpoint containing its digest, and synchronizes the directory.
After restart the provider checks these records against the enrolled store
identity and agreement before authorizing a read.

For a revocation, the provider first saves a private intent containing the
previous checkpoint digest and the exact revocation signed by the folder
authority. It makes that intent durable before changing SQLite. From this
point a restart keeps the old credential from reading either item. If the
provider stops before it has durable evidence of the request, the previous
grant can still apply. Bytes already handed to the transport can still arrive.

The provider recognizes three combinations when a valid intent is present.

| Signed record in SQLite | Checkpoint digest | On restart |
| --- | --- | --- |
| Previous grant | Previous grant | Keep reads disabled while waiting for the same signed revocation. |
| Revocation | Previous grant | Keep reads disabled. An exact retry can finish the checkpoint. |
| Revocation | Revocation | Keep reads disabled. An exact retry can finish removing the intent. |

For the retry the provider verifies the authority signature, scope and exact
bytes against the intent. It finishes the SQLite and checkpoint writes as
needed, removes the intent, synchronizes the directory and then returns
`GRANT_ACK` with the digest of the revocation. A lost reply after this point
can be confirmed by resending the same signed bytes. Missing, inconsistent
or unrecognized evidence keeps the store offline for reconciliation.
An older database paired with a newer checkpoint keeps reads disabled. A
whole-volume snapshot can roll back all of the provider's local evidence.

The Linux store publishes the intent from an unnamed file. This requires a
filesystem supporting `O_TMPFILE` and access to `/proc/self/fd`.
[Linux unnamed temporary files](https://man7.org/linux/man-pages/man2/open.2.html)

This loopback trial assumes a trusted host clock and uses preloaded sealed items.
The subprocess restart exercises grant recovery. Upload receipts, volume
power-loss behavior and clock reconciliation remain separate trials.
[Restart test and build requirements](readiness.md)

## The trial volume

Use local Linux ext4 with ordered data, write barriers and a storage device
that honors flushes as the **first provider test configuration**. Place the
SQLite index, its rollback journal and both sealed files on this volume. Use
SQLite journal mode `DELETE` and synchronous mode `EXTRA`. Verify the settings
on every connection before enabling the provider. SQLite then synchronizes the
journal's directory after removing the journal at commit. External files also
require successful synchronization of their contents and directory entries.
[SQLite synchronization modes](https://www.sqlite.org/pragma.html#pragma_synchronous) ·
[Linux file and directory synchronization](https://man7.org/linux/man-pages/man2/fsync.2.html) ·
[ext4 data modes and barriers](https://www.kernel.org/doc/html/latest/admin-guide/ext4.html)

Record the filesystem, mount options, SQLite version and storage stack with
the future trial results. A process restart tests application recovery. An
abrupt shutdown of a disposable test volume with a documented power-loss
model tests persistence. Repeat both on each OS and filesystem before enabling
that provider configuration. Windows and macOS clients can be evaluated
independently of their ability to issue durable provider receipts.

## Provider-owned records

| Record | What the provider keeps |
| --- | --- |
| Store | A private root bound to the provisioned volume and store ID, an index schema version, fixed capacity limits and a single running provider instance. Persist newly created directory entries up to the existing parent at initialization. |
| Agreement | Provider, uploader and folder authority public keys, opaque agreement and folder IDs, physical quota, expiry and grace, write authority, highest accepted recovery generation and signed grant state. Keep final closure durable. |
| Upload | One unique row per provider, authenticated uploader, agreement and request ID. Keep the exact manifest and fingerprint, reservation deadline, physical capacity commitment, per-item stage, terminal state, health and original receipt bytes if committed. |
| Item | Two distinct slots per upload, one object and one catalog. Keep role, opaque locator, expected sealed length and digest, provider-generated filename and ownership. Enforce locator exclusivity within the agreement across reservations and committed items. |

Keep the private index and its files accessible only to the service account.
Resolve generated filenames relative to the already-open private directory.
Create each file exclusively and verify that entries are regular, singly
linked files on the intended volume before reuse. A peer-supplied locator
is an index lookup. File names from Ana's catalog stay encrypted. State or
files from a different volume cannot initialize an existing store.
[Local identity and grant custody](identity-custody.md)

## Reserving the last space

The first admission trial starts with an empty, dedicated Linux provider store,
one locally enrolled agreement and one uploader. `RESERVE` and `STATUS` use
the authenticated device identity. One SQLite transaction claims both item
slots, records the exact manifest and accounts for space before a pending
reply can leave the provider. On reconnect the client can query the same ID
and fingerprint or resend the original manifest. The deadline and charge
belong to the original admission. A reservation alone publishes no data.
The trial uses a trusted lab clock and retains expired IDs. Each staged
item uses the original reservation. `FINALIZE` commits the signed receipt on the
manifest row that owns both files. On retry the provider checks both files and
returns the stored receipt alongside the current availability.

This store retains up to 64 request IDs, subject to a lower local limit.
It rounds each item separately and budgets extra blocks for file extents
and directory entries. On a 4 KiB allocation unit the five-byte example
therefore holds 16 KiB for the pair. SQLite and its rollback journal have
a separate allowance. The admission records a commitment to future writes.
Actual file allocation starts when the uploader sends an item.

1. Set the host cap and free-space margin in physical bytes. Bound live
   requests, retained terminal IDs and concurrent writers. Reserve host-wide
   space for the maximum permitted index, rollback journal, directory growth
   and cleanup transactions. Cap the database page count and allow journal
   growth separately. Assign each agreement an allowance for its upload and
   terminal metadata within that host budget. If these bounds cannot fit,
   decline the reservation before accepting file bytes.
   [SQLite database limits](https://www.sqlite.org/limits.html) · [Storage agreements](storage.md)
2. Estimate each sealed item's peak allocation separately, rounding up to the
   volume's allocation unit and allowing for extent metadata and preallocation.
   A partial item occupies its reserved slot. Restart it in that same slot
   after durable discard. Never budget for two files and silently create a
   third copy. Keep already allocated bytes charged through durable cleanup.
3. Under a single host admission lock, reconcile actual allocated bytes and the
   **remaining unallocated part** of each commitment, including metadata. Their
   sum plus the new commitment must fit the host cap. Repeat for the agreement
   and requester limits. The volume's currently available bytes for this account
   must cover all unallocated commitments including the new one, plus the free
   margin. Check available inodes for prospective files as well. Persist the
   reservation before releasing the lock. [Linux filesystem capacity](https://man7.org/linux/man-pages/man3/statvfs.3.html)
4. Serialize allocation accounting across writers. Recheck available space,
   the margin and each budget before granting another bounded write or metadata
   allocation. Account for outstanding writes. On shortage or a write/flush
   error, stop the affected write. Query the recovered state before retrying.
   Space consumed independently by other applications can exhaust a volume
   after admission. It cannot turn an incomplete upload into a confirmed one.

For the five-byte fixture, a 4 KiB allocation unit gives two rounded payload
allowances of 4 KiB each for the 21-byte object and the 250-byte catalog.
Additional filesystem and index overhead still needs its budget. Charging
271 physical bytes would admit more work than this host can store.

Pausing offers blocks new reservations. If Luis **lowers a cap** below an
existing commitment, keep all committed files and report the overage. Hold
new writes and finalizations that exceed the revised budget, including those
under a pending reservation. The uploader sees a capacity shortage and can
wait or cancel. Keep the original deadline and all occupied bytes charged.
Raising the cap can let the same eligible upload resume. Reducing it cannot
evict a confirmed copy during its retention agreement.

## Which clock decides expiry?

Use the provider's trusted UTC time for persisted deadlines. The write window
includes the start instant and excludes the agreement expiry. A reservation
can finalize strictly before its own deadline and the agreement expiry. Read
grace starts exactly at agreement expiry and ends exactly seven days later,
with that last instant excluded. During grace, stored data remains read-only
and signed grant changes can still restrict who reads it. Persist observed expiry or final closure
before reclaiming data. A later clock change cannot reopen a terminal
reservation or closed agreement. The fixed-term trial uses no renewal.
[LAN deadlines and status](lan-protocol.md) · [Agreement policy](storage.md)

Keep the greatest trusted UTC value used for lifecycle decisions in the index,
in the same transaction as any resulting state change. Validate UTC at startup
and after sleep. A controlled lab clock or independent trusted time reference
suffices for the trial. During operation compare wall-clock movement with
monotonic elapsed time. If the clock falls behind the durable value or makes
an unexplained jump, suspend time-dependent reads, writes and cleanup until
the time is verified again. An authenticated request that depends on this
decision sees `UNAVAILABLE`. Keep reservations and occupied bytes accounted
for in the meantime. In particular, an implausible future date cannot trigger
garbage collection. Verify time again before destructive expiry cleanup.
Stream and challenge timeouts use elapsed time and do not survive restart.

## From reservation to publication

For `PUT`, each request has one writer for either item. The Linux provider
writes into an unnamed file on its private volume. It links that file to
the reserved slot only after verifying the declared length, SHA-256, the
final `END` and the orderly end of the request. It flushes the file, links
it, flushes the directory and records the staged bit before replying.
Closing an interrupted transfer drops its unnamed file. The other item's
slot and the original reservation remain available. After a restart, the
provider checks each linked slot against its manifest before reporting it.

| Boundary | Required order |
| --- | --- |
| Admit | Deduplicate the scoped request, allocate both slots and commit its capacity and ownership in SQLite. A conflicting manifest leaves the original intact. Only then create files in their final private locations. |
| Stage | Hold one writer per request. Verify size and digest of each finished file, synchronize the file, then synchronize its parent directory. Commit its staged marker. Read authorization consults committed membership only. |
| Publish | Serialize finalization with cancellation, expiry and authorization changes. Recheck both stages and write eligibility. In one SQLite transaction publish both locator memberships, replace reserved usage with stored usage and save the exact signed receipt. Reply after durable commit. An ambiguous commit requires reopening SQLite before reporting an outcome. |
| Cancel or expire | Durably mark the request terminal before reclaiming its files. Stop and close its writers, unlink only its owned files, synchronize the parent directory, then settle capacity in SQLite. Repeat unfinished cleanup after restart. After commit the storage agreement governs retention. |
| Close | After the agreement and read grace end, deny reads and writes before cleanup. Keep a closure record that prevents agreement ID reuse. Account for open file handles until the last reader is closed and cleanup completes. |

## Reopening the store

Before serving the affected state, acquire the instance lock, verify the
expected volume and store identity, open the existing index, let SQLite
recover its journal, check index integrity and reconcile ownership, capacity
and grants. Missing or unreadable index state on a provisioned root puts it
in maintenance. Recreating an empty index would lose acknowledged revocations
and permit double booking. An old whole-volume snapshot can also roll back
an apparently valid index. The trial assumes an honest provider and storage
stack and does not establish rollback resistance against a host restoring
such a snapshot. [SQLite recovery and storage assumptions](https://www.sqlite.org/atomiccommit.html)

Recheck expected lengths and SHA-256 digests, using bounded streaming and one
file at a time in this trial. For a pending item, adopt fully matching bytes
after completing both synchronizations and the staged commit. Discard an
incomplete or mismatched owned item, clear its staged marker and request that
item again while the reservation is valid. For a committed pair, either file
missing or damaged marks the copy unavailable. Retain the receipt, disable
reads of the pair and preserve its ownership and applicable charges. A status
retry cannot silently replace lost data or sign a second receipt.

During the trial, recheck the committed pair before returning a healthy status
or starting a read. Authenticate the request first, apply the host's I/O limits
and verify the same opened files that will be served. An I/O failure during
streaming aborts that read. Ana still verifies the complete catalog and object
before publishing recovered plaintext.

Track unexpected entries as a local maintenance condition. Keep them in place,
account for occupied space and suspend admissions until ownership and budget
are reconciled. A missing mount, unexpected link, failed integrity check or
unknown schema version requires local attention. Reconciliation must use a
bounded queue and honor the host's disk bandwidth limit. Restart discards all
outstanding read challenges and reloads the persisted grant generation before
issuing a new one. [LAN operations and limits](lan-protocol.md)

## Evidence to obtain from the first provider

| ID | Interrupt or change | Expected observation on reopening |
| --- | --- | --- |
| D1 | Stop after the file or directory sync, before its staged marker. Repeat with a truncated item. | Matching bytes become staged after verification. Only the affected incomplete item restarts at byte zero. Both locators stay private until publication. |
| D2 | Stop between the two staged markers and final publication, then separately drop the reply after commit. | The first run exposes zero published locators. In the second, repeated status returns the same receipt, one charge and two readable items for a valid grant. |
| D3 | Race two requests for the last capacity. Then exhaust space during the winner's flush. | One reservation at most. No confirmation for the incomplete write. A restart or cancelled cleanup still accounts for every owned allocation until released. |
| D4 | Damage a committed catalog, disconnect the trial volume or introduce an unowned file. | Damaged data reports unavailable and retains its receipt. A missing volume never looks like a fresh store. Unknown files stay untouched and occupy accounted space. |
| D5 | Restart immediately after acknowledging revocation. Independently interrupt removal of an expired pair before the cleanup accounting commit. | The old recovery key receives zero catalog and object bytes. Repeated cleanup frees only the owned pair and releases capacity once. |
| D6 | Lower the cap below a reservation while it is live. Separately restart with the clock behind its last recorded time, then set an unverified date past grace. | The lowered cap blocks finalization and keeps the reserved charge. With uncertain time the provider returns unavailable for time-dependent requests and retains files and charges. Restore trusted time at exactly expiry to check read-only grace, then at its end to check closure. |

All six rows describe future observations. Record statuses, byte counts,
quota before and after restart, and the digest of the synthetic receipt.
Keep private paths, keys, recovery packages and payloads out of the report.
