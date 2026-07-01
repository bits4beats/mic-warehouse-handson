# Phase 04 — Warehouse BC: adapters & dual-write

> Italian version: [`README-IT.md`](./README-IT.md)

```
   ___  _                          ___ _ __ _  ___ _   _
  / __|| | ___  __ _ _ _      ___ / __| '__| |/ __| | | |
 | (__ | |/ -_)/ _` | ' \    /___|\__ \ |  | | (__| |_| |
  \___||_|\___|\__,_|_||_|        |___/_|  |_|\___|\___/

  Phase 04 — Where the BC meets the legacy database.
```

Estimated time: ~1h build in breakout rooms, then a shared restitution.

```text
CP1    You mapped MIC (the legacy monolith)
CP2    We chose the Warehouse BC to extract
CP3    You built its Go domain layer + repository port
CP4    You give that port real adapters and run them beside legacy   <-- you are here
CP5+   Use cases + dispatcher + HTTP + ...
```

> **Plenary opening (before you start):** we walk the canonical CP3 domain together
> ([`reference-design.md`](../phase-03-skeleton/solutions/reference-design.md)) and introduce dual-write
> and the Anti-Corruption Layer. Then you build.

---

## Your mission

CP3 gave the Warehouse BC a repository **port** — `ArticleRepository`, a contract with five methods and
**no implementation**. Now the BC must run **beside the legacy database**: every write lands in *both*
stores, so nothing breaks while traffic is still on the monolith — without leaking the legacy schema into
the clean domain.

**You write two files. Everything else is given.**

```
repositories/
  article_repository.go              MySQLArticleRepository       ← GIVEN (worked example, BC schema)
  in_memory_article_repository.go    InMemoryArticleRepository    ← GIVEN (test fake)
  legacy_article_repository.go       LegacyMySQLArticleRepository → TASK 1  (the legacy ACL adapter)
  dual_write_article_repository.go   DualWriteArticleRepository   → TASK 2  (dual-write / single-read)
