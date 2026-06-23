# Dependency Map — Warehouse in the MIC Monolith

Concrete dependency map between the **Warehouse** BC (the extraction target) and its neighbours, anchored to the actual PHP code under `phase-01-monolith/php-app/`.

This document answers three questions:

1. **Where does Warehouse live today inside MIC?** (which tables, which files)
2. **Who depends on Warehouse?** (in-bound: who reads/writes its data)
3. **What does Warehouse depend on?** (out-bound: who it reads from)

The consolidated answer determines the **extraction safety**: if Warehouse is a sink (everyone reads from it, it reads from nobody) then we can extract it first without breaking dependencies down-stream.

---

## 1. Where Warehouse lives today

### Shared storage (intentional anti-pattern)

The entire MIC domain lives in two generic tables (see `phase-01-monolith/database/schema.sql`):

- `business_data` — N business entities identified by the `record_type` column.
- `business_relations` — N polymorphic relationships between entities, identified by the `relation_type` column.

### What belongs to the Warehouse BC

| Warehouse concept                                | Where it lives                                                    | Detail                                                                                                                                                                        |
| ------------------------------------------------ | ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Article** (stockable catalog item)             | `business_data WHERE record_type='articolo'`                      | `code=SKU`, `name=nome`, `amount_1=prezzo_listino`, `amount_2=qta_minima`, `text_1=categoria`, `text_2=iva_default`                                                           |
| **Warehouse Location** (_magazzino_)             | `business_data WHERE record_type='magazzino'`                     | `code=warehouse code`, `name=name`                                                                                                                                            |
| **Stock movement** (_movimento_; in/out movement ledger) | `business_data WHERE record_type='movimento'`                     | `date_1=date`, `amount_1=qty`, `text_1=type` (in/out) — _conceptually Warehouse-internal but **descoped from the Go BC** for this exercise; see `tactical-ddd-warehouse.md`._ |
| **Movement ↔ Article**                           | `business_relations WHERE relation_type='movimento_di_articolo'`  | `amount=quantity moved`                                                                                                                                                       |
| **Movement ↔ Location**                          | `business_relations WHERE relation_type='movimento_in_magazzino'` | —                                                                                                                                                                             |

### PHP controllers that implement Warehouse

| File                                              | What it does                                                                                                             | HTTP routes                                           |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------- |
| `php-app/src/Controllers/ArticleController.php`   | Article CRUD + mapping `record_type='articolo'` → clean DTO                                                              | `/api/articles[/:id]`                                 |
| `php-app/src/Controllers/MagazzinoController.php` | Location CRUD + **inventory level** aggregation (JOIN on `business_relations` + `business_data`)                         | `/api/magazzini[/:id]`, `/api/magazzini/:id/giacenze` |
| `php-app/src/Controllers/MovimentoController.php` | Creates movements + links them to article and location via `relate()` — **stays in PHP** (movement ledger not migrated) | `/api/movimenti[/:id]`                                |

`ArticleController` explicitly flags its extractability (lines 7–21): _"BC candidate: Warehouse (extraction target). This controller is in scope for migration to the Go Warehouse BC starting in Phase 06."_

---

## 2. Who depends on Warehouse (in-bound)

Every entry is verified against the code. Line numbers refer to the repo state at `phase-01-monolith/php-app/src/`.

### 2.1 Orders → Warehouse 🔴 (strong dependency)

**File:** `Controllers/OrderController.php`

#### Read when creating an order line

- **`addRiga()` (lines 121–176)** — when the monolith adds a `riga_ordine` to an `ordine`:
  - `$this->repo->findById($art_id)` (line 131) → loads the article to read `code` (SKU) and `amount_1` (default price if not provided in body).
  - Also reads `articolo->text2` for the VAT code, used to look up `aliquota_iva` (out-bound towards Pricing — see section 3).
  - `$this->repo->relate($riga->id, $art_id, 'articolo_in_riga_ordine', $qta)` (line 166) → creates the persistent relation between line and article.

#### Read when listing order lines

- **`righe()` (lines 85–118)** — when the monolith serves `/api/orders/:id/righe`:
  ```sql
  SELECT bd.id, bd.code, bd.name
    FROM business_relations br
    JOIN business_data bd ON bd.id = br.target_id
   WHERE br.source_id = ? AND br.relation_type = 'articolo_in_riga_ordine'
   LIMIT 1
  ```
  → for each line, retrieves the SKU and name of the related article.

**Coupling type:** read-only on Warehouse anagrafica + writes a relation `articolo_in_riga_ordine` (semantic ownership: the relation lives in Orders but points to a Warehouse aggregate).

### 2.2 Listino (Catalog/Pricing) → Warehouse 🟠 (moderate dependency)

**File:** `Controllers/ListinoController.php`

- **`addVoce()` (lines 55–86)** — when adding a _voce_ to a _listino_:
  - `$this->repo->findById($articolo_id)` (line 64) + check `$art->recordType !== 'articolo'` (line 65) — validates that the target is a Warehouse aggregate.
  - `$this->repo->relate($voce->id, $articolo_id, 'articolo_di_voce', $prezzo)` (line 81) — creates the relation.

**Coupling type:** read-only (existence check) + writes a relation `articolo_di_voce`.

### 2.3 Invoicing → Warehouse 🟡 (transitive dependency)

**File:** `Controllers/InvoiceController.php`

