# ADR-006: MIC PHP Monolith Baseline

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

The exercise starts from `phase-01-monolith/`, a deliberately messy PHP
monolith called MIC. MIC is a fictional invoicing SaaS for Italian SMBs and is
the "before" state that the Strangler Fig migration gradually replaces.

The current implementation is deliberately small and explicit. It is a PHP 8.2
application where routing, controller dispatch, persistence, and cross-domain
coupling are visible in the repository instead of hidden behind framework
conventions. It has:

- one nginx + php-fpm container;
- a custom front controller in `php-app/index.php`;
- a tiny custom router in `php-app/src/Router.php`;
- controller classes under `php-app/src/Controllers/`;
- a generic repository over two polymorphic tables;
- MySQL 8.0 seeded from `database/schema.sql` and `database/seed.sql`;
- a no-op nginx facade scaffold under `facade/`.

Phase 01 exists to make coupling visible before learners attempt to extract a
Warehouse Business Capability into Go. The monolith intentionally mixes
Catalog, Logistics/Warehouse, Orders, Invoicing, Payments, Customers, Pricing,
and System data in the same persistence model.

The two core tables are:

- `business_data`: one row shape for 16+ business record types;
- `business_relations`: one relation table for all cross-record links.

This anti-pattern is the point of the phase. Learners should observe how a
single generic schema makes ownership boundaries hard to see, then use Phase 02
to turn that chaos into DDD boundaries and an extraction plan.

---

## Decision

We will keep Phase 01 as a **plain PHP MIC monolith with a polymorphic schema**
rather than a framework-based reference service.

This local exercise decision gives learners:

1. A concrete legacy baseline to run in Docker.
2. A visible database anti-pattern: many domains hidden in two generic tables.
3. Direct cross-domain coupling to inspect in PHP code and SQL.
4. A stable legacy `/api/*` contract to compare against later phases.
5. A no-op Strangler Fig facade available from day one.

### Architecture Decision

```text
┌────────────────────────────────────────────┐
│ Browser / smoke-test client                │
├────────────────────────────────────────────┤
│ nginx                                      │
│ - serves static SPA assets                 │
│ - forwards API requests to PHP-FPM         │
├────────────────────────────────────────────┤
│ PHP front controller (`php-app/index.php`) │
│ - custom router                            │
│ - controller dispatch                      │
├────────────────────────────────────────────┤
│ Controllers                                │
│ - map friendly DTO fields                  │
│ - interpret amount_N/text_N/date_N columns │
│ - contain intentionally coupled logic      │
├────────────────────────────────────────────┤
│ Generic Repository                         │
│ - `business_data`                          │
│ - `business_relations`                     │
├────────────────────────────────────────────┤
│ MySQL 8.0                                  │
└────────────────────────────────────────────┘
```

### Key Design Choices

**Small explicit PHP application**

The application uses a tiny router and direct controllers. This keeps the
participant's attention on coupling and migration: learners can see exactly
where a request enters, which controller handles it, and where database reads
cross candidate context boundaries.

**Polymorphic database model**

All business records live in `business_data`; the meaning of `amount_1`,
`text_2`, `date_1`, and similar columns depends on `record_type`.

Example mappings:

| `record_type` | Generic columns with business meaning |
|---|---|
| `articolo` | `code = SKU`, `name = article name`, `amount_1 = list price`, `amount_2 = minimum stock`, `text_1 = category`, `text_2 = VAT code` |
| `ordine` | `code = order number`, `amount_1 = taxable amount`, `amount_2 = VAT`, `amount_3 = total`, `status = workflow state` |
| `riga_ordine` | `parent_id = order`, `amount_1 = quantity`, `amount_2 = unit price`, `amount_3 = discount`, `amount_4 = line total` |
| `magazzino` | `code = warehouse code`, `name = warehouse name`, `text_1 = city`, `text_2 = postal code` |

**Direct cross-domain reads**

Orders directly read Warehouse/Catalog data. For example,
`OrderController::addRiga()` loads an `articolo` row to derive the default unit
price and reads `articolo.text_2` to look up the VAT rate. That is the coupling
learners must notice before extracting Warehouse.

**Legacy contract, not platform-compliant BC contract**

Phase 01 keeps the legacy `/api/*` paths and the simple legacy error envelope:

```json
{ "error": "not_found" }
```

This intentionally differs from the platform target standards:

- Platform ADR0014 expects Business Capability API standardization.
- Platform ADR0015 expects major-versioned REST paths.
- Platform ADR0016 expects Problem Details for HTTP API errors.

Phase 01 observes those gaps; later phases decide which parts to practice,
mock, defer, or implement.

**Facade scaffold from day one**

`phase-01-monolith/facade/` ships a no-op nginx reverse proxy on port 8081. In
Phase 01 every route still goes to the monolith. Later phases can flip route
groups without introducing a new component under pressure.

---

## Implementation

### File Structure

```text
phase-01-monolith/
├── Dockerfile
├── docker-compose.yml
├── nginx.conf
├── openapi.yaml
├── routes-map.md
├── database/
│   ├── schema.sql
│   └── seed.sql
├── facade/
│   ├── README.md
│   ├── docker-compose.facade.yml
│   ├── feature-toggles.yaml
│   ├── nginx.conf
│   └── proxy_pass_headers.inc
└── php-app/
    ├── index.php
    ├── public/
    │   ├── index.html
    │   ├── css/
    │   └── js/
    └── src/
        ├── Database.php
        ├── Repository.php
        ├── Router.php
        ├── Controllers/
        └── Models/
```

### Runtime Stack

`docker compose up --build` starts:

