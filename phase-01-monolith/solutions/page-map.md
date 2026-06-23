# Page map — MIC

> Reference deliverable (Phase 01 solution): what a strong room converges on. Open only after you
> have produced your own. The GUI is a vanilla-JS SPA with a hash router (`#/dashboard`,
> `#/orders`, ...); each section is one JS module in `php-app/public/js/`.

| Section | Route | What it is for | Backed by |
|---|---|---|---|
| **Dashboard** | `#/dashboard` | KPIs (monthly revenue, active orders, articles below min stock, overdue payments) + charts + last audit activity | `dashboard.js` → `GET /api/dashboard/kpi` |
| **Customers** | `#/customers` | Customer list + detail with tabs (master data, orders, invoices, payments) | `customers.js` → `/api/customers`, `/api/customers/:id/orders`, `/api/customers/:id/invoices` |
| **Suppliers** | `#/suppliers` | Supplier list + master-data form | `suppliers.js` → `/api/suppliers` |
| **Articles** | `#/articles` | Catalog: list, status filter, create/edit (SKU, category, VAT, price, min stock) | `articles.js` → `/api/articles` |
| **Price lists (Listini)** | `#/listini` | Price lists + line items (`voci`), add-item form | `listini.js` → `/api/listini`, `/api/listini/:id/voci` |
| **Orders (Ordini)** | `#/orders` | Order header (customer, price list, agent, status) + inline lines (`righe`) + "generate invoice" | `orders.js` → `/api/orders`, `/api/orders/:id/righe` |
| **Invoices** | `#/invoices` | Invoice list + detail + "Send to SDI" (mock) + record payment | `invoices.js` → `/api/invoices`, `/api/invoices/:id/invia-sdi` |
| **Warehouse (Magazzino)** | `#/magazzino` | Warehouses, stock levels (`giacenze`), movements | `magazzino.js` → `/api/magazzini`, `/api/magazzini/:id/giacenze`, `/api/movimenti` |
| **Discounts (Sconti)** | `#/sconti` | Promo codes (CRUD) | `sconti.js` → `/api/sconti` |
| **Agents (Agenti)** | `#/agenti` | Agent master data (zone, commission %) | `agenti.js` → `/api/agenti` |
| **Settings** | `#/settings` | Article categories, VAT rates (`iva-rates`), users | `settings.js` → `/api/categories`, `/api/iva-rates`, `/api/users` |

**What to notice:** the GUI presents ten clean business areas. The database underneath does **not**
mirror them: every screen above reads and writes the same two generic tables (see
[`coupling-map.md`](./coupling-map.md)). The page map is the "intended" decomposition; the coupling
map is the real one.
