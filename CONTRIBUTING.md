# 🤝 Contributing to HiveScope

Thanks for your interest in improving HiveScope! This document explains how to contribute cleanly and safely.

## Workflow

1. **Fork** the repository and clone it.
2. Create a branch from `dev`: `git checkout -b feat/short-name`
3. Make your changes following the conventions below.
4. Open a **Pull Request** against the `dev` branch (never directly against `main`).
5. Wait for review and a green CI.

## Commit convention (Conventional Commits)

```
feat:     new functionality
fix:      bug fix
docs:     documentation only
refactor: code change without behavior change
chore:    maintenance (deps, config, CI)
test:     tests
```

Example: `feat: add particle animation on message links`

## 🔒 Hygiene checklist — MANDATORY before every push

- [ ] No API keys, tokens, or passwords (grep for `sk-`, `key`, `token`, `password`).
- [ ] No real ChatDev logs or content from personal projects.
- [ ] No absolute paths from your machine (e.g. `C:/Users/...` or `/home/yourname/...`).
- [ ] All sample data in `examples/` is 100% synthetic.
- [ ] `.env.example` contains variable names only — never real values.
- [ ] `gitleaks detect --source . -v` runs clean.

> ⚠️ A secret pushed once lives in the Git history forever and must be considered
> compromised (rotate it immediately). That is why this checklist is not optional.

## Code standards

- **Frontend:** strict TypeScript, functional components, hooks. No unjustified `any`.
- **Bridge (Python):** Python 3.10+, `asyncio`, type hints, no heavyweight dependencies.
- **Language:** all code, commits, and identifiers in English.

## Definition of Done

A PR can be merged when it:

1. Passes CI (lint + build).
2. Passes the hygiene checklist.
3. Includes documentation for any user-visible behavior change.
4. Has been reviewed by at least one maintainer.

## Questions?

Open an [Issue](../../issues) with the `question` label. Every question is welcome!
