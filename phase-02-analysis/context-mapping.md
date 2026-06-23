# Context Mapping — Warehouse and its Neighbours

A Bounded Context never floats in isolation. It has neighbours, and the relationship between them follows a small set of named patterns. This document maps the Warehouse BC's relationships to the other BCs that share the legacy monolith.

The patterns come from Eric Evans' _DDD_. We split them by **intent**: who pays the cost of integration?

> 💡 **Why this matters.** The Ubiquitous Language ([`ubiquitous-language.md`](./ubiquitous-language.md)) is _ubiquitous_ only inside its BC. The moment two BCs need to exchange information, you cross a boundary where the two ULs could no longer agree on terms. Context mapping is how you document and choose **who translates** — and that decision constrains everything downstream (versioning, deprecation, schema design, deployment cadence).

---

## Pattern key

DDD context-mapping patterns fall into **three families**, organised by the kind of relationship the two BCs have. Read this first; the rest of the document assumes you can pick a pattern by name and family.

| Pattern                         | Family              | Who translates the model?                                                                                        | Coupling                | When to choose it                                                                                                                                        |
| ------------------------------- | ------------------- | ---------------------------------------------------------------------------------------------------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Partnership**                 | Cooperation         | Both teams jointly design and evolve the integration                                                             | 🟡 Medium               | Two teams whose success is mutually dependent. They share planning, releases and tests for the boundary.                                                 |
| **Shared Kernel**               | Cooperation         | Both teams share _a subset of the model_ (code, schemas, libraries) and co-own its evolution                     | 🔴 High                 | Two teams whose models genuinely overlap in a small, stable area. Cheap when stable, expensive when either side changes.                                 |
| **Customer/Supplier**           | Upstream/Downstream | Negotiated between the two teams. The downstream (customer) needs influence over the upstream (supplier) roadmap | 🟡 Medium               | The downstream has leverage (budget, criticality, contract). Both teams can succeed independently but cooperate explicitly.                              |
| **Conformist**                  | Upstream/Downstream | No one — the downstream accepts the upstream model **as is**                                                     | 🔴 High                 | The upstream is dominant. The downstream has _no real ability_ to influence it, so it conforms and pays the coupling cost.                               |
| **Anti-Corruption Layer (ACL)** | Upstream/Downstream | The **downstream** translates the upstream model into its own clean model                                        | 🟢 Low                  | The downstream wants to protect its model from the upstream's shape (legacy, vendor, foreign domain). The price is writing and maintaining a translator. |
| **Open Host Service (OHS)**     | Upstream/Downstream | The **upstream** publishes a stable protocol for many consumers                                                  | 🟢 Low on consumer side | The upstream wants to decouple its many consumers from its internal model by publishing a stable API contract.                                           |
| **Published Language**          | Upstream/Downstream | Both sides speak a _formalised shared dialect_ (often industry-standard)                                         | 🟢 Low after investment | Industry standards (EDIFACT, HL7, FHIR, ISO 20022) or a purpose-built common schema (Hermes Data Product schemas).                                       |
| **Separate Ways**               | No relationship     | Each team owns its own implementation; no integration                                                            | 🟢 Lowest               | The two BCs don't need to integrate. Cost-effective when the overlap is illusory.                                                                        |

> 💡 **Cooperation vs Upstream/Downstream.** In _cooperation_ both teams design the boundary together; in _upstream/downstream_ there is a power asymmetry — the upstream produces, the downstream consumes. **Conformist is the specialisation of Customer/Supplier where the downstream has lost (or never had) negotiation leverage.**

> 💡 **Mnemonics.**
> Partnership = _"we co-design"_ · Shared Kernel = _"we share a module"_ · Customer/Supplier = _"we negotiate"_ · Conformist = _"I adapt to you"_ · ACL = _"I put a buffer"_ · OHS = _"I publish a stable API"_ · Published Language = _"we both speak this dialect"_ · Separate Ways = _"we don't talk"_

### External canonical references

If you want to go deeper before continuing:

