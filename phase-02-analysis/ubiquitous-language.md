# Ubiquitous Language — Warehouse Bounded Context

> Strategic DDD artifact: the shared vocabulary of the Warehouse BC. One term per concept. Definitions are normative.

---

## What the Ubiquitous Language is

> 💡 **Ubiquitous Language (UL)** — the single language shared by domain experts, product owners, designers and developers within one Bounded Context. Its goal is to eliminate the cost of constantly translating between _business model_ and _technical model_. Each term has exactly one meaning inside its BC.

Three properties (Evans, _DDD_ 2003, ch. 2):

1. **Precise.** No assumptions about what a word means. If two people interpret "stock" differently, the word is broken — split it.
2. **Consistent.** One concept = one term. If the domain expert says "stock" and the developer says "inventory" for the same thing, the UL is leaking.
3. **Used by everyone.** The same words appear in conversations with the customer, in tickets, in commits, in code, in test names, in API contracts, in event payloads.

> ⚠ **The UL is owned by the whole team — not the architect, not the lead developer.** A UL is the _main output of knowledge crunching_: the joint work of domain experts and engineers crystallising what the business actually does.

### Why "ubiquitous"

The language must be ubiquitous **inside** its Bounded Context. **Outside** the BC the same word may mean something different — in a Catalog BC "Article" might mean an editorial/news item, not a stockable item. The BC stops the UL.

> 💡 **A Bounded Context defines the applicability of a UL.** Integration with other BCs uses **context mapping** patterns (see [`context-mapping.md`](./context-mapping.md)).

---

## Can we have a bilingual vocabulary?

The simplest case is when the team agrees on a single language for everything — conversations, tickets, commits, code. If that is possible, the UL table has no translation column and you are done.

The harder case — handled here — is when **domain experts reasonably prefer to speak their own language** rather than the one used in code. This is the common situation in many product teams: domain experts are most precise and most agile when they speak their native language, while the codebase is in English (frameworks, libraries, hiring market, external documentation). Forcing domain experts to speak English slows down knowledge crunching. In this exercise that pair is Italian (customer) and English (code), but the pattern applies to any combination.

The team then has to keep two conversations consistent at all times:

- With the customer / domain expert, in their language: _"Quando rettifico la giacenza di un articolo..."_
- In the codebase, ADRs, commit messages, in the code language: _"`Article.AdjustInventory(...)` records an `InventoryAdjusted` event"_

The bilingual UL provides the **negotiated, accepted translations** that bridge the two. The code-language column is _canonical_: it drives identifiers, event payloads, test names. The customer-language column is the team-and-customer-agreed phrase — nothing more, nothing less. Without this table every developer translates ad hoc, and you end up with competing synonyms across PRs, tickets and customer emails for the same concept.

> 💡 **The translation column is not _literal_ — it is _negotiated_.** Some mappings are obvious (Article → Articolo). Some are not (Inventory Level → _Giacenza_, never _disponibilità_; SKU → _Codice articolo_, never _ID prodotto_). Agreeing on these mappings _is_ part of the knowledge-crunching work.

If your team operates in more than two languages, replicate the column — the structure (one canonical column, N translation columns, one definition column) stays the same.

---

## The Warehouse vocabulary

| Term (EN, canonical)   | Italian (con il cliente) | Definition                                                                                                                       |
| ---------------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| **Article**            | Articolo                 | A stockable item identified by a unique SKU, with a name, a price, and zero-or-more inventory levels across warehouse locations. |
| **Inventory Level**    | Giacenza                 | The count of units of a specific Article kept at a specific Warehouse Location.                                                  |
| **Money**              | Importo monetario        | A monetary value paired with a currency. Always handled together; never as a bare number.                                        |
| **Price**              | Prezzo                   | The selling price assigned to an Article. Expressed as a Money value. Owned by the Warehouse BC as the canonical purchase price. |
| **Quantity**           | Quantità                 | A non-negative integer measuring units of stock.                                                                                 |
| **SKU**                | Codice articolo          | _Stock Keeping Unit_. The globally unique alphanumeric identifier of an Article.                                                 |
| **Stock Reservation**  | Prenotazione di stock    | A logical reservation of stock units against an Inventory Level on behalf of an Order.                                           |
| **Warehouse Location** | Magazzino fisico         | A physical place where stock is kept (e.g., the Milan warehouse), identified by an alphanumeric code.                            |

