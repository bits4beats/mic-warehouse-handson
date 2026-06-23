# ADR-001: Strangler Fig Pattern per l'estrazione del Warehouse BC

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

L'applicazione monolitica PHP accoppia strettamente tre Business Capability critiche: Orders, Invoicing e Warehouse. Questi moduli condividono tabelle di database, logica di business ed endpoint API, creando dipendenze forti che rendono impossibile evolverli in modo indipendente.

Il modulo Warehouse, responsabile di inventory management, stock tracking e definizione degli articoli, è un candidato naturale per l'estrazione in un bounded context separato. Tuttavia, una riscrittura "big-bang" comporta rischi sostanziali:

- **Downtime:** coordinare un cutover completo richiede di sincronizzare Orders, Invoicing e Warehouse nello stesso momento.
- **Rischio di regressione:** migrare anni di logica legacy aumenta la superficie in cui possono apparire bug.
- **Perdita di contesto tra team:** due team che lavorano in parallelo senza authority condivisa sul deploy generano overhead di coordinamento.
- **Complessità di rollback:** se il nuovo servizio fallisce, tornare al monolite diventa architetturalmente complicato.

Il business richiede:

- **Estrazione zero-downtime:** Orders e Invoicing continuano a funzionare senza interruzioni.
- **Costruzione graduale della fiducia:** il comportamento del nuovo servizio viene validato contro il servizio legacy prima del cutover completo.
- **Rollback semplice:** se il servizio estratto fallisce in qualunque phase, si torna al monolite senza frizione.
- **Affidabilità production-grade:** gestione sicura di richieste concorrenti da più client.

Il team non ha ancora esperienza pratica con refactoring architetturali incrementali su larga scala. Questa ADR stabilisce un pattern collaudato che insegna sia l'esecuzione tecnica sia il change management organizzativo.

---

## Decision

Adotteremo lo **Strangler Fig Pattern** per sostituire gradualmente il modulo Warehouse con un nuovo servizio Go, mantenendo piena backward compatibility e operatività zero-downtime.

Lo Strangler Fig Pattern prevede:

1. Costruire il nuovo servizio accanto al monolite.
2. Instradare una percentuale di traffico verso il nuovo servizio, mantenendo il monolite come fallback.
3. Aumentare gradualmente la percentuale di traffico man mano che cresce la fiducia.
4. Smantellare il modulo Warehouse del monolite solo dopo il cutover completo.

Questo approccio è comprovato in migrazioni reali verso microservizi (Amazon, Netflix, Shopify) e offre:

- **Safety:** il dual-write garantisce consistenza dei dati mentre si valida il comportamento del nuovo servizio.
- **Confidence building:** il percentile routing permette di validare in produzione con blast radius limitato.
- **Semplicità di rollback:** qualunque phase può tornare immediatamente al monolite.
- **Autonomia dei team:** i team Orders e Invoicing continuano a deployare in modo indipendente.

---

## Implementation

### Phase 1: Parallel Operation (settimane 1-3)

**Obiettivo:** il nuovo servizio Go accetta traffico in modalità read-only mentre il monolite continua a gestire tutte le operazioni.

**Setup tecnico:**

- Deploy del servizio Go Warehouse, inizialmente ancora instabile.
- Implementazione del dual-write a livello monolite: le scritture vanno sia allo stato legacy sia allo stato interno del servizio Go.
- Deploy di un routing proxy layer che:
  - instrada le richieste `GET /warehouse/*` al servizio Go, con fallback al monolite in caso di errore;
  - instrada le richieste `POST/PUT/DELETE /warehouse/*` esclusivamente al monolite;
  - registra tutte le decisioni di routing per audit trail.
- Implementazione di health check: il servizio Go comunica al routing proxy la propria percentuale di readiness.

**Validazione:**

- I test unitari sul nuovo servizio Go passano senza dipendenze esterne.
- I test di integrazione validano la sincronizzazione dei dati tra monolite e servizio Go.
- I load test read-only confrontano i tempi di risposta tra monolite e servizio Go.
- Shadow traffic da staging: payload simili alla produzione vengono inviati al servizio Go, scartando le risposte.
- QA manuale: gli sviluppatori possono abilitare una feature flag per instradare il proprio traffico al servizio Go.

