# Phase 03 — Reference design (solution)

> ⚠ **Spoiler.** This is the shape a good Warehouse domain skeleton converges on, plus the rationale.
> Open it **only after** you have built your own. The value of CP3 is in making these design
> decisions yourself.

This is also the **facilitator reference** for the CP3 restitution: the set of decisions a room
should land on, and the questions to probe. The full reference **implementation** is the Go code in
`entities/`, `events/`, and `interfaces/` (the arrival point).

---

## Module & layout

`warehouse.local/core` (Go 1.22, Echo v4 for the health endpoint).

```
entities/    article.go (aggregate root) · sku.go · money.go (value objects) · inventory.go (entity)
events/      events.go (DomainEvent interface + ArticleCreated / InventoryAdjusted / StockReserved)
interfaces/  repository.go (ArticleRepository — interface only)
main.go      Echo server, GET /health → {"status":"ok"} on :8081
```

## Building blocks & invariants

| Block | Construct via | Invariants | Why |
|---|---|---|---|
| **Article** (aggregate root) | `NewArticle(id, sku, name, description, price)` | `id` non-empty; `name` non-empty (trimmed); `price.AmountCents > 0` | The root guards the whole boundary; callers never touch inventory directly |
| **SKU** (value object) | `NewSKU(code)` | matches `^[A-Z0-9-]{3,32}$` | A code is defined entirely by its value: no identity, equality by value |
| **Money** (value object) | `NewMoney(amountCents, currency)` | `amountCents >= 0`; currency `^[A-Z]{3}$` (ISO-4217) | Integer cents avoid float errors; equality by value+currency |
| **InventoryLevel** (entity) | `NewInventoryLevel(id, articleID, locationCode, quantity, reserved)` | ids non-empty; `quantity >= 0`; `reserved >= 0`; **`reserved <= quantity`** | Has identity (one per article+location) but is owned by Article: no repository of its own |

**Article behaviour:**
- `ChangePrice(newPrice)` — rejects non-positive price; **rejects currency change** (currency
  migration is a separate flow); no-op if equal; otherwise mutates and **records** a private
  `ArticlePriceChanged` fact.
- `PendingEvents()` returns a **defensive copy**; `ClearPendingEvents()` drains. The aggregate
  **records, does not publish** — the publishing layer arrives in CP5. (ADR-011.)
- `Money(0, "EUR")` is a valid Money, but a zero-price Article is not — the stricter rule lives on
  the aggregate, not the value object.

## Domain events (`events/`)

`DomainEvent` interface = `Name() string` + `OccurredAt() time.Time`. Three past-tense facts:

- `ArticleCreated` (ArticleID, SKU, **ArticleName**, PriceCents, Currency, At) — the field is
  `ArticleName`, not `Name`, to avoid colliding with the `Name()` method.
- `InventoryAdjusted` (ArticleID, LocationCode, Delta, NewQuantity, Reason, At).
- `StockReserved` (ArticleID, LocationCode, Quantity, ReservationID, OrderID, At).

## Repository port (`interfaces/`)

One interface, **aggregate-only**:

```
ArticleRepository:
  Save(ctx, *Article) error        // idempotent for the same aggregate state
  FindByID(ctx, id) (*Article, error)
  FindBySKU(ctx, skuCode) (*Article, error)
  List(ctx) ([]*Article, error)    // admin/back-office only
  Delete(ctx, id) error
```

There is deliberately **no `InventoryRepository`**: loading or saving an `InventoryLevel`
independently would let a caller put the aggregate into an invalid state.

## Reference tests (19) — NOT given to participants

These belong to the **reference solution**, not the participant path. They are deliberately withheld
because they pin the internal structure (package layout `entities`/`events`, exact type and field
names, constructor signatures, the cents-as-int64 representation via `AsDecimal`). Handing them over
would dictate the design instead of letting each room choose it. Participants write their own tests
against their own API; these come out at the restitution as the canonical reference.

- **SKU** (3): accepts valid, rejects invalid, equality by value.
- **Money** (6): valid, rejects negative amount, rejects empty currency, rejects invalid currency, equality by value, decimal view.
- **Article** (7): valid inputs, rejects empty id, rejects empty name, rejects zero price; `ChangePrice` records event, rejects zero, rejects currency mismatch.
- **Events** (3): `ArticleCreated`, `InventoryAdjusted`, `StockReserved` each satisfy `DomainEvent`.

(`InventoryLevel` has no reference test — a gap in the reference, not a participant task.)

## Restitution checklist

- [ ] **Article** is the aggregate root, and inventory is reached only through it (no free-standing inventory API).
- [ ] **SKU** and **Money** have no identity and compare by value; both are built through fail-loud factories.
- [ ] **`reserved <= quantity`** (and non-negative stock) is enforced at construction.
- [ ] Events are **recorded on the aggregate**, not published/serialized in this phase.
- [ ] Exactly one repository, for the **Article aggregate**; no `InventoryRepository`.
- [ ] No database driver or web framework imported into the domain packages; `main.go` only serves `/health`.

Common divergences to surface: an `InventoryRepository` (breaks the boundary); price as a float
(should be integer cents); SKU/Money carrying an `ID`; events published immediately instead of
recorded; validation in the caller instead of the factory.
