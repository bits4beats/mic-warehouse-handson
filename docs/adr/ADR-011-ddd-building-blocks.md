# ADR-011: DDD Building Blocks as the Code Vocabulary

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

ADR-010 commits the team to Event Storming as the method for finding BCs. Event Storming produces aggregates and past-tense events. Once the canvas is drawn, that vocabulary must survive into the code, otherwise the workshop output is decoration.

The current Go skeleton (`phase-03-skeleton/entities/`) uses neutral nouns: `Article`, `Inventory`. Those names hide which object is the unit of change, which is identity-bearing, which is replaceable, and which represents an event. As a result, future maintainers cannot tell from a file name whether modifying a struct breaks an invariant or merely replaces a value.

Module 2.3 prescribes four building blocks. The repository must adopt them by name:

- **Aggregate** — cluster of related objects treated as one transactional unit. One aggregate root controls access. The aggregate enforces invariants.
- **Entity** — object with persistent identity. Two entities with the same data and different ids are different things.
- **Value Object** — object defined by its data alone. No id. Replaced rather than mutated.
- **Domain Event** — past-tense fact emitted by an aggregate. Immutable. Other BCs can subscribe.

## Decision

The Go code under `phase-03-skeleton/` and onward uses these four names directly:

- `entities/article.go` declares `Article` as the **aggregate root** of the Warehouse aggregate. Outside callers only ever talk to the root.
- `entities/sku.go` declares `SKU` as a **value object** (no id, equality by data, validated on construction).
- `entities/money.go` declares `Money` as a **value object**.
- `entities/inventory.go` declares `Inventory` as an **entity** within the aggregate (persistent identity, but accessed through `Article`).
- A new `events/` package declares one type per **domain event**: `ArticleCreated`, `InventoryAdjusted`, `StockReserved`. Each event has a struct, a past-tense name, and an `OccurredAt time.Time` field.

Repositories load and save aggregates only. There is no `SKURepository` or `MoneyRepository` — value objects are persisted as part of the aggregate they belong to.

Use cases mutate aggregates and return `(Article, []DomainEvent)` so that the HTTP layer (or an event publisher) can fan events out. Aggregates do not publish events themselves; they record them.

## Consequences

**Positive:**
- A reader of the code can answer "what is this thing?" from the file name alone.
- The link from Event Storming canvas → code is direct: every orange sticky note becomes a struct in `events/`.
- Refactoring is constrained: a value object cannot grow an id without becoming an entity, which is a code review signal.

**Negative:**
- The team must learn four new terms. Mitigation: ADR-011 defines the building blocks; `phase-02-analysis/tactical-ddd-warehouse.md` hosts the worked Warehouse mapping, and `phase-02-analysis/ubiquitous-language.md` is the canonical business vocabulary.
- Renaming existing entities touches every phase from 03 onward. Mitigation: that refactor is sequenced as Stream B Plan 2.

## Implementation Reference

- `phase-02-analysis/ubiquitous-language.md` — canonical Warehouse business vocabulary (strategic DDD).
- `phase-02-analysis/tactical-ddd-warehouse.md` — DDD tactical building blocks (Aggregate / Entity / VO / Domain Event) applied to Warehouse.
- `phase-03-skeleton/entities/` — aggregate root + value objects *(planned — Stream B Plan 2)*.
- ADR-014 (Data Products) consumes domain events as published facts.
