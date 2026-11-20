# Working on HiveScope

These repository instructions apply to human-assisted and automated changes.
They do not grant authority to publish, change permissions or install tools.
Session instructions and the user's explicit authorization take precedence.
Spanish guidance: [contribution guide](docs/es/contributing.md) and
[AI-assisted development](docs/es/ai-assisted-development.md).

## Current scope

HiveScope is in design and documentation. Use the bilingual roadmap to
distinguish proposals from shipped features and measured results. The first
prototype targets recovery of one encrypted file on a replacement device.
Go is the first candidate for the desktop node.

An open network lets people request resources without contributing hardware.
Community directories advertise services. Nodes authenticate one another and
admit reservations under the host's local profile. Only the folder owner
authorizes readers. In a future general-purpose computing session the requester
chooses the software. Release session keys only after the client verifies
the required execution environment, including a GPU when used. If verification
fails, the request waits and the requester keeps the keys.

## Before changing anything

1. Read the relevant design, roadmap and contribution documents.
2. Inspect existing files and verify technical claims against primary sources.
3. For changes to trust, cryptography, retention, protocol or permissions,
   propose a documented design decision before implementation.
4. Keep the change bounded. Avoid unrelated cleanup and new tool dependencies.

## Documentation and languages

- English is the editorial reference; English and Spanish are both maintained.
- Update both language versions for a behavior or policy change.
- Keep matching `docs-pair` identifiers and revision numbers. Increase the
  revision in both files when their shared meaning changes.
- Translate prose, not code identifiers, command names, paths or protocol fields.
- Add future languages through the translation guide; label incomplete work.
- Keep status claims honest: proposed, implemented, verified and released are
  different states.

## Verification and review

- Run `npm ci` and `npm run check` for documentation changes.
- Review banners at full and narrow widths, with reduced motion enabled and
  disabled, after changing SVGs.
- When implementation begins, add meaningful tests for changed behavior and
  failure modes. Do not invent test commands before a toolchain exists.
- Report what was actually checked and any unresolved limitations.
- Never publish keys, recovery bundles, credentials, user files or identifying
  production logs. Use synthetic examples.
- Do not silently weaken encryption, authentication, quotas or authorization.
- Preserve license notices for reused material.

## Optional assistant tooling

ECC is a reference and an optional contributor tool, not a project dependency.
Do not install it, add hooks, enable MCP services, persist session memory or
change a contributor's global assistant settings as part of ordinary work.
Do not add automatic delegation requirements. All contributions must remain
reviewable without a particular assistant, model or paid service.