```

The test suite ships **red on purpose**: the two starters return `TODO` errors and their tests fail. Your
job is to make them green. (`entities/`, `events/`, the BC adapter and the fake are already green.)

### Why an ACL — the two stores

The two databases model the same thing differently, on purpose:

| | `legacy_db` (3306) | `warehouse_db` (3307) |
|---|---|---|
| price | `price DECIMAL(10,2)` — `29.99` | `price_cents BIGINT` + `currency CHAR(3)` — `2999`, `EUR` |
| currency | none — assume `EUR` | explicit column |

An **Anti-Corruption Layer** is an adapter that *contains the legacy mess*: it converts `29.99 ↔ 2999`,
invents `currency = EUR` on read, and drops it on write — so `entities.Article` always gets a proper
`Money`, and the decorator never learns a legacy detail. That is Task 1.

---

## The consegna

### ▸ Task 1 — `repositories/legacy_article_repository.go` (the ACL adapter)

Implement the five port methods against `legacy_db`, **plus the two price-conversion helpers**:

- **Save** — idempotent upsert (`INSERT ... ON DUPLICATE KEY UPDATE`); `Money.AmountCents → "29.99"`; drop currency.
- **FindByID / FindBySKU** — `SELECT`, then rebuild the `Article`: DECIMAL → cents, default `EUR`, rebuild
  `SKU`/`Money` through their **domain factories**; `ErrArticleNotFound` when there is no row.
- **List** — every legacy row, rebuilt the same way.
- **Delete** — `ErrArticleNotFound` when no row was deleted (check `RowsAffected`).
- **No `float64`** — money is integer cents + DECIMAL strings; do it in `centsToDecimal` / `decimalToCents`.
- **`Article.Inventories` is out of scope** — persist only the article row.

✅ **Green when:** `repositories/legacy_conversions_test.go` passes (pure functions, no database needed).

### ▸ Task 2 — `repositories/dual_write_article_repository.go` (the decorator)

Implement the wrapper that runs the two adapters side-by-side. From the outside it *is* an
`ArticleRepository`; inside it holds the legacy + BC adapters:

- **Save / Delete** — write to **both**, **legacy first**. Legacy fails → return, don't touch BC. BC fails
  → return the error (legacy keeps the article; no rollback in this exercise).
- **FindByID / FindBySKU / List** — **single read**, routed to one store by mode (`ReadFromLegacy` default
  → `ReadFromBC` after cutover). No comparison, no reconciliation.

✅ **Green when:** `repositories/dual_write_article_repository_test.go` passes (in-memory, no database).

### ▸ Done when

```bash
docker compose run --rm test                                                    # whole suite green
docker compose --profile demo run --build --rm seed write   demo-1 ABC-001 "Widget" 2999 EUR
docker compose --profile demo run --build --rm seed read    demo-1
docker compose --profile demo run --build --rm seed compare demo-1
```

`seed write` prints `✓ legacy write OK` / `✓ BC write OK` with a confirmed read-back, `seed read` returns
`price=2999 EUR`, and `seed compare` prints `OK: stores aligned`. (No API to `curl` yet — HTTP is CP6; the
`seed` CLI runs the same `dual.Save` / `dual.FindByID` a handler will.)

---

## Run

> Prerequisites: Rancher Desktop with the Docker engine and `curl`. Local Go (1.22+) is optional.

**Step 0 — free the ports** (Phase 04 binds 8081, 8082, 3306, 3307; stop the previous phase):

```bash
cd ../phase-01-monolith && docker compose down && docker rm -f mic-facade 2>/dev/null || true
cd ../phase-03-skeleton && docker compose down
docker ps --filter "publish=3306" --filter "publish=3307" --filter "publish=8081" --filter "publish=8082"  # expect no rows
```

**Step 1 — start the dual stack:**

```bash
cd phase-04-db
docker compose up --build              # two MySQL containers + Go app + Adminer
curl -s http://localhost:8081/health   # {"status":"ok","mode":"legacy"}
```

Before coding, open **http://localhost:8082** (Adminer) and look at `articles` in both databases — server
`legacy-mysql` (`legacy_user`/`legacy_pass`/`legacy_db`) and `warehouse-mysql`
(`warehouse_user`/`warehouse_pass`/`warehouse_db`). Same concept, different shape: that's the gap your ACL
crosses. Shut down with `docker compose down` (add `-v` to drop volumes).

---

## Scope & how to work

Build **only** the two starters. Don't touch `entities/`, `events/`, `interfaces/repository.go`, the BC
adapter, or the in-memory fake — and don't build use cases (CP5), HTTP (CP6), or auth/policy/events
(CP7–CP10). If the AI proposes editing the domain or the port, **stop** — it's the wrong layer.

Work in a tight loop with your AI agent: **small prompt → small diff → run the matching test → interpret**.
Do **Task 1 before Task 2**, conversions first (their tests are instant). Keep each diff confined to the
file you're completing (`git diff`).

### Two hints while you work

> 💡 **Watch the database with Adminer.** Keep `http://localhost:8082` open while you build. After each
> `seed write` or `seed compare`, refresh `articles` in *both* stores and see what your adapter actually
> wrote: `legacy_db.price = 29.99` next to `warehouse_db.price_cents = 2999, currency = EUR`. The database
> is the ground truth; the terminal only summarizes it.

> 💡 **Turn Go into a language you know.** You do not need to be fluent in Go. When a snippet is opaque,
> ask the AI for an *equivalent* in a language you know (C#, Java, TypeScript, or plain pseudocode):
> *"translate this Go function to C#, keep the behaviour, do not change the design."* Understand it there,
> then come back to the Go.

### Optional reading

- [`ADR-013 — Dual-Write`](../docs/adr/ADR-013-dual-write-pattern.md) — write order and read modes (this
  exercise keeps dual-write + single read; divergence detection is out of scope).
- [`context-mapping.md` §"Pattern 3 — ACL"](../phase-02-analysis/context-mapping.md) — the Anti-Corruption Layer.

---

## Solutions

Reference implementations: [`solutions/legacy_article_repository.expected.go.txt`](./solutions/legacy_article_repository.expected.go.txt)
(Task 1), [`solutions/dual_write_article_repository.expected.go.txt`](./solutions/dual_write_article_repository.expected.go.txt)
(Task 2), walkthrough in [`solutions/README.md`](./solutions/README.md). **Open them only after your suite
is green** — use as a checklist, not a copy source.

---

## Next phase

→ **Phase 05** adds the **use cases** layer on top of the repositories you just built:
`CreateArticleUseCase`, `AdjustInventoryUseCase`, and an event dispatcher.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
