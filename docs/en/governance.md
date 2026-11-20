<!-- docs-pair: governance; revision: 2 -->

[English](governance.md) | [Español](../es/governance.md) | [Index](README.md)

# Governance and maintenance

The initial maintainer is [RubioKo](https://github.com/RubioKo). No steering group,
foundation, corporate sponsorship or additional maintainer is implied.

## Decisions

Use public Issues for non-sensitive proposals and pull requests for reviewable
changes. Describe the problem, alternatives, chosen behavior, risks and acceptance
criteria. Major decisions must update the English and Spanish design documents.
Sensitive findings follow the [security policy](security.md).

Seek technical agreement through evidence and review. The initial maintainer has
final merge and scope authority while the project has one maintainer. Participants
can record a reasoned disagreement without making personal attacks. Future
maintainers are added through a public, reviewed governance change with their consent.

## Branches and review

`main` is the reference branch. Work happens on short-lived branches or forks and
arrives through pull requests. Maintainers review scope, correctness, language
parity, licensing and the checks required for the change. A green documentation
check is not a security audit. Direct administrative changes are reserved for
repository maintenance and must be described when they affect contributors.

Before merging to `main`, resolve review conversations and pass the
`Markdown, links and language parity` check from GitHub Actions. The active rule
blocks force pushes and branch deletion and has no bypass. While one maintainer
has write access, the approval minimum is zero. When a second maintainer gains
write access, require one independent approval as part of granting that access.

Dependabot checks npm dependencies and GitHub Actions weekly and can open
security update PRs between scheduled checks. The repository requires actions
to use full commit SHAs and allows two external actions, `actions/checkout`
and `actions/setup-node`. To add another, explain the need and requested
permissions in a PR so the maintainer can review the change to the allowlist.

## Future releases

There are no supported software versions yet. A release must identify its scope,
license, platform support, checks and known limitations. Before publishing
executables, document the release verification and update trust model. Do not
create release tags that suggest a usable storage service for documentation alone.

Repository administration is separate from P2P operation: being a maintainer does
not confer authority over another participant's files or storage agreements.
