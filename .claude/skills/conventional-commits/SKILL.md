---
name: conventional-commits
description: Use when writing or reviewing git commit messages for this repo. Formats commits per the Conventional Commits v1.0.0 spec (https://www.conventionalcommits.org/en/v1.0.0/) — type(scope): description, with body/footers and breaking-change markers. Trigger on "commit message", "conventional commit", "write a commit", or when about to run `git commit`.
---

# Conventional Commits

Format commit messages according to the [Conventional Commits v1.0.0](https://www.conventionalcommits.org/en/v1.0.0/) spec.

## Structure

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## Types

- `feat` — new feature (MINOR semver bump)
- `fix` — bug fix (PATCH semver bump)
- `build`, `chore`, `ci`, `docs`, `style`, `refactor`, `perf`, `test` — recommended, not required by the spec

Pick the type that matches what actually changed; don't default to `chore` for everything.

## Rules

- **Scope**: optional, parenthesized noun for the affected area, e.g. `fix(parser): ...`
- **Description**: required, short summary right after `type: ` or `type(scope): `
- **Body**: optional, starts one blank line after the description, free-form, can be multiple paragraphs
- **Footers**: optional, start one blank line after the body, `Token: value` or `Token #value` format, tokens use hyphens instead of spaces (e.g. `Reviewed-by: ...`, `Refs #123`)
- Nothing is case-sensitive **except** `BREAKING CHANGE`, which must be uppercase

## Breaking changes

Mark either way (or both):
1. `!` right before the colon: `feat!: drop support for Node 6`
2. A footer: `BREAKING CHANGE: <description>`

Breaking changes map to a MAJOR semver bump.

## Examples

```
feat(auth): add OAuth2 login flow
```

```
fix: prevent race condition in request handler

Guard the shared counter with a mutex; concurrent requests
were occasionally double-incrementing it.

Refs #42
```

```
feat!: remove deprecated context.Value key lookup

BREAKING CHANGE: callers must use the typed accessor functions
instead of raw context.Value keys.
```

## Applying this in the repo

- Check `git log --oneline -20` for how recent commits are actually styled before assuming this repo already follows the spec strictly.
- When drafting a commit message per the user's global commit workflow (see CLAUDE.md / top-level instructions on commit style), still shape the subject line as `<type>[optional scope]: <description>`.
