<!-- docs-pair: upload-confirmation; revision: 2 -->

[English](upload-confirmation.md) | [Español](../es/upload-confirmation.md) | [Index](README.md)

# When can Ana count on a stored copy?

**Proposed contract for one file on one LAN provider.** Ana sends two items:
an encrypted object and its signed, encrypted catalog. Even if the progress
bar reaches 100%, her upload stays **awaiting confirmation** until her device
verifies and saves the provider's receipt for both items. The receipt records
what the provider accepted at that time. A successful recovery on a clean
device is a separate milestone 1 test. [File and catalog example](file-catalog.md)

## One request for both items

Before sending anything, Ana's device durably records a fresh random 16-byte
request ID, the intended provider, the manifest below and where it can reread
the original sealed bytes. It reuses them after a timeout or restart. The
provider scopes the ID to its own identity, the authenticated uploading device
and the agreement. Only that uploader, authorized under the agreement, can
write or query its upload status. A recovery grant authorizes separate reads.

The manifest fingerprint is SHA-256 of these raw fields in order. Quoted
domains are ASCII, integers are unsigned big-endian and digests are 32 raw bytes.

| Order | Manifest fields |
| --- | --- |
| 1 | `HiveScope/upload/v1`, one zero byte, provider and uploader public keys (32 bytes each) |
| 2 | Folder ID, agreement ID and request ID (16 bytes each) |
| 3 | Object locator (32), sealed size (8), SHA-256 of the sealed object |
| 4 | Catalog locator (32), sealed size (8), SHA-256 of the complete signed catalog |

The provider compares both lengths with the [trial format limits](file-catalog.md)
and reserves both locators under this agreement. It rejects collisions with a
different live request or stored item. In one durable SQLite transaction it
records the manifest, an absolute reservation deadline no later than the agreement
expiry, ownership of the prospective files
and capacity commitment. Admission is serialized against host and agreement
quotas, requester limits, the remaining physical space and the host's free-space
margin. Include filesystem allocation, temporary files and index/journal
overhead. As staged bytes occupy space, they consume the existing reservation.
The provider enforces its limits again as it writes. [Storage quotas](storage.md)

## What survives an interruption

The two files have exclusive, provider-generated locations in its storage
directory. The provider can keep them there from first write to final commit.
Only the committed index makes them visible to authorized readers. One writer
at a time owns a request. Cancellation, cleanup and commit serialize with it.
If commit wins, a later cancellation returns the receipt and leaves the stored
pair under its retention agreement.

| Last durable state | What the provider does | What a retry learns |
| --- | --- | --- |
| No reservation | A status query makes no changes. Only a reservation request under a valid agreement may allocate space. | Unknown request. Ana can repeat her original reservation with the same ID and manifest. |
| Reserved, possibly receiving data | Stream within the declared size. For each item, check exact length and digest, flush file contents and persist its directory entry. Only then durably mark that item staged. | The reservation and its deadline survive. Return the list of complete staged items. Discard a partial item before resending that item from byte zero. Neither item is readable yet. |
| Both items staged | Under the same request lock, recheck the write authorization, agreement and reservation deadline. In one durable SQLite commit publish both locator mappings and agreement membership, convert the held quota to stored usage, and record the signed receipt and committed status. | Until that commit succeeds, Ana still sees pending. If both items remain staged and eligible, retry finalization without retransmitting them. |
| Committed | Send the receipt only after the database reports durable success. Keep the original receipt bytes. | The same request returns that exact receipt. Reconnects and duplicate finalizations create zero new files or quota charges. |
| Expired or cancelled before commit | Mark terminal, stop writes, unlink only files owned by the request and make the cleanup durable. Release the occupied space after cleanup. | Return the terminal status. A new upload requires a new request ID and fresh admission. |

On startup, recover SQLite and reconcile the recorded file locations before
serving affected reads, reporting status or admitting new work. If an item
survived without a staged marker, verify its exact bytes and finish the flushes
before marking it staged, or discard it and request retransmission. Recheck
staged markers against their files. A failed flush or uncertain database commit
produces no success response: reopen the durable state and follow the matching
row. Until a missing or damaged staged item is replaced and verified, clear its
staged marker and report it as needed. If a previously committed file is missing or corrupt, report **unavailable**
and retain its historical receipt. Missing or inconsistent index state requires
reconciliation before treating an ID as unknown or reusing the space.

