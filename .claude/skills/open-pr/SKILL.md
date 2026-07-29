---
name: open-pr
description: Use whenever opening a pull request in this repo (gh pr create, "open a PR", "create the PR"). Enforces a PR description with Motivation first, then Summary, then Test plan — never an empty or guessed Motivation.
---

# Opening a PR

PR descriptions in this repo lead with **Motivation**, before Summary and Test plan.

```markdown
## Motivation
<why this change exists — the reasoning, not the mechanics>

## Summary
- <bullet points of what changed>

## Test plan
- [ ] <bulleted checklist>
```

## Motivation is the why, not the trigger

One or two sentences on the reasoning behind the change — the problem, gap, or
risk that made it worth doing. Not who asked or what ticket it came from, just
the why. E.g.:

> The README was a placeholder and CLAUDE.md said no Go source existed, which
> made onboarding harder than it needed to be.

not:

> User asked me to add a README on 2026-07-29.

## Never leave Motivation empty or invented

Before running `gh pr create`, resolve Motivation in this order:

1. **Infer it from context** — the conversation, commit messages on the
   branch, or a linked issue/task usually already state the why. If it's
   clearly established, write it straight in. Don't interrupt to confirm
   something already obvious.
2. **If it's genuinely unclear** — e.g. you're picking up someone else's
   branch cold, running as a subagent with no upstream context, or the
   commits only describe *what* changed, not *why* — stop and ask instead
   of guessing:
   - If you're the main agent talking to the user directly, ask them
     (`AskUserQuestion` or a plain question).
   - If you're a subagent spawned for this task, ask the orchestrating
     agent/parent for the motivation rather than fabricating one or
     shipping the PR without it.
3. Never fall back to a vague placeholder ("various improvements", "see
   commits") or an empty section just to keep moving.

## Base branch: default to main, and check ancestry first

`main` is the base branch for PRs in this repo — don't default to stacking
on another feature/integration branch (e.g. `unit_test`) just because the
current branch happened to fork from it.

Before running `gh pr create`, check whether the base you're about to use
actually gives a clean diff:

```
git fetch origin
git log --oneline <your-branch> ^origin/main
```

If that list contains commits you didn't write for this PR (i.e. the branch
forked from something that's ahead of `main` but hasn't been merged to
`main` yet), the diff against `main` will bundle in unrelated, already
reviewed work. Don't silently pick whichever base makes the diff look
smaller — tell the user what you see (which branch is ahead of `main` by
what) and let them choose: merge the intermediate branch into `main` first
(cleanest), or open against `main` accepting the wider diff.

## Rest of the flow

Summary and Test plan follow the existing repo convention (see the global
git/PR instructions): short bullets for what changed, a checklist for how
it was verified. Keep using `gh pr create --body "$(cat <<'EOF' ... EOF)"`
so formatting survives.
