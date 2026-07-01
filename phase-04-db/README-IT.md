# Fase 04 — Warehouse BC: adapter e dual-write

> Versione inglese: [`README.md`](./README.md)

```
   ___  _                          ___ _ __ _  ___ _   _
  / __|| | ___  __ _ _ _      ___ / __| '__| |/ __| | | |
 | (__ | |/ -_)/ _` | ' \    /___|\__ \ |  | | (__| |_| |
  \___||_|\___|\__,_|_||_|        |___/_|  |_|\___|\___/

  Fase 04 — Dove il BC incontra il database legacy.
```

Tempo stimato: ~1h di build nelle breakout room (dopo una breve apertura in plenaria), poi una restituzione condivisa.

```text
CP1    Hai mappato MIC (il monolite legacy)
CP2    Abbiamo scelto il Warehouse BC da estrarre
CP3    Hai costruito il suo domain layer Go + la porta del repository
CP4    Dai a quella porta degli adapter reali e li fai girare in parallelo al legacy   <-- sei qui
CP5+   Use case + dispatcher + HTTP + ...
```

---

## Apertura (plenaria) — lo scheletro canonico su cui costruisci

CP3 era *costruiscilo-tu*: ogni room ha prodotto un domain layer leggermente diverso. Prima di aggiungere
la persistenza, ci allineiamo su **un unico scheletro canonico** — il reference design in
[`phase-03-skeleton/solutions/reference-design.md`](../phase-03-skeleton/solutions/reference-design.md).
Assicurati di saper indicare ogni pezzo:

- **`Article`** — l'aggregate root, l'unico oggetto con cui i chiamanti parlano. Identità tramite `ID`.
- **`SKU`**, **`Money`** — value object (nessuna identità, uguaglianza per valore). `Money` è **centesimi
  interi + valuta** (`2999`, `"EUR"`), mai un float.
- **`InventoryLevel`** — un'entità che vive *dentro* `Article`.
- **`ArticleRepository`** (in `interfaces/`) — la **porta**: un contratto che elenca `Save`, `FindByID`,
  `FindBySKU`, `List`, `Delete`, **senza implementazione**. Il dominio non sa nulla dei database.

La Fase 04 è il punto in cui quella porta smette di essere astratta.

### Lo scenario di migrazione, e perché serve una ACL

Il Warehouse BC deve girare **fianco a fianco con il database legacy** durante la migrazione: ogni
scrittura deve atterrare in **entrambi** gli store così nulla si rompe mentre il traffico è ancora sul
monolite. Ma i due store modellano la stessa cosa in modo diverso — di proposito:

| | `legacy_db` (porta 3306) | `warehouse_db` (porta 3307) |
|---|---|---|
| prezzo | `price DECIMAL(10,2)` — es. `29.99` | `price_cents BIGINT` + `currency CHAR(3)` — es. `2999`, `EUR` |
| valuta | assente — si assume `EUR` | colonna esplicita |

Un **adapter** è la cosa concreta che soddisfa la porta `ArticleRepository` contro uno store. Un
**Anti-Corruption Layer (ACL)** è un adapter il cui compito esplicito è *contenere il disordine legacy*
così che non raggiunga mai il dominio pulito: converte `29.99 ↔ 2999`, inventa `currency = EUR` in
uscita e la elimina in entrata. Così `entities.Article` vede sempre un `Money` corretto, anche se la
tabella legacy non sa rappresentarlo. **Quella ACL — e il dual-write che la usa — è ciò che costruisci.**

---

## Cosa costruisci (l'obiettivo)

L'adapter lato BC e il fake in memoria sono **già scritti per te** — leggi
`repositories/article_repository.go` (`MySQLArticleRepository`) come esempio svolto di "un adapter contro
MySQL". Completi due starter:

```
repositories/
  article_repository.go              MySQLArticleRepository      ← DATO (esempio svolto, schema BC)
  in_memory_article_repository.go    InMemoryArticleRepository   ← DATO (fake di test)
  legacy_article_repository.go       LegacyMySQLArticleRepository ← TASK 1 (l'adapter ACL)
  dual_write_article_repository.go   DualWriteArticleRepository   ← TASK 2 (dual-write / single-read)
```

> La suite di test parte **rossa di proposito**: i due starter restituiscono errori `TODO` e i loro test
> falliscono. Il tuo compito è farli diventare verdi. Tutto il resto (`entities/`, `events/`, l'adapter
> BC, il fake in memoria) è già verde e resta tale.

### Task 1 — l'adapter ACL legacy (`LegacyMySQLArticleRepository`)

Implementa i cinque metodi della porta contro `legacy_db`, **inclusa la conversione del prezzo**:

- **Save** — **upsert** idempotente (`INSERT ... ON DUPLICATE KEY UPDATE`). Converti `Money.AmountCents`
  in una stringa DECIMAL (`2999 → "29.99"`). Elimina la valuta — legacy non ha colonna.
- **FindByID / FindBySKU** — `SELECT`, poi **reidrata** un `Article`: da DECIMAL a centesimi, valuta di
  default **`EUR`**, ricostruisci `SKU` e `Money` **tramite le factory di dominio**. Restituisci
  `ErrArticleNotFound` quando non c'è riga.
- **List** — ogni riga legacy, reidratata allo stesso modo.
- **Delete** — per id; restituisci `ErrArticleNotFound` quando **nessuna riga** è stata eliminata
  (controlla `RowsAffected`).
- **Mai `float64`** — il denaro è centesimi interi e stringhe DECIMAL; i float corrompono silenziosamente
  la valuta. Fai la conversione con aritmetica intera/stringa, nelle helper `centsToDecimal` /
  `decimalToCents`.
- **`Article.Inventories` è fuori scope** — persisti solo la riga dell'articolo.

**Come sai che è giusto:** i test puri di conversione in `repositories/legacy_conversions_test.go`
(`2999 ↔ "29.99"`, arrotondamento, round-trip) diventano verdi — senza database.

### Task 2 — il decorator dual-write / single-read (`DualWriteArticleRepository`)

È il **decorator** che fa girare i due adapter fianco a fianco. Dall'esterno *è* un `ArticleRepository`;
dentro tiene gli adapter legacy + BC e:

- **Save / Delete** — scrive su **entrambi** gli store, **legacy per primo**. Se legacy fallisce, fermati
  e restituisci l'errore (non toccare BC). Se BC fallisce, restituisci l'errore (legacy mantiene
  l'articolo — nessun rollback in questo esercizio).
- **FindByID / FindBySKU / List** — **lettura singola**: instrada verso **uno** store scelto da un flag
  di modalità (`ReadFromLegacy` di default → `ReadFromBC` dopo il cutover). Nessun confronto, nessuna
  riconciliazione.

**Come sai che è giusto:** i test del decorator in
`repositories/dual_write_article_repository_test.go` diventano verdi (in memoria, senza DB), e poi il
flusso end-to-end `seed` qui sotto funziona contro i due database reali.

### Fatto = la suite è verde e la cucitura si accende

Con lo stack su, da `phase-04-db/`:

```bash
docker compose run --rm test                                                    # tutti i package verdi
docker compose --profile demo run --build --rm seed write   demo-1 ABC-001 "Widget" 2999 EUR
docker compose --profile demo run --build --rm seed read    demo-1
docker compose --profile demo run --build --rm seed compare demo-1
```

Hai **finito** quando la suite di test è verde, `seed write` stampa `✓ legacy write OK` / `✓ BC write OK`
con una conferma di lettura, `seed read demo-1` restituisce `price=2999 EUR`, e `seed compare demo-1`
stampa `OK: stores aligned`. (L'HTTP arriva in CP6, quindi non c'è ancora un'API da `curl` — la CLI
`seed` esegue gli stessi `dual.Save` / `dual.FindByID` che chiamerà un handler.)

---

## Esecuzione

> Prerequisiti: Rancher Desktop con l'engine Docker e `curl`/`curl.exe`. Go locale (1.22+) è opzionale —
> i test girano nel servizio Compose `test`.

### Step 0 — Libera prima le porte

La Fase 04 usa **8081** (app), **8082** (Adminer), **3306** (MySQL legacy), **3307** (MySQL BC). Le Fasi
01 e 03 usano anche 8081/3306, quindi ferma prima la fase precedente:

```bash
cd ../phase-01-monolith && docker compose down && docker rm -f mic-facade 2>/dev/null || true
cd ../phase-03-skeleton && docker compose down
docker ps --filter "publish=3306" --filter "publish=3307" --filter "publish=8081" --filter "publish=8082"
# atteso prima dell'avvio: nessuna riga
```

### Step 1 — Avvia lo stack doppio

```bash
cd phase-04-db
docker compose up --build              # due container MySQL + app Go + Adminer
curl -s http://localhost:8081/health   # {"status":"ok","mode":"legacy"}
```

Prima di scrivere codice, sfoglia entrambi i database su **http://localhost:8082** (Adminer) e *vedi* la
discrepanza che stai per colmare — server `legacy-mysql` (`legacy_user` / `legacy_pass` / `legacy_db`) e
`warehouse-mysql` (`warehouse_user` / `warehouse_pass` / `warehouse_db`). Apri `articles` in entrambi: lo
stesso concetto, una forma diversa. Quel divario è esattamente ciò che l'adapter ACL esiste per colmare.

### Step 2 — Costruisci, poi verifica

Implementa il Task 1, poi il Task 2, eseguendo i test corrispondenti dopo ciascuno, e chiudi con il flusso
`seed` qui sopra. Spegni con `docker compose down` (aggiungi `-v` per eliminare i volumi).

---

## Scope: fermati agli adapter e al decorator

Costruisci **solo** i due starter. **Non** toccare — e **non** costruire ora:

| Non ora / non toccare | Perché · arriva in |
|---|---|
| `entities/`, `events/`, `interfaces/repository.go` | il dominio e la porta sono fissati in CP3; cambiarli è il layer sbagliato |
| `MySQLArticleRepository`, `InMemoryArticleRepository` | dati come esempio svolto + fake di test |
| Persistere `Article.Inventories` | persistenza delle righe figlie — una fase successiva |
| Use case + dispatch degli eventi | CP5 |
| Handler HTTP `/articles` | CP6 |
| Auth, policy, eventi sul filo, observability | CP7–CP10 |

> Se l'AI propone di modificare il dominio o la porta mentre completi un adapter, **fermati** — sta
> spostando il problema nel layer sbagliato.

---

## Come lavorare

Piccole breakout room, il tuo **agente di coding AI** come motore, in un ciclo stretto:

```text
prompt piccolo → diff piccolo → esegui il test corrispondente → interpreta il risultato
```

Fai il **Task 1 prima del Task 2**, e dentro il Task 1 implementa **un metodo alla volta** (prima le
conversioni, dato che i loro test sono puri e istantanei). Tieni ogni diff confinato al file che stai
completando — controlla con `git diff`. Una room ha finito quando la suite è verde su adapter che la room
ha scritto e sa spiegare.

### Due hint mentre lavori

> 💡 **Guarda il database con Adminer.** Tieni aperto `http://localhost:8082` mentre costruisci. Dopo ogni
> `seed write` o `seed compare`, ricarica `articles` in *entrambi* gli store e osserva cosa ha davvero
> scritto il tuo adapter: `legacy_db.price = 29.99` accanto a `warehouse_db.price_cents = 2999,
> currency = EUR`. Il database è la verità; il terminale ne è solo un riassunto.

> 💡 **Trasforma il Go in un linguaggio che conosci.** Non serve essere fluenti in Go. Quando uno snippet
> è oscuro, chiedi all'AI un *equivalente* in un linguaggio che conosci (C#, Java, TypeScript o
> pseudocodice): *"traduci questa funzione Go in C#, mantieni il comportamento, non cambiare il design."*
> Capiscilo lì, poi torna al Go.

