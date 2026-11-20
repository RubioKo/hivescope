<!-- docs-pair: security; revision: 2 -->

[English](security.md) | [Español](../es/security.md) | [Index](README.md)

# Security and threat model

HiveScope has no supported software release. This design has not received a
cryptographic audit. Never use an experimental build as the only copy of a file.

## Trust boundaries

| Party | Authorized access | Remaining risk |
| --- | --- | --- |
| Owner-authorized device | Folder plaintext, keys and permitted edits. | Compromise exposes its folders and may publish harmful changes. |
| Approved storage provider | Encrypted fragments and bounded maintenance metadata. | May drop data, lie, collude or observe traffic patterns. |
| Delegated repair worker | Limited ciphertext reconstruction and replacement. | Can waste allowed resources or fail to repair. |
| Relay | Connection and traffic metadata; encrypted payload transport. | Can observe peers, delay or deny traffic. |
| Maintainer | Repository review and release administration. | A compromised distribution channel can ship malicious software. |

Peer approval limits admission; it does not prove honesty. The initial alpha
does not solve permissionless Sybil resistance or globally enforce reciprocity.

## Required defenses

- Use existing cryptographic and transport libraries; do not invent primitives.
- Authenticate peers and independently authorize every operation and folder role.
- Keep folder keys separate from device identities and repair permissions.
- Encrypt file names, paths and catalogs; verify ciphertext commitments and AEAD
  before making data visible. A signed digest establishes integrity, not availability.
- Reject malformed, replayed, oversized and unauthorized requests with bounded
  parsing, queues, streams and per-peer concurrency.
- Treat paths as untrusted and keep writes inside the configured roots.
- Protect local secrets with an explicit unlock and encrypted-at-rest design;
  OS file permissions alone do not protect against an unlocked compromised host.
- Keep exported recovery material encrypted with a reviewed passphrase-based
  format. Require explicit user action to export or reveal secrets.
- Use epochs, signed state and remembered version heads to limit rollback.
  Recovery from a stale export without an honest up-to-date peer can still miss
  newer changes; signatures alone do not prove freshness.

## Limits of the privacy claim

The intended confidentiality property is that storage providers cannot read file
content without keys. This is not a claim that all metadata is hidden or that a
zero-knowledge proof system is implemented. IP addresses, sizes, timing, placement
relationships and repair activity remain observable to relevant peers.

Encryption cannot recover deleted data, undo a compromised owner's edits, retract
downloaded plaintext or guarantee service from volunteers. Simultaneous household
outages, expired agreements and loss of keys can defeat recovery. Distributed
inference has different privacy properties and must not inherit storage claims.

## Proposed decision on transcription

Treat a person's recording as private by default. Transcribe it only on the
owner's device. Audio, excerpts, derived features, instructions and the
transcript stay there during processing. Logs, crash reports and telemetry
must exclude that content, private file paths and keys. If the model is
unavailable, memory runs out or the task fails, report the failure locally.
Retries follow the same rule. A separately requested storage backup may send
encrypted fragments under a storage agreement.

For volunteer computing, the proposed input is an item already published in
an approved public catalog. Remote requests select a public item by its catalog
identifier and an approved operation. The operator can read the public input
and its result. Approving a host or checking a sharing box cannot make a private
file eligible. Remote processing of private material remains a research question.
Someone who compromises the owner's device could still read local data.

**Future acceptance test.** Give a synthetic private recording a distinctive
spoken phrase and file name. With the local model installed, transcribe it
offline. Then offer a remote worker, exhaust local resources and retry after
reconnecting. Inspect outgoing requests at the sender and the worker's queue.
The first transcript must stay local and neither attempt may request remote work.
Check logs, crash reports and telemetry for the spoken phrase and the private
file path. Repeat with an unclassified recording.

## Reporting a concern

Use [GitHub private vulnerability reporting](https://github.com/RubioKo/hivescope/security/advisories/new)
for sensitive findings. Include the affected document or future version, a
synthetic reproducer, expected behavior and impact. Do not include real files,
credentials, recovery exports or personal logs. English and Spanish are accepted.

Use public [Issues](https://github.com/RubioKo/hivescope/issues) for non-sensitive
design questions. There is no guaranteed response time or bounty program.
Maintainers will assess impact, coordinate a fix where applicable and document
verified outcomes without claiming an audit that did not occur.
