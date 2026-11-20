<!-- docs-pair: ai-development; revision: 1 -->

[English](ai-assisted-development.md) | [Español](../es/ai-assisted-development.md) | [Index](README.md)

# Optional AI-assisted development and ECC

You can contribute using a text editor and the documented checks. AI tools are
optional. Contributors remain responsible for the correctness, licensing and
confidentiality of everything they submit.

## What ECC contributes

[ECC](https://github.com/affaan-m/ECC) is an external project offering workflows,
rules and skills for coding assistants. It can help a contributor organize
research, planning, implementation, review and verification. It is not the P2P
network, storage engine, shared compute service or security audit of HiveScope.

This repository contains original project instructions in [AGENTS.md](../../AGENTS.md),
not an ECC installation. No ECC code, hooks, skills, memory or MCP configurations
are bundled or required. We do not claim measured productivity improvements.

## Optional setup

1. Read the [official ECC README](https://github.com/affaan-m/ECC#readme) and its
   current support information for your assistant. Capabilities differ by tool.
2. Choose one documented installation method for that assistant, inspect the
   permissions and files it changes, and follow its official instructions.
3. Keep credentials, hooks, private memory and personal settings outside this
   repository. Do not commit global assistant configurations.
4. Start with planning, documentation or review assistance. Evaluate Go guidance
   when the implementation milestone begins.
5. Run the HiveScope checks independently of what an assistant reports.

The upstream project can change installation commands and supported features.
This guide intentionally links to its maintained instructions instead of pinning
an untested installer. ECC installation does not authorize repository publication
or change HiveScope's contribution and review rules.

## Applying the ideas to HiveScope

Research the existing design first. Propose small changes with clear acceptance
criteria. Review with a fresh perspective and verify the actual result. For
documentation, that means language parity, links and truthful status statements.
For future code, it includes the relevant failure and behavior tests.

Adapt advice to the component: browser security checklists do not cover P2P trust,
quota abuse, secret recovery, corrupted fragments or churn. Generic Go patterns
are suggestions to evaluate, not architecture mandates. Do not introduce automatic
delegation or a required model subscription.

## Data and attribution

Do not send real user files, keys, credentials or recovery exports to an assistant.
Use synthetic examples and inspect generated changes yourself. Cite sources and
identify substantive tool assistance where it helps reviewers understand how the
work was checked. Do not describe generated output as independently audited.
ECC remains subject to its own [MIT license](https://github.com/affaan-m/ECC/blob/main/LICENSE).
