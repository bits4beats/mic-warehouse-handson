# ADR-007: Implementazione Go Clean Architecture - dallo skeleton al servizio completo

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Contesto

Il pattern Strangler Fig (ADR-001) richiede di estrarre la capability Warehouse dal monolite PHP in un servizio Go separato. Il servizio Go deve crescere phase dopo phase, mantenendo l'esercizio eseguibile e comprensibile a ogni checkpoint.

Il servizio deve:

1. Girare in modo indipendente dalla Phase 03 in poi.
2. Preservare API e semantica business Warehouse introdotte dal baseline monolitico.
3. Tenere le regole di dominio isolate da HTTP, database, autenticazione, autorizzazione e trasporto eventi.
4. Insegnare Clean Architecture in modo incrementale, invece di presentare subito un servizio completo.
5. Rendere ogni layer testabile attraverso confini espliciti.

Go è scelto perché supporta un runtime deployabile piccolo, dependency inversion tramite interface, testing lineare e pattern di servizio production-grade senza richiedere un framework pesante.

Il repository corrente usa una **struttura piatta per phase** invece di un albero `internal/`. È una scelta intenzionale per l'esercizio: i partecipanti possono vedere direttamente i layer dentro ogni directory di phase.

---

## Decisione

Implementeremo il servizio Warehouse come applicazione Go in Clean Architecture, costruita incrementalmente lungo le Phase 03-10.

Le decisioni architetturali chiave sono:

1. `Article` è l'Aggregate root di Warehouse.
2. `InventoryLevel` è posseduto dall'aggregate `Article` e viene modificato tramite comportamento a livello di aggregate.
3. `ArticleRepository` è il repository port per persistere e caricare l'aggregate.
4. Gli use case orchestrano costruzione dell'aggregate, mutazioni, persistenza e dispatch degli eventi.
5. Gli handler HTTP sono adapter: traducono request in input degli use case e output degli use case in response.
6. Autenticazione, autorizzazione, observability e pubblicazione eventi sono responsabilità di edge, cablate fuori dal dominio.
7. Il servizio resta eseguibile phase per phase anche quando alcune concern production sono rappresentate da adapter locali o scope gap espliciti.

### Progressione delle Phase

```text
Phase 03: Fondazione di dominio
  entities/, events/, interfaces/, main.go

Phase 04: Adapter di persistenza
  repositories/

Phase 05: Use case e confine event dispatcher
  usecases/, dispatcher/

Phase 06: Adapter API HTTP
  handlers/

Phase 07: Autenticazione e request context
  middleware/, iam-mock/

Phase 08: Autorizzazione e decisioni OPA/Rego
  policies/, middleware/

Phase 09: Pubblicazione eventi Hermes/Data Product
  events/, schemas/

Phase 10: Servizio integrato
  handlers/, middleware/, policies/, events/, observability/, tests/, backoffice/
```

---

## Forma Architetturale

I layer non sono implementati come deployable indipendenti. Sono package dentro lo stesso servizio Go, con dipendenze dirette verso l'interno.

```text
HTTP / Auth / Policy / Observability / Events
                 |
              Use case
                 |
          Repository port
                 |
       Entity e Value Object
```

I package esterni possono dipendere dai package interni. I package interni non devono dipendere dai package esterni.

### Responsabilità dei Package

| Package | Responsabilità | Introdotto |
|---|---|---|
| `entities/` | Aggregate root, entity, value object, invarianti locali | Phase 03 |
| `events/` | Domain event e, più avanti, adapter Hermes/Data Product | Phase 03, esteso in Phase 09 |
| `interfaces/` | Repository port per la persistenza dell'aggregate | Phase 03 |
| `repositories/` | Adapter MySQL, adapter in-memory, decoratore dual-write, divergence logger | Phase 04 |
| `dispatcher/` | Event dispatch port usato dagli use case | Phase 05 |
| `usecases/` | Workflow applicativi e orchestrazione business | Phase 05 |
| `handlers/` | Adapter HTTP Echo e registrazione route | Phase 06 |
| `middleware/` | Auth, M2M, correlation, request context | Phase 07 |
| `policies/` | Policy enforcer in-memory e adapter OPA client | Phase 08 |
| `schemas/` | Schemi Data Product locali usati dall'adapter Hermes | Phase 09 |
| `observability/` | Request logging e supporto trace/correlation | Phase 10 |
| `tests/` | Test in stile E2E per il servizio integrato | Phase 10 |

