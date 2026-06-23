# Phase 02 — Solutions

> ⚠ **Spoiler.** Questa directory contiene worked answer e reasoning per le domande in [`../README-IT.md`](../README-IT.md).
> Aprila **solo dopo** aver tentato le domande da solo. Il valore dell'esercizio sta nell'attrito, non nella risposta.

> English version: [`README.md`](./README.md)

---

## Format

Ogni domanda segue lo stesso layout a tre blocchi, usato in tutte le phase:

- **🎯 Answer (TL;DR)** — 1–2 frasi, la risposta diretta
- **🧠 Reasoning** — il _perché_: rinforzo del concetto, edge case, collegamenti ad altri artefatti
- **🔬 Verification** — puntatori agli artefatti di Phase 02, ADR e slide del corso che ancorano la risposta

---

## Step 1 — Event Storming

Domande di riferimento in [`../README-IT.md` Step 1](../README-IT.md#step-1--leggi-il-canvas-dellevent-storming-15-min).

### D1 — Perché lo Step 1 (_enumerate_) vieta il filtering?

**🎯 Answer (TL;DR)**

Filtrare durante l'enumerazione mescola due conversazioni diverse (_cos'è successo?_ vs _cosa importa?_) e la seconda affoga la prima, quindi sinonimi ed eventi di edge case scompaiono prima che qualcuno possa contestarli. Finisci col modello che la voce dominante nella stanza ha già immaginato — non quello che il business effettivamente esibisce.

**🧠 Reasoning**

L'Event Storming ha tre passi per una ragione. Ogni passo è un'attività cognitiva diversa:

| Passo        | Attività                                                              | Cosa produce                                |
| ------------ | --------------------------------------------------------------------- | ------------------------------------------- |
| 1. Enumerate | _Pensiero divergente_ — scrivi tutto, senza giudizio                  | Lista grezza di eventi past-tense           |
| 2. Cluster   | _Pensiero convergente_ — raggruppa, fai emergere duplicati e tensioni | Aggregati e BC candidati                    |
| 3. Name      | _Sintesi_ — dai un'etichetta a ciascun cluster                        | BC candidati con responsabilità in una riga |

Se provi a fare tutti e tre i passi insieme, le voci più forti della stanza (spesso senior architect o PM più opinionati) iniziano a _pre-clusterizzare_: _"nah, `CustomerEmailChanged` è lo stesso di `CustomerAddressChanged`"_. Questo cortocircuita lo step divergente. Il giudizio _"è la stessa cosa"_ può essere corretto — o può essere sbagliato perché il team legal ha vincoli sui cambi di PII che l'architect non conosceva.

> 💡 **L'artefatto che perdi saltando lo Step 1 è la tensione fra eventi simili.** Quella tensione _è informazione sul dominio_; rimandare il filtering la preserva.

Failure mode concreto sul canvas Warehouse: immagina che qualcuno avesse collassato `InventoryStocked`, `InventoryReplenished` e `InventoryAdjustedManually` in un unico `InventoryChanged` durante l'enumerazione. Lo Step 4 di decomposizione avrebbe perso la distinzione fra _"il magazzino ha ricevuto una consegna"_ (`InventoryReplenished`) e _"un operatore ha corretto un errore"_ (`InventoryAdjustedManually`) — due operazioni di business molto diverse con implicazioni di audit e finance diverse.

**🔬 Verification**

- **Artefatto:** [`event-storming-canvas.md`](../event-storming-canvas.md) - "Don't filter at Step 1"
- **ADR:** [ADR-010 — Event Storming methodology](../../docs/adr/ADR-010-event-storming-methodology.md)
- **Slide del corso:** Modulo 2.3, _Event Storming workshop_ — _"Step 1 = throw everything on the wall; resist the urge to clean up"_

---

## Step 2 — Ubiquitous Language

