# Phase 03 — Warehouse BC: il domain layer in Go

> English version: [`README.md`](./README.md)

```
   ___       ___ _        _      _
  / __|___  / __| |_  ___| | ___| |_ ___ _ _
 | (_ / _ \ \__ \ / // -_) |/ -_)  _/ _ \ ' \
  \___\___/ |___/_\_\\___|_|\___|\__\___/_||_|

  Phase 03 — Dove il dominio Warehouse incontra Go.
```

Tempo stimato: ~1h di build in breakout room, poi una restituzione condivisa.

```text
CP1    Hai capito MIC (il monolite legacy)
CP2    Abbiamo scelto la BC Warehouse da estrarre
CP3    Costruisci il suo domain layer in Go   <-- sei qui
CP4+   Persistenza, dual-write, HTTP, ...
```

---

## La tua missione

In CP1 hai mappato il monolite MIC; in CP2 abbiamo scelto la Bounded Context **Warehouse** da
estrarre. Ora scrivi il **primo codice del nuovo servizio**: il suo **domain layer** in Go.

E' il cuore di business della BC, e nient'altro per ora: **niente database, niente API HTTP, niente
framework**. Solo i tipi che modellano il dominio Warehouse e le regole che devono sempre valere. La
persistenza e la superficie HTTP arrivano nelle prossime lezioni; se le costruisci ora stai
costruendo la cosa sbagliata.

Hai un **agente AI di coding**. Usalo per scrivere il codice con te, ma le decisioni di design e
soprattutto gli invarianti sono tuoi. **Non c'e' uno skeleton nella repo da leggere ne' una suite di
test a cui conformarsi**: la forma del codice (package, tipi, rappresentazione) la scegli tu. La
specifica e' il comportamento qui sotto, non una particolare struttura Go.

---

## Cosa costruisci (il target)

Un modulo Go (suggerito: `warehouse.local/core`, Go 1.22, Echo per l'endpoint di health), con questo
layout:

```
entities/     Article (aggregate root), SKU + Money (value object), InventoryLevel (entity)
events/       i domain event + un'interfaccia DomainEvent
interfaces/   ArticleRepository (la porta di persistenza — solo interfaccia, nessuna implementazione)
main.go       un server Echo che espone solo GET /health
```

### I building block

| Blocco | Tipo | Responsabilita' | Identita' & uguaglianza |
|---|---|---|---|
| **Article** | Aggregate root | Possiede SKU, prezzo e livelli di inventario; l'unico oggetto con cui i chiamanti parlano | identita' per `ID` |
| **SKU** | Value object | Il codice articolo | nessuna identita'; uguaglianza per valore |
| **Money** | Value object | Un prezzo: centesimi interi + valuta ISO-4217 | nessuna identita'; uguaglianza per valore |
| **InventoryLevel** | Entity (dentro Article) | Stock in una location: quantita' + riservato | identita' per `ID`, ma caricato/salvato solo via Article |
| **ArticleCreated / InventoryAdjusted / StockReserved** | Domain event | Fatti al passato che l'aggregato registra | n/a |

### Le regole che devono valere (i tuoi criteri di accettazione)

- **Article** — `id` e `name` non vuoti; **prezzo > 0**; costruito tramite una factory che fallisce
  in modo esplicito. Un'operazione `ChangePrice` mantiene la valuta (cambiare valuta e' un flusso di
  migrazione separato, non questo metodo) e **registra un fatto** invece di pubblicarlo.
- **SKU** — rispetta `^[A-Z0-9-]{3,32}$` (lettere maiuscole, cifre, trattini; lunghezza 3–32). La
  factory rifiuta tutto il resto.
- **Money** — importo in **centesimi ≥ 0**; valuta di tre lettere maiuscole (ISO-4217). La factory
  rifiuta input non validi. (Nota: `Money(0, "EUR")` e' un Money valido, ma un Article con prezzo
  zero no — la regola piu' stretta vive sull'aggregato.)
- **InventoryLevel** — id non vuoti; quantita' ≥ 0; riservato ≥ 0; **riservato ≤ quantita'**. Uno per
  (articolo, location). Raggiungibile solo via Article; **non ha un repository proprio**.
- **Domain event** — fatti al passato, immutabili. L'aggregato li **registra** in una lista pending;
  un layer successivo (CP5) li drena e pubblica. **Non** pubblicare ne' serializzare qui.