**Criteri di successo:**

- Il 100% delle letture può essere servito dal servizio Go con differenza di latenza inferiore al 5%.
- Nessuna perdita dati durante 48 ore continuative di dual-write.
- Tutti i test di integrazione passano in ambiente staging.

### Phase 2: Percentile Routing (settimane 4-6)

**Obiettivo:** instradare percentuali crescenti di traffico di produzione verso il servizio Go, includendo letture e scritture.

**Setup tecnico:**

- Estensione del dual-write per includere `POST/PUT/DELETE` nel servizio Go accanto alle scritture nel monolite.
- Il routing proxy instrada in base a percentuali configurabili: 10% -> 25% -> 50% -> 75% -> 90%.
- Implementazione del distributed tracing: correlare le risposte del monolite e del servizio Go per la stessa richiesta logica.
- Metrics dashboard: confrontare latenza, error rate e metriche di consistenza tra i servizi.

**Validazione:**

- Load test a ogni livello percentuale: 10%, 25%, 50%.
- Chaos engineering: far fallire intenzionalmente il servizio Go al 10% di traffico e verificare che il fallback funzioni.
- Controlli di consistenza dati: campionare casualmente operazioni e verificare che servizio Go e monolite raggiungano lo stesso stato finale.
- Business user acceptance testing: eseguire workflow di esempio, come ricezione ordine, aggiornamento inventory e fatturazione, sul 10% del traffico.

**Criteri di successo:**

- Il 50% del traffico di produzione è instradato al servizio Go senza perdita dati.
- Gli error rate restano entro lo 0,1% della baseline del monolite.
- La latenza p99 resta entro il 10% del monolite su tutte le operazioni.
- Il fallback al monolite funziona entro 100 ms per qualunque errore del servizio Go.

### Phase 3: Fallback & Monitoring (settimane 7-8)

**Obiettivo:** validare che il rollback sia sicuro e automatico; finalizzare il monitoring per il cutover permanente.

**Setup tecnico:**

- Implementazione del fallback automatico: se l'error rate del servizio Go supera l'1% per 30 secondi, il 100% del traffico torna al monolite.
- Implementazione di un consistency checker: un job in background verifica che lo stato del servizio Go corrisponda al monolite per il 100% delle operazioni.
- Alert sulle divergenze: se viene rilevata qualunque inconsistenza, viene avvisato immediatamente l'on-call engineer.
- Finalizzazione dell'observability: log, metriche e trace per tutti i critical path del servizio Go.

**Validazione:**

- Failure test controllati: terminare i pod del servizio Go, verificare il fallback automatico e misurare il recovery time.
- Verifica dell'accuratezza del consistency checker: iniettare inconsistenze intenzionali e assicurarsi che vengano rilevate entro 1 minuto.
- Chaos at scale: simulare network partition, failure di database ed errori a cascata.
- Long-duration soak test: 72 ore di traffico di produzione con routing al 90% verso il servizio Go.

**Criteri di successo:**

- Il fallback automatico si attiva e completa entro 100 ms.
- Il consistency checker gira continuamente con zero falsi positivi.
- L'accuratezza degli alert di monitoring supera il 99%.
- Nessuno scenario di failure produce impatto business.

### Phase 4: Permanent Cutover (settimana 9+)

**Obiettivo:** completare la migrazione e smantellare il modulo Warehouse del monolite.

**Approccio tecnico:**

- Instradare il 100% del traffico Warehouse esclusivamente al servizio Go.
- Disattivare il dual-write nel monolite, interrompendo le scritture allo stato interno legacy.
- Rimuovere il routing proxy; il servizio Go diventa il modulo Warehouse canonico.
- Archiviare il codice Warehouse del monolite, conservandolo in version control come riferimento storico.
- Opzionale: dismettere le tabelle Warehouse del database monolitico dopo un retention period di 30 giorni.

**Strategia di rollback:**

- Se si verifica un incidente critico post-cutover, riabilitare immediatamente dual-write e percentile routing.
- Tornare a routing 50/50 mentre l'indagine è in corso.
- Nessuna perdita dati, perché il servizio Go resta authoritative.

---

## Consequences

### Esiti positivi

