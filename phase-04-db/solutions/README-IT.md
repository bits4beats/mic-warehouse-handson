# Fase 04 — Soluzioni

> Versione inglese: [`README.md`](./README.md)

> ⚠ **Spoiler.** Risposte ragionate per i due task di coding in [`../README-IT.md`](../README-IT.md).
> Apri questo file **solo dopo che la tua suite di test è verde**. Il valore della fase sta nello scrivere
> tu stesso gli adapter e il decorator, non nel leggere l'obiettivo.

Le due implementazioni di riferimento sono i file `.go.txt` accanto a questo README (l'estensione `.txt`
impedisce a Go di compilarli):

- **Task 1** → [`legacy_article_repository.expected.go.txt`](./legacy_article_repository.expected.go.txt)
- **Task 2** → [`dual_write_article_repository.expected.go.txt`](./dual_write_article_repository.expected.go.txt)

Ogni sezione qui sotto usa gli stessi tre blocchi: **🎯 TL;DR · 🧠 Ragionamento · 🔬 Verifica.**

---

## Task 1 — `LegacyMySQLArticleRepository` (l'adapter ACL)

### 🎯 TL;DR

Implementa i cinque metodi di `ArticleRepository` contro `legacy_db`, più le due helper di conversione del
prezzo. L'adapter è un **Anti-Corruption Layer**: converte `Money` (centesimi + valuta) da/verso la colonna
legacy `price DECIMAL`, **inventa `EUR` in lettura**, **elimina la valuta in scrittura**, e reidrata
`SKU`/`Money` tramite le factory di dominio — così lo schema legacy non trapela mai in `entities.Article`.

### 🧠 Ragionamento

Le decisioni non ovvie, ognuna codificata nella soluzione di riferimento:

- **`Save` idempotente via upsert.** `INSERT ... ON DUPLICATE KEY UPDATE` mantiene il contratto `Save`
  della Fase 03: chiamarlo due volte con lo stesso stato dell'aggregate finisce nella stessa riga, che
  esistesse o no. Nessuna race `SELECT`-poi-branch.
- **Conversione senza `float64`.** `29.99` non è rappresentabile esattamente come float, e `29.99 * 100`
  può dare `2998.999…`. Per il denaro è un bug. `centsToDecimal` / `decimalToCents` usano invece
  aritmetica intera/stringa: dividi sul punto decimale, riempi/arrotonda la parte frazionaria a due cifre,
  proteggi dall'overflow. I test puri in `legacy_conversions_test.go` fissano il comportamento esatto
  (`2999 ↔ "29.99"`, `"1" → 100`, `"1.5" → 150`, round-trip).
- **Reidratazione, non creazione.** In lettura l'adapter ricostruisce l'`Article` con il suo `ID` e i
  timestamp persistiti tramite uno struct literal — **non** chiama `NewArticle` (che è il workflow di
  creazione per input nuovo). Ma **ricostruisce** `SKU` e `Money` tramite `NewSKU` / `NewMoney`, perché i
  valori che rientrano nel dominio devono comunque essere validi.
- **La valuta è inventata in lettura.** Legacy non ha colonna valuta, quindi l'adapter mette `EUR` di
  default. Quel "vuoto riempito" è la corruzione contenuta: il dominio riceve sempre un `Money` completo.
- **`Article.Inventories` è fuori scope.** Si persiste solo la riga `articles`.
- **`Delete` controlla `RowsAffected`.** Zero righe eliminate → `ErrArticleNotFound`, non un successo
  silenzioso.

### 🔬 Verifica

- I test di conversione diventano verdi: `docker compose run --rm test` (oppure i casi puri
  `centsToDecimal` / `decimalToCents` in `repositories/legacy_conversions_test.go`).
- End-to-end (dopo il Task 2): `seed write demo-1 ABC-001 "Widget" 2999 EUR` poi `seed compare demo-1`
  → `OK: stores aligned`; Adminer mostra `legacy_db.articles.price = 29.99` e
  `warehouse_db.articles.price_cents = 2999, currency = EUR`.
