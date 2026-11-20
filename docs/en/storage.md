<!-- docs-pair: storage; revision: 3 -->

[English](storage.md) | [Español](../es/storage.md) | [Index](README.md)

# Encrypted storage and maintenance

Milestone 1 will store one complete encrypted object with one provider. Recovery
on a replacement device must reproduce the original bytes. A verified copy at
that stage depends on the continuing availability of that provider. The following
redundancy and repair policy applies from milestone 3. Its parameters need validation.

## File protection

- Split a stable file version into blocks of at most 4 MiB.
- Use the candidate Go implementation's standard AES-256-GCM library. Use a fresh
  key for each immutable file version and unique nonces for every block under
  that key. Bind version and block position as authenticated associated data.
- Apply Reed–Solomon to the ciphertext, including the information needed for
  authenticated decryption, using an existing library such as `klauspost/reedsolomon`.
- The first redundancy profile is two data fragments plus one parity fragment per block.
  Any two valid fragments reconstruct the encrypted block. Corrupt fragments
  count as unavailable; parity alone does not authenticate content.
- Place the three fragments on distinct providers selected under the node's
  resource policy. For controlled trials, confirm separate operators and
  households. In an open network multiple peer identities can belong to one
  operator, so report the evidence available for independent placement.

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

Under the redundancy profile a version is `pending` until its placements and metadata are acknowledged. It is
`protected` with all three valid placements and metadata confirmations, `degraded`
when only two remain available, and `unavailable` when fewer than two can be read.
These are per-block requirements: the whole file needs every block and its catalog.
Unavailable does not necessarily mean permanently lost; providers may reconnect.

A receipt proves that a provider acknowledged an obligation, not that the bytes
will remain available forever. Capacity shortages leave edits pending and visible.

## Agreements and physical capacity

Anyone may request storage without offering hardware or earning credits.
Compatible nodes accept an agreement automatically under the host's local
profile. It records both parties, physical byte quota, permitted operations,
expiry and transfer limits. An expiring directory advertisement only helps find
the host. The host checks real space and commitments before making a reservation.

The first LAN trial provisions a fixed 30-day agreement locally and tests a
seven-day read grace period after expiry. Its wire protocol has no renewal
operation. The open-network design adds automatic negotiation and a daily
renewal attempt while reachable. An offline client cannot renew its lease.
New writes need a valid agreement. Existing data remains readable during grace;
providers may garbage-collect it afterwards. Loss of every owner's device beyond
that horizon can therefore cause data loss even with a recovery export.

Quota accounting includes live data, retained versions, metadata, incoming
reservations and temporary bytes. Charge a request once across retries. Reserve
atomically so competing requests cannot consume the same bytes. Keep a configurable
margin of free space on the offered volume. Pausing stops new reservations.
When a reduced quota leaves insufficient capacity, stop new writes. Keep
existing data under its retention agreement even if the host stops advertising.
Migration needs replacement providers in the later redundancy milestone.
[First provider's capacity and persistence rules](provider-persistence.md)

## Health checks and repair

- Check reachability every five minutes. Three consecutive failures mark a
  provider as suspect and trigger repair evaluation if redundancy is reduced.
- Start repair without a fixed 48-hour wait when only two valid fragments remain.
  A replacement must meet the local placement policy and confirm an agreement
  with free quota before it receives data.
- The owner can delegate bounded read/reconstruct/write authority over encrypted
  fragments to an explicitly delegated repair worker without disclosing decryption keys.
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
