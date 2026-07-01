# ADR-014: I Data Product come contratti Hermes versionati

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

L'ADR-004 vincola il team a Hermes per l'event streaming. Quell'ADR tratta gli eventi come messaggi: payload tipizzati emessi dal Warehouse e consumati dalle parti interessate. Il Modulo 2.2 della formazione alza l'asticella: gli eventi su Hermes non sono soltanto messaggi, sono **Data Product** — definizioni di dati validate da schema, versionate, contribuite al datalake, con garanzie di replay e un ciclo di vita pubblicato.

Oggi il codice pubblica `ArticleCreated` e `InventoryUpdated` come semplici struct Go. Non c'è uno schema registry, non c'è una politica di versionamento, non c'è un ciclo di vita per le versioni di evento deprecate. Un consumer che legge il codice non ha modo di sapere se un campo sia obbligatorio, se stia arrivando una nuova versione, o se il producer abbia intenzione di rimuovere un campo nel prossimo trimestre.

## Decision

Ogni domain event (ADR-011) che attraversa un confine di BC su Hermes viene trattato come un Data Product, con i seguenti obblighi:

1. **Schema.** Ogni evento ha un JSON Schema (Draft 2020-12) versionato in `phase-09-events/schemas/`. Lo schema è il contratto. I producer validano prima di pubblicare; i consumer validano prima di consumare.
2. **Versioning.** Gli schema sono versionati (`v1`, `v2`, …). Aggiungere campi opzionali è un cambiamento non breaking. Rinominare o rimuovere campi richiede una nuova major version.
3. **Lifecycle.** Ogni schema dichiara nei metadata il proprio stage di lifecycle (`stable`, `deprecated`, `sunset`). Gli schema deprecated portano un campo `sunset_date`.
4. **Replay.** Hermes garantisce il replay storico dall'event log. I consumer possono ricostruire lo stato da `t=0` per ogni data product a cui sono iscritti.

L'artefatto pubblicato è lo schema, non la struct Go. I consumer in altri linguaggi (PHP, Python) leggono lo schema, non il sorgente del producer.

## Consequences

**Positive:**
- L'integrazione cross-team diventa una negoziazione di contratto, non un messaggio su Slack.
- I consumer rilevano i cambiamenti incompatibili del producer al deploy (la validazione schema fallisce) anziché in produzione (il parsing crasha).
- Chi entra in ritardo può ricostruire lo stato storico facendo il replay dell'event log contro il validatore dello schema più recente.

**Negative:**
- I producer non possono far evolvere gli schema con leggerezza. Un nuovo campo required è un bump di major version.
- Il team deve mantenere uno schema registry. Per l'esercizio, il registry è la directory `schemas/` più la storia git. In produzione sarebbe un registry gestito.

**Neutral:**
- L'ADR-016 (Pact + openapi-diff + Deprecation) governa i contratti API. L'ADR-014 governa i contratti di evento. I due ADR condividono di proposito il vocabolario di lifecycle.

## Implementation Reference

- `phase-09-events/schemas/article-dp-v1.json` — schema v1 dell'Article Data Product *(planned — Stream B Plan 3)*.
- `phase-09-events/schemas/inventory-dp-v1.json` — schema v1 dell'Inventory Data Product *(planned — Stream B Plan 3)*.
- `phase-09-events/events/data_product.go` — middleware di validazione schema *(planned — Stream B Plan 3)*.
- `lab-2B-contracts-decoupling/01-data-products/` — variante standalone per il lab con demo di consumer e replay *(planned — Stream A Plan 5)*.
