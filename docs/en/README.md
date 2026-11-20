<!-- docs-pair: index; revision: 7 -->

[English](README.md) | [Español](../es/README.md) | [Project home](../../README.md)

# HiveScope documentation

HiveScope is in design. Start with the [roadmap](roadmap.md) for the first
encrypted recovery test and the milestones for automatic resource allocation,
private computing and hosting. [Offer my computer](resource-offers.md) follows
one request from the person asking for resources to the machine lending them.

## Design

- [Vision and boundaries](vision.md)
- [Offer my computer](resource-offers.md)
- [Architecture and interfaces](architecture.md)
- [First recovery test and open design decisions](readiness.md)
- [Recovery package and signed reads](recovery-format.md)
- [How recovery identifies and verifies the file](file-catalog.md)
- [When a provider can confirm an upload](upload-confirmation.md)
- [Encrypted storage and maintenance](storage.md)
- [Folder synchronization](synchronization.md)
- [Security and threat model](security.md)
- [Roadmap and acceptance criteria](roadmap.md)
- [Glossary](glossary.md)

## Participation

- [Contributing](contributing.md)
- [Governance and maintenance](governance.md)
- [Code of conduct](code-of-conduct.md)
- [Translations](translations.md)
- [Optional AI-assisted development and ECC](ai-assisted-development.md)
- [Licensing](licensing.md)

For local documentation checks, use Node.js 22 or newer, run `npm ci` in the
repository root, then `npm run check`. These commands check documentation only.
