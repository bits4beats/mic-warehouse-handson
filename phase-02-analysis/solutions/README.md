# Phase 02 — Solutions

> ⚠ **Spoiler.** This directory contains worked answers and reasoning for the questions in [`../README.md`](../README.md)
> Open it **only after** you have attempted the questions yourself. The value of the exercise is in the friction, not in the answer.

> Italian version: [`README-IT.md`](./README-IT.md)

---

## Format

Every question follows the same three-block layout, used across all phases:

- **🎯 Answer (TL;DR)** — 1–2 sentences, the direct answer
- **🧠 Reasoning** — the _why_: concept reinforcement, edge cases, links to other artefacts
- **🔬 Verification** — pointers to the Phase 02 artefacts, ADRs and course slides that anchor the answer

---

## Step 1 — Event Storming

Reference questions in [`../README.md` Step 1](../README.md#step-1--read-the-event-storming-canvas-15-min).

### Q1 — Why does Step 1 (_enumerate_) forbid filtering?

**🎯 Answer (TL;DR)**

Filtering during enumeration mixes two different conversations (_what happened?_ vs _what matters?_) and the second drowns the first, so synonyms and edge-case events disappear before anyone can challenge them. You end up with the model the dominant voice in the room already imagined — not the one the business actually exhibits.

**🧠 Reasoning**

Event Storming has three steps for a reason. Each step is a different cognitive activity:

| Step         | Activity                                                       | What it produces                           |
| ------------ | -------------------------------------------------------------- | ------------------------------------------ |
| 1. Enumerate | _Divergent thinking_ — write everything, no judgement          | Raw list of past-tense events              |
| 2. Cluster   | _Convergent thinking_ — group, surface duplicates and tensions | Candidate aggregates and BCs               |
| 3. Name      | _Synthesis_ — give each cluster a label                        | BC candidates with one-line responsibility |

If you try to do all three at once, the room's loudest voices (often the senior architects or the most opinionated PM) start to _pre-cluster_: "_nah, `CustomerEmailChanged` is the same as `CustomerAddressChanged`_". That short-circuits the divergent step. The "_same as_" judgement might be correct — or it might be wrong because the legal team has constraints on PII changes the architect didn't know about.

> 💡 **The artefact you lose by skipping Step 1 is the tension between similar events.** That tension is _information about the domain_; deferring filtering preserves it.

Concrete failure mode in the Warehouse canvas: imagine someone had collapsed `InventoryStocked`, `InventoryReplenished` and `InventoryAdjustedManually` into a single `InventoryChanged` during enumeration. The Step 4 decomposition would have lost the distinction between _"warehouse received a delivery"_ (`InventoryReplenished`) and _"operator corrected an error"_ (`InventoryAdjustedManually`) — two very different business operations with different audit and finance implications.

**🔬 Verification**

- **Artefact:** [`event-storming-canvas.md`](../event-storming-canvas.md) - "Don't filter at Step 1"
- **ADR:** [ADR-010 — Event Storming methodology](../../docs/adr/ADR-010-event-storming-methodology.md)
- **Course slides:** Module 2.3, _Event Storming workshop_ — _"Step 1 = throw everything on the wall; resist the urge to clean up"_

---

## Step 2 — Ubiquitous Language

Reference questions in [`../README.md` Step 2](../README.md#step-2--read-the-ubiquitous-language-20-min).

### Q2 — Why is "Article" in Warehouse different from "Article" in Catalog?

**🎯 Answer (TL;DR)**

Because the _invariants_ and _responsibilities_ are different. Catalog's Article is a merchandising entity (categories, descriptions, photos); Warehouse's Article is a stockable entity (SKU, inventory levels). Same word, two BCs, two different concepts. The legacy `business_data` table doesn't distinguish them because it has no notion of BC boundaries — that is _the_ anti-pattern the extraction is meant to fix.

**🧠 Reasoning**

This is the canonical example of a **homonym across Bounded Contexts**. The Ubiquitous Language is _ubiquitous inside_ the BC; it does not extend across the boundary. _Within_ Warehouse, "Article" always means "a stockable item with a SKU and inventory levels". _Within_ Catalog, "Article" always means "a merchandising entry with a name, description and category".

| Property                   | Catalog's Article                | Warehouse's Article                       |
| -------------------------- | -------------------------------- | ----------------------------------------- |
| Identity carrier           | Internal `id`; SKU is metadata   | UUID `ID`; SKU is the business identifier |
| Invariant on price         | None (Pricing owns price)        | `price > 0`, currency stable              |
| Invariant on stock         | None (not tracked)               | `quantity ≥ 0`, `reserved ≤ quantity`     |
| Lifecycle events           | Created, Renamed, Decommissioned | InventoryAdjusted, StockReserved          |
| What changes invalidate it | Description, photo, category     | Price, inventory level, reservation count |

Two different things. The fact that the _string_ "Article" is used twice is fine — _as long as_ each BC keeps its own meaning explicit and the integration uses a context-mapping pattern that translates between them (in this exercise: Customer/Supplier for Orders↔Warehouse, Conformist for Invoicing↔Catalog).

> 💡 **A homonym across BCs is not a problem — pretending it doesn't exist is.** The UL discipline says: name the homonym, give each side its own definition, and translate at the boundary. The legacy `business_data` table fails this discipline; the Go BC enforces it.

**🔬 Verification**

- **Artefact:** [`ubiquitous-language.md`](../ubiquitous-language.md)
- **Mapping:** [`tactical-ddd-warehouse.md`](../tactical-ddd-warehouse.md)
- **Context Mapping:** [`context-mapping.md`](../context-mapping.md)

---

### Q3 — What dangers does allowing synonyms for a UL term expose the team to?

**🎯 Answer (TL;DR)**

Synonyms violate the UL's two core properties — _precise_ and _consistent_ (Evans, _DDD_ ch. 2) — and silently undo the knowledge crunching that produced the vocabulary in the first place. Concrete dangers: (1) communication between domain experts and engineers degrades, because different people use different words for the same concept without realising it; (2) the team's negotiated agreement on _one canonical term per concept_ is lost without anyone deciding to lose it; (3) Bounded Context boundaries blur because synonyms often leak in from neighbouring BCs; (4) in a bilingual setup, the negotiated translation column stops being authoritative.

**🧠 Reasoning**

Evans names two properties the UL must have to do its job (_DDD_, ch. 2): the language must be _precise_ (one meaning per word) and _consistent_ (one word per meaning). Allowing synonyms violates _consistent_ outright and usually erodes _precise_ too. The downstream consequences are DDD failures, not tooling failures:

| If the UL admitted...                               | The DDD failure is...                                                                                                                                                                                                                                         |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `quantity`, `qty`, `amount` for the same concept    | **Knowledge crunching wasted.** The team negotiated _one_ canonical term for the concept "Quantity" during the workshop. Allowing three names for it silently undoes that agreement — every new arrival has to rediscover that the words mean the same thing. |
| `availability` alongside `InventoryLevel`           | **BC boundary erodes.** _Availability_ is a Catalog concept (can-be-sold? in-region?). Admitting it inside Warehouse implies Warehouse cares about visibility, which it doesn't. The neighbour BC's vocabulary leaks across the boundary.                     |
| `prodotto` alongside `Article`                      | **Bilingual UL drifts.** The team negotiated _Article ↔ Articolo_. Admitting _prodotto_ reopens that decision — when the customer says _"il prodotto"_ a developer no longer knows which canonical English term it maps to.                                   |
| `stock` meaning both _on-hand_ and _reserved_ units | **Loss of precision.** One word, two meanings. Evans's first UL property breaks. Conversations and code now require context to disambiguate — exactly the translation cost the UL was supposed to eliminate.                                                  |

> 💡 **The deeper cost.** The UL is the _shared output_ of the knowledge crunching between domain experts and engineers. Every synonym admitted is a piece of that shared understanding silently lost. After six months the team is back to where it was _before_ the workshop, except now everybody _thinks_ there is a UL.

**🔬 Verification**

- **Artefact:** [`ubiquitous-language.md`](../ubiquitous-language.md)
- **Reference:** Eric Evans, _DDD_ 2003, ch. 2 — _Communication and the Use of Language_. The UL is defined exactly by these properties; remove them and you have a glossary, not a UL.

---

## Step 3 — Context Mapping

Reference questions in [`../README.md` Step 3](../README.md#step-3--read-the-context-mapping-15-min).

### Q4 — Customer/Supplier vs Conformist: what's the precise difference?

**🎯 Answer (TL;DR)**

Both are Upstream/Downstream patterns; the difference is **power balance**. In Customer/Supplier the downstream **has** influence over the upstream's roadmap (negotiation, joint planning, contract reviews). In Conformist the downstream has **lost** (or never had) that influence and simply accepts the upstream model as is. **Conformist is the specialisation of Customer/Supplier where the negotiation channel is closed.**

**🧠 Reasoning**

The two patterns sit on a spectrum, not in two separate boxes:

```text
Customer/Supplier  ←──── continuum ────→  Conformist
  ↑                                          ↑
  has leverage                               no leverage
  negotiates                                 accepts
  contract evolves                           contract fixed
  joint roadmap                              upstream's roadmap only
```

The same relationship can **slide** from one to the other:

- A startup using an internal API of a partner company starts in Customer/Supplier (the partner needs them). After the startup is acquired and demoted in priority, the relationship slides into Conformist.
- A team building on a vendor SDK starts in Conformist (the vendor controls the SDK). When the team becomes large enough to influence the vendor's roadmap (big customer, strategic relationship), it can slide into Customer/Supplier.

In the Warehouse exercise the relationship Orders↔Warehouse is **Customer/Supplier**: the two teams negotiate the API and the event schemas. If the Warehouse team were a separate company that didn't return calls, Orders would slide to Conformist (or build an ACL, see Q5).

**🔬 Verification**

- **Artefact:** [`context-mapping.md`](../context-mapping.md)
- **External:** Evans, _DDD_ (2003)

---

### Q5 — Why ACL instead of Conformist between Go Warehouse and the legacy schema?

**🎯 Answer (TL;DR)**

The legacy `business_data`/`business_relations` schema uses generic columns (`amount_1`, `text_2`, `payload`) that _would corrupt_ the Go BC's model if exposed directly. Conformist would force every Go struct to carry legacy column names; ACL pays the cost of a translator at the boundary and lets the Go BC model the domain cleanly. The trade-off is **translator maintenance cost** vs **a polluted domain model** — for a long-lived BC that will outlive the legacy, ACL wins.

**🧠 Reasoning**

Conformist would produce code like:

```go
type Article struct {
    ID       string
    Code     string   // = SKU (legacy "code")
    Amount1  int64    // = price in cents (legacy "amount_1")
    Amount2  int64    // = min quantity (legacy "amount_2")
    Text1    string   // = category (legacy "text_1")
    Text2    string   // = VAT code (legacy "text_2")
    Payload  string   // = JSON blob
}
```

This is _technically_ a faithful translation of the legacy schema. But:

1. **Readability is destroyed.** A new joiner cannot tell what `Amount2` is without reading legacy migration notes.
2. **The UL is violated.** `Amount1` is not a UL term; the UL forbids `amount` for non-Money fields.
3. **Refactor cost rises.** Any business operation now has to manipulate generic-named fields and accidentally exposes them in API responses, events, logs.

ACL pays a translator cost at exactly **one place** (later lessons, planned) and the rest of the Go code lives in the UL. Trade-off: when the legacy schema changes (rare in MIC's case — it's the part the exercise wants to extract from), the translator must be updated. The cost of _one_ translator file vs _every_ line of Go code is the right asymmetry to bet on.

> 💡 **ACL is the bet that the Go BC outlives the legacy schema.** If the legacy were going to outlive the Go BC, Conformist would be cheaper. For a Strangler Fig extraction the assumption is reversed.

**🔬 Verification**

- **Artefact:** [`context-mapping.md`](../context-mapping.md) "Pattern 3 — Anti-Corruption Layer"
- **Tactical mapping:** [`tactical-ddd-warehouse.md`](../tactical-ddd-warehouse.md)
- **ADR:** ADR-013 — Dual-write pattern

---

## Step 4 — Tactical DDD

Reference questions in [`../README.md` Step 4](../README.md#step-4--read-the-tactical-ddd-building-blocks-15-min).

### Q6 — Why does `InventoryLevel` have an `ID` but `SKU` does not?

**🎯 Answer (TL;DR)**

Because **`InventoryLevel` is an Entity** (identity-bearing) and **`SKU` is a Value Object** (defined by data alone). Two `InventoryLevel`s with the same `(ArticleID, LocationCode, Quantity)` but different `ID`s are _different things_ (e.g., two different ledger rows that happen to be in the same state). Two `SKU`s with the same `Code` are _the same thing_ — there's no "this SKU instance vs that SKU instance".

**🧠 Reasoning**

The Entity/VO distinction is one of the most common stumbling blocks in DDD. The litmus test is:

> _"If I create another one of these with exactly the same data, is it the same thing or a different thing?"_

For SKU: two `SKU{Code: "WGT-001"}` are the _same_ SKU — there's no instance identity. Hence VO.
For InventoryLevel: two `InventoryLevel{ArticleID: "a-1", LocationCode: "IT-MILANO1", Quantity: 100}` could be two distinct rows in different transactional contexts; they need IDs to be distinguishable. Hence Entity.

The practical consequences:

| Property       | Entity (`InventoryLevel`)                       | VO (`SKU`)                        |
| -------------- | ----------------------------------------------- | --------------------------------- |
| Identity field | `ID` (UUID)                                     | None                              |
| Equality       | By `ID`                                         | By all data fields                |
| Mutability     | Mutable (through aggregate root)                | Immutable — replaced, not mutated |
| Construction   | `NewInventoryLevel(id, ...)` — requires an `id` | `NewSKU(code)` — no `id` argument |

> 💡 **The Entity/VO choice constrains your equality operator and your persistence layer.** Get it wrong and the database accumulates duplicate "values" with different IDs, or two distinct entities collide because they happen to have identical content. The aggregate root reconciles these decisions at the boundary.

**🔬 Verification**

- **Artefact:** [`tactical-ddd-warehouse.md`](../tactical-ddd-warehouse.md) "Entity" and "Value Object"
- **Code:** `phase-03-skeleton/entities/inventory.go:13-21` (the struct has `ID`) vs `phase-03-skeleton/entities/sku.go` (no `ID` field)
- **ADR:** [ADR-011](../../docs/adr/ADR-011-ddd-building-blocks.md) — definitions of Entity and Value Object

---

## Step 5 — Dependency Map

Reference questions in [`../README.md` Step 5](../README.md#step-5--read-the-dependency-map-15-min).

### Q7 — Why is Warehouse called "a sink"? What does that mean for extraction order?

**🎯 Answer (TL;DR)**

A "sink" is a node with **many in-bound edges and zero out-bound edges**: everyone reads from it; it reads from no one. In section 4, Warehouse has 3 in-bound dependencies (Orders, Listino, Invoicing) and 0 out-bound dependencies on other BCs' tables. That's the textbook _safe-to-extract first_ condition in a Strangler Fig — extracting Warehouse cannot break anything else, because Warehouse doesn't depend on anything else.

**🧠 Reasoning**

In a Strangler Fig migration the order of extraction matters because:

- Extracting a BC that **reads from** other BCs forces you to design the read contracts first (otherwise the extracted BC can't function alone).
- Extracting a BC that **only is read from** by other BCs only requires that you expose a contract for the readers — you control the timing.

Warehouse is in the second category. The extraction sequence becomes:

1. Stand up the Go BC Article/inventory HTTP surface (Phase 06).
2. Harden that same surface with authentication and policy before trusting it for routed traffic (Phase 07-08).
3. Use a cutover runbook to move consumers only after parity, policy, and observability checks are in place.
4. Decommission the legacy Warehouse path after the integration evidence says it is safe.

No step requires Warehouse to call any other BC — so no other BC has to be ready before Warehouse can be extracted. That's the "sink" property paying off.

> 💡 **The sink property is rare and valuable.** Most BCs in a monolith have mutual dependencies. Picking a sink as the first extraction target is the easiest mode and a smart pedagogical choice for this exercise. Real-world extractions typically have to deal with mixed dependencies and pick the BC with the _cheapest_ (not necessarily zero) out-bound calls.

**🔬 Verification**

- **Artefact:** [`DEPENDENCY-MAP.md`](../DEPENDENCY-MAP.md)
- **External:** Sam Newman, _Building Microservices_ (2nd ed.)

---

## Step 6 — Extraction Plan

No fixed Q/A — the Extraction Plan is a read-only artefact for this phase. Walk through it on your own; revisit it after Phase 03 has implemented the domain layer to see which decisions paid off.

---

## Step 7 — Stretch

No fixed Q/A. The stretch is open-ended on purpose:

- Running the UL validation prompts on real code is the muscle memory the exercise wants you to leave with.
- Sketching an Event Storming canvas for _your_ project is the only way to internalise the three-step rhythm.

Photograph the canvas and bring it to the Phase 03 review.
