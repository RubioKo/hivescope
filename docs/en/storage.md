<!-- docs-pair: storage; revision: 1 -->

[English](storage.md) | [Español](../es/storage.md) | [Index](README.md)

# Encrypted storage and maintenance

This is the proposed alpha storage policy. Values are design defaults to validate,
not measurements or guarantees from a deployed network.

## File protection

- Split a stable file version into blocks of at most 4 MiB.
- Encrypt each block with AES-256-GCM using the Go standard library. Use a fresh
  key for each immutable file version and unique nonces for every block under
  that key. Bind version and block position as authenticated associated data.
- Apply Reed–Solomon to the ciphertext, including the information needed for
  authenticated decryption, using an existing library such as `klauspost/reedsolomon`.
- The initial profile is two data fragments plus one parity fragment per block.
  Any two valid fragments reconstruct the encrypted block. Corrupt fragments
  count as unavailable; parity alone does not authenticate content.
- Store the three fragments on three distinct approved providers. Separate
  operator and household failure domains through explicit operator information;
  three peer identities alone do not prove independence.

This consumes about 1.5 times the encrypted content, plus padding, metadata and
retained versions. The original plaintext copies on the owner's devices are
additional. A later 10 + 4 profile would consume about 1.4 times the ciphertext
and require fourteen independent placements to tolerate four provider losses.

## Catalogs, receipts and state

Providers receive opaque fragment IDs and integrity commitments, not plaintext
paths or decryption keys. Signed catalogs contain encoding information and
encrypted references needed to retrieve the data. Catalog versions and owner
authorization records are replicated on all three selected metadata providers.
Readers retain the newest verified state they have seen to detect older replays.

A version is `pending` until its placements and metadata are acknowledged. It is
`protected` with all three valid placements and metadata confirmations, `degraded`
when only two remain available, and `unavailable` when fewer than two can be read.
These are per-block requirements: the whole file needs every block and its catalog.
Unavailable does not necessarily mean permanently lost; providers may reconnect.

A receipt proves that a provider acknowledged an obligation, not that the bytes
will remain available forever. Capacity shortages leave edits pending and visible.

## Agreements and physical capacity

Each mutually accepted agreement specifies parties, physical byte quota, permitted
operations, expiry and upload/download limits. There is no automatic credit reward
for advertised free capacity. Reciprocity is negotiated, not globally enforced.

Default validity is 30 days. Attempt renewal daily while reachable. A seven-day
grace period follows expiry; an offline client does not silently renew its lease.
New writes need a valid agreement. Existing data remains readable during grace;
providers may garbage-collect it afterwards. Loss of every owner's device beyond
that horizon can therefore cause data loss even with a recovery export.

Quota accounting includes live data, retained versions, metadata, incoming
reservations and temporary bytes. Repeated requests must not double-charge.
Keep a configurable free-disk safety reserve. Reducing a quota stops new writes
and requests migration; it does not immediately erase valid obligations.

## Health checks and repair

- Check reachability every five minutes. Three consecutive failures mark a
  provider as suspect and trigger repair evaluation if redundancy is reduced.
- Start repair without a fixed 48-hour wait when only two valid fragments remain.
  A replacement must be approved and have an applicable agreement and free quota.
- The owner can delegate bounded read/reconstruct/write authority over encrypted
  fragments to an approved repair worker, without disclosing decryption keys.
  It does not grant folder editing, agreement signing or unrestricted deletion.
- Workers reconstruct ciphertext, validate signed digests and obtain replacement
  receipts. Receipt announcements are eventually reconciled by authorized devices.
- Use stable task IDs, capacity reservations, backoff and cancellation so duplicate
  repair attempts cannot consume unbounded space. Existing valid copies remain
  until replacement is confirmed and their obligations allow removal.
- With fewer than two valid fragments, wait and report unavailable; do not fabricate
  a repair. Without a replacement destination, report degraded and keep retrying.

Storage audits request unpredictable samples of actual bytes with Merkle
authentication paths against signed commitments. Cached leaf hashes alone are
not an acceptable response. Audit frequency and sample sizes require measurement
before the repair milestone freezes the protocol. These are probabilistic checks;
they do not prove independent physical storage or prevent collusion and outsourcing.

## Sources to evaluate during implementation

- [Authenticated encryption in Go](https://pkg.go.dev/crypto/cipher)
- [Reed–Solomon library](https://github.com/klauspost/reedsolomon)
- [Tahoe-LAFS architecture and repair](https://tahoe-lafs.readthedocs.io/en/stable/architecture.html)
