# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A hands-on teaching lab: extracting the **Warehouse** Bounded Context out of **MIC**, a legacy PHP
invoicing monolith, into a clean **Go** microservice, using the Strangler Fig pattern. It is organised
as sequential **phases** (checkpoints), each a self-contained folder. This branch (`lezione-8-fase-4`)
is the **Phase 4** starting point — participant does the work; solutions live on the parallel
`-soluzione` branches (see [README.md](README.md)).

Key consequence for an agent: **open and work inside a single phase folder, not the repo root.** The
phase folders are independent (each has its own `docker-compose.yml`, DB init, go.mod / php-app). Active
work for this branch is [phase-04-db/](phase-04-db/).

- `phase-01-monolith/`, `phase-03-skeleton/` — the PHP monolith (Lesson 7 context).
- `phase-02-analysis/` — DDD analysis docs, no code.
- `phase-04-db/` — the Go Warehouse service (this lesson).

## Phase 04 — the Go service ([phase-04-db/](phase-04-db/))

Module `warehouse.local/core`, Go 1.22, Echo v4 for HTTP, `go-sql-driver/mysql`. Clean architecture in
dependency order:

- `entities/` — domain: `Article` aggregate root (owns `InventoryLevel`s), value objects `SKU`, `Money`.
  Invariants enforced in constructors and mutators. Aggregates **record** `PendingEvent`s, they do not
  publish them.
- `events/` — past-tense domain event structs (`ArticleCreated`, `InventoryAdjusted`, …).
- `interfaces/repository.go` — the `ArticleRepository` **port**: aggregate-level ops only (Save,
  FindByID, FindBySKU, List, Delete). No per-entity methods.
- `repositories/` — adapters implementing the port:
  - `in_memory_article_repository.go` — test fake (green).
  - `article_repository.go` — BC adapter → `warehouse_db` (`price_cents BIGINT` + `currency CHAR(3)`).
  - `legacy_article_repository.go` — **ACL adapter** → `legacy_db` (`price DECIMAL(10,2)`, no currency).
    Translates legacy schema to the domain model; keeps legacy details out of the aggregate.
  - `dual_write_article_repository.go` — decorator: writes to **both** stores (legacy first, then BC),
    reads from **one** store chosen by `ReadMode` (`ReadFromLegacy` | `ReadFromBC`). Transitional per
    ADR-013, deleted after cutover.
- `main.go` — composition root. Wires both DBs from env vars, exposes only `/health` on :8081 (HTTP
  handlers arrive in Phase 06). Reads config from `LEGACY_DB_*`, `WAREHOUSE_DB_*`, `DUAL_WRITE_READ_MODE`.
- `cmd/seed/` — demo CLI that exercises the dual-write decorator against both real DBs (`write` / `read`
  / `compare` subcommands). Stand-in for the not-yet-existing HTTP layer.

### The participant's task (Phase 4)

Two starter files return sentinel TODO errors and must be completed:
- `legacy_article_repository.go` (`ErrLegacyRepositoryTODO`) — Task 1, the ACL adapter.
- `dual_write_article_repository.go` (`ErrDualWriteTODO`) — Task 2, the dual-write decorator.

The tests in `repositories/` are the spec; make them green. `entities/`, `events/`, the BC adapter, and
the in-memory fake are already green.

### Commands (run from `phase-04-db/`)

```bash
go test ./...                        # run the suite locally (needs Go toolchain)
go test ./repositories/ -run TestX   # single test / package
docker compose run --rm test         # whole suite green in a container (test profile)
docker compose up --build            # legacy MySQL + warehouse MySQL + Go app + Adminer (:8082)
docker compose down                  # stop (add -v to drop volumes)

# dual-write demo CLI (demo profile, needs the two MySQL containers up):
docker compose --profile demo run --build --rm seed write   demo-1 ABC-001 "Widget" 2999 EUR
docker compose --profile demo run --build --rm seed read    demo-1
docker compose --profile demo run --build --rm seed compare demo-1
```

Ports: app `8081`, legacy MySQL `3306`, warehouse MySQL `3307`, Adminer `8082`. Before starting Phase 04
containers, shut down other phases' stacks to avoid port clashes.

## Reference

Architecture Decision Records in [docs/adr/](docs/adr/) drive the design (each has an `-IT.md` Italian
twin): Strangler Fig (001), Clean Architecture layers (002), MIC PHP baseline (006), Go clean
architecture (007), Event Storming (010), DDD building blocks (011), Dual-Write (013). When asked *why* a
pattern is shaped a certain way, cite the ADR rather than guessing.

Docs come in English + Italian (`README.md` / `README-IT.md`); keep both in sync when editing prose.