Domande di riferimento in [`../README-IT.md` Step 2](../README-IT.md#step-2--leggi-la-ubiquitous-language-20-min).

### D2 — Perché "Article" in Warehouse è diverso da "Article" in Catalog?

**🎯 Answer (TL;DR)**

Perché gli _invarianti_ e le _responsabilità_ sono diversi. L'Article di Catalog è un'entità di merchandising (categorie, descrizioni, foto); l'Article di Warehouse è un'entità stockabile (SKU, inventory level). Stessa parola, due BC, due concetti diversi. La tabella legacy `business_data` non li distingue perché non ha nozione di confini BC — è *l'*anti-pattern che l'estrazione vuole correggere.

**🧠 Reasoning**

Questo è l'esempio canonico di un **omonimo cross Bounded Context**. La Ubiquitous Language è _ubiquita dentro_ il BC; non si estende oltre il confine. _Dentro_ Warehouse, "Article" significa sempre "un item stockabile con SKU e inventory level". _Dentro_ Catalog, "Article" significa sempre "una entry di merchandising con nome, descrizione e categoria".

| Proprietà                      | Article in Catalog                   | Article in Warehouse                       |
| ------------------------------ | ------------------------------------ | ------------------------------------------ |
| Portatore di identità          | `id` interno; SKU è metadata         | `ID` UUID; SKU è l'identificatore business |
| Invariante sul prezzo          | Nessuno (Pricing possiede il prezzo) | `price > 0`, currency stabile              |
| Invariante sullo stock         | Nessuno (non tracciato)              | `quantity ≥ 0`, `reserved ≤ quantity`      |
| Eventi del ciclo di vita       | Created, Renamed, Decommissioned     | InventoryAdjusted, StockReserved           |
| Cosa lo invalida quando cambia | Descrizione, foto, categoria         | Prezzo, inventory level, reservation count |

Due cose diverse. Il fatto che la _stringa_ "Article" sia usata due volte va bene — _purché_ ogni BC mantenga il proprio significato esplicito e l'integrazione usi un context-mapping pattern che traduca fra loro (in questo esercizio: Customer/Supplier per Orders↔Warehouse, Conformist per Invoicing↔Catalog).

> 💡 **Un omonimo cross-BC non è un problema — fingere che non esista lo è.** La disciplina UL dice: nomina l'omonimo, dai a ogni lato la sua definizione, traduci al confine. La tabella legacy `business_data` fallisce questa disciplina; il Go BC la enforza.

**🔬 Verification**

- **Artefatto:** [`ubiquitous-language.md`](../ubiquitous-language.md)
- **Mapping:** [`tactical-ddd-warehouse.md`](../tactical-ddd-warehouse.md)
- **Context Mapping:** [`context-mapping.md`](../context-mapping.md)

---

### D3 — Quali pericoli espone il definire dei sinonimi per un termine della UL?

**🎯 Answer (TL;DR)**

I sinonimi violano le due proprietà fondamentali della UL — _precise_ e _consistent_ (Evans, _DDD_ cap. 2) — e disfano silenziosamente il knowledge crunching che ha prodotto il vocabolario. Pericoli concreti: (1) la comunicazione fra esperti di dominio e ingegneri si degrada perché persone diverse usano parole diverse per lo stesso concetto senza accorgersene; (2) l'accordo del team su _un termine canonico per concetto_ va perso senza che nessuno decida di perderlo; (3) i confini dei Bounded Context si sfumano perché i sinonimi spesso colano dentro da BC vicini; (4) in un setup bilingue, la colonna delle traduzioni negoziate smette di essere autoritaria.

**🧠 Reasoning**

Evans nomina due proprietà che la UL deve avere per funzionare (_DDD_, cap. 2): il linguaggio dev'essere _precise_ (un significato per parola) e _consistent_ (una parola per significato). Ammettere sinonimi viola _consistent_ in modo netto e di solito erode anche _precise_. Le conseguenze sono fallimenti DDD, non fallimenti di tooling:

| Se la UL ammettesse...                                    | Il fallimento DDD è...                                                                                                                                                                                                                                                               |
| --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `quantity`, `qty`, `amount` per lo stesso concetto        | **Knowledge crunching sprecato.** Il team ha negoziato _un_ termine canonico per "Quantity" durante il workshop. Ammettere tre nomi per lo stesso concetto disfa silenziosamente quell'accordo — ogni nuovo arrivato deve riscoprire da sé che le parole significano la stessa cosa. |
| `availability` accanto a `InventoryLevel`                 | **Il confine del BC erode.** _Availability_ è un concetto Catalog (vendibile? in-region?). Ammetterlo dentro Warehouse implica che Warehouse si occupi di visibilità, ma non è così. Il vocabolario del BC vicino sconfina.                                                          |
| `prodotto` accanto a `Article`                            | **La UL bilingue va in deriva.** Il team ha negoziato _Article ↔ Articolo_. Ammettere _prodotto_ riapre quella decisione — quando il cliente dice _"il prodotto"_ uno sviluppatore non sa più su quale termine inglese canonico mapparlo.                                            |
| `stock` per indicare sia _giacenza_ sia _unità riservate_ | **Perdita di precisione.** Una parola, due significati. La prima proprietà UL di Evans si rompe. Conversazioni e codice ora richiedono il contesto per disambiguare — esattamente il costo di traduzione che la UL doveva eliminare.                                                 |

> 💡 **Il costo profondo.** La UL è l'_output condiviso_ del knowledge crunching fra esperti di dominio e ingegneri. Ogni sinonimo ammesso è un pezzo di quella comprensione condivisa perso silenziosamente. Dopo sei mesi il team è tornato al punto _prima_ del workshop, tranne che adesso tutti _pensano_ che ci sia una UL.

**🔬 Verification**

- **Artefatto:** [`ubiquitous-language.md`](../ubiquitous-language.md)
- **Riferimento:** Eric Evans, _DDD_ 2003, cap. 2 — _Communication and the Use of Language_. La UL è definita esattamente da queste proprietà; toglile e hai un glossario, non una UL.

---

## Step 3 — Context Mapping

Domande di riferimento in [`../README-IT.md` Step 3](../README-IT.md#step-3--leggi-il-context-mapping-15-min).

### D4 — Customer/Supplier vs Conformist: qual è la differenza precisa?

**🎯 Answer (TL;DR)**

Entrambi sono pattern Upstream/Downstream; la differenza è il **bilanciamento di potere**. Nel Customer/Supplier il downstream **ha** influenza sulla roadmap dell'upstream (negoziazione, planning congiunto, contract review). Nel Conformist il downstream ha **perso** (o non ha mai avuto) quell'influenza e semplicemente accetta il modello upstream così com'è. **Conformist è la specializzazione di Customer/Supplier dove il canale di negoziazione è chiuso.**

**🧠 Reasoning**

I due pattern stanno su uno spettro, non in due scatole separate:

```text
Customer/Supplier  ←──── continuum ────→  Conformist
  ↑                                          ↑
  ha leva                                    no leva
  negozia                                    accetta
  contratto evolve                           contratto fisso
  roadmap congiunta                          solo roadmap upstream
```

La stessa relazione può **scivolare** da uno all'altro:

- Una startup che usa l'API interna di un partner inizia in Customer/Supplier (il partner ha bisogno di loro). Dopo che la startup viene acquisita e retrocessa di priorità, la relazione scivola in Conformist.
- Un team che costruisce su un SDK di un vendor inizia in Conformist (il vendor controlla l'SDK). Quando il team diventa abbastanza grande da influenzare la roadmap del vendor (grande cliente, relazione strategica), può scivolare in Customer/Supplier.

Nell'esercizio Warehouse la relazione Orders↔Warehouse è **Customer/Supplier**: i due team negoziano l'API e gli event schema. Se il team Warehouse fosse un'azienda separata che non risponde alle chiamate, Orders scivolerebbe in Conformist (o costruirebbe un ACL, vedi D5).

**🔬 Verification**

- **Artefatto:** [`context-mapping.md`](../context-mapping.md)
- **Esterno:** Evans, _DDD_ (2003)

---

### D5 — Perché ACL invece di Conformist fra Go Warehouse e schema legacy?

**🎯 Answer (TL;DR)**

Lo schema legacy `business_data`/`business_relations` usa colonne generiche (`amount_1`, `text_2`, `payload`) che _corromperebbero_ il modello del Go BC se esposte direttamente. Conformist forzerebbe ogni struct Go a portare i nomi delle colonne legacy; ACL paga il costo di un translator al confine e lascia al Go BC modellare il dominio pulitamente. Il trade-off è **costo di manutenzione del translator** vs **un modello di dominio inquinato** — per un BC long-lived che sopravvivrà al legacy, vince ACL.

**🧠 Reasoning**

Conformist produrrebbe codice tipo:

```go
type Article struct {
    ID       string
    Code     string   // = SKU (legacy "code")
    Amount1  int64    // = prezzo in cents (legacy "amount_1")
    Amount2  int64    // = min quantity (legacy "amount_2")
    Text1    string   // = categoria (legacy "text_1")
    Text2    string   // = codice IVA (legacy "text_2")
    Payload  string   // = blob JSON
}
```

Tecnicamente è una traduzione fedele dello schema legacy. Ma:

1. **La leggibilità è distrutta.** Un nuovo joiner non può dire cos'è `Amount2` senza leggere le note di migrazione legacy.
2. **La UL è violata.** `Amount1` non è un termine UL; la UL vieta `amount` per campi non-Money.
3. **Il costo di refactor sale.** Ogni operazione business ora deve manipolare campi nominati generici e li espone accidentalmente in response API, eventi, log.

ACL paga il costo del translator in esattamente **un punto** (later lessons, planned) e il resto del codice Go vive nella UL. Trade-off: quando lo schema legacy cambia (raro nel caso MIC — è la parte da cui l'esercizio vuole estrarre), il translator va aggiornato. Il costo di _un_ file translator vs _ogni_ riga di codice Go è l'asimmetria giusta su cui scommettere.

> 💡 **ACL è la scommessa che il Go BC sopravviverà allo schema legacy.** Se fosse il legacy a sopravvivere al Go BC, Conformist sarebbe più economico. Per un'estrazione Strangler Fig l'assunzione è rovesciata.

**🔬 Verification**

- **Artefatto:** [`context-mapping.md`](../context-mapping.md) "Pattern 3 — Anti-Corruption Layer"
- **Mapping tattico:** [`tactical-ddd-warehouse.md`](../tactical-ddd-warehouse.md)
- **ADR:** ADR-013 — Dual-write pattern

---

## Step 4 — Tactical DDD

Domande di riferimento in [`../README-IT.md` Step 4](../README-IT.md#step-4--leggi-i-building-block-del-ddd-tattico-15-min).

### D6 — Perché `InventoryLevel` ha un `ID` ma `SKU` no?

**🎯 Answer (TL;DR)**

Perché **`InventoryLevel` è una Entity** (identity-bearing) e **`SKU` è un Value Object** (definito dai dati soli). Due `InventoryLevel` con stessi `(ArticleID, LocationCode, Quantity)` ma `ID` diversi sono _cose diverse_ (es. due righe diverse del ledger che capitano di essere nello stesso stato). Due `SKU` con stesso `Code` sono _la stessa cosa_ — non c'è una distinzione "questa istanza di SKU vs quella istanza di SKU".

**🧠 Reasoning**

La distinzione Entity/VO è uno degli inciampi più comuni nel DDD. Il litmus test è:

> _"Se creo un altro di questi con esattamente gli stessi dati, è la stessa cosa o una cosa diversa?"_

Per SKU: due `SKU{Code: "WGT-001"}` sono lo _stesso_ SKU — non c'è identità di istanza. Quindi VO.
Per InventoryLevel: due `InventoryLevel{ArticleID: "a-1", LocationCode: "IT-MILANO1", Quantity: 100}` potrebbero essere due righe distinte in contesti transazionali diversi; servono `ID` per essere distinguibili. Quindi Entity.

Le conseguenze pratiche:

| Proprietà      | Entity (`InventoryLevel`)                       | VO (`SKU`)                             |
| -------------- | ----------------------------------------------- | -------------------------------------- |
| Campo identità | `ID` (UUID)                                     | Nessuno                                |
| Equality       | Per `ID`                                        | Per tutti i campi data                 |
| Mutabilità     | Mutable (attraverso aggregate root)             | Immutable — sostituito, non mutato     |
| Construction   | `NewInventoryLevel(id, ...)` — richiede un `id` | `NewSKU(code)` — niente argomento `id` |

> 💡 **La scelta Entity/VO vincola il tuo operatore di equality e il tuo layer di persistence.** Sbagliarla e il database accumula "value" duplicati con ID diversi, o due entity distinte collidono perché capitano di avere contenuto identico. L'aggregate root riconcilia queste decisioni al confine.

**🔬 Verification**

- **Artefatto:** [`tactical-ddd-warehouse.md`](../tactical-ddd-warehouse.md) "Entity" e "Value Object"
- **Codice:** `phase-03-skeleton/entities/inventory.go:13-21` (la struct ha `ID`) vs `phase-03-skeleton/entities/sku.go` (nessun campo `ID`)
- **ADR:** [ADR-011](../../docs/adr/ADR-011-ddd-building-blocks.md) — definizioni di Entity e Value Object

---

## Step 5 — Dependency Map

Domande di riferimento in [`../README-IT.md` Step 5](../README-IT.md#step-5--leggi-la-dependency-map-15-min).

### D7 — Perché Warehouse è chiamato "un sink"? Cosa implica per l'ordine di estrazione?

**🎯 Answer (TL;DR)**

Un "sink" è un nodo con **molti edge in-bound e zero edge out-bound**: tutti leggono da lui; lui non legge da nessuno. In sezione 4, Warehouse ha 3 dipendenze in-bound (Orders, Listino, Invoicing) e 0 dipendenze out-bound da tabelle di altri BC. È la condizione _safe-to-extract first_ da manuale in uno Strangler Fig — estrarre Warehouse non può rompere nient'altro, perché Warehouse non dipende da nient'altro.

**🧠 Reasoning**

In una migrazione Strangler Fig l'ordine di estrazione importa perché:

- Estrarre un BC che **legge da** altri BC forza a progettare prima i contratti di read (altrimenti il BC estratto non funziona da solo).
- Estrarre un BC che **è solo letto da** altri BC richiede solo di esporre un contratto per i reader — il timing lo controlli tu.

Warehouse è nella seconda categoria. La sequenza di estrazione diventa:

1. Stand up della superficie HTTP Article/inventory del Go BC (Phase 06).
2. Rafforza la stessa superficie con autenticazione e policy prima di fidarti del routing del traffico (Phase 07-08).
3. Usa un cutover runbook per spostare i consumer solo dopo controlli di parity, policy e observability.
4. Decommissiona il path Warehouse legacy quando l'evidenza di integrazione dice che è sicuro.

Nessuno step richiede a Warehouse di chiamare un altro BC — quindi nessun altro BC deve essere pronto prima che Warehouse possa essere estratto. È la proprietà "sink" che ripaga.

> 💡 **La proprietà sink è rara e preziosa.** La maggior parte dei BC in un monolite ha dipendenze mutue. Scegliere un sink come primo target di estrazione è la modalità più semplice e una scelta pedagogica intelligente per questo esercizio. Le estrazioni nel mondo reale devono tipicamente gestire dipendenze miste e scegliere il BC con le chiamate out-bound _più economiche_ (non necessariamente zero).

**🔬 Verification**

- **Artefatto:** [`DEPENDENCY-MAP.md`](../DEPENDENCY-MAP.md)
- **Esterno:** Sam Newman, _Building Microservices_ (2nd ed.)

---

## Step 6 — Extraction Plan

Nessuna Q/A fissa — l'Extraction Plan è un artefatto di sola lettura per questa phase. Percorrilo da solo; tornaci sopra dopo che Phase 03 ha implementato il domain layer per vedere quali decisioni hanno pagato.

---

## Step 7 — Stretch

Niente Q/A fisse. Lo stretch è open-ended apposta:

- Eseguire i prompt di validazione UL su codice reale è la memoria muscolare con cui l'esercizio vuole farti uscire.
- Sketchare un canvas di Event Storming per _il tuo_ progetto è l'unico modo per interiorizzare il ritmo a tre step.

Fotografa il canvas e portalo alla review di Phase 03.
