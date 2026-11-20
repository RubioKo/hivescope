<!-- docs-pair: roadmap; revision: 3 -->

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
additional abuse and accounting designs. The community AI pilot would use recordings
from an approved public catalog, complete open models and separate compute agreements.
For private recordings, the goal is transcription on the owner's device. Before
enabling it, use synthetic audio marked private to test that a local failure or retry
keeps the audio and transcript on the device, even with a volunteer host available.
Partitioned inference is a later research question. Training a model is a separate initiative. None of these
capabilities is part of the current documentation delivery.

### Community game server trial

Explore round-the-clock play with low latency and no mandatory rental fee. Choose
a game whose license permits community hosting of a dedicated server. Agree on an
active host and a standby. Disconnect the active host mid-game and restore the
latest available save on the standby. Check that players can reconnect and only
one server accepts changes, even when the original host returns. Measure downtime,
lost progress and player latency by region before and after the handover. Record
electricity and connectivity costs and who covers them. Use the results to decide
whether to trial other persistent services.
