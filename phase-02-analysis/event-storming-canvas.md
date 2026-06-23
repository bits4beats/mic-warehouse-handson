# Event Storming Canvas — Warehouse Extraction from the MIC Monolith

> **Method:** ADR-010 (Event Storming). Three steps in this order: **enumerate → cluster → name**.
>
> **Domain narrative source:** the MIC monolith. Its `business_data` table holds 16+ entity types; this canvas extracts only those relevant to the Warehouse BC.

---

## What Event Storming is

> 💡 **Event Storming** — a low-tech workshop activity for a heterogeneous group of people to brainstorm and rapidly model a business process. A _tool and a methodology_ for **sharing domain knowledge across roles**.

The technique was introduced by Alberto Brandolini around 2013. Its premise is simple: the cheapest way to learn how a business actually works is to put the people who run it, build it, sell it and support it in the same room with a wall, a marker, and a stack of sticky notes — and ask them to name **every past-tense event** they can think of, in any order. The chaos is the point; the wall reveals the seams.

### Who should participate

Ideally a _heterogeneous_ group:

- **Domain experts** (operations, sales, support, finance, ...) — they know what actually happens.
- **Engineers** — they will turn the narrative into code, so they need to hear it first-hand.
- **Product owners** / business analysts — they translate strategy into features.
- **QA / testers** — they will trace the events as test scenarios.
- **UI/UX designers** — the events surface the screens and workflows.
- Sometimes **legal**, **finance**, **security** — when the domain has constraints they own.

The more diverse the group, the richer the canvas. **Avoid groups larger than ~10 people** — at that size individual voices get drowned out and the workshop converts back into a meeting. For bigger orgs, run multiple parallel storms on slices of the business.

### Why it works

1. **Past-tense events are unambiguous.** _"Order placed"_ is something that either happened or didn't — there's no room for opinion. Nouns and processes ("management of the cart") collapse into discussion; events keep the conversation grounded.
2. **The wall is the bottleneck.** When two people argue about whether _"Customer registered"_ and _"Account created"_ are the same event, that argument **is the work**. Doing it at a wall (cheap) is far better than discovering it in production (expensive).
3. **No technology constraints.** No database schema, no UML, no class diagrams. The narrative is in the language of the business — which is also the seed of the Ubiquitous Language ([`ubiquitous-language.md`](./ubiquitous-language.md)).

### What it produces

Two artefacts:

- A _timeline_ of past-tense **domain events**.
- A clustering of those events into **candidate Bounded Contexts**.

The wall is intentionally low-fidelity. It is **not** a UML diagram and should not be promoted into one. Photographing the wall is the closest you get to _documenting_ an Event Storming session.

---

## Event Storming, the Ubiquitous Language, and Bounded Contexts

Event Storming is also where the **Ubiquitous Language** ([`ubiquitous-language.md`](./ubiquitous-language.md)) is first surfaced. The sticky notes on the wall are written _in the language the business uses_ — they don't say `OrderEntity.create()`, they say _"the order was placed"_. By the time you cluster events and draw BC boundaries, you already have a candidate vocabulary for each BC.

> 💡 **A Bounded Context defines the applicability of a Ubiquitous Language and of the model it carries.** When you draw a circle around a cluster and call it "Warehouse", you are also declaring: _the words on these sticky notes mean what they mean **only inside this circle**._ Outside the circle, the same word may mean something else (Catalog's "Article" ≠ Warehouse's "Article"). This is why defining the **scope** of a UL — i.e., the BC boundary — is itself a strategic design decision.

The UL is then **refined** after the workshop:

1. Remove synonyms (the wall surfaces them; the refinement picks one).
2. Define each term in one line (definition is also a way to test whether you understood it).
3. Assign each term to BC.
4. Validate against the code being shipped (Phase 03 onwards).

Bounded Contexts surfaced by Event Storming also help enforce **loose coupling externally**: if Orders and Warehouse are separate BCs, every cross-BC interaction goes through an _explicit_ contract ([`context-mapping.md`](./context-mapping.md)) rather than a casual table read.

---

## Step 1 — Enumerate (events, past-tense, no filtering)

The following events were observed by walking through the existing PHP controllers and reading the seed data of the MIC monolith. They are deliberately granular — Step 1 is the cheapest insurance against bad boundaries.

```text
ArticleCreated
ArticleRenamed
ArticlePriceChanged
ArticleDescriptionChanged
ArticleSuspended
ArticleReactivated
ArticleDecommissioned

WarehouseRegistered
WarehouseRenamed
WarehouseDecommissioned

InventoryStocked
InventoryAdjustedManually
InventoryReplenished
InventoryDepleted
InventoryTransferred

StockReserved
StockReservationConfirmed
StockReservationReleased
StockReservationExpired

StockMovementRecorded
StockMovementCorrected

ListinoPublished
ListinoVoceAdded
ListinoVoceRepriced

OrderPlaced
OrderConfirmed
OrderItemAdded
OrderItemRemoved
OrderShipped
OrderCancelled

InvoiceIssued
InvoiceSent
InvoicePaid
InvoiceVoided

CustomerRegistered
CustomerAddressChanged
```

**Total: 36 events.** Module 2.3 says aim for 15–25 — we're well over because the MIC narrative covers six BCs at once; only Cluster C is in scope for extraction.