### Letture opzionali

- [`ADR-013 — Dual-Write Pattern`](../docs/adr/ADR-013-dual-write-pattern.md) — ordine di scrittura e
  modalità di lettura (nota: questo esercizio mantiene dual-write + single read; la variante con
  rilevamento delle divergenze è fuori scope qui).
- [`phase-02-analysis/context-mapping.md` §"Pattern 3 — ACL"](../phase-02-analysis/context-mapping.md) —
  l'Anti-Corruption Layer che questa fase implementa.
- [`ADR-002`](../docs/adr/ADR-002-clean-architecture-layers.md) / [`ADR-007`](../docs/adr/ADR-007-go-clean-architecture-implementation.md)
  — perché gli adapter stanno fuori dal dominio.

---

## Soluzioni

Le implementazioni di riferimento sono in [`solutions/`](./solutions/):
[`legacy_article_repository.expected.go.txt`](./solutions/legacy_article_repository.expected.go.txt) (Task 1)
e [`dual_write_article_repository.expected.go.txt`](./solutions/dual_write_article_repository.expected.go.txt)
(Task 2), con la spiegazione in [`solutions/README-IT.md`](./solutions/README-IT.md).

**Aprile solo dopo che la tua suite è verde.** Usale come checklist — upsert idempotente, `2999 → "29.99"`,
parsing DECIMAL senza `float64`, reidratazione via factory, scrivi-su-entrambi/leggi-da-uno — non come
qualcosa da copiare.

---

## Fase successiva

→ La **Fase 05** aggiunge il layer degli **use case** (Layer 2 della Clean Architecture) sopra i repository
che hai appena completato: `CreateArticleUseCase`, `AdjustInventoryUseCase`, e un dispatcher di eventi che
drena gli eventi di dominio che l'aggregate registra.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
