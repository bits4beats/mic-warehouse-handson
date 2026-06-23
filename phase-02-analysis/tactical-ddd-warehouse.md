# Tactical DDD — Warehouse Bounded Context

> This file maps the four DDD tactical building blocks (Aggregate, Entity, Value Object, Domain Event — see [ADR-011](../docs/adr/ADR-011-ddd-building-blocks.md)) onto the concrete Warehouse domain.
>
> Use it as the **structural contract** for the Go code in phases 03 onward: any struct, method or event that touches Warehouse must be classifiable as one of the building blocks below.

---

## Where this fits in the DDD picture

DDD has two halves. The **strategic** half (often skipped, but actually really important) is about knowledge crunching and discovering the right boundaries: subdomains, Bounded Contexts, Ubiquitous Language, context mapping. The **tactical** half is about the building blocks _inside_ one BC: Aggregate, Entity, Value Object, Domain Event, Repository.

> 💡 **Common pitfall.** Teams jump straight to _"let's implement Aggregates and Value Objects"_ without the strategic work — Knowledge crunching, subdomain discovery, BC design, UL. The tactical patterns then become decoration: code that _looks_ DDD but does not solve the right problem. The strategic work is what makes the tactical decisions defensible.

This file presupposes the strategic work has been done:

- The Warehouse BC has been identified (Cluster C of the Event Storming, see [`event-storming-canvas.md`](./event-storming-canvas.md)).
- The Ubiquitous Language has been discovered (see [`ubiquitous-language.md`](./ubiquitous-language.md)).
- Relationships with neighbouring BCs are mapped (see [`context-mapping.md`](./context-mapping.md)).

If you read this file looking for _"what does this code do?"_, read [`ubiquitous-language.md`](./ubiquitous-language.md) first — that is the **business** vocabulary. This file translates that vocabulary into **structural** patterns.

> 📖 The patterns of integration **between** Bounded Contexts (Partnership, Shared Kernel, Customer/Supplier, Conformist, Anti-Corruption Layer, Open Host Service, Published Language, Separate Ways) are in [`context-mapping.md`](./context-mapping.md). They do not appear here because the boundaries below are all **internal** to the Warehouse BC.

---

## Aggregate

> 💡 **Aggregate.** A cluster of objects treated as a single unit of change. Has one **aggregate root** which is the only entry point for callers. The aggregate enforces **invariants** — business rules that must be true _at every moment_ of the aggregate's life.

| Aggregate   | Root      | What it owns                                                                                                                                                                                | Invariants                                                                                                                                                                      |
| ----------- | --------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Article** | `Article` | `SKU` (VO), `Money` (VO), `name` and `description` (primitives), zero-or-more `InventoryLevel` entities, the `Reserved` counters within them (precursor of `StockReservation` in Phase 05+) | `price > 0`, `SKU` matches `^[A-Z0-9-]{3,32}$`, `quantity ≥ 0`, `reserved ≥ 0`, `reserved ≤ quantity`, `name` is non-empty, `price.currency` does not change after construction |

The Article aggregate is the **only** aggregate exposed by Warehouse. Inventory levels and reservations live inside it; they are never persisted, fetched, or modified independently.

> 💡 **Why one aggregate, not three?** Event Storming Step 4 (see [`event-storming-canvas.md`](./event-storming-canvas.md)) surfaced three conceptual aggregates: Article, StockReservation, StockMovement. For the 20-hour extraction exercise we collapse them into a single `Article` aggregate. The boundary between "inventory level" and "stock reservation" is well-modelled as entities inside the same root, and the StockMovement ledger is **descoped entirely** from the Go BC (see _What is not in the Warehouse BC_ below).

---

## Entity

> 💡 **Entity.** A thing with persistent identity. Two entities with the same data and different `ID`s are **different** things. Entities are mutable; mutation goes **through the aggregate root**.