| Service | Container | Purpose | Host port |
|---|---|---|---|
| `mic-app` | `mic-app` | nginx + PHP-FPM application | `8080` |
| `mysql` | `mic-mysql` | MySQL 8.0 database | `3306` |
| `adminer` | `mic-adminer` | Browser-based DB inspection | `8082` |

The facade overlay:

```bash
docker compose -f docker-compose.yml -f facade/docker-compose.facade.yml up
```

adds:

| Service | Container | Purpose | Host port |
|---|---|---|---|
| `facade` | `mic-facade` | no-op Strangler Fig reverse proxy | `8081` |

### Schema

`database/schema.sql` creates only two business tables:

```sql
CREATE TABLE business_data (...);
CREATE TABLE business_relations (...);
```

`database/seed.sql` loads representative customers, articles, price lists,
orders, invoices, stock movements, warehouses, users, and audit records.

### API Contract

The runtime API is routed under `/api/*`.

Common response envelopes:

| Endpoint type | Response |
|---|---|
| List | `{ "data": [...], "meta": { "total": n, "limit": n, "offset": n } }` |
| Detail | `{ "data": {...} }` |
| Create | HTTP `201` + `{ "data": {...} }` |
| Delete | HTTP `200` + `{ "data": { "id": n, "deleted": true } }` |
| Error | HTTP `4xx/5xx` + `{ "error": "..." }` |

`phase-01-monolith/openapi.yaml` documents the Warehouse-relevant legacy routes
that are candidates for Strangler Fig routing.

### Coupling Example

`php-app/src/Controllers/OrderController.php` is the clearest Phase 01 coupling
artifact:

- order lines are stored as `record_type = 'riga_ordine'`;
- each line relates to an `articolo` via `articolo_in_riga_ordine`;
- unit price defaults to `articolo.amount_1`;
- VAT code comes from `articolo.text_2`;
- VAT percentage is looked up in `record_type = 'aliquota_iva'`;
- order totals are recalculated in the same controller.

That flow crosses Orders, Warehouse/Catalog, and Pricing inside one controller
and one database transaction boundary. Later phases isolate those concerns.

### Facade Scaffold

`facade/feature-toggles.yaml` declares legacy route groups that make the intended Strangler Fig boundary visible from Phase 01:

| Route group | Legacy routes | Why it exists in Phase 01 |
|---|---|---|
| `articles` | `/api/articles*` | The Article aggregate is the first Go BC HTTP surface introduced in Phase 06. |
| `warehouses` | `/api/magazzini*` | Legacy Warehouse-adjacent routes kept visible for facade planning; the current Go progression does not implement these as a separate Phase 07 migration. |
| `inventory_levels` | `/api/magazzini/:id/giacenze` | Legacy stock-level route kept visible for parity thinking; the Go BC models inventory through Article aggregate operations such as `/articles/:article_id/inventory/adjust`. |

In Phase 01 every group is `mode: legacy`; nginx implements only passthrough. The toggle file intentionally carries no phase numbers; confirm implemented routes in the phase README and code.

---

## Consequences

### Positive Outcomes

- Learners can run and inspect a realistic "legacy enough" baseline quickly.
- The generic schema makes hidden bounded contexts visible through friction.
- The code contains concrete cross-domain reads instead of abstract examples.
- Adminer gives a no-local-MySQL path for inspecting rows and relationships.
- The facade shape is present before any route depends on it.
- Later Go phases can compare against a specific legacy API surface.

### Tradeoffs and Intentional Gaps

- The monolith is intentionally not platform-compliant.
- The application avoids framework conventions that would hide the request flow.
- There is no PHPUnit suite in the current Phase 01 artifact; smoke tests in
  the README are the participant-facing verification path.
- The schema is intentionally not the normalized Warehouse schema used by later
  Go phases.
- Authentication, authorization, OpenTelemetry, CloudEvents, and Problem
  Details are out of scope for Phase 01.
- Some Warehouse-adjacent routes, such as stock movements, remain in PHP for
  this exercise even though they are conceptually part of the broader domain.

---

## References

- `phase-01-monolith/README.md` - participant path for Phase 01.
- `phase-01-monolith/database/schema.sql` - polymorphic MIC schema.
- `phase-01-monolith/php-app/src/Repository.php` - generic persistence boundary.
- `phase-01-monolith/php-app/src/Controllers/ArticleController.php` - Warehouse/Catalog candidate route.
- `phase-01-monolith/php-app/src/Controllers/MagazzinoController.php` - Warehouse and stock-level candidate routes.
- `phase-01-monolith/php-app/src/Controllers/OrderController.php` - Orders-to-Warehouse coupling.
- `phase-01-monolith/openapi.yaml` - legacy Warehouse-relevant API contract.
- `phase-01-monolith/routes-map.md` - legacy route inventory and historical facade planning metadata.
- `phase-01-monolith/facade/README.md` - facade scaffold.
- `docs/adr/ADR-001-strangler-fig-pattern.md` - migration strategy.
- `docs/adr/ADR-012-facade-feature-toggle-canary.md` - facade and toggle design.
- `docs/platform-adr/README.md` - platform standards reference pack.

---

## Implementation Checkpoints

Phase 01 is aligned when:

1. `docker compose config` succeeds in `phase-01-monolith/`.
2. `docker compose -f docker-compose.yml -f facade/docker-compose.facade.yml config` succeeds.
3. `docker compose up --build` starts `mic-app`, `mysql`, and `adminer`.
4. `GET http://localhost:8080/api/dashboard/kpi` returns `{ "data": ... }`.
5. `GET http://localhost:8081/facade/health` returns facade health when the overlay is running.
6. The participant can explain why `OrderController::addRiga()` depends on article and VAT data owned by other candidate contexts.