> 💡 **What the table holds is domain nouns** — the concepts a domain expert names out loud: _Article_, _Inventory Level_, _SKU_, _Price_, _Stock Reservation_, ... Things you will _not_ find in the table on purpose:
>
> - **Operations** (_Create article_, _Adjust inventory_, _Reserve stock_, ...). These are inferred from the nouns: once the team agrees that _Article_ and _Price_ exist as first-class concepts, "change the price of an article" is self-evident. Listing operations separately would duplicate intent and quickly go stale.
> - **Past-tense domain events** (`ArticleCreated`, `InventoryAdjusted`, `StockReserved`, ...). These are _facts that emerge_ from domain operations — they were discovered during the Event Storming workshop ([`event-storming-canvas.md`](./event-storming-canvas.md)) and they get mapped to `DomainEvent` structs in code ([`tactical-ddd-warehouse.md`](./tactical-ddd-warehouse.md) §"Domain Event"). Their names re-use UL nouns in past tense; they are not separate UL terms.
> - **Tactical / architectural classification** (which UL term becomes an Aggregate, an Entity, a Value Object, a Repository, ...). That decision lives in [`tactical-ddd-warehouse.md`](./tactical-ddd-warehouse.md). The UL is the vocabulary; the tactical mapping is _how we implement it_.

---

## What is NOT in the Warehouse UL

Not every term that surfaces in a domain conversation belongs here. Some concepts are owned by neighbouring BCs and must not bleed into the Warehouse vocabulary. For example: _Order_ and _Invoice_ belong to the Orders and Invoicing BCs respectively — the Warehouse only knows that stock gets reserved on behalf of something, not what that something is. Similarly _Category_ (_Categoria_) is a Catalog concern, and _VAT rate_ (_IVA_) lives in Pricing/Tax.

If a term from another BC appears in Warehouse code or tickets, treat it as a context leak. See [`context-mapping.md`](./context-mapping.md) for how the Warehouse integrates with neighbouring BCs without absorbing their vocabulary.

---

## AI prompts for the UL lifecycle

Three short prompts that cover the lifecycle of a UL: **produce it** from raw domain material, **audit code** against it, **generate new code** that respects it. Self-contained: paste each into any AI tool.

### Prompt 1 — Produce (or refresh) the UL

> 🤖 Use this to bootstrap a UL from raw domain material, or to refresh an existing UL with terms found in a new conversation / ticket.

```text
You are producing the Ubiquitous Language for the Warehouse Bounded Context.

Inputs:
- Domain material to extract vocabulary from (an Event Storming canvas, a
  domain narrative, a customer interview transcript, a feature request,
  meeting notes, ...):

<<<
{{DOMAIN_INPUT}}
>>>

- (Optional) the existing UL, if you are refreshing rather than creating
  from scratch. If empty, generate from scratch:

<<<
{{EXISTING_UL_OR_EMPTY}}
>>>

Task: extract the canonical vocabulary and produce a UL table in this
EXACT format (Markdown), nothing else, no extra columns:

| Term (EN, canonical) | Italian (con il cliente) | Definition          |
| -------------------- | ------------------------ | ------------------- |
| **<noun>**           | <negotiated Italian>     | <one sentence>      |
| ...                  | ...                      | ...                 |

Rules — what goes in:
- ONLY nouns (domain concepts: entities, value-shaped concepts, terms the
  team uses to name things in the domain).
- One canonical term per concept; no synonyms.
- Sort alphabetically by canonical EN term.
- If extending an existing UL, mark new rows with `[NEW]` and modified
  rows with `[CHANGED]` in the Term column; leave untouched rows unmarked.

Rules — what does NOT go in:
- Operations / verbs (e.g., _Create article_, _Adjust inventory_). These
  are inferred from the nouns above — once the vocabulary is agreed, the
  operations are self-evident. Listing them separately duplicates intent.
- Domain events (e.g., `ArticleCreated`, `InventoryAdjusted`). These are
  past-tense FACTS that emerge from domain operations; they live in the
  Event Storming canvas and the tactical mapping, not in the UL.
- Tactical / architectural names (Aggregate, Entity, Value Object,
  Repository, Save, DTO, ID, ...). These are implementation decisions,
  not vocabulary.

For the Italian column prefer the *negotiated* term the customer actually
uses, not a literal translation. Flag any term where you had to guess by
appending "(guess)" to the Italian cell.

Output ONLY the UL table. Do not write code, ADRs, tactical mappings, or rationale.
```

