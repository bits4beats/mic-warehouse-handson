# Phase 04 — Solutions

> Italian version: [`README-IT.md`](./README-IT.md)

> ⚠ **Spoiler.** Worked answers for the two coding tasks in [`../README.md`](../README.md).
> Open this **only after your test suite is green**. The value of the phase is in writing the
> adapters and the decorator yourself, not in reading the target.

The two reference implementations are the `.go.txt` files next to this README (the `.txt` extension
keeps Go from compiling them):

- **Task 1** → [`legacy_article_repository.expected.go.txt`](./legacy_article_repository.expected.go.txt)
- **Task 2** → [`dual_write_article_repository.expected.go.txt`](./dual_write_article_repository.expected.go.txt)

Each section below uses the same three blocks: **🎯 TL;DR · 🧠 Reasoning · 🔬 Verification.**

---

## Task 1 — `LegacyMySQLArticleRepository` (the ACL adapter)

### 🎯 TL;DR

Implement the five `ArticleRepository` methods against `legacy_db`, plus the two price-conversion
helpers. The adapter is an **Anti-Corruption Layer**: it converts `Money` (cents + currency) to/from the
legacy `price DECIMAL` column, **invents `EUR` on read**, **drops currency on write**, and rehydrates
`SKU`/`Money` through their domain factories — so the legacy schema never leaks into `entities.Article`.

### 🧠 Reasoning

The non-obvious decisions, each of which the reference solution encodes:

- **Idempotent `Save` via upsert.** `INSERT ... ON DUPLICATE KEY UPDATE` keeps the Phase-03 `Save`
  contract: calling it twice with the same aggregate state ends in the same row, whether it existed or
  not. No `SELECT`-then-branch race.
- **Conversion without `float64`.** `29.99` is not exactly representable as a float, and `29.99 * 100`
  can yield `2998.999…`. For money that's a bug. `centsToDecimal` / `decimalToCents` do integer/string
  math instead: split on the decimal point, pad/round the fractional part to two digits, guard overflow.
  The pure tests in `legacy_conversions_test.go` pin the exact behaviour (`2999 ↔ "29.99"`, `"1" → 100`,
  `"1.5" → 150`, round-trip).
- **Rehydration, not creation.** On read the adapter rebuilds the `Article` with its persisted `ID` and
  timestamps via a struct literal — it does **not** call `NewArticle` (that's the creation workflow for
  fresh input). But it **does** rebuild `SKU` and `Money` through `NewSKU` / `NewMoney`, because values
  crossing back into the domain must still be valid.
- **Currency is invented on read.** Legacy has no currency column, so the adapter defaults to `EUR`.
  That "filled gap" is the corruption being contained: the domain always gets a complete `Money`.
- **`Article.Inventories` is out of scope.** Only the `articles` row is persisted.
- **`Delete` checks `RowsAffected`.** Zero rows deleted → `ErrArticleNotFound`, not a silent success.

### 🔬 Verification

- Conversion tests go green: `docker compose run --rm test` (or the pure `centsToDecimal` /
  `decimalToCents` cases in `repositories/legacy_conversions_test.go`).
- End-to-end (after Task 2): `seed write demo-1 ABC-001 "Widget" 2999 EUR` then `seed compare demo-1`
  → `OK: stores aligned`; Adminer shows `legacy_db.articles.price = 29.99` and
  `warehouse_db.articles.price_cents = 2999, currency = EUR`.
- Checklist vs [`legacy_article_repository.expected.go.txt`](./legacy_article_repository.expected.go.txt):
  idempotent upsert · `2999 → "29.99"` · DECIMAL parsing without `float64` · factory rehydration ·
  `EUR` default on read · `ErrArticleNotFound` on missing row.

---

## Task 2 — `DualWriteArticleRepository` (dual-write / single-read decorator)

### 🎯 TL;DR

A **decorator** that, from the outside, *is* an `ArticleRepository`, but inside holds the legacy and BC
adapters. It **writes to both** stores (legacy first) and **reads from one**, chosen by a `ReadMode`
flag (`ReadFromLegacy` by default → `ReadFromBC` after cutover). No comparison, no reconciliation.

### 🧠 Reasoning

- **Write order is deliberate.** `Save`/`Delete` go **legacy first**: legacy is the system of record
  during migration. If legacy fails, BC is never touched and the error returns. If BC fails after legacy
  succeeded, the error returns and legacy keeps the value — there is **no rollback** in this exercise
  (rolling back a confirmed write is worse than a temporary gap; reconciliation is a later concern).
- **Single read by mode.** All three reads route to exactly one store. `ReadFromLegacy` is the safe
  default early in the migration; `ReadFromBC` is the post-cutover state. The mode is read from the
  `DUAL_WRITE_READ_MODE` env var in `main.go`, so the same binary moves through the migration with a
  restart, not a redeploy.
- **The decorator only works because the port is an interface.** Its two fields are typed
  `interfaces.ArticleRepository`, and it satisfies the same interface itself — that's what lets it stand
  in for one repository while delegating to two, and lets the tests wire two in-memory fakes in place of
  real MySQL. A concrete struct could not be substituted this way.

> **Scope note.** This exercise keeps the decorator to *dual-write + single-read*. The
> divergence-detection variant (read both, compare, log mismatches) is intentionally **out of scope** —
> see the scope decision in the lesson plan. ADR-013 still describes the fuller pattern.

### 🔬 Verification

- Decorator tests go green: the in-memory tests in
  `repositories/dual_write_article_repository_test.go` (writes hit both stores; legacy-fail aborts BC;
  BC-fail still keeps legacy; reads route by mode).
- End-to-end: `seed write` prints `✓ legacy write OK` / `✓ BC write OK` with a confirmed read-back;
  `seed read demo-1` returns `price=2999 EUR`.
- Checklist vs [`dual_write_article_repository.expected.go.txt`](./dual_write_article_repository.expected.go.txt):
  write-both-legacy-first · abort-BC-on-legacy-failure · read-one-by-mode · interface-typed collaborators.

---

## Common AI pitfalls (reject these diffs)

While completing either task, stop the AI if it proposes to:

- change `entities/`, `events/`, or `interfaces/repository.go` (the domain and the port are fixed);
- use `float64` anywhere in the price conversion;
- add a `currency` column to the legacy schema instead of defaulting `EUR` in the adapter;
- teach `DualWriteArticleRepository` about SQL or legacy-schema details (that belongs in the adapter);
- persist `Article.Inventories` (out of scope this phase);
- add auto-reconcile / auto-fix of divergences (out of scope; also wrong by ADR-013).
