# ADR-006: Baseline del monolite PHP MIC

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

L'esercizio parte da `phase-01-monolith/`, un monolite PHP volutamente disordinato
chiamato MIC. MIC è un SaaS fittizio di fatturazione per PMI italiane ed è lo
stato "before" che la migrazione Strangler Fig sostituisce gradualmente.

L'implementazione attuale è volutamente piccola ed esplicita. È
un'applicazione PHP 8.2 in cui routing, dispatch verso i controller,
persistenza e coupling cross-domain sono visibili nel repository invece di
essere nascosti da convenzioni di framework. Ha:

- un container nginx + php-fpm;
- un front controller custom in `php-app/index.php`;
- un piccolo router custom in `php-app/src/Router.php`;
- controller sotto `php-app/src/Controllers/`;
- un repository generico sopra due tabelle polimorfiche;
- MySQL 8.0 seedato da `database/schema.sql` e `database/seed.sql`;
- uno scaffold nginx facade no-op sotto `facade/`.

Phase 01 esiste per rendere visibile il coupling prima che i partecipanti
provino a estrarre una Warehouse Business Capability in Go. Il monolite mescola
intenzionalmente dati di Catalog, Logistics/Warehouse, Orders, Invoicing,
Payments, Customers, Pricing e System nello stesso modello di persistenza.

Le due tabelle centrali sono:

- `business_data`: una sola forma di riga per 16+ tipi di record di business;
- `business_relations`: una sola tabella relazionale per tutti i collegamenti cross-record.

Questo anti-pattern è il punto della phase. I partecipanti devono osservare come
uno schema generico renda difficili da vedere i confini di ownership, poi usare
Phase 02 per trasformare quel caos in bounded context DDD e in un extraction plan.

---

## Decision

Manterremo Phase 01 come **monolite MIC in PHP puro con schema polimorfico**,
anziché come servizio di riferimento basato su framework.

Questa decisione locale dell'esercizio offre ai partecipanti:

1. Un baseline legacy concreto da eseguire in Docker.
2. Un anti-pattern database visibile: molti domini nascosti in due tabelle generiche.
3. Coupling cross-domain diretto da ispezionare nel codice PHP e in SQL.
4. Un contratto legacy stabile `/api/*` da confrontare con le phase successive.
5. Una facade Strangler Fig no-op disponibile dal primo giorno.

### Architecture Decision

```text
┌────────────────────────────────────────────┐
│ Browser / smoke-test client                │
├────────────────────────────────────────────┤
│ nginx                                      │
│ - serve asset statici della SPA            │
│ - inoltra le richieste API a PHP-FPM       │
├────────────────────────────────────────────┤
│ PHP front controller (`php-app/index.php`) │
│ - router custom                            │
│ - dispatch verso i controller              │
├────────────────────────────────────────────┤
│ Controller                                 │
│ - mappano campi DTO leggibili              │
│ - interpretano colonne amount_N/text_N     │
│ - contengono logica volutamente accoppiata │
├────────────────────────────────────────────┤
│ Repository generico                        │
│ - `business_data`                          │
│ - `business_relations`                     │
├────────────────────────────────────────────┤
│ MySQL 8.0                                  │
└────────────────────────────────────────────┘
```

### Key Design Choices

**Applicazione PHP piccola ed esplicita**

L'applicazione usa un piccolo router e controller diretti. Questo mantiene
l'attenzione del partecipante sul coupling e sulla migrazione: si vede
esattamente dove entra una richiesta, quale controller la gestisce e dove le
letture database attraversano i confini dei candidate context.

**Modello database polimorfico**

Tutti i record di business vivono in `business_data`; il significato di
`amount_1`, `text_2`, `date_1` e colonne simili dipende da `record_type`.

Esempi di mapping:

| `record_type` | Colonne generiche con significato di business |
|---|---|
| `articolo` | `code = SKU`, `name = nome articolo`, `amount_1 = prezzo listino`, `amount_2 = scorta minima`, `text_1 = categoria`, `text_2 = codice IVA` |
| `ordine` | `code = numero ordine`, `amount_1 = imponibile`, `amount_2 = IVA`, `amount_3 = totale`, `status = stato workflow` |
| `riga_ordine` | `parent_id = ordine`, `amount_1 = quantità`, `amount_2 = prezzo unitario`, `amount_3 = sconto`, `amount_4 = totale riga` |
| `magazzino` | `code = codice magazzino`, `name = nome magazzino`, `text_1 = città`, `text_2 = CAP` |

**Letture cross-domain dirette**

Orders legge direttamente dati Warehouse/Catalog. Per esempio,
`OrderController::addRiga()` carica una riga `articolo` per derivare il prezzo
unitario di default e legge `articolo.text_2` per cercare l'aliquota IVA. Questo
è il coupling che i partecipanti devono notare prima di estrarre Warehouse.

**Contratto legacy, non contratto BC platform-compliant**

Phase 01 mantiene i path legacy `/api/*` e il semplice envelope errore legacy:

```json
{ "error": "not_found" }
```

Questo differisce intenzionalmente dagli standard target di piattaforma:

- Platform ADR0014 si aspetta la standardizzazione delle Business Capability API.
- Platform ADR0015 si aspetta path REST con major version esplicita.
- Platform ADR0016 si aspetta Problem Details per gli errori HTTP API.

Phase 01 osserva questi gap; le phase successive decidono quali parti praticare,
mockare, differire o implementare.

**Scaffold facade dal primo giorno**

`phase-01-monolith/facade/` pubblica un reverse proxy nginx no-op sulla porta
8081. In Phase 01 ogni rotta va ancora al monolite. Le phase successive possono
cambiare i route group senza introdurre un nuovo componente sotto pressione.

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

`docker compose up --build` avvia:

| Service | Container | Scopo | Porta host |
|---|---|---|---|
| `mic-app` | `mic-app` | Applicazione nginx + PHP-FPM | `8080` |
| `mysql` | `mic-mysql` | Database MySQL 8.0 | `3306` |
| `adminer` | `mic-adminer` | Ispezione DB da browser | `8082` |

L'overlay della facade:

```bash
docker compose -f docker-compose.yml -f facade/docker-compose.facade.yml up
```

aggiunge:

| Service | Container | Scopo | Porta host |
|---|---|---|---|
| `facade` | `mic-facade` | Reverse proxy Strangler Fig no-op | `8081` |

### Schema

`database/schema.sql` crea solo due tabelle di business:

```sql
CREATE TABLE business_data (...);
CREATE TABLE business_relations (...);
```

`database/seed.sql` carica clienti, articoli, listini, ordini, fatture,
movimenti di stock, magazzini, utenti e audit record rappresentativi.

### API Contract

La runtime API è esposta sotto `/api/*`.

Envelope di risposta comuni:

| Tipo endpoint | Risposta |
|---|---|
| List | `{ "data": [...], "meta": { "total": n, "limit": n, "offset": n } }` |
| Detail | `{ "data": {...} }` |
| Create | HTTP `201` + `{ "data": {...} }` |
| Delete | HTTP `200` + `{ "data": { "id": n, "deleted": true } }` |
| Error | HTTP `4xx/5xx` + `{ "error": "..." }` |

`phase-01-monolith/openapi.yaml` documenta le rotte legacy Warehouse-relevant
candidate al routing Strangler Fig.

### Coupling Example

`php-app/src/Controllers/OrderController.php` è l'artefatto di coupling più
chiaro di Phase 01:

- le righe ordine sono salvate come `record_type = 'riga_ordine'`;
- ogni riga è collegata a un `articolo` tramite `articolo_in_riga_ordine`;
- il prezzo unitario di default arriva da `articolo.amount_1`;
- il codice IVA arriva da `articolo.text_2`;
- la percentuale IVA viene cercata in `record_type = 'aliquota_iva'`;
- i totali ordine vengono ricalcolati nello stesso controller.

Questo flow attraversa Orders, Warehouse/Catalog e Pricing dentro lo stesso
controller e lo stesso confine transazionale del database. Le phase successive
isolano queste responsabilità.

### Facade Scaffold

`facade/feature-toggles.yaml` dichiara route group legacy che rendono visibile il confine Strangler Fig già in Phase 01:

