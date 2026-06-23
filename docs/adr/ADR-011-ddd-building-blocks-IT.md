# ADR-011: I building block del DDD come vocabolario del codice

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

L'ADR-010 vincola il team a Event Storming come metodo per individuare i BC. Event Storming produce aggregati ed eventi al passato. Una volta disegnato il canvas, quel vocabolario deve sopravvivere fino al codice, altrimenti l'output del workshop diventa pura decorazione.

L'attuale scheletro Go (`phase-03-skeleton/entities/`) usa nomi neutri: `Article`, `Inventory`. Tali nomi nascondono quale oggetto sia l'unità di cambiamento, quale porti un'identità, quale sia rimpiazzabile e quale rappresenti un evento. Di conseguenza, chi mantiene il codice in futuro non riesce a capire dal nome del file se modificare una struct rompe un'invariante o semplicemente rimpiazza un valore.

Il Modulo 2.3 prescrive quattro building block. Il repository deve adottarli per nome:

- **Aggregate** — cluster di oggetti correlati trattato come una singola unità transazionale. Un aggregate root controlla l'accesso. L'aggregate fa rispettare le invarianti.
- **Entity** — oggetto con identità persistente. Due entity con gli stessi dati e id diversi sono cose diverse.
- **Value Object** — oggetto definito esclusivamente dai propri dati. Niente id. Si rimpiazza, non si muta.
- **Domain Event** — fatto al passato emesso da un aggregate. Immutabile. Altri BC possono sottoscriverlo.

## Decision

Il codice Go a partire da `phase-03-skeleton/` in avanti usa direttamente questi quattro nomi:

- `entities/article.go` dichiara `Article` come **aggregate root** dell'aggregate Warehouse. I chiamanti esterni parlano sempre e solo con il root.
- `entities/sku.go` dichiara `SKU` come **value object** (senza id, uguaglianza per dati, validato in costruzione).
- `entities/money.go` dichiara `Money` come **value object**.
- `entities/inventory.go` dichiara `Inventory` come **entity** all'interno dell'aggregate (identità persistente, ma accesso attraverso `Article`).
- Un nuovo package `events/` dichiara un tipo per ogni **domain event**: `ArticleCreated`, `InventoryAdjusted`, `StockReserved`. Ogni evento è una struct, ha un nome al passato e un campo `OccurredAt time.Time`.

I repository caricano e salvano solo aggregate. Non esistono `SKURepository` o `MoneyRepository`: i value object si persistono come parte dell'aggregate a cui appartengono.

Gli use case mutano aggregate e ritornano `(Article, []DomainEvent)` così che il layer HTTP (o un event publisher) possa distribuire gli eventi. Gli aggregate non pubblicano eventi da soli: li registrano.

## Consequences

**Positive:**
- Chi legge il codice sa rispondere "che cos'è questa cosa?" già dal nome del file.
- Il legame dal canvas di Event Storming al codice è diretto: ogni post-it arancione diventa una struct in `events/`.
- Il refactoring è vincolato: un value object non può guadagnare un id senza diventare un'entity, e questo è un segnale chiaro per la code review.

**Negative:**
- Il team deve imparare quattro termini nuovi. Mitigazione: ADR-011 definisce i building block; `phase-02-analysis/tactical-ddd-warehouse.md` ospita la mappatura applicata al Warehouse e `phase-02-analysis/ubiquitous-language.md` è il vocabolario business canonico.
- Rinominare le entity esistenti tocca ogni fase a partire dalla 03. Mitigazione: quel refactor è schedulato come Stream B Plan 2.

## Implementation Reference

- `phase-02-analysis/ubiquitous-language.md` — vocabolario business canonico del Warehouse (DDD strategico).
- `phase-02-analysis/tactical-ddd-warehouse.md` — building block DDD tattici (Aggregate / Entity / VO / Domain Event) applicati al Warehouse.
- `phase-03-skeleton/entities/` — aggregate root e value object *(planned — Stream B Plan 2)*.
- ADR-014 (Data Products) consuma i domain event come fatti pubblicati.
