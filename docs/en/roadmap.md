<!-- docs-pair: roadmap; revision: 4 -->

[English](roadmap.md) | [Español](../es/roadmap.md) | [Index](README.md)

# Roadmap and acceptance criteria

HiveScope is in design. Each milestone below describes a proposed delivery and
the evidence contributors will need to publish. Release dates depend on
review, available hardware and volunteers.

## Milestone 0 — Design and public documentation

Publish the vision, resource profiles, architecture, threat model and roadmap in
English and Spanish. Preserve the language selector, illustrated story,
contribution routes and private security reporting in both project READMEs.

**Acceptance.** Ask someone new to HiveScope what they can request, what they
can lend, who controls software inside a session, who sets physical limits and
what happens if a suitable protected host cannot be found. Record gaps in the
explanation and revise both versions. [Offer my computer](resource-offers.md)

## Milestone 1 — First recoverable copy

Start with a desktop CLI, one synthetic file, one storage provider on a LAN and
a clean replacement device. Specify the protocol messages, resource limits,
key and nonce handling, local secret protection and recovery package before
implementing them. The draft recovery design records how a replacement proves
its right to read and how a provider remembers revocation.
[Recovery design PR](https://github.com/RubioKo/hivescope/pull/1)

**Acceptance.** Turn off the original device. Using the saved recovery package,
its password and a reachable provider, recover the file byte for byte. Revoke
the recovery credential, wait for the provider's confirmation, restart the
provider and retry with the old package. Both catalog and object reads must
return zero bytes. Also test interrupted transfers, a wrong password and corrupt
data. A failed recovery must preserve an existing destination file. One provider
establishes a verified copy with a single point of failure.

## Milestone 2 — Automatic allocation on an open network

Use controlled test machines and synthetic data with the same invitation-free
flow intended for public participation. Add expiring directory advertisements,
replaceable directories, authenticated peers, local profiles, time-limited
reservations and host-wide budgets that also cover temporary data.

**Acceptance.** A requester that lends nothing completes a copy on a stranger's
host without personal approval. Exclude a candidate outside its schedule. Race
two requests for one remaining slot, interrupt after the host commits but before
the requester receives the receipt, and retry. Only one slot may be consumed.
Pause the host and take the first directory offline. The requester must be able
to use another directory or see why the operation is waiting and cancel it.
Repeated requests and multiple peer identities must stay within the host's
total capacity.

## Milestone 3 — Resilient storage and folder synchronization

Distribute each encrypted block across three providers using two data fragments
and one parity fragment. Replicate catalog metadata, issue receipts and measure
the cost and coverage of byte-sampling checks. Add bounded repair, signed file
versions, conflict copies, deletion markers and 30-day recoverable history.
[Storage](storage.md) · [Synchronization](synchronization.md)

**Acceptance.** Report redundant protection only after confirming all three
placements per block and the catalog. Remove one provider and restore the file
with two valid fragments per block. Report degraded protection until a compatible
spare confirms repair. With fewer than two valid fragments for any block,
or without the required catalog, report the file unavailable. Repair uses
ciphertext and tolerates retries and restarts within quota. Reconcile offline
edits and delete/edit conflicts, retaining both competing edits and any newer
local work. Verify that a missing mount or incompatible file name causes a
visible error without silent deletion.

## Milestone 4 — Desktop alpha

Exercise the CLI and background service on Windows, Linux and macOS using
household connections. Test with two owner-authorized devices, three storage
providers in different households and a spare. Build the graphical interface
on the same measured operations. Phones and tablets will first be evaluated
as access and recovery clients.

**Acceptance.** Publish direct, relayed and failed connection results, and
exercise restarts, low disk space, battery pauses and recovery after losing
the original device. Agree on resource budgets before the measurements. On a
four-core PC with 8 GB of RAM, report idle and transfer memory, idle and active
CPU, recovery throughput and time to pause. Go is the first candidate for the
node. Revisit Rust if measurements under equivalent conditions show a meaningful
gain against a missed budget. For the network, report directory renewals,
reconnect bursts, relay use and repair traffic separately. The reference PC
is for measuring the desktop node. Private session hosting requires its own
compatible hardware.

## Milestone 5 — Private general-purpose compute

Offer CPU and RAM in a confidential virtual machine. The requester selects the
software, loads their own program and retrieves its result. Evaluate GPU
support separately, including the protected connection between CPU and GPU.
[Security model](security.md)

**Acceptance.** Run a requester-chosen program within the host profile.
On missing, stale or incompatible attestation, verify that keys and private
inputs stay with the requester and the request can wait or be cancelled.
Exercise a missing GPU attestation and a retry on a different host. Stop a
running session, release the allocated resources and report exactly which
results or checkpoints are recoverable.

## Milestone 6 — Persistent services and games

A community chooses a service, such as an application or a game, and the
resources it needs. The nodes arrange a compatible active host and standby
under their respective profiles.

### Community game server trial

Choose a game whose license permits community hosting. Project Zomboid is
one candidate. Record which groups and regions participate and who covers
electricity, connectivity and relay costs.

**Acceptance.** Disconnect the active host during play, verify the replacement
environment and restore the latest valid save. Reconnect players and bring the
former host back online. Only one instance may accept new changes throughout.
Publish latency by region, interruption length, time to reconnect and any lost
progress before setting availability expectations.