- Checklist vs [`legacy_article_repository.expected.go.txt`](./legacy_article_repository.expected.go.txt):
  upsert idempotente · `2999 → "29.99"` · parsing DECIMAL senza `float64` · reidratazione via factory ·
  default `EUR` in lettura · `ErrArticleNotFound` su riga mancante.

---

## Task 2 — `DualWriteArticleRepository` (decorator dual-write / single-read)

### 🎯 TL;DR

Un **decorator** che, dall'esterno, *è* un `ArticleRepository`, ma dentro tiene gli adapter legacy e BC.
**Scrive su entrambi** gli store (legacy per primo) e **legge da uno**, scelto da un flag `ReadMode`
(`ReadFromLegacy` di default → `ReadFromBC` dopo il cutover). Nessun confronto, nessuna riconciliazione.

### 🧠 Ragionamento

- **L'ordine di scrittura è deliberato.** `Save`/`Delete` vanno **legacy per primo**: legacy è il sistema
  di riferimento durante la migrazione. Se legacy fallisce, BC non viene mai toccato e l'errore torna. Se
  BC fallisce dopo che legacy è riuscito, l'errore torna e legacy mantiene il valore — **nessun rollback**
  in questo esercizio (annullare una scrittura confermata è peggio di un vuoto temporaneo; la
  riconciliazione è una preoccupazione successiva).
- **Lettura singola per modalità.** Tutte e tre le letture instradano verso esattamente uno store.
  `ReadFromLegacy` è il default sicuro all'inizio della migrazione; `ReadFromBC` è lo stato post-cutover.
  La modalità è letta dalla variabile d'ambiente `DUAL_WRITE_READ_MODE` in `main.go`, così lo stesso
  binario attraversa la migrazione con un riavvio, non un redeploy.
- **Il decorator funziona solo perché la porta è un'interfaccia.** I suoi due campi sono di tipo
  `interfaces.ArticleRepository`, e lui stesso soddisfa la stessa interfaccia — è ciò che gli permette di
  sostituirsi a un repository delegando a due, e permette ai test di cablare due fake in memoria al posto
  di MySQL reale. Uno struct concreto non potrebbe essere sostituito così.

> **Nota di scope.** Questo esercizio mantiene il decorator a *dual-write + single-read*. La variante con
> rilevamento delle divergenze (leggi entrambi, confronta, logga le discrepanze) è intenzionalmente **fuori
> scope** — vedi la decisione di scope nel piano della lezione. ADR-013 descrive comunque il pattern più
> completo.

### 🔬 Verifica

- I test del decorator diventano verdi: i test in memoria in
  `repositories/dual_write_article_repository_test.go` (le scritture colpiscono entrambi gli store; il
  fallimento legacy interrompe BC; il fallimento BC mantiene comunque legacy; le letture instradano per
  modalità).
- End-to-end: `seed write` stampa `✓ legacy write OK` / `✓ BC write OK` con conferma di lettura;
  `seed read demo-1` restituisce `price=2999 EUR`.
- Checklist vs [`dual_write_article_repository.expected.go.txt`](./dual_write_article_repository.expected.go.txt):
  scrivi-su-entrambi-legacy-per-primo · interrompi-BC-su-fallimento-legacy · leggi-da-uno-per-modalità ·
  collaboratori tipizzati come interfaccia.

---

## Trappole comuni con l'AI (rifiuta questi diff)

Mentre completi uno dei due task, ferma l'AI se propone di:

- cambiare `entities/`, `events/`, o `interfaces/repository.go` (il dominio e la porta sono fissati);
- usare `float64` da qualche parte nella conversione del prezzo;
- aggiungere una colonna `currency` allo schema legacy invece di mettere `EUR` di default nell'adapter;
- insegnare a `DualWriteArticleRepository` dettagli di SQL o dello schema legacy (è compito dell'adapter);
- persistere `Article.Inventories` (fuori scope in questa fase);
- aggiungere auto-riconciliazione / auto-fix delle divergenze (fuori scope; e sbagliato per ADR-013).
