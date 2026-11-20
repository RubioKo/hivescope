<!-- docs-pair: vision; revision: 5 -->

[English](vision.md) | [Español](../es/vision.md) | [Index](README.md)

# Vision and boundaries

HiveScope aims to let anyone with a compatible computer use it as a small community
server. They could lend space for encrypted files, run a task for someone else or
keep a service online for a group. Private storage is the first planned service.
Folder owners keep their keys and decide which devices can read each folder.

## How people lend resources

The owner would choose which services to run, who can use them, when they are
available and how much disk, CPU, memory, available GPU capacity and bandwidth
to offer. For example, Luis could offer only disk space during the day and help
transcribe public recordings at night. Someone who needs resources would choose a compatible
offer from their approved contacts. They could request a service even if their own
device only acts as a client, provided the other party agrees. Each task would use
only the data and resources covered by that agreement.
The proposed flow shows the agreement before it is sent and what happens if the
host shuts down during service. [Offer my computer](resource-offers.md)

## Problems we want to solve

- **My only copy is on a computer that could fail.** A family could protect its photos,
  and a student could protect a thesis or a project's source code. The first
  test will turn off the original computer and recover an identical file on another
  device using the recovery package, its password and a reachable provider whose
  agreement allows retrieval.
- **I work on two computers and have lost track of the latest version.** A later milestone
  will let the owner's authorized devices synchronize a folder. If the same document is
  edited on both computers while they are offline, the test is whether the owner can
  find both versions after reconnecting and choose which one to keep.
- **Our archive needs transcripts of public recordings.** For the proposed pilot,
  an archive volunteer could select a published public-domain recording from a
  community-reviewed catalog. A host could use spare computing time to turn it into
  text. A creator transcribing an unpublished interview would use their own device.
- **I want to help and still need my own computer.** Someone could offer disk space at
  night and keep their computing capacity for their own work. We can test whether
  HiveScope respects their schedule and disk quota and requires a separate computing
  agreement before running a transcription.
- **Our game world goes offline when its creator leaves.** A group of players could
  take turns hosting on volunteer computers and keep playing in the same world
  without depending on one person's schedule.

## Game servers and other community services

To test continuous hosting, we propose a game whose dedicated server can save and restore a
world. The community suggested Project Zomboid as an example. In the pilot, one approved
computer would host the game and another would be ready to take over from the last
valid save. Only one would accept changes to the
world at a time. Each host would opt in to that service separately and choose which
resources and hours to offer. The host running the server can access the world
data it processes.

The aim is for the group to be able to join at any hour, with low latency and no
required server rental fee. To evaluate those goals we would measure how often
players can actually join and play, response times from their own connections
and the resources each host contributes. We would also test a scheduled handoff
and an unexpected outage to find out how long it takes players to rejoin and how
much progress they can lose. Power, connectivity and any relays count toward the
cost even when volunteers provide them.

To choose a host, we propose comparing round-trip times from players to each
candidate with available capacity. Recording the results by group will show when
a good average hides a slow connection.
[Example of host selection based on player latency](https://docs.aws.amazon.com/gameliftservers/latest/developerguide/queues-design-latency.html)

If the pilot works, the same approach could be evaluated for other services, such
as a club's events calendar. Each service would need approved software, people
responsible for maintaining it and a recovery test for its own data. The
[roadmap](roadmap.md) describes the pilot.

## First useful service

The first prototype will test manual upload and recovery of a sample file. Private
folder synchronization between the owner's authorized devices follows later in the
[roadmap](roadmap.md). Approved community providers retain encrypted fragments and
help restore availability.
An approved provider does not automatically become a reader of the folder.

The initial network uses bilateral agreements. Each provider decides whom to
serve, for how much physical capacity, for how long and at which transfer limits.
There is no network-wide balance, token, blockchain or universal exchange rate.
Approval is a local decision rather than an account in a central registry.

Some community nodes are expected to be regularly online. Offline editing remains
possible, but remote recovery needs enough reachable fragments and metadata.
The alpha will not advertise a durability or availability SLA.

## Participant control

Sharing is opt-in. A participant must configure the storage directory, quota,
upload/download limits and schedule before accepting data. The design includes
manual pause and battery-aware pause. Detecting games or video calls is later work.
Departing providers should allow migration during their agreement's remaining
term; abrupt failures are still part of the threat and reliability model.

## Meaning of free

The goal is no mandatory service fee. Hardware, electricity, connectivity,
maintenance and relay traffic still have costs paid by participants or donors.
An offer of disk space does not guarantee access to scarce GPU capacity.
The project does not promise unlimited storage, bandwidth or inference.

## Later community AI

Personal recordings are private by default. The proposed transcription tool would
process them on the owner's device and keep the transcript there. If the recording's
status is unknown, the same rule applies. Insufficient resources or a failed attempt
would leave the job local for a retry. Folder backup remains a separate operation
that sends encrypted fragments to storage providers.

For the community pilot, a requester would choose an approved host and a public
recording from the reviewed catalog. The host would receive the catalog identifier,
retrieve the published recording and run a complete open model in an approved
runtime. The host can hear that audio and read the resulting text. This request accepts catalog identifiers
only. Local files, pasted notes and private prompts cannot accompany it. Selecting
a trusted host or marking a file as shareable does not add it to the catalog.
Each model needs a license review to establish how its weights can be used and
redistributed.

Additional kinds of work need a design and security review before inclusion.
After measuring the first service, the community can evaluate splitting a model
across providers. Model training requires a separate data, licensing, evaluation
and compute plan.

## Outside the first implementation

Public download links, multi-user editing, mobile clients, transparent cloud-only
files, global credits, permissionless admission, model training and anonymous
networking are outside the first storage implementation. The initial architecture
targets Windows, Linux and macOS with a Go daemon and CLI; a GUI follows later.
