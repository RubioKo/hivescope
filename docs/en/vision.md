<!-- docs-pair: vision; revision: 6 -->

[English](vision.md) | [Español](../es/vision.md) | [Index](README.md)

# Vision and boundaries

HiveScope is designing an open network where anyone can request storage or computing
capacity. People may participate solely as requesters. A volunteer decides which
resources to lend and how much of their computer they need to keep available for
themselves. Their computers would find each other and agree on work automatically.

## What a person could do

**Keep a recoverable copy.** A student chooses the folder with their thesis, notes
and photos. Their computer encrypts file names and contents before transfer. The
student chooses which of their own devices may read that folder. After the laptop
fails, the first prototype must recover a sample file from a reachable provider on
a clean replacement device with a valid recovery credential. A later milestone
will test repair across independent hosts and preserve both versions when offline
edits conflict. [Storage and repair](storage.md) · [Folder synchronization](synchronization.md)

**Use a private computer session.** A creator requests CPU and RAM, then installs the
tools of their choice inside their environment. They might build a project, render
a scene or run their own model. The proposed confidential virtual machine protects
the workload while it runs. Before releasing session keys, the requester's device
must verify the environment and bind those keys to that session. A GPU workload
also requires evidence for the GPU and its connection to the protected environment.
When the protection cannot be verified, the request waits or the person cancels.
Private inputs and keys stay with the requester. A host can still observe
operational information such as session duration or traffic volume and can go offline.
[Security model](security.md)

**Host a service with a community.** A gaming group could request hosting for its
world across volunteers' schedules. Project Zomboid is a suggested example. One
active host accepts changes. A compatible standby could start from the last verified
save after a handoff. The trial would record how long players take to reconnect,
what progress they lose and the latency from their actual connections. A club
calendar or another persistent service would need its own state recovery test.
[Roadmap](roadmap.md)

## What happens between the computers

A host sets a profile once, choosing disk, CPU, RAM, GPU or network bandwidth and
setting a schedule and limits. For example, Luis may lend 20 GB of disk in the
evening while keeping all computing capacity for himself. Ana can ask for a copy
without adding Luis as a contact. Replaceable community directories would carry
expiring service announcements. File names, session contents and keys remain
outside those announcements.

The requester finds candidates and connects to a host. The host checks its local
profile, physical capacity and existing commitments, then accepts a reservation
with an expiry. If the final space is contested, only one reservation can succeed.
A retry returns the state of the same request. A pause stops new reservations.
Existing storage follows its retention and migration agreement. A compute session
must report the last confirmed save when interrupted, so its owner can see what work
survived. [Offer my computer](resource-offers.md) · [Architecture](architecture.md)

## Participation and cost

The goal is access without a required rental fee. People may ask for help regardless of
the hardware they contribute. The pilot will rely on voluntary capacity and local
quotas, without required credits, balances or cryptocurrency. Participants or donors
bear the cost of hardware, electricity, maintenance, connectivity and relay traffic.
When capacity is unavailable, the requester can see why and choose to wait or cancel.

## How the project will test its promises

1. Start with an encrypted file, a single provider and recovery on a new device.
   Test a valid recovery credential and the same credential after a confirmed
   revocation and provider restart. Record this result separately from redundancy.
2. Exercise automatic reservations with unfamiliar peers before expanding the
   storage trial to two data fragments and one parity fragment, repairs and
   synchronization. Test directory failure, a host pausing and a lost reply.
3. Test the desktop node on Windows, Linux and macOS. A four-core PC with 8 GB of
   RAM is the reference for measuring resource use during storage, including pause
   latency. Measure directory and relay demand separately. Phones and tablets
   will first be evaluated as clients for access and recovery.
4. Trial general-purpose CPU and RAM sessions on hardware that can prove the
   required protection. Later trials will cover a verified GPU path, then persistent
   hosting with a compatible standby and a recoverable saved state.

The repository publishes the design, its assumptions and acceptance criteria.
Implementation and measured results belong to subsequent milestones. [Roadmap](roadmap.md)