| Route group | Rotte legacy | Perché esiste in Phase 01 |
|---|---|---|
| `articles` | `/api/articles*` | L'Aggregate Article è la prima superficie HTTP del Go BC introdotta in Phase 06. |
| `warehouses` | `/api/magazzini*` | Rotte legacy vicine a Warehouse mantenute visibili per pianificare la facade; la progressione Go corrente non le implementa come migrazione separata in Phase 07. |
| `inventory_levels` | `/api/magazzini/:id/giacenze` | Rotta legacy di stock-level mantenuta visibile per ragionare sulla parity; il Go BC modella l'inventory tramite operazioni sull'Aggregate Article, come `/articles/:article_id/inventory/adjust`. |

In Phase 01 ogni group è `mode: legacy`; nginx implementa solo passthrough. Il file dei toggle non contiene intenzionalmente numeri di phase; conferma le rotte implementate nel README e nel codice della phase.

---

## Consequences

### Esiti positivi

- I partecipanti possono eseguire e ispezionare rapidamente un baseline abbastanza legacy da essere realistico.
- Lo schema generico rende visibili bounded context nascosti attraverso la frizione.
- Il codice contiene letture cross-domain concrete, non esempi astratti.
- Adminer offre un percorso senza client MySQL locale per ispezionare righe e relazioni.
- La forma della facade è presente prima che qualunque rotta dipenda da essa.
- Le phase Go successive possono confrontarsi con una specifica superficie API legacy.

### Tradeoff e gap intenzionali

- Il monolite è intenzionalmente non platform-compliant.
- L'applicazione evita convenzioni di framework che nasconderebbero il request flow.
- Non esiste una suite PHPUnit nell'artefatto corrente di Phase 01; gli smoke test
  nel README sono il percorso di verifica per i partecipanti.
- Lo schema è intenzionalmente diverso dallo schema Warehouse normalizzato usato
  dalle phase Go successive.
- Authentication, authorization, OpenTelemetry, CloudEvents e Problem Details
  sono fuori scope per Phase 01.
- Alcune rotte vicine a Warehouse, come i movimenti di stock, restano in PHP per
  questo esercizio anche se concettualmente appartengono al dominio più ampio.

---

## References

- `phase-01-monolith/README-IT.md` - percorso partecipante per Phase 01.
- `phase-01-monolith/database/schema.sql` - schema MIC polimorfico.
- `phase-01-monolith/php-app/src/Repository.php` - confine di persistenza generico.
- `phase-01-monolith/php-app/src/Controllers/ArticleController.php` - rotta candidata Warehouse/Catalog.
- `phase-01-monolith/php-app/src/Controllers/MagazzinoController.php` - rotte candidate Warehouse e stock-level.
- `phase-01-monolith/php-app/src/Controllers/OrderController.php` - coupling Orders-to-Warehouse.
- `phase-01-monolith/openapi.yaml` - contratto API legacy Warehouse-relevant.
- `phase-01-monolith/routes-map.md` - inventario delle rotte legacy e metadati storici di pianificazione della facade.
- `phase-01-monolith/facade/README.md` - scaffold facade.
- `docs/adr/ADR-001-strangler-fig-pattern-IT.md` - strategia di migrazione.
- `docs/adr/ADR-012-facade-feature-toggle-canary-IT.md` - design di facade e toggle.
- `docs/platform-adr/README.md` - reference pack degli standard di piattaforma.

---

## Implementation Checkpoints

Phase 01 è allineata quando:

1. `docker compose config` passa in `phase-01-monolith/`.
2. `docker compose -f docker-compose.yml -f facade/docker-compose.facade.yml config` passa.
3. `docker compose up --build` avvia `mic-app`, `mysql` e `adminer`.
4. `GET http://localhost:8080/api/dashboard/kpi` ritorna `{ "data": ... }`.
5. `GET http://localhost:8081/facade/health` ritorna la health della facade quando l'overlay è in esecuzione.
6. Il partecipante sa spiegare perché `OrderController::addRiga()` dipende da dati articolo e IVA posseduti da altri candidate context.
