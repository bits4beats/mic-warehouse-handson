# ADR-010: Event Storming come metodo per individuare i Bounded Context

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

Il Modulo 2.3 di Tech Track 2 stabilisce che una Business Capability si *trova*, non si *riceve già definita*. La parte difficile della decomposizione è tracciare la linea fra un BC e l'altro. Oggi il repository documenta lo Strangler Fig (ADR-001) e i layer della clean architecture (ADR-002), ma non documenta *come* si individuano in prima battuta i BC candidati.

Senza un metodo esplicito, i team scivolano in due modalità di fallimento:

- **Nominare per primi.** Si scrive una lista di nomi di BC ("Catalogo", "Ordini", "Resi") riproducendo la struttura che si immagina il sistema abbia, non quella che il business effettivamente esprime.
- **Partire dallo schema.** Si guardano le tabelle del database e si raggruppano. Il risultato è software che il database capisce, non software che il business capisce.

La formazione prescrive Event Storming — una tecnica di workshop nata nella community DDD — come la polizza assicurativa più economica contro confini sbagliati. Produce due artefatti (una lista di eventi di dominio al passato e un clustering di tali eventi in BC candidati) prima che chiunque dia un nome a un BC o disegni una tabella.

## Decision

Adottiamo **Event Storming** come metodo canonico per individuare i Bounded Context in questo esercizio e in qualunque futuro lavoro di estrazione di un BC.

Il metodo prevede esattamente tre passi, in quest'ordine:

1. **Enumerare.** Elencare ogni evento di business significativo a partire dalla narrativa di dominio, al passato, senza filtrare.
2. **Raggruppare.** Aggregare gli eventi che stanno bene insieme. Ogni cluster traccia un aggregato.
3. **Nominare.** Ogni cluster diventa un BC candidato. La nominatura avviene per ultima.

L'output è un canvas a due sezioni: gli eventi di dominio a sinistra, i Bounded Context a destra.

In questo repository, la Phase 02 (`phase-02-analysis/`) ospita il canvas canonico per l'estrazione del Warehouse. Il Lab 2.A (later labs) ospita un esercizio cartaceo costruito sulla narrativa ShopRight, utilizzato come artefatto didattico.

## Consequences

**Positive:**
- I confini dei BC sono difesi da un artefatto (il canvas), non da un'opinione.
- Il vocabolario usato in Event Storming (eventi al passato, aggregati) si trasferisce direttamente nel codice (ADR-011).
- Una nuova persona che entra nel team può leggere il canvas e ricostruire il ragionamento senza dover rifare il workshop.

**Negative:**
- Il metodo richiede accesso al dominio. Se il team che conduce il workshop non ha stakeholder di business in stanza, il canvas è una congettura.
- Il passo 1 (enumerare senza filtrare) sembra dispersivo ed è il più frequentemente saltato. Il piano deve segnalarlo esplicitamente.

**Neutral:**
- L'output è intenzionalmente a bassa fedeltà (post-it, righe di spreadsheet). Non è un diagramma UML e non va promosso a tale.

## Implementation Reference

- `phase-02-analysis/event-storming-canvas.md` — canvas canonico per l'estrazione del Warehouse.
- `phase-02-analysis/ubiquitous-language.md` — la Ubiquitous Language fatta emergere dal canvas, raffinata nel vocabolario canonico del Warehouse (con prompt AI per validare il codice).
- ADR-011 riprende il vocabolario DDD che Event Storming fa emergere.
