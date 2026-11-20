<!-- docs-pair: home; revision: 7 -->

English | [Español](README.es.md)

![HiveScope — community-powered storage, open source and in design. Contributors welcome.](assets/banner.svg)

# HiveScope

HiveScope is an open-source project to help people use their own computers as
**small community servers**. Someone with spare disk space could keep an encrypted copy
of a friend's work. A more powerful computer could help caption recordings from a public archive.
A group of players could take turns keeping their shared world online. Each person decides
what to lend, to whom and for how long.

**Project status — design and documentation.** The first planned prototype is a
command-line tool for storing and recovering a file. This repository contains the design,
security model and tests we intend to run. [Project vision](docs/en/vision.md) · [Roadmap](docs/en/roadmap.md)

## What the community could use it for

- **Protect files and work across devices.** Choose folders containing family photos,
  coursework, music, source code or other files. Encrypt them on the owner's device
  before sending them to approved storage providers. Recovery comes first. Later,
  authorized devices would synchronize changes and keep both versions when offline edits conflict.
- **Transcribe a private recording.** HiveScope would treat personal recordings as
  private by default. A podcast creator could transcribe an interview on their own
  computer. The audio and transcript would stay on that device during the job.
  If the computer lacks the resources, HiveScope would explain why and offer a local retry.
- **Lend computing power for public work.** In the first community AI pilot, a
  volunteer could run an approved open model on an already published recording, such
  as a recorded story a library makes available for transcription. The request would identify an
  entry in an approved public catalog. The host could access that recording and the
  resulting transcript.
- **Keep a service running together.** The proposed game-hosting trial would give a group
  one active host and an agreed standby. If the active host goes offline, the standby
  could restore the last valid world save and the players could reconnect. The host can
  read the world data it processes. Project Zomboid is one suggested example. A club's
  events calendar is another possible use, subject to its own hosting and recovery tests.

The aim is to make services available without a mandatory rental fee. Participants or
donors would still cover hardware, electricity, connectivity and any relay traffic.
For games, the pilot would measure when people can actually join, latency from players'
connections, time to reconnect and how much progress a handoff loses. Those measurements
will tell a group how close it can get to round-the-clock play. [Community hosting trial](docs/en/roadmap.md#community-game-server-trial)

## What it means to offer your computer

Luis might offer Ana 20 GB of disk space in the evenings and keep his GPU for his own
work. In the proposed **Offer my computer** flow, he chooses one service and one approved
contact, sets a schedule and expiry, and reviews the limits. A disk offer sets a quota
and transfer limits. A computing offer can cap CPU, memory, GPU use and concurrent jobs.
The offer stays pending until Ana accepts the terms. She can participate solely as a
requester if someone agrees to help her.

If Luis needs his computer back, he can pause new requests. Stopping a public catalog transcription
already in progress may lose unsaved work. Taking a storage host offline makes its
fragments temporarily unreachable. A game handoff depends on another authorized host
and a valid save. The proposed shutdown screen shows which of these applies before
Luis decides what to do. [Resource offers](docs/en/resource-offers.md)

## How a private folder would work

### Ana chooses which devices can read her files

![Ana points to a family photo beside a document and an audio waveform on her laptop.](assets/story/hivescope-story.jpg)

Ana wants to open her thesis draft, a photo and a recorded interview on both her laptop
and her desktop. She authorizes both devices for `Studies`. For `Personal documents`,
she authorizes only the laptop. Each choice applies to every file in that folder.

### Luis stores the encrypted copy

![Luis adjusts a slider beneath rows of colored blocks representing encrypted data on his screen.](assets/story/encrypted-storage.jpg)

Ana's computer encrypts the contents and file names before sending data. Luis stores
encrypted fragments and can observe details such as transfer size and timing. The keys
that open the files stay with Ana's authorized devices. If Ana transcribes her private
interview, the model would run on her own computer. Luis would continue to hold only
encrypted fragments of her backup. [Security model](docs/en/security.md)

### Ana recovers her work on a replacement computer

![Ana holds a USB drive beside her closed laptop. The photo, document and recording reappear on a desktop monitor.](assets/story/recovering-files.jpg)

With her original laptop off, Ana unlocks the recovery package she kept separately
using its password. Her new computer needs a reachable catalog and enough valid
fragments from providers whose agreements still allow retrieval. The first prototype
must recover a sample file byte for byte. [Recovery design](docs/en/architecture.md#ownership-and-recovery) · [Storage and repair](docs/en/storage.md)

## Development and platforms

1. **Secure core.** Build a Go command-line prototype for device identity, peer approval,
   encrypted transfer and recovery on a new computer.
2. **Resilient storage and folder sync.** Distribute each encrypted block across three
   approved hosts so that any two valid fragments can recover it. Test repair when a
   host goes offline, then test offline edits and conflicts across the owner's devices.
   [Folder synchronization](docs/en/synchronization.md)
3. **Closed desktop alpha.** Validate the command-line tool on Windows, macOS and Linux
   over real household connections. The graphical interface would follow a usable
   background service. Phone and tablet clients are exploratory designs.

Community computing and persistent hosting are later proposals. For technical contributors,
the design describes a Go background service, authenticated peer connections using libp2p
and a local SQLite index on each node. [Architecture](docs/en/architecture.md) · [Documentation index](docs/en/README.md)

## Contribute

- **Review a failure case.** What should a user see if a storage provider disappears
  halfway through recovery? Describe the expected behavior using sample files. [Propose a scenario](https://github.com/RubioKo/hivescope/issues/new?template=proposal.yml)
- **Design a home hosting trial.** Describe the equipment, hours and bandwidth a
  volunteer could offer, or how a play group would check that its world survives a
  handoff. [Open a design discussion in Issues](https://github.com/RubioKo/hivescope/issues)
- **Improve the documentation.** English and Spanish are maintained together. You can
  clarify an explanation, check a translation or help add another language. [Translation guide](docs/en/translations.md)

Start a working branch from `development` and target `development` with a pull request.
Sensitive security findings have a private reporting channel. [Contributing](docs/en/contributing.md) · [Security reporting](SECURITY.md) · [Governance](docs/en/governance.md) · [Code of conduct](docs/en/code-of-conduct.md)

Contributor instructions and optional assistant tooling are documented separately.
[Contributor instructions](AGENTS.md) · [Optional AI and ECC guidance](docs/en/ai-assisted-development.md)

## License and attribution

New HiveScope material is licensed under **AGPL-3.0-only**. The animated banners,
adapted from RubioKo's original design, retain their **MIT** license and attribution.
[Project license](LICENSE) · [Licensing guide](docs/en/licensing.md) · [Third-party notices](THIRD_PARTY_NOTICES.md)
