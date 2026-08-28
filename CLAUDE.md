# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

context-debug is a Go package (`github.com/iruvan/context-debug`) that lets a
service collect per-dependency-call debug data (name, request, response,
error, duration) on a `context.Context`, and read it back later — e.g. to
attach a debug trail to an HTTP response only when debug mode is enabled.

## Architecture

- `contextdebug.go` — the whole package. A `store` (guarded by a mutex) is
  attached to a context via an unexported context key.
  - `New(ctx, opt ...Option) context.Context` — attaches a fresh store;
    `Option{EntriesLimit}` optionally caps how many entries it retains.
  - `Enabled(ctx) bool` — whether `ctx` has an active store.
  - `Collect(ctx, Entry)` — appends an entry; no-op if debug mode isn't
    enabled, so call sites don't need to guard every call.
  - `Snapshot(ctx) []Entry` — returns a copy of everything collected.
- `example/` — a standalone Go module (`go.mod` replaces
  `github.com/iruvan/context-debug` with `../`) demonstrating usage in an
  HTTP service that queries Postgres and an external API through
  instrumented dependency methods. `docker-compose.yml` (Postgres only),
  `db/init.sql` (schema + seed data) and a `Makefile` make it runnable
  with `make run` alone.
- `docs/` — design docs (`TRD.pdf`) and `decision_log/`.

## Commands

```
go test ./...              # run the package tests (root module)
cd example && make run     # seeded Postgres in Docker, service on the host
cd example && go run .     # run against your own DB (needs POSTGRES_DSN)
```

There is no separate lint config; rely on `go vet` and standard `gofmt`.
