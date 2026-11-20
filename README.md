<!-- docs-pair: home; revision: 13 -->

English | [Español](README.es.md)

![HiveScope — community-powered storage, open source and in design. Contributors welcome.](assets/banner.svg)

# HiveScope

HiveScope is an open-source project for sharing storage and computing power across the
world. If you need to protect a folder, run your own software or keep a server online
for your community, the app will look for computers that can help. If you want to lend
resources, you choose what to share, how much and when. The computers coordinate the work.

**HiveScope can restore an encrypted file locally.** You choose the destination and
whether to replace an existing file. The reader checks the entire file first. In the
interruption test, stopping the process just before replacement leaves the original
file intact. The offline examples cover an empty file, five bytes and 4 MiB + 1 byte.
Recovery on a replacement computer and revocation at a running provider are the
goals of the first desktop prototype. Private computing and service hosting come later.
[First prototype and test results](docs/en/readiness.md) · [Project vision](docs/en/vision.md) · [Roadmap](docs/en/roadmap.md)

![A world map connects three scenes. A traveler opens a private folder on a phone, a creator works on a laptop and a volunteer offers resources from a desktop.](assets/story/worldwide-network.jpg)

*A vision of HiveScope. You choose which devices can open your folders. When your own
computer is too small for a job, you could request capacity from the community.*

## What could you do with HiveScope?

### Protect the files you choose

![A masked owner views the same private folder on a laptop and a tablet. Both show a lakeside cabin photo, a recording and a document.](assets/story/private-files.jpg)

Back up folders containing photos, documents, music, source code or other work. You
hold the keys and choose which of your devices may read each folder. The first test
will recover one file on a replacement computer. Folder synchronization and trials on
phones and tablets come later.

### Run your own software

![A designer renders a green helmet. On a separate computer, the person lending the GPU sees two meters showing GPU and memory use.](assets/story/shared-compute.jpg)

You could request CPU and memory for a private environment where you install your own
tools, compile a project, render a scene or run a model. Before sending session keys,
your device would verify the remote environment, including any GPU used. If verification
fails or a compatible host is unavailable, the keys stay on your device. You can wait
or cancel. [Security model](docs/en/security.md)

### Host a service for a community

![Two players explore the same lighthouse from separate screens. Beside the mini PC hosting their game, a small display shows the server online.](assets/story/game-hosting.jpg)

A gaming group could keep a shared world going across volunteers' schedules. One host
would accept changes at a time. A compatible standby could take over from a verified
save. Project Zomboid is one possible trial. The group would measure latency, downtime
and lost progress before making an availability claim. [Hosting trial](docs/en/roadmap.md)

## What happens when you ask for or offer resources?

You can participate solely as a requester. Choose what you need and your computer will
look in community directories for compatible hosts. People who lend resources set a
profile once. Luis, for example, might offer 20 GB of disk space in the evenings and
keep his GPU for himself. His computer checks its schedule, free space and existing
commitments before accepting a reservation. Ana and Luis can be complete strangers.

Luis can pause new reservations. Existing copies follow their retention agreements.
If capacity runs out, Ana sees why her request is pending. Retrying the same request
preserves its state without reserving the space twice. Participation is voluntary. The
aim is access without a required rental fee. Participants or donors still cover
hardware, electricity, connectivity and relay traffic. [Offer my computer](docs/en/resource-offers.md)

## How Ana would recover a private folder

### Ana chooses which devices may read her files

Ana authorizes her laptop and desktop to read `Studies`, which holds a thesis draft,
a photo and a recorded interview. For `Personal documents`, she authorizes only her
laptop. Each choice covers all files in that folder.

### Luis's computer stores the encrypted copy

Ana's computer encrypts the contents and names before sending the copy. Luis's storage
profile admits the transfer automatically. He can observe its size and timing. The keys
that open Ana's files stay under her control.

### Ana restores her work on a replacement computer

With her original laptop off, Ana uses her password to unlock the recovery package she
kept separately. In the first planned test, a reachable provider checks her recovery
credential, and the new computer restores a sample file byte for byte. Later, the
design will place each encrypted block across three providers so any two valid
fragments can reconstruct it. [Recovery design](docs/en/architecture.md) · [Storage and repair](docs/en/storage.md)

## Development and devices

The first deliverable is a desktop command-line tool. Go is the first choice for the
node. A four-core PC with 8 GB of RAM is the reference for measuring memory use, CPU,
recovery speed and time to pause. Confidential sessions need separate tests on compatible
hardware. The desktop alpha will test Windows, Linux and macOS and add a graphical
interface after measuring the background service. Phones and tablets will first be
evaluated for access and recovery.
[Architecture](docs/en/architecture.md) · [Documentation index](docs/en/README.md)

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
