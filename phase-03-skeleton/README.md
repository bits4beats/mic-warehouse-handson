# Phase 03 — Warehouse BC: the Go domain layer

> Italian version: [`README-IT.md`](./README-IT.md)

```
   ___       ___ _        _      _
  / __|___  / __| |_  ___| | ___| |_ ___ _ _
 | (_ / _ \ \__ \ / // -_) |/ -_)  _/ _ \ ' \
  \___\___/ |___/_\_\\___|_|\___|\__\___/_||_|

  Phase 03 — Where the Warehouse domain meets Go.
```

Estimated time: ~1h build in breakout rooms, then a shared restitution.

```text
CP1    You understood MIC (the legacy monolith)
CP2    We chose the Warehouse BC to extract
CP3    You build its Go domain layer   <-- you are here
CP4+   Persistence, dual-write, HTTP, ...
```

---

## Your mission

In CP1 you mapped the MIC monolith; in CP2 we picked the **Warehouse** Bounded Context to extract.
Now write the **first code of the new service**: its **domain layer** in Go.

This is the business heart of the BC, and nothing else yet: **no database, no HTTP API, no
framework**. Just the types that model the Warehouse domain and the rules they must always enforce.
Persistence and the HTTP surface arrive in the next lessons; if you build them now you are building
the wrong thing.

You have an **AI coding agent**. Use it to write the code with you, but you own the design decisions
and, above all, the invariants. There is **no skeleton in the repo to read and no test suite to
conform to**: the shape of the code (packages, types, representation) is yours to choose. The spec is
the behaviour below, not a particular Go structure.

---

## What you build (the target)

A Go module (suggested: `warehouse.local/core`, Go 1.22, Echo for the health endpoint), laid out as:

```
entities/     Article (aggregate root), SKU + Money (value objects), InventoryLevel (entity)
events/       the domain events + a DomainEvent interface
interfaces/   ArticleRepository (the persistence port — interface only, no implementation)
main.go       an Echo server exposing only GET /health
```

### The building blocks

| Block | Kind | Responsibility | Identity & equality |
|---|---|---|---|
| **Article** | Aggregate root | Owns its SKU, price, and inventory levels; the only object callers talk to | identity by `ID` |
| **SKU** | Value object | The article code | no identity; equality by value |
| **Money** | Value object | A price: integer cents + ISO-4217 currency | no identity; equality by value |
| **InventoryLevel** | Entity (inside Article) | Stock at one location: quantity + reserved | identity by `ID`, but loaded/saved only through Article |
| **ArticleCreated / InventoryAdjusted / StockReserved** | Domain events | Past-tense facts the aggregate records | n/a |

### The rules that must hold (your acceptance criteria)

- **Article** — `id` and `name` non-empty; **price > 0**; constructed through a factory that fails
  loud. A `ChangePrice` operation keeps the currency (a currency change is a separate migration
  flow, not this method) and **records a fact** rather than publishing it.
- **SKU** — matches `^[A-Z0-9-]{3,32}$` (uppercase letters, digits, hyphens; length 3–32). The
  factory rejects anything else.
- **Money** — amount in **cents ≥ 0**; currency is three uppercase letters (ISO-4217). The factory
  rejects bad input. (Note: `Money(0, "EUR")` is a valid Money, but an Article with a zero price is
  not — the stricter rule lives on the aggregate.)
- **InventoryLevel** — ids non-empty; quantity ≥ 0; reserved ≥ 0; **reserved ≤ quantity**. One per
  (article, location). Reached only through Article; it has **no repository of its own**.
- **Domain events** — past-tense, immutable facts. The aggregate **records** them on a pending list;
  a later layer (CP5) drains and publishes them. Do **not** publish or serialize here.
- **Repository** — exactly one port, `ArticleRepository`, with **aggregate-level operations only**
  (a `Save` that is idempotent for the same state, plus find/list/delete). **No
  `InventoryRepository`** — that would break the aggregate boundary.
- **Clean architecture** — the domain packages import no database driver and no web framework.
  `main.go` only boots the server and answers `GET /health` with `{"status":"ok"}` on `:8081`.

