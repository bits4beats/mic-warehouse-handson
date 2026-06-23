# Worst-of list — MIC antipatterns

> Reference deliverable (Phase 01 solution): what a strong room converges on. Open only after you
> have produced your own.

## Top 3

### 1. One polymorphic schema for everything
All 19 record types live in `business_data` + `business_relations`, with meaning encoded in
`record_type` and positional columns (`amount_1` = price for an `articolo`, taxable total for an
`ordine`, ...). **Why it hurts:** no foreign keys, no type safety, no per-entity constraints; the
database can enforce *nothing* about a business record. Every query starts by filtering on
`record_type`, and any field that does not fit the generic columns ends up in `payload_json`.
**Cost of keeping it:** every new field and every integrity rule has to be re-implemented in PHP and
policed by hand; data corruption is a `WHERE` clause away. This is the root antipattern that makes
the next two possible.

### 2. No domain layer: business logic lives in SQL and controllers
The dependency chain is `controller → repository → raw SQL`. Invariants (order total = sum of
lines, price > 0, VAT lookup, stock never negative) are written inline inside controller methods
(e.g. `OrderController::addRiga` / `recalcOrdine`). **Why it hurts:** nothing *owns* a business
rule, so the same rule is re-implemented or quietly skipped per call site, and there is no place to
unit-test it. **Cost of keeping it:** behavior drifts between endpoints; a refactor has no safety
net.

### 3. No boundaries: domains read each other's raw data
`OrderController::addRiga()` loads the article record and reads its price from `amount_1` and its
VAT code from `text_2`, then looks up `aliquota_iva` directly. **Why it hurts:** Orders is coupled
to the *internal column layout* of Warehouse and Pricing, with no contract in between. **Cost of
keeping it:** any change to how an article stores price or VAT silently breaks Orders; you cannot
extract or replace one area without touching the others. (Full evidence in
[`coupling-map.md`](./coupling-map.md).)

## Honorable mentions (real problems, smaller blast radius)

- **Unversioned API** (`/api/*`): no way to evolve a contract without breaking callers.
- **Ad-hoc error envelope** (`{ "error": "..." }`): no standard problem details, no error-code contract.
- **No authentication or authorization anywhere**: every route is open.
- **No automated tests**: nothing pins current behavior before a refactor.
- **Detail endpoints keyed by the internal numeric `id`** (not SKU/business code): leaks a storage detail into the API.

> These used to be handed to participants as an "intentional scope gaps" list in the README. In the
> reworked Phase 01 they are not told: they should fall out of the exploration and land here.
