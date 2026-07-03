# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A hands-on teaching lab: extracting the **Warehouse** Bounded Context out of **MIC**, a legacy PHP
invoicing monolith, into a clean **Go** microservice, using the Strangler Fig pattern. It is organised as
sequential **phases** (checkpoints), each a self-contained folder. This branch (`lezione-8-fase-5`) is the
**Phase 5** starting point — participant does the work; solutions live on the parallel `-soluzione`
branches and, for Phase 5, also in [phase-05-usecases/solutions/](phase-05-usecases/solutions/) (see
[README.md](README.md)).

Key consequence for an agent: **open and work inside a single phase folder, not the repo root.** The phase
folders are independent (each has its own `docker-compose.yml`, `go.mod` / `php-app`). Active work for this
branch is [phase-05-usecases/](phase-05-usecases/).

- `phase-01-monolith/`, `phase-03-skeleton/` — the PHP monolith (Lesson 7 context).
- `phase-02-analysis/` — DDD analysis docs, no code.
- `phase-04-db/` — the persistence checkpoint (legacy ACL adapter + dual-write decorator; two MySQL DBs).
- `phase-05-usecases/` — **this lesson**: application use cases + a thin HTTP API. In-memory only.

## Phase 05 — use cases + HTTP API ([phase-05-usecases/](phase-05-usecases/))

Module `warehouse.local/core`, Go 1.22, Echo v4 for HTTP. **No database** — runs entirely on in-memory
adapters (persistence was Phase 4's job); data lives for the process run and resets on restart. Clean
architecture in dependency order (inner layers know nothing of outer ones):

- `entities/` — domain: `Article` aggregate root (owns `InventoryLevel`s), value objects `SKU`, `Money`.
  Invariants enforced in constructors/mutators. Aggregates **record** `PendingEvent`s; they do not
  dispatch them.
- `events/` — past-tense domain event structs (`ArticleCreated`, `ArticlePriceChanged`, …).
- `interfaces/repository.go` — the `ArticleRepository` **port**: aggregate-level ops only (Save, FindByID,
  FindBySKU, List, Delete). No per-entity methods.
- `dispatcher/` — the `EventDispatcher` **port** + `InMemoryDispatcher` fake. Aggregates record events; the
  **use case** calls `Dispatch` after `Save`; the dispatcher owns transport.
- `usecases/` — application workflows, one per file. Each `…UseCase.Execute` orchestrates: build/fetch
  aggregate → `Save` → `Dispatch` → `ClearPendingEvents`. `in_memory_article_repository.go` and `mocks.go`
  are the given test scaffolding.
- `handlers/` — HTTP layer. A handler does exactly **one** job: translate HTTP ↔ use case (bind DTO, call
  `Execute`, map result/status). `router.go` registers routes; `openapi.go` serves Swagger UI at `/docs`.
- `main.go` — composition root. Wires in-memory repo + dispatcher → use cases → handlers → Echo. Exposes
  `/health`, the article routes, and `/docs`. No env config.

### The slice pattern

Every request flows through one job per layer:

```
HTTP request -> handler (bind JSON, call use case, map result/status)
             -> use case (validate, ask aggregate, Save, Dispatch)
             -> aggregate (business rule) -> repository + dispatcher ports -> JSON response
```

`CreateArticle` (`POST /articles`) is the **given worked example**, wired end to end — read it first; it is
the exact shape every other slice follows.

### The participant's task (Phase 5)

Take one use case fully around the loop (**code → Go test green → handler → uncomment route → API test at
`/docs`**) before starting the next.

- **Loop A (everyone):** `GetArticleUseCase.Execute` in `usecases/get_article.go`; `GetArticle` handler
  (404 on `usecases.ErrArticleNotFound`, else 200); uncomment `GET /articles/:id` in `router.go`.
- **Loop B (optional):** `ChangeArticlePriceUseCase.Execute` (no-op if the price is unchanged); handler;
  uncomment `PUT /articles/:id/price`.

The `*_test.go` files in `usecases/` are the spec; make them green. Given code is already green.

### Rules that matter here

- **Save first, dispatch after; failures never dispatch.** If `Save` succeeds but `Dispatch` fails, state
  changed and the event is lost — a real gap left open on purpose (closed later by the outbox/Hermes work,
  CP9). Notice it; don't fix it in Phase 5.
- **Stay in your layer.** Do **not** touch `entities/`, `events/`, `interfaces/`, `dispatcher/`, or the
  given `handlers/openapi.go`. If the AI reaches into the domain or a port while completing a loop, stop —
  it's the wrong layer.
- The HTTP layer here is deliberately reduced: no auth/JWT, no middleware beyond logger+recover, no
  list/pagination, no dual-write/MySQL wiring. Those arrive in Phase 06.

### Commands (run from `phase-05-usecases/`)

```bash
go test ./...                              # run the suite locally (needs Go toolchain)
go test ./usecases/ -run TestGetArticle    # single test / package
docker compose run --rm test               # whole suite (test profile), verbose
docker compose up --build                  # serve the API + Swagger UI on :8081
docker compose down                        # stop
```

Test the live API from the browser at **http://localhost:8081/docs** (Swagger UI — Try it out → Execute).
`POST /articles` works from the start; `GET` and `PUT` light up as you build them. Port `8081` clashes with
Phase 04 — run `cd ../phase-04-db && docker compose down` first.

## Phase 04 — persistence ([phase-04-db/](phase-04-db/))

Previous checkpoint, included as context. Same module/domain, but with two real MySQL databases and a
`repositories/` layer of `ArticleRepository` adapters:

- `article_repository.go` — BC adapter → `warehouse_db` (`price_cents BIGINT` + `currency CHAR(3)`).
- `legacy_article_repository.go` — **ACL adapter** → `legacy_db` (`price DECIMAL(10,2)`, no currency).
  Translates the legacy schema to the domain model, keeping legacy details out of the aggregate.
- `dual_write_article_repository.go` — decorator: writes to **both** stores (legacy first, then BC), reads
  from **one** chosen by `ReadMode`. Transitional per ADR-013, deleted after cutover.
- `cmd/seed/` — demo CLI exercising the dual-write decorator (`write`/`read`/`compare`).

Commands run from `phase-04-db/`; `main.go` reads config from `LEGACY_DB_*`, `WAREHOUSE_DB_*`,
`DUAL_WRITE_READ_MODE`. Ports: app `8081`, legacy MySQL `3306`, warehouse MySQL `3307`, Adminer `8082`.

## Reference

Architecture Decision Records in [docs/adr/](docs/adr/) drive the design (each has an `-IT.md` Italian
twin): Strangler Fig (001), Clean Architecture layers (002), MIC PHP baseline (006), Go clean architecture
(007), Event Storming (010), DDD building blocks (011), Dual-Write (013), Data Products on Hermes (014).
When asked *why* a pattern is shaped a certain way, cite the ADR rather than guessing.

Docs come in English + Italian (`README.md` / `README-IT.md`); keep both in sync when editing prose.