SQLite guarantees atomicity within its own database. The two ciphertext files
need separate durability operations. On Linux, syncing a new file also requires
syncing its parent directory for the new name to survive a crash. Validate file,
directory and database durability on each filesystem and OS offered by the
prototype before advertising this guarantee. [SQLite atomic commit](https://www.sqlite.org/atomiccommit.html) · [SQLite durability settings](https://www.sqlite.org/pragma.html#pragma_synchronous) · [Linux fsync](https://man7.org/linux/man-pages/man2/fsync.2.html)

## What the receipt means on retry

Sign, using the provider's pinned Ed25519 identity, the concatenation of
`HiveScope/stored/v1`, one zero byte, the manifest fingerprint, the commit
timestamp (8) and the agreement expiry at commit (8), both in UTC Unix seconds.
Append the 64-byte pure Ed25519 signature. Save these exact bytes as part of
the commit. Ana verifies the signature, recomputes the fingerprint from her
saved manifest, checks the agreement expiry against the agreed terms and
persists the receipt before marking the copy confirmed when the authenticated
provider reports committed. If it instead reports unavailable, show **copy
unavailable** even with a valid historical receipt.

- Same scoped ID and same manifest: return the existing state before attempting
  new admission, even if the provider has paused new reservations. The same ID
  with a different manifest returns a conflict and leaves the original untouched.
- A pause stops new reservations. It preserves an existing reservation's deadline.
  An expired agreement cannot finalize a new write during its read grace period.
  An authorized uploader can still retrieve the historical receipt during grace.
- Keep receipts and terminal IDs through the agreement's lifetime, renewals
  and read grace. Bound the number and size of records at admission. After
  final closure, reject writes to that agreement permanently and never reuse its
  ID. A delayed retry cannot recreate the upload after collection.

## Walkthrough and prototype acceptance

For the [five-byte fixture](file-catalog.md), Ana sends **21 sealed object
bytes and 250 signed catalog bytes**. These 271 payload bytes occupy one
reservation. The provider accounts separately for actual disk overhead.

| Interrupt the trial here | Expected result after restart |
| --- | --- |
| The object is staged. Half the catalog arrived. | Report only the object as staged. Ana resends the 250-byte catalog under the original reservation. |
| Both files are durable. Stop just before the database transaction commits. | Neither locator is readable. While eligible, retry finalization and publish the pair once. |
| Commit succeeds. Drop the reply, restart the provider and retry twice. | Ana first sees awaiting confirmation. Both retries return the identical signed receipt. There is one committed upload, two locator mappings, 271 sealed bytes and one quota conversion. Recovery with the valid package returns the original five bytes. |
| Two clients compete for capacity sufficient for only one pair. | At most one reservation wins the last space across the host. |
| A concurrent retry uses the same scoped ID with a changed digest. | Return a conflict. The original manifest, files and quota commitment stay intact. |
| The reservation expires during upload, or disk fills before the last flush. | No receipt is issued. Expiry starts cleanup. After a write failure, report the copy as pending and retry the affected item under the same reservation only while it is valid. Account for occupied bytes until cleanup. |

### Reference receipt for the lost reply

Use provider key TEST 2 and uploader key TEST SHA(abc) from RFC 8032 §7.1.
Use the folder, agreement, locators, sizes and both digests from the
`hive.txt` fixture above. The request ID is sixteen bytes of hex `15`.
The commit time is `1700000000` and the agreed expiry is `1702592000`.
The 276-byte manifest has fingerprint
`5cef2907ccccffa568c547206a2a4324294367cb9d54695ba33671d48a149e03`.
The first 68 bytes of the receipt are signed. With the signature appended,
the 132-byte receipt has SHA-256
`ff68ed2f38d36637e790a353415e3d69c92cdf6eabcfa5639250ae48b2175e2d`.
[Public Ed25519 test fixtures](https://www.rfc-editor.org/rfc/rfc8032#section-7.1) · [Recovery fixture IDs](recovery-format.md)

The reference signature is shown in two consecutive halves.

```text
d391894a4b1772edefcca619d2ffad9cc9d67e223a3f0f8f643bfc1a4db85377
8fc23139c2702f0713d2d5d9635ea5222935924ecfac6d115465a0b228510405
```

The example signature was verified with the public provider key. Changing
only the object length from 21 to 22 changed the fingerprint and invalidated
the original signature for that manifest. The Linux upload trial exercises
interrupted staging with this same object and catalog. The pair is still
pending after both items arrive. Publication and a signed receipt over
the network remain acceptance criteria. [Interrupted upload trial](readiness.md)