- **Eric Evans, _Domain-Driven Design: Tackling Complexity in the Heart of Software_** (2003). Primary source.
- **Vaughn Vernon, _Implementing Domain-Driven Design_** (2013).
- **Vlad Khononov, _Learning Domain-Driven Design_** (2021, O'Reilly).
- Free DDD Reference (Evans, 2015).
- Microsoft Learn: [DDD-Oriented Microservices](https://learn.microsoft.com/en-us/azure/architecture/microservices/model/domain-analysis) — the _Context maps_ section summarises the patterns with diagrams.

For the building blocks **inside** a BC (Aggregate, Entity, Value Object, Domain Event), see [`tactical-ddd-warehouse.md`](./tactical-ddd-warehouse.md) and [ADR-011](../docs/adr/ADR-011-ddd-building-blocks.md).

---

## The neighbours

The legacy monolith hosts at minimum the following candidate BCs. All currently share the `business_data` and `business_relations` tables:

- **Catalog** — articles in their merchandising sense (categories, descriptions, units of measure).
- **Pricing** — price lists, VAT rates, discounts.
- **Customers** — customer data.
- **Orders** — orders, order lines.
- **Invoicing** — invoices, credit notes, payments.
- **Warehouse** — articles _as stockable items_, warehouse locations, inventory levels — _the BC we are extracting_. Stock movements would also fit in this BC but are descoped (see [`tactical-ddd-warehouse.md`](./tactical-ddd-warehouse.md)).

For this exercise, **only Warehouse is extracted** into Go. The others remain in PHP as the _legacy host_ the Strangler Fig grows around.

> 💡 **The same noun in two BCs could be two different things.** Catalog's "Article" is a marketing entity (descriptions, categories, photos); Warehouse's "Article" is a stockable entity (SKU, inventory levels). Both are legitimate; both are different. The UL of each BC ([`ubiquitous-language.md`](./ubiquitous-language.md)) makes this explicit.

---

## Pattern 1 — Customer/Supplier: Orders → Warehouse

**Direction:** Orders depends on Warehouse.
**Family:** Upstream/Downstream.
**Power balance:** Symmetric — both teams can succeed independently, but coordinate explicitly.

Orders is a **Customer** of Warehouse's contract; Warehouse is the **Supplier**.

When Orders creates an order line it needs to:

1. Verify the article exists and retrieve its base price (variations for customers, IVA, discounts are in the Pricing BC).
2. Check available stock at a chosen location.
3. Reserve stock for the order.

In the legacy monolith Orders does this via direct table reads. After extraction, Orders calls Warehouse's API (or consumes its events). The contract — endpoints, event schemas, deprecation policy — is **negotiated** between the two teams.

What this means in practice:

- Warehouse **should listen** to Orders' needs. Extending an endpoint to add a missing field is fair game.
- Warehouse **must not** bend its model to fit Orders' implementation details. Adding an `order_total_so_far` field would be Orders-domain pollution.
- Orders **must not** scrape internal Warehouse tables; it goes through the contract.
- A breaking change requires **mutual agreement** and a deprecation window.

The contract artefacts live in:

- `phase-01-monolith/openapi.yaml` (Warehouse API contract — baseline).

> 💡 **Customer/Supplier vs Conformist:** in Customer/Supplier the downstream still **has** influence on the upstream's roadmap (planning sync, joint design sessions, contract reviews). The moment that influence disappears — the upstream stops returning calls, the team is too small to push back, the vendor is fixed — you slide into Conformist.

## Pattern 2 — Conformist: Invoicing → Catalog → Warehouse (indirect)

**Direction:** Invoicing reads from Catalog (description, IVA), which in turn reads from Warehouse (price, SKU).
**Family:** Upstream/Downstream.
**Power balance:** Asymmetric — Invoicing has _no leverage_ to renegotiate Catalog's data shape; Catalog has no real incentive to support Invoicing's needs.

> 💡 **Conformist is a _specialisation_ of Customer/Supplier where the power balance has tilted entirely upstream.** The downstream still consumes; the difference is that it cannot negotiate. The upstream just publishes the contract on its own terms and the downstream accepts it.

For this exercise, Conformist is **documented but does not require code**: Invoicing reads from Catalog through the legacy monolith's existing PHP code. We flag it as a relationship to monitor when (or if) Catalog is extracted.

If Invoicing were unwilling to accept Catalog's shape, the next step would be inserting an **Anti-Corruption Layer** (Pattern 3) between them — at Invoicing's cost.

## Pattern 3 — Anti-Corruption Layer: Warehouse Go BC ↔ legacy PHP fields

**Direction:** When the Go Warehouse BC reads from or writes to the legacy `business_data` schema during dual-write (ADR-013), it does **not** expose the legacy generic-column names internally.
**Family:** Upstream/Downstream — the legacy schema is the (effectively immutable) upstream; the Go BC is the downstream that protects its model.
**Power balance:** Asymmetric, **but** the downstream chooses to pay translation cost rather than conform.

The translation layer lives in later lessons _(planned, Plan 2B)_:

```text
legacy.business_data.amount_1   →   Article.Price = Money(amount_1 * 100, "EUR")
legacy.business_data.code       →   Article.SKU   = SKU(code)
legacy.business_data.payload    →   not exposed
```

The Anti-Corruption Layer gives the Go BC freedom to model its domain cleanly without inheriting the legacy schema's generic-column anti-pattern.

> 💡 **ACL vs Conformist:** facing the same legacy schema, _Conformist_ would adopt the generic columns (`amount_1`, `text_2`) as Go field names and live with them forever. _ACL_ refuses, builds a translator, and pays the maintenance cost in exchange for a clean domain model. The choice depends on team budget and how long you expect to coexist with the legacy.

## Patterns not used in this BC (yet)

The following patterns are part of the canonical map (above) but do not occur in the Warehouse extraction today. They are listed so a future maintainer can recognise them:

- **Partnership** — could emerge if the Warehouse team and the Orders team work jointly on a major feature (e.g., a backorder workflow that requires changes on both sides simultaneously). Today Orders and Warehouse are in a Customer/Supplier relationship, not a Partnership.
- **Shared Kernel** — would apply if Warehouse and Catalog co-owned a small library (e.g., a shared `Identifier` type). We do not share code across BCs in this exercise; each owns its model end-to-end.
- **Open Host Service (OHS)** — Warehouse's REST API in Phase 06 _resembles_ an OHS in spirit (stable URI shape, versioning, content negotiation), but with a single consumer (Orders) we do not formally promote it to OHS until a third consumer arrives.
- **Published Language** — Phase 09 introduces Hermes Data Product schemas (`article-dp-v*.json`). Those schemas are a published language for downstream subscribers; today they exist only as planned artifacts (see ADR-014).
- **Separate Ways** — applies to Warehouse vs Customers: the two BCs never need to talk directly. Customers data appears in Warehouse code only as an opaque `customerId` field inside `StockReserved` events.

---

## Context map (visualisation)

The diagram below shows **only** the relationships discussed in this document, with the pattern explicitly named on each edge. Read every edge as: _"BC A relates to BC B via pattern P; family F; coupling C"_.

```text
   ┌──────────────┐
   │  Invoicing   │
   └──────┬───────┘
          │  Conformist (Upstream/Downstream)
          │  Invoicing accepts Catalog's shape; no leverage.
          ↓
   ┌──────────────┐                          ┌────────────────┐
   │   Catalog    │                          │    Orders      │
   │ (in monolith)│                          │ (downstream of │
   │              │                          │   Warehouse)   │
   └──────┬───────┘                          └───────┬────────┘
          │                                          │
          │  reads SKU + base price                  │  Customer/Supplier
          │  (makes Invoicing's dependency           │  (Upstream/Downstream)
          │   on Warehouse transitive)               │  Negotiated contract.
          │                                          │
          └─────────────────┬────────────────────────┘
                            ↓
                    ┌────────────────┐         ┌────────────────────┐
                    │   Warehouse    │ ← ACL → │  legacy schema     │
                    │  (Go BC,       │         │  business_data,    │
                    │   Supplier)    │         │  business_relations│
                    └────────┬───────┘         └────────────────────┘
                             │
                             │  Open Host Service (planned)
                             │  Stable contract for many consumers.
                             ↓
                    ┌────────────────┐
                    │     Hermes     │   (Published Language —
                    │  Data Products │    Hermes DP schemas,
                    │  consumers     │    planned)
                    └────────────────┘
```

**Reading the diagram:**

- Direction of the arrow = direction of the dependency (consumer → provider).
- Label on the arrow = name of the pattern (with its family).
- Pattern 2 (Conformist, transitive) is rendered as a two-hop chain: `Invoicing → Catalog → Warehouse`. Invoicing has no direct link to Warehouse — it inherits the dependency through Catalog.
- The dashed-style ACL (ASCII: `← ACL →`) sits **between** the Go BC and the legacy schema because it is bidirectional during dual-write (ADR-013): the Go BC translates legacy fields on read and produces legacy-compatible writes during the migration window.

---

## What this document is for

When making any decision that crosses BC boundaries — a new endpoint, a new event, a schema change, a deprecation — open this map first. The pattern named on the relationship **constrains** the decision:

| If the relationship is…      | Then you should…                                                                              |
| ---------------------------- | --------------------------------------------------------------------------------------------- |
| **Partnership**              | Plan and release the change jointly. Pair-design the contract.                                |
| **Shared Kernel**            | Coordinate the kernel evolution; both sides ship the kernel update together.                  |
| **Customer/Supplier**        | Negotiate; respect a deprecation window; the downstream's voice matters.                      |
| **Conformist**               | Adapt; do not assume the upstream will change for you.                                        |
| **ACL**                      | Update the translator at the boundary; do not let the upstream language leak into your model. |
| **OHS / Published Language** | Version the public schema; document migration paths for consumers.                            |
| **Separate Ways**            | Don't add a dependency just because it would be convenient.                                   |

If you find yourself unable to classify the relationship, that _is_ the signal — re-storm with the other team to surface what's really going on.
