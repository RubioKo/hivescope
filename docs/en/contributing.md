<!-- docs-pair: contributing; revision: 2 -->

[English](contributing.md) | [Español](../es/contributing.md) | [Index](README.md)

# Contributing

HiveScope welcomes documentation, protocol analysis, security review, translation
and future implementation work. English and Spanish are accepted. You do not need
ECC, an AI assistant, a subscription or a GPU to contribute.

## Choose a bounded task

Read the [vision](vision.md), [architecture](architecture.md), [roadmap](roadmap.md)
and [code of conduct](code-of-conduct.md). Use an Issue to describe the problem,
suggested result and acceptance criteria. Small corrections can go straight to a
pull request. Changes to cryptography, trust, protocols, retention or licensing
need a design discussion before implementation.

Good initial work includes a reproducible failure scenario, a threat-model review,
a clearer design explanation, translation review or a test-hardware description
without identifying personal information. Do not claim a task is implemented
because a proposal or mockup exists.

## Local workflow

1. Fork the repository and clone your fork.
2. Fetch `development` from `RubioKo/hivescope` and create a short-lived branch
   from its tip, even if your fork initially contains only `main`.
3. Make the smallest complete change, including its English and Spanish text.
4. With Node.js 22 or newer, run `npm ci` and `npm run check` in the repository root.
5. Push your branch to your fork. Open a pull request using the template and
   select `development` in `RubioKo/hivescope` as the base.
6. Address review. Maintainers merge accepted work into `development` and
   promote validated milestones to `main` as described in [Governance](governance.md).

These npm commands install and run documentation tools, not a HiveScope service.
Do not add empty source trees, placeholder executables or build instructions for
unimplemented software.

## Documentation quality

- Explain outcomes and limitations in plain language; cite primary technical sources.
- Update paired files and their shared `docs-pair` revision when meaning changes.
- Preserve relative links, accessible image descriptions and license attribution.
- Use synthetic paths and data. Never commit user files, keys, credentials or recovery material.
- For banner changes, check the full and narrow layout, animation and reduced-motion mode.
- Keep code identifiers and command names in English; prose can be English or Spanish.
- New languages follow the [translation guide](translations.md).

Use Conventional Commit prefixes such as `docs:`, `fix:`, `chore:` or, when code
exists, `feat:` and `test:`. English summaries help international search; Spanish
discussion and explanations remain welcome.

## Review and licensing

A pull request should state the problem, resulting behavior, checks actually run
and unresolved concerns. Changes need maintainer review and passing documentation
checks. Future executable changes also require meaningful behavior and failure
tests appropriate to their component.

Submit only material you have the right to contribute. Your contribution uses
the license applicable to the affected file; disclose third-party sources and
preserve their notices. Newly authored project material defaults to AGPL-3.0-only.
The animated banners are an explicitly documented MIT exception.

AI assistance is optional. You are responsible for accuracy, attribution,
confidentiality and review of your submission regardless of the tool used.
See [AI-assisted development](ai-assisted-development.md) and [AGENTS.md](../../AGENTS.md).
