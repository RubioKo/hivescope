<!-- docs-pair: security; revision: 1 -->

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

## Reporting a concern

Use [GitHub private vulnerability reporting](https://github.com/RubioKo/hivescope/security/advisories/new)
for sensitive findings. Include the affected document or future version, a
synthetic reproducer, expected behavior and impact. Do not include real files,
credentials, recovery exports or personal logs. English and Spanish are accepted.

Use public [Issues](https://github.com/RubioKo/hivescope/issues) for non-sensitive
design questions. There is no guaranteed response time or bounty program.
Maintainers will assess impact, coordinate a fix where applicable and document
verified outcomes without claiming an audit that did not occur.