| Entity           | Identity  | Lifetime                                                                                   | Notes                                             |
| ---------------- | --------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------- |
| `Article`        | UUID `ID` | Created on first publication, never physically deleted (logical de-activation when needed) | Aggregate root.                                   |
| `InventoryLevel` | UUID `ID` | One per `(Article, WarehouseLocation)` pair                                                | Inner entity. Loaded and saved through `Article`. |

> 💡 **`StockReservation` is planned but not yet a Go entity.** Phase 03–04 model reservations as a non-negative `Reserved integer` field on `InventoryLevel`. Phase 05+ promotes it to a full entity. The UL ([`ubiquitous-language.md`](./ubiquitous-language.md)) defines the term fully; only the Go implementation is simplified in early phases.

---

## Value Object

> 💡 **Value Object.** A thing defined by its data alone. No identity. Two VOs with the same data are the **same** thing. Replaced rather than mutated. Constructed via a factory that **fails loudly** on invalid input.

| Value Object                                                                                  | Data                                   | Validation on construction                              |
| --------------------------------------------------------------------------------------------- | -------------------------------------- | ------------------------------------------------------- |
| `SKU`                                                                                         | `string Code`                          | Matches `^[A-Z0-9-]{3,32}$`                             |
| `Money`                                                                                       | `int64 AmountCents`, `string Currency` | `AmountCents ≥ 0`, `Currency` is a valid ISO 4217 code  |
| `WarehouseLocation` _(planned VO; today stored as `LocationCode` string in `InventoryLevel`)_ | `string Code`                          | Matches `^[A-Z]{2}-[A-Z0-9]{3,8}$` (e.g., `IT-MILANO1`) |
| `Quantity` _(conceptual; today modelled as `integer` field inside `InventoryLevel`)_          | `integer Value`                        | `Value ≥ 0`                                             |

The aggregate trusts that any VO instance it holds is valid by construction. Validity checks at use-sites are a smell.

> 💡 **Aggregates can tighten VO constraints.** `Money` accepts `AmountCents ≥ 0` (zero is a valid monetary amount), but the `Article` aggregate refuses `price.AmountCents = 0` (a zero-price article is not valid in our domain). VOs define what is _expressible_; aggregates define what is _acceptable in this context_.

---

## Domain Event

> 💡 **Domain Event.** A past-tense fact recorded by an aggregate when an important business thing has happened. Immutable once recorded. Other Bounded Contexts can subscribe.

Naming rule: `<Subject><PastVerb>` — e.g., `ArticleCreated`, _not_ `CreateArticle` (an imperative is a command, not an event) and _not_ `ArticleCreation` (a noun phrase is a description, not a fact).

| Event                      | Emitted when                                                | Payload (canonical fields)                                                        |
| -------------------------- | ----------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `ArticleCreated`           | A new `Article` is published                                | `articleId`, `sku`, `articleName`, `priceCents`, `currency`, `occurredAt`         |
| `ArticlePriceChanged`      | An `Article`'s price changes (currency unchanged)           | `articleId`, `oldPriceCents`, `newPriceCents`, `currency`, `occurredAt`           |
| `InventoryAdjusted`        | An `InventoryLevel.Quantity` changes (positive or negative) | `articleId`, `locationCode`, `delta`, `newQuantity`, `reason`, `occurredAt`       |
| `StockReserved`            | A reservation is created against an `InventoryLevel`        | `articleId`, `locationCode`, `quantity`, `reservationId`, `orderId`, `occurredAt` |
| `StockReservationReleased` | A reservation expires, is cancelled or confirmed            | `articleId`, `locationCode`, `quantity`, `reservationId`, `reason`, `occurredAt`  |

Events are **recorded** by the aggregate, not published. The use case (or repository) drains the aggregate's `pendingEvents` and publishes them after `Save` succeeds. This separation is why the aggregate is purely consistent and free of side effects beyond mutating itself.

---

## Mapping to PHP monolith fields (for migration)

