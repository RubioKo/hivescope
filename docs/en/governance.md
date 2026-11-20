<!-- docs-pair: governance; revision: 3 -->

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

`main` is the stable default branch. Start a short-lived branch from
`development` and open a pull request back to `development` to integrate your
work. Maintainers review scope, correctness, language parity, licensing and
the checks required for the change. When the milestone meets its
[roadmap criteria](roadmap.md), open a PR from `development` to `main` with the
results and use **Create a merge commit** to preserve the shared history.

Both permanent branches require a PR, resolved review conversations and a
passing `Markdown, links and language parity` check from GitHub Actions.
Both block force pushes and deletion with no bypass. GitHub automatically deletes
merged work branches hosted in this repository. While one maintainer has write
access, the approval minimum is zero. When a second maintainer gains
write access, require one independent approval as part of granting that access.

Dependabot updates and urgent fixes may go to `main` through a PR. After
merging one, open a PR from `main` to `development` and use a merge commit to
bring the fix back into ongoing work. Maintainers document administrative
changes that affect contributors. The documentation check covers Markdown,
local links and language pairs.

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
