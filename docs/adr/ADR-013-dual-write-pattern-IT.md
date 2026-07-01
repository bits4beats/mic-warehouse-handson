# ADR-013: Pattern dual-write per la migrazione dei dati durante lo Strangler Fig

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

Un Bounded Context possiede i propri dati. Il monolite no. Estrarre un BC dal monolite significa estrarre le sue tabelle dal database condiviso e portarle in uno store privato controllato dal BC. Il Modulo 2.3 prescrive due pattern per farlo — dual-write e sincronizzazione event-based — ma i due non sono intercambiabili.

Questo ADR vincola il team a usare il **dual-write** come pattern dati per la migrazione del Warehouse. La sincronizzazione event-based è documentata nell'ADR-014 (Data Products) ed è destinata ai subscriber a valle, non alla migrazione in sé.

La scelta dipende dal requisito di consistency del consumer, non dal BC. Una richiesta che deve leggere ciò che ha appena scritto (per esempio: creare un articolo e subito recuperarlo per visualizzarlo) non tollera la latenza di una pipe di eventi asincrona. L'integrazione primaria del Warehouse con Orders è una lettura-dopo-scrittura nella stessa richiesta, quindi l'eventual consistency è insicura durante la migrazione.

## Decision

Durante la migrazione, ogni scrittura sui dati di proprietà del Warehouse va su **entrambi** i database:

```
Application
   ↓        ↓
Legacy DB   New BC DB
```

L'implementazione è un decorator a livello di repository (`DualWriteArticleRepository`) che incapsula sia il repository legacy sia quello nuovo. Su `Save`:

1. Apre la transazione sul legacy DB. Scrive. Commit.
2. Apre la transazione sul nuovo BC DB. Scrive. Commit.
3. Se il passo 2 fallisce, logga un evento di divergenza e (a seconda della policy) compensa facendo rollback del passo 1, oppure accetta la divergenza e lascia che un reconciler la rilevi.

Le letture vengono instradate dalla facade (ADR-012):

- mode `legacy`: lettura dal legacy DB.
- mode `canary` / `bc`: lettura dal nuovo BC DB.
- mode `dual-read`: lettura da entrambi, confronto, log della divergenza, restituzione della risposta del legacy.

Quando il 100% delle scritture sul nuovo BC è in salute e il tasso di divergenza è zero per un'intera finestra di osservazione, le scritture verso il legacy DB cessano. A quel punto il nuovo BC diventa il system of record.

## Consequences

**Positive:**
- Strong consistency: una lettura-dopo-scrittura all'interno della stessa richiesta vede la riga appena scritta indipendentemente dal DB su cui finisce la lettura.
- Il rollback è banale: si interrompono le scritture verso il nuovo DB e il legacy è ancora completo.
- L'unica cosa che deve cambiare è l'applicazione. Niente CDC, niente trigger, niente replication.

**Negative:**
- La doppia scrittura aumenta la latenza. Carichi reali possono richiedere una coda di scrittura con concorrenza limitata.
- La gestione dei fallimenti parziali non è banale. Il piano deve specificare la policy di compensazione in modo esplicito per ogni rotta.
- L'applicazione si fa carico della logica di migrazione, quindi i test devono coprire quattro stati: entrambi ok, fallisce legacy, fallisce il nuovo, falliscono entrambi.

**Neutral:**
- Il dual-write è *transitorio*. Una volta completato il cutover, il decorator si cancella. Lasciarlo permanentemente in produzione è un sintomo di problema.

## Implementation Reference

- `phase-04-db/repositories/dual_write_article_repository.go` — implementazione del decorator *(planned — Stream B Plan 2)*.
- `phase-04-db/repositories/divergence_logger.go` — cattura e logga le discrepanze *(planned — Stream B Plan 2)*.
- `lab-2A-event-storming-ddd/03-strangler-safety/dual-write/` — variante standalone per il lab *(planned — Stream A Plan 4)*.