- **Repository** — esattamente una porta, `ArticleRepository`, con **sole operazioni a livello di
  aggregato** (un `Save` idempotente per lo stesso stato, piu' find/list/delete). **Nessun
  `InventoryRepository`** — romperebbe il confine dell'aggregato.
- **Clean architecture** — i package di dominio non importano driver di database ne' web framework.
  `main.go` avvia solo il server e risponde a `GET /health` con `{"status":"ok"}` su `:8081`.

### Come saprai che e' giusto

Non c'e' una suite di test prescritta da soddisfare: fisserebbe la forma interna del tuo codice, e la
forma e' tua. Gli **invarianti qui sopra sono la specifica**. Dimostra che valgono come faresti su
una BC reale: scrivi i tuoi test sulla tua API. Una room ha finito quando ogni invariante e' coperto
e verde, su un design scelto dalla room. La soluzione di riferimento e i suoi test escono alla
restituzione, dopo che ti sei impegnato su un design.

### Consegna anche: le decisioni dietro il tuo codice

Il Go che scrivi codifica scelte architetturali, quasi tutte implicite: perche' **Article** e' l'unico
aggregate root, perche' **SKU** e **Money** non hanno identita', perche' c'e' una sola porta di
repository e **nessun** `InventoryRepository`, perche' gli eventi vengono **registrati** e non
pubblicati qui. Rendile esplicite, cosi' il codice smette di essere una scatola nera.

Produci un breve **`design-decisions.md`** (mezza pagina basta). Per ogni scelta non ovvia del tuo
skeleton, indica la **decisione**, la **regola o il confine che protegge**, l'**alternativa che hai
scartato** e **come si allinea al target di clean architecture** consegnato in CP2. Alla restituzione
difendi questo design, non la sintassi, ed e' il filo che la lezione successiva riprende quando la BC
mette su un vero layer di persistenza.

> Prerequisiti: Rancher Desktop con il motore Docker. Go locale (1.22+) e' opzionale — i test
> possono girare nel servizio Compose `test`.

Hai **finito** quando il servizio parte e i tuoi test per ogni invariante sono verdi:

```bash
cd phase-03-skeleton
docker compose up --build          # poi: curl http://localhost:8081/health  → {"status":"ok"}
docker compose run --rm test       # esegue i tuoi test   (oppure, con Go locale: go test ./... -v)
```

> **Libera prima le porte.** La Phase 03 usa `8081` (app) e `3306` (MySQL), le stesse dello stack
> della Phase 01. Esegui `docker compose down` in `phase-01-monolith/` (e `docker rm -f mic-facade`
> se resta appeso) prima di avviare qui. Controlla con
> `docker ps --filter "publish=8081" --filter "publish=3306"` — nessuna riga prima di `up`.

Il `docker-compose.yml` avvia gia' un container MySQL anche se `main.go` non vi si connette: e'
voluto, la Phase 04 usera' lo stesso file senza setup aggiuntivo.

---

## Scope: fermati al dominio

Costruisci **solo** il domain layer. Questi appartengono a lezioni successive — **non** costruirli
ora:

| Non ora | Arriva in |
|---|---|
| Implementazione MySQL del repository + decorator dual-write | Lezione 8 · CP4 |
| Use case + dispatch degli eventi | CP5 |
| Handler HTTP `/articles` | CP6 |
| Auth, policy, eventi sul filo, observability | CP7–CP10 |

Due semplificazioni di scope da CP2: **StockReservation** e' per ora solo un contatore `Reserved` su
`InventoryLevel` (promosso a entity a parte piu' avanti), e il **ledger dei movimenti di stock**
resta nel monolite PHP — non fa parte di questa BC Go.

---

## Come lavorare

Piccole breakout room, il vostro **agente AI di coding** come motore. Costruite lo skeleton voi. Confrontiamo gli
skeleton alla restituzione: Article e' venuto fuori come aggregate root? SKU e Money non hanno
identita'? `reserved ≤ quantity` e' garantito? Qualcuno ha fatto trapelare persistenza o HTTP nel
dominio?

### Letture opzionali

- [`ADR-011 — DDD building blocks`](../docs/adr/ADR-011-ddd-building-blocks.md) — Aggregate, Entity, Value Object, Domain Event.
- [`ADR-007 — Go Clean Architecture`](../docs/adr/ADR-007-go-clean-architecture-implementation.md) — il layering che questa fase pratica.

---

## Soluzioni

Sul branch **`lezione-7`** (quello che cloni) `phase-03-skeleton/` contiene una copia del monolite
MIC: e' da li' che estrai il dominio Warehouse, e far funzionare la build Go e Docker fa parte del tuo
lavoro. La **soluzione di riferimento** e' il branch **`lezione-7-soluzione`**: lo skeleton di dominio
Go sviluppato (`entities/`, `events/`, `interfaces/`, `main.go`, piu' build Go e setup Compose), con
le motivazioni di design in `solutions/reference-design.md`.

**Passa alla soluzione solo dopo** aver costruito il tuo. Il valore di questa fase sta nel prendere
tu le decisioni di design, non nel copiare un target.

---

## Cosa viene dopo

La lezione successiva da' al tuo `ArticleRepository` la prima implementazione reale (MySQL) piu' un
**decorator dual-write** (DB legacy + nuovo) e un divergence logger: il seam inizia ad accendersi.
Quel lavoro vive in una lezione successiva, non in questa repo.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