### How you'll know it's right

There is no prescribed test suite to satisfy — that would pin the internal shape of your code, and
the shape is yours. The **invariants above are the spec**. Prove they hold the way you would on a
real BC: write your own tests against your own API. A room is done when every invariant is covered
and green, on a design the room chose. The reference solution and its tests come out at the
restitution, after you have committed to a design.

### Also deliver: the decisions behind your code

The Go you write encodes architectural choices, most of them implicit: why **Article** is the only
aggregate root, why **SKU** and **Money** carry no identity, why there is a single repository port
and **no** `InventoryRepository`, why events are **recorded** and not published here. Make them
explicit so the code stops being a black box.

Produce a short **`design-decisions.md`** (half a page is enough). For each non-obvious choice in
your skeleton, state the **decision**, the **rule or boundary it protects**, the **alternative you
rejected**, and **how it lines up with the clean-architecture target** handed to you in CP2. At the
restitution you defend this design, not the syntax, and it is the thread the next lesson picks up
when the BC grows a real persistence layer.

> Prerequisites: Rancher Desktop with the Docker engine. Local Go (1.22+) is optional — tests can
> run inside the Compose `test` service.

You are **done** when the service boots and your own tests for every invariant are green:

```bash
cd phase-03-skeleton
docker compose up --build          # then: curl http://localhost:8081/health  → {"status":"ok"}
docker compose run --rm test       # runs your tests   (or, with local Go: go test ./... -v)
```

> **Free the ports first.** Phase 03 uses `8081` (app) and `3306` (MySQL), the same as Phase 01's
> stack. Run `docker compose down` in `phase-01-monolith/` (and `docker rm -f mic-facade` if it
> lingers) before starting here. Check with
> `docker ps --filter "publish=8081" --filter "publish=3306"` — expect no rows before `up`.

The `docker-compose.yml` already brings up a MySQL container even though `main.go` does not connect
to it: that is deliberate, Phase 04 will use the same file with no extra setup.

---

## Scope: stop at the domain

Build **only** the domain layer. These belong to later lessons — do **not** build them now:

| Not now | Arrives in |
|---|---|
| MySQL repository implementation + dual-write decorator | Lesson 8 · CP4 |
| Use cases + event dispatch | CP5 |
| HTTP `/articles` handlers | CP6 |
| Auth, policy, events-on-the-wire, observability | CP7–CP10 |

Two scope simplifications carried from CP2: **StockReservation** is just a `Reserved` counter on
`InventoryLevel` for now (promoted to its own entity later), and the **stock-movement ledger** stays
in the PHP monolith — it is not part of this Go BC.

---

## How to work

Small breakout rooms, your **AI coding agent** as your engine. Build the skeleton yourselves. We compare
skeletons at the restitution: did Article come out as the aggregate root? Do SKU and Money have no
identity? Is `reserved ≤ quantity` enforced? Did anyone leak persistence or HTTP into the domain?

### Optional reading

- [`ADR-011 — DDD building blocks`](../docs/adr/ADR-011-ddd-building-blocks.md) — Aggregate, Entity, Value Object, Domain Event.
- [`ADR-007 — Go Clean Architecture`](../docs/adr/ADR-007-go-clean-architecture-implementation.md) — the layering this phase practices.

---

## Solutions

On the **`lezione-7`** branch (the one you clone) `phase-03-skeleton/` holds a copy of the MIC
monolith: that is what you extract the Warehouse domain from, and wiring up the Go build and Docker is
part of your job. The **reference solution** is the **`lezione-7-soluzione`** branch: the developed Go
domain skeleton (`entities/`, `events/`, `interfaces/`, `main.go`, plus the Go build and Compose
setup), with the design rationale in `solutions/reference-design.md`.

**Switch to the solution only after you have built your own.** The value of this phase is in
making the design decisions yourself, not in copying a target.

---

## What comes next

The next lesson gives your `ArticleRepository` its first real implementation (MySQL) plus a
**dual-write decorator** (legacy + new DB) and a divergence logger: the seam starts turning on. That
work lives in a later lesson, not in this repository.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