The legacy monolith stores warehouse data in `business_data` (with `record_type='articolo'`) and `business_relations`, both with generic columns. The Anti-Corruption Layer in later lessons translates legacy fields into the Warehouse UL:

| Legacy column     | Legacy meaning                         | Maps to                                                       |
| ----------------- | -------------------------------------- | ------------------------------------------------------------- |
| `id`              | row id                                 | `Article.ID` (UUID derived from legacy id during dual-write)  |
| `code`            | SKU string                             | `Article.SKU.Code`                                            |
| `name`            | article name                           | `Article.Name`                                                |
| `description`     | description                            | `Article.Description`                                         |
| `amount_1`        | `prezzo_listino` (list price, decimal) | `Article.Price.AmountCents` (× 100)                           |
| `amount_2`        | `qta_minima` (min quantity)            | not migrated (out of Warehouse BC scope)                      |
| `text_1`          | `categoria`                            | not migrated (Catalog BC concern)                             |
| `text_2`          | `iva_default`                          | not migrated (Pricing BC concern)                             |
| `payload.unita`   | unit of measure                        | not migrated (Catalog BC concern)                             |
| `payload.peso_kg` | weight                                 | not migrated (Catalog BC concern)                             |
| `status`          | `attivo`/`sospeso`/`cessato`           | not migrated initially; reintroduced as a VO only when needed |

The legacy schema does **not** distinguish reserved-vs-available quantity; the Go BC introduces `InventoryLevel.Reserved` as a fresh concept.

---

## What is **not** in the Warehouse BC

These appear in the legacy monolith or the broader domain narrative but belong **elsewhere**:

| Out-of-scope concept                              | Belongs to                          | Why it's not Warehouse                                                              |
| ------------------------------------------------- | ----------------------------------- | ----------------------------------------------------------------------------------- |
| Categoria, IVA, listino prezzi                    | Catalog / Pricing                   | Categorisation, tax rules and multi-price tables are not stocking concerns.         |
| Stock movement ledger (_movimento di magazzino_) | _Deliberately descoped_ — see below | Conceptually a Warehouse-internal concern, but **not migrated** in this exercise.   |
| Customer, Order, Invoice                          | Customers / Orders / Invoicing      | They _consume_ Warehouse data via API or events; they are not Warehouse aggregates. |

> ℹ **About stock movements.** The legacy monolith hosts `record_type='movimento'` rows that record stock entrate/uscite. Event Storming surfaced two candidate events (`StockMovementRecorded`, `StockMovementCorrected`) that would normally live in a Warehouse-internal _StockMovement_ aggregate. **For this 20-hour extraction we descope them entirely** — they remain in the PHP monolith and are not represented in the Go BC. This is a deliberate simplification: the `InventoryAdjusted` event already covers the most common use case (a stock level changed), and the movement-ledger aspect can be reconstructed from the event stream when needed. See [`event-storming-canvas.md`](./event-storming-canvas.md) for the original three-aggregate sketch.

Keeping the BC small is deliberate: it makes the migration exercise tractable in the available phases (~20 hours) and surfaces the Strangler Fig pattern more clearly than a bigger extraction would.

---

## References

- [`ubiquitous-language.md`](./ubiquitous-language.md) — the strategic (business) vocabulary this tactical mapping implements
- [`event-storming-canvas.md`](./event-storming-canvas.md) — where the aggregates came from
- [`context-mapping.md`](./context-mapping.md) — relationships **between** BCs (the patterns above are all internal to Warehouse)
- [`docs/adr/ADR-011-ddd-building-blocks.md`](../docs/adr/ADR-011-ddd-building-blocks.md) — the architectural decision that the code uses these four block names
- `phase-03-skeleton/entities/` — concrete Go implementation of Article + SKU + Money + InventoryLevel
- `phase-03-skeleton/events/` — concrete Go implementation of the three currently-shipped events