### Prompt 2 — Audit the codebase against the UL

> 🤖 Use this to check that the code mirrors the UL — both directions, in one shot.

```text
You are auditing a Go codebase for Ubiquitous Language conformance.

Inputs:
- The UL: `phase-02-analysis/ubiquitous-language.md` (the vocabulary table).
- Code: all .go files under `phase-XX-skeleton/`.

Task:
1. Extract every canonical English term from the UL.
2. Extract every exported identifier (type, method, struct field, event
   name, package-level variable) from the code.
3. Produce two tables:
   A. UL term → matched code identifier (or "MISSING in code").
   B. Code identifier → matched UL term (or "NOT IN UL — propose:
      rename / add to UL / move to another BC").

Be strict: a UL has zero synonyms. If two identifiers refer to the same
concept, flag it.
```

### Prompt 3 — Generate code that respects the UL

> 🤖 Use this when asking an assistant to write or modify code in the Warehouse BC. Treat the UL as part of the system prompt.

```text
You are writing Go code for the Warehouse Bounded Context.

Hard constraint: `phase-02-analysis/ubiquitous-language.md` is the canonical
lexicon. Every identifier you introduce — types, methods, struct fields,
event names, test names — must be a UL canonical term or a trivial
composition of them (e.g., articleRepository, newInventoryLevel).

If you need a concept that is not in the UL: stop. State that the concept
is missing, propose a one-line definition plus Italian translation, and
ask the user to confirm adding it to the UL before writing code.

Do not invent synonyms. Do not use foreign-BC terms.

Task:
<<< {{INSTRUCTION}} >>>
```

> 💡 **Treat the prompts as a starting point, not a fixed recipe.** Tweak the wording and the output format until you have a workflow you trust well enough to run after every PR/MR — that is the cadence at which lexical drift actually surfaces.

---

## Where the UL was born — and where it grows

The Warehouse UL emerged from the Event Storming workshop documented in [`event-storming-canvas.md`](./event-storming-canvas.md): the past-tense events on the wall (_ArticleCreated_, _InventoryAdjusted_, ...) and the candidate Bounded Contexts surfaced the vocabulary's first nouns. For this reason it is important to avoid synonyms from the beginning. The team and the domain experts then refined it and negotiating the Italian translations row by row.

> 💡 **The UL is a living document.** Each new ticket, each customer call, each retro can add or refine a term. A PR/MR that introduces a new domain concept must update this file in the same change — otherwise the UL drifts and the code stops being a mirror of the domain.

---

## References

- [`event-storming-canvas.md`](./event-storming-canvas.md) — where the UL was first surfaced
- [`tactical-ddd-warehouse.md`](./tactical-ddd-warehouse.md) — Aggregate / Entity / VO / Domain Event mapping (the _implementation_ of the UL)
- [`context-mapping.md`](./context-mapping.md) — relationships with neighbouring BCs (the UL stops at these boundaries)
- [`docs/adr/ADR-010-event-storming-methodology.md`](../docs/adr/ADR-010-event-storming-methodology.md) — Event Storming as the method that surfaces the UL
- [`docs/adr/ADR-011-ddd-building-blocks.md`](../docs/adr/ADR-011-ddd-building-blocks.md) — tactical DDD names used by the code

External canonical references:

- Eric Evans, _Domain-Driven Design: Tackling Complexity in the Heart of Software_ (2003)
- Vaughn Vernon, _Implementing Domain-Driven Design_ (2013)
- Vlad Khononov, _Learning Domain-Driven Design_ (2021)
- Free DDD Reference (Evans, 2015)
