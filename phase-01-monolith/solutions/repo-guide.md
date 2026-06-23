# Repo guide — `phase-01-monolith/`

> Reference deliverable (Phase 01 solution): what a strong room converges on. Open only after you
> have produced your own.

## Top level

| Path | What it is |
|---|---|
| `docker-compose.yml` | The stack: `mic-app` (nginx + php-fpm), `mysql` 8.0, `adminer`. Mounts `database/*.sql` as init scripts. |
| `Dockerfile` | Builds the `mic-app` image (nginx + php-fpm 8.2 supervised in one container). |
| `nginx.conf` | Web server: static assets from `public/`, everything else to the PHP front controller via fast-cgi. |
| `openapi.yaml` | The legacy API contract (unversioned `/api/*`, `{ data }` / `{ data, meta }` / `{ error }` envelopes). |
| `php-app/` | The application (see below). |
| `database/` | `schema.sql` (the two generic tables) + `seed.sql` (demo data). |

## `php-app/` — the application

| Path | What it does |
|---|---|
| `index.php` | **Front controller.** Routes `/api/*` to the Router + controllers; serves the SPA shell and static assets otherwise. Registers a CRUD route set per resource. |
| `src/Router.php` | Tiny regex router (`/api/foo/:id`). No framework. |
| `src/Database.php` | PDO connection from `DB_*` env vars. |
| `src/Repository.php` | The data-access layer: `findById`, `create`, `update`, `relate`, `rawOne`, `rawAll`. **All persistence goes through `business_data` / `business_relations`.** |
| `src/Models/BusinessData.php` | Active-record-ish wrapper over one `business_data` row (generic columns). |
| `src/Models/BusinessRelation.php` | Wrapper over one `business_relations` row. |
| `src/Controllers/` | One controller per resource (18 of them). Each declares its `record_type` and maps generic columns ↔ a domain-shaped DTO. |
| `public/index.html` | SPA shell. |
| `public/css/style.css` | Styles. |
| `public/js/app.js` | SPA core: hash router, table/modal helpers, view registry. |
| `public/js/<section>.js` | One module per section (`dashboard`, `customers`, `suppliers`, `articles`, `listini`, `orders`, `invoices`, `magazzino`, `sconti`, `agenti`, `settings`). |

## Controllers (the 18)

`Agente`, `Article`, `Audit`, `Base`, `Category`, `Customer`, `Dashboard`, `Invoice`, `Iva`,
`Listino`, `Magazzino`, `Movimento`, `NotaCredito`, `Order`, `Pagamento`, `Sconto`, `Supplier`,
`User`. `BaseController` holds the shared CRUD + relation-lookup helpers; each subclass sets its
`type()` (the `record_type`) and its `toDto` / `fromPayload` column mapping.

**What to notice:** there is **no domain layer**. The dependency direction is
`controller → repository → raw SQL → two generic tables`. Business rules (order totals, VAT lookup,
price defaulting) live inside controller methods, not in any model that owns them. See
[`architecture.md`](./architecture.md) and [`worst-antipatterns.md`](./worst-antipatterns.md).