✅ **Evoluzione zero-downtime:** Orders e Invoicing non subiscono outage; i clienti non percepiscono la riorganizzazione interna.

✅ **Fiducia tramite esposizione graduale:** validiamo il comportamento del nuovo servizio al 10%, 25% e 50% del traffico prima del cutover completo, intercettando problemi prima che impattino tutti i clienti.

✅ **Semplicità di rollback:** qualunque phase può tornare al monolite in meno di 100 ms con fallback automatico. Non serve coordinamento manuale.

✅ **Pratica realistica:** il team sperimenta pattern di production safety usati da aziende tier-1. Lo stesso pattern si applica a future estrazioni di BC.

✅ **Apprendimento organizzativo:** insegna il cambiamento incrementale invece delle riscritture big-bang. Costruisce fiducia nei distributed system.

✅ **Validazione della consistenza dati:** dual-write con verifica periodica aumenta la fiducia nella correttezza del nuovo servizio prima dello smantellamento del monolite.

### Tradeoff e sfide

⚠️ **Duplicazione temporanea:** per le settimane 1-4, il modulo Warehouse esiste sia nel monolite sia nel servizio Go. Questo raddoppia temporaneamente lo storage e complica il deployment.

⚠️ **Complessità del dual-write:** il monolite deve scrivere in modo consistente su due sistemi durante le phase 1-3. Se uno dei due fallisce, bisogna gestire scritture parziali e recovery.

⚠️ **Overhead operativo:** mantenere routing proxy, consistency checker e logica di fallback aumenta il carico di observability, anche se diventa infrastruttura riutilizzabile.

⚠️ **Complessità di test:** validare dual-write e scenari di fallback richiede test di integrazione sofisticati e competenza di load testing.

⚠️ **Rischio temporale:** se il servizio Go presenta bug in produzione durante il percentile routing, il blast radius è proporzionale alla percentuale instradata. Mitigazione: monitoring aggressivo e fallback automatico.

⚠️ **Mismatch di data modeling:** se lo schema del servizio Go diverge dal monolite, il dual-write può fallire silenziosamente o introdurre inconsistenze. Serve una governance rigida dello schema.

---

## Learning Goals

I partecipanti capiranno:

1. **Perché l'estrazione incrementale batte le riscritture big-bang:** sicurezza psicologica, garanzie di rollback, costruzione graduale della fiducia.
2. **Come implementare lo Strangler Pattern in pratica:** routing proxy, dual-write, fallback, monitoring.
3. **Pattern operativi per distributed system:** percentile routing, chaos testing, consistency checking.
4. **Strategie di migrazione dati:** dual-write, verifica della consistenza, cutover finale.
5. **Dinamiche organizzative:** gestione del rischio tecnico, fiducia degli stakeholder, coordinamento tra team.

Completando i checkpoint 1-4, il team avrà esperienza hands-on con:

- costruzione di un servizio in Clean Architecture che può ricevere traffico in parallelo al monolite;
- implementazione del dual-write senza causare perdita dati;
- setup del monitoring per intercettare bug sottili, come inconsistenze e regressioni di latenza;
- esecuzione sicura di una migrazione production.

Questa base abilita l'estrazione sicura di ulteriori bounded context, come Invoicing e Orders, usando lo stesso pattern collaudato.

---

## References

- **Sam Newman, "Building Microservices" (2nd ed.)** - Il capitolo 3 tratta lo Strangler Fig Pattern con esempi reali.
- **Martin Fowler, "Strangler Fig Application"** (https://martinfowler.com/bliki/StranglerFigApplication.html)
- **Transizione Amazon da monolite a servizi** - documentata nel blog "All Things Distributed".
- **Deployment pattern:** feature flag, canary release, blue-green deployment come tecniche complementari.

---

## Implementation Reference

Questa ADR è implementata nei checkpoint dell'esercizio come segue:

### Phase 02: Analysis & Design (Checkpoint 2)

- **File:** `phase-02-analysis/DEPENDENCY-MAP.md`
- **Focus:** capire le dipendenze correnti del monolite e progettare i confini di estrazione.
- **Artefatti chiave:** dependency map che mostra Orders -> Warehouse, inventario degli endpoint API.