- `lookupRelTarget()` (lines 55–65) — JOIN on `business_relations br JOIN business_data bd ON bd.id = br.target_id` to retrieve entities related to the invoice (customer, order). It does **not** touch Warehouse directly, but reads `ordine` which in turn is linked to `riga_ordine`, which points to `articolo` via `articolo_in_riga_ordine`.

**Coupling type:** transitive read — the chain is Invoice → Order → OrderItem → Article. Invoicing never sees the SKU/price of the article directly; it could need to in the future if it printed detailed line items on invoices (it does not today at the invoicing level).

---

## 3. What Warehouse depends on (out-bound)

Verified against the three Warehouse controllers (`ArticleController`, `MagazzinoController`, `MovimentoController`):

### ArticleController

- **Zero out-bound queries.** It only maps `record_type='articolo'` to/from a DTO. It does not call `findById` on any other entity.

### MovimentoController

- Reads `articolo` and `magazzino` (`lookupRelTarget`, lines 46–56) — **both Warehouse-domain**. Not out-bound; in-domain.

### MagazzinoController.giacenze()

- Query (lines 54–67) JOINing `business_data + business_relations` to aggregate `movimenti` by `(articolo, magazzino)`. **All in-domain Warehouse.** Out-bound: zero.

### ⚠ Note: the VAT query in OrderController is _not_ Warehouse's

`OrderController::addRiga()` lines 138–141 executes:

```sql
SELECT id, amount_1 FROM business_data WHERE record_type='aliquota_iva' AND code=? LIMIT 1
```

This is **Orders reading from Pricing**, not Warehouse. We mention it here for clarity: Warehouse reads `text_2` (VAT code, a string) as metadata on the article, but the numeric VAT percentage is owned by Pricing (`aliquota_iva` record_type).

---

## 4. Summary: Warehouse is a sink

```text
  ┌────────────────┐   ┌──────────────────┐   ┌──────────────────┐
  │  Invoicing BC  │   │ Catalog/Pricing  │   │    Orders BC     │
  └───────┬────────┘   └────────┬─────────┘   └────────┬─────────┘
          ·                     │                       │
     (transitive           findById articolo       findById articolo
      via Order)           articolo_di_voce         articolo_in_riga_ordine
          · · · · · · ·         │                       │
                                └──────────┬────────────┘
                                           │
                                           ▼
                               ┌───────────────────────┐
                               │     Warehouse BC      │
                               │  articolo + magazzino │
                               └───────────────────────┘

  Orders BC ───── aliquota_iva lookup ────► Pricing BC   (not a Warehouse dependency)
```

**3 in-bound, 0 out-bound.** Warehouse does not read from any other BC in its controllers. This is the textbook definition of a _safe-to-extract first_ candidate in a Strangler Fig scenario.

---

## 5. Implications for the extraction

Once Warehouse lives as a Go service (the outcome of CP6–CP10), the three dependent BCs must migrate their access:

| Dependent BC | Pre-extraction call pattern                                                | Post-extraction target pattern                                                                                                         |
| ------------ | -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Orders       | `$repo->findById($art_id)` (direct SQL on `business_data`)                 | `GET /api/articles/{id}` HTTP call to the Go BC                                                                                        |
| Orders       | `$repo->relate(rigaId, artId, 'articolo_in_riga_ordine')` (cross-BC write) | Relation stays in Orders' schema, but `art_id` becomes an **opaque** Warehouse ID — existence check via HTTP `HEAD /api/articles/{id}` |
| Listino      | `$repo->findById($articolo_id)` + check `recordType==='articolo'`          | `GET /api/articles/{id}` (404 = invalid)                                                                                               |
| Listino      | `$repo->relate(voceId, artId, 'articolo_di_voce')`                         | Same as Orders: local relation, opaque `art_id`                                                                                        |
| Invoicing    | Transitive read via Order                                                  | No change: it already does not touch Warehouse directly                                                                                |

### Transition strategy (Strangler Fig)

Once the Go BC owns the Article/inventory HTTP surface, that surface is progressively hardened (authentication, policy enforcement, event publication) while the monolith keeps serving direct SQL, until a cutover runbook moves consumers. That hardening and cutover are the work of the following lessons.

None of the three in-bound dependencies in section 2 requires Warehouse to call another BC. The Go BC still needs a stable Article contract before consumers can move, but the precise facade flip is a later step.

---

## 6. Independent verification

> 🤖 An AI tool with access to the repo files can re-verify this document in one shot. Example starting point — adapt the scope and output format to taste:

```text
Read `phase-02-analysis/DEPENDENCY-MAP.md` and the PHP controllers under
`phase-01-monolith/php-app/src/Controllers/`. For each factual claim in the
document, check it against the code and report: ✅ correct | ❌ wrong | ⚠ stale.
```

If the files change over time and this document has not been updated, **trust the code**: update this file via a PR and, if material, open a documentation follow-up with the updated references.

---

## 7. Related documents

- [`event-storming-canvas.md`](./event-storming-canvas.md) — events Warehouse emits/consumes
- [`ubiquitous-language.md`](./ubiquitous-language.md) — canonical vocabulary of the Warehouse BC
- [`tactical-ddd-warehouse.md`](./tactical-ddd-warehouse.md) — Aggregate / Entity / Value Object mapping for Warehouse
- [`context-mapping.md`](./context-mapping.md) — Customer/Supplier, Conformist, ACL relationships between Warehouse and its neighbours
- `phase-01-monolith/README.md` — the MIC monolith from the participant's perspective
