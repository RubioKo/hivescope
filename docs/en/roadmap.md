<!-- docs-pair: roadmap; revision: 2 -->

[English](roadmap.md) | [Español](../es/roadmap.md) | [Index](README.md)

# Roadmap and acceptance criteria

There are no scheduled release dates. Progress depends on confirmed contributors,
test hardware and review. A milestone is complete only with reviewable evidence.

## Current baseline: repository and design

Deliver English and Spanish documentation, contributor policies, optional ECC
guidance, animated banners and documentation checks. This does not deliver an
executable. Useful first contributions include reviewing failure scenarios,
clarifying specifications, documenting test hardware and improving translations.

## Milestone 1: secure core

Review the [prototype scope, open decisions and planned failure cases](readiness.md).

Specify exact protocol messages, resource limits, nonce/key derivation, local
secret protection, enrollment and recovery-export format before implementing them.
Build device identity, authenticated transfer, agreements and manual encrypted
upload/retrieval through a Go CLI.

Accept when synthetic content round-trips exactly; unauthorized peers and corrupt
data are rejected; interrupted transfers resume safely; quotas include temporary
data; and a recovery export can restore access using a new device identity.

## Milestone 2: resilient storage

Add 2 + 1 coding, replicated catalogs, receipts, agreement expiry, byte-sampling
audits and delegated repair. Specify audit sample sizes, costs and replay defenses
from experiments before freezing the protocol.

Accept when any one of three providers can be lost while recovery succeeds;
missing/corrupt fragments trigger bounded repair to an approved spare; fewer than
two valid fragments produce an unavailable state; and duplicate work, expiry and
restart do not cause unsafe deletion or quota overflow. Repair must work without
giving the worker file-decryption keys.

## Milestone 3: folder synchronization

Add stable capture, signed causal versions, notifications plus scans, atomic local
application, conflict copies, deletion markers and 30-day recoverable history.

Accept when disconnected concurrent edits preserve both versions; delete/edit
conflicts retain edited content; reconnecting devices do not resurrect acknowledged
deletions; local edits during download are preserved; and incompatible names are
reported without silent data loss. Test directory races and missing mounts too.

## Milestone 4: multiplatform closed alpha

Deliver actual CLI binaries and documented operation for Windows, Linux and macOS.
Use two authorized synchronization devices and three storage providers across
different households; add a spare provider for replacement repair. Include
stable and intermittent machines, restrictive NAT and real internet transfers.

Accept after direct connections, relay fallback and total connection failure are
exercised; restart and low-disk tests pass; resource limits, schedules and battery
pause are observed; and encrypted recovery succeeds without the original device
or its database. Publish measured results and unresolved problems, not an SLA.

## Later work

A GUI follows a usable daemon. Public admission and global credits require
additional abuse and accounting designs. Community AI starts with complete existing
models on capable providers under separate compute agreements, then evaluates
partitioned inference. Training a model is a separate initiative. None of these
capabilities is part of the current documentation delivery.