> 💡 **Don't filter at Step 1.** The temptation is to _"clean up"_ the list as you go — combining synonyms, dropping events that _"don't seem important"_, renaming sloppy past-tense forms. Resist it. The filtering step has its own name: **Step 2 (cluster)**. Doing them at the same time merges two different conversations and you lose the artefacts that show _which_ synonyms were in tension.

---

## Step 2 — Cluster (events that hang together)

Group events that share an aggregate's lifecycle. Each cluster is a candidate Bounded Context.

### Cluster A — Catalog

- `ArticleCreated`
- `ArticleRenamed`
- `ArticleDescriptionChanged`
- `ArticleSuspended`
- `ArticleReactivated`
- `ArticleDecommissioned`

### Cluster B — Pricing

- `ArticlePriceChanged`
- `ListinoPublished`
- `ListinoVoceAdded`
- `ListinoVoceRepriced`

### Cluster C — Warehouse _(this is the BC we extract)_

- `WarehouseRegistered`
- `WarehouseRenamed`
- `WarehouseDecommissioned`
- `InventoryStocked`
- `InventoryAdjustedManually`
- `InventoryReplenished`
- `InventoryDepleted`
- `InventoryTransferred`
- `StockReserved`
- `StockReservationConfirmed`
- `StockReservationReleased`
- `StockReservationExpired`
- `StockMovementRecorded`
- `StockMovementCorrected`

### Cluster D — Orders

- `OrderPlaced`
- `OrderConfirmed`
- `OrderItemAdded`
- `OrderItemRemoved`
- `OrderShipped`
- `OrderCancelled`

### Cluster E — Invoicing

- `InvoiceIssued`
- `InvoiceSent`
- `InvoicePaid`
- `InvoiceVoided`

### Cluster F — Customers

- `CustomerRegistered`
- `CustomerAddressChanged`

---

## Step 3 — Name (each cluster becomes a BC candidate)

| Cluster | BC name       | Responsibility                                                                                        | Events owned |
| ------- | ------------- | ----------------------------------------------------------------------------------------------------- | ------------ |
| A       | Catalog       | Article identity, naming, lifecycle                                                                   | 6            |
| B       | Pricing       | List prices, listini, repricing                                                                       | 4            |
| C       | **Warehouse** | Physical inventory, stock reservations _(reservations and movements partially descoped — see Step 4)_ | 14           |
| D       | Orders        | Customer orders lifecycle                                                                             | 6            |
| E       | Invoicing     | Invoice issuance and payment                                                                          | 4            |
| F       | Customers     | Customer anagrafica                                                                                   | 2            |

Six BCs fall out of the canvas. **Warehouse is the heaviest** at 14 events, which is one reason we extract it first: its operational complexity (stock reservations, movements) deserves its own deployment lifecycle.

> 💡 **Naming comes _last_.** You can spot a team that storms in the wrong order: they begin with _"let's design the Catalog service"_ and then look for events that fit. The result is the structure they already imagined — not the structure the business actually exhibits.

---

## Step 4 (bonus) — Aggregate decomposition inside Warehouse

Within Cluster C, three conceptual aggregates emerge:

1. **Article (Warehouse perspective)** — `ArticleCreated` is _not_ in this cluster (it's Catalog's). Warehouse holds a _projection_ of Article (id, sku, name, price snapshot) plus its own state. Events that the Warehouse-side Article aggregate emits:
   - `InventoryAdjusted` (covers stocked / replenished / depleted / adjusted manually)
   - `InventoryTransferred`

2. **StockReservation** — its own aggregate because reservations have an independent lifecycle from inventory levels:
   - `StockReserved`
   - `StockReservationConfirmed`
   - `StockReservationReleased`
   - `StockReservationExpired`

3. **StockMovement** — append-only movement ledger:
   - `StockMovementRecorded`
   - `StockMovementCorrected`

### What this exercise actually implements

For the 20-hour exercise budget we make two deliberate simplifications:

- **Reservations are folded into the `Article` aggregate** as the `Reserved` counter on `InventoryLevel` (Phase 03–04), then promoted to a full entity in Phase 05+. They are not promoted to a separate aggregate.
- **The StockMovement ledger is descoped entirely.** It remains in the PHP monolith and is not represented in the Go BC. `InventoryAdjusted` already records the most common case (a stock level changed); a fully-fledged movement ledger can be reconstructed from the event stream later.

The net result: **one Go aggregate (`Article`) with one inner entity (`InventoryLevel`)**. The Warehouse BC therefore _includes_ reservations (modelled as an integer counter, later an entity) but _excludes_ the stock-movement ledger.

This is documented in:

- [`ubiquitous-language.md`](./ubiquitous-language.md) — `StockReservation` is marked 🚧 partial; `StockMovement` is not in the UL at all.
- [`tactical-ddd-warehouse.md`](./tactical-ddd-warehouse.md) — the _"What is not in the Warehouse BC"_ section confirms StockMovement is out of scope.

The simplification is reversible: a future release can split the aggregate and add the movement ledger when load demands it.

---

## What this canvas is for

If anyone proposes a Warehouse feature whose triggering event isn't in Cluster C, push back: it likely belongs to a different BC. If the event is in Cluster C but doesn't fit the `Article` aggregate above, the aggregate model is wrong — re-storm.