---

## Mappa Implementativa

### Phase 03: Fondazione di Dominio

**Obiettivo:** stabilire il modello di dominio e il primo confine di dependency inversion.

File chiave:

- `phase-03-skeleton/entities/article.go`
- `phase-03-skeleton/entities/inventory.go`
- `phase-03-skeleton/entities/money.go`
- `phase-03-skeleton/entities/sku.go`
- `phase-03-skeleton/events/events.go`
- `phase-03-skeleton/interfaces/repository.go`
- `phase-03-skeleton/main.go`

Scelte importanti:

- `Article` è l'Aggregate root.
- `Money` e `SKU` sono Value Object con validazione locale.
- `InventoryLevel` non viene salvato tramite un repository indipendente.
- `ArticleRepository` espone metodi a livello di aggregate: `Save`, `FindByID`, `FindBySKU`, `List`, `Delete`.
- Gli aggregate registrano pending event; non li pubblicano.
- `main.go` espone solo un health endpoint Echo minimale, così lo skeleton resta eseguibile.

## Strategia di Test

L'esercizio usa i test per validare ogni confine al livello in cui conta:

| Area | Stile di test |
|---|---|
| Entity e Value Object | Unit test su invarianti e comportamento di mutazione |
| Repository port | Check compile-time dell'interface e test repository mirati |
| Use case | Unit test con repository e dispatcher in-memory |
| Handler | Test HTTP handler con cablaggio use case in-memory |
| Middleware | Test su token, M2M, correlation e skip path |
| Policy | Test policy in-memory e test decisioni OPA client |
| Eventi | Test su validazione schema ed envelope CloudEvents |
| Integrazione | Flusso E2E con `httptest` in Phase 10 |

I test devono preservare la regola Clean Architecture: i layer interni vengono testati senza adapter esterni.

---

## Conseguenze

### Risultati Positivi

- I partecipanti possono ispezionare una nuova concern architetturale per phase.
- Il modello di dominio resta indipendente da database, HTTP, auth, policy e dettagli di trasporto eventi.
- Gli use case restano testabili perché repository, dispatcher e decisioni policy sono rappresentati da confini espliciti.
- La regola dell'Aggregate root viene rinforzata facendo passare i cambi inventory attraverso `Article`.
- Le concern platform successive possono essere insegnate tramite adapter locali senza cambiare il modello di dominio.

### Tradeoff

- La struttura piatta dei package è più didattica di un layout production convenzionale con `internal/`.
- Alcune integrazioni production sono rappresentate da adapter locali o mock.
- Il dual-write non è completo finché non esiste un vero adapter di traduzione legacy.
- I partecipanti devono distinguere i layer Clean Architecture dagli adapter cross-cutting come auth, policy, observability ed eventi.

---

## Riferimenti

**ADR Exercise Correlate**

- ADR-001: Strangler Fig Pattern
- ADR-002: Clean Architecture Layers
- ADR-006: MIC PHP Monolith Baseline
- ADR-008: MySQL Repository Pattern
- ADR-009: HTTP Handlers + Echo
- ADR-011: DDD Building Blocks
- ADR-013: Dual-Write Pattern
- ADR-014: Data Products on Hermes
- ADR-016: Pact Testing and API Deprecation
- ADR-017: Observability

**Riferimenti Phase Chiave**

- `phase-03-skeleton/README.md`

---

## Criteri di Successo

Alla fine dell'esercizio:

1. Un partecipante sa spiegare perché `Article` è l'Aggregate root.
2. Un partecipante sa indicare il repository port e la sua implementazione MySQL.
3. Un partecipante sa spiegare perché gli handler chiamano gli use case invece dei repository.
4. Un partecipante sa spiegare perché OPA, Hermes, auth e observability restano al bordo.
5. Un partecipante sa eseguire i test per ogni layer e interpretare quale confine protegge ciascun test.
6. Un partecipante sa identificare lo scope gap dual-write corrente e l'adapter legacy mancante necessario per chiuderlo.
