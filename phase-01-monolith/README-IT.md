# Phase 01 — MIC Monolith Baseline

> English version: [`README.md`](./README.md)

```
   __  __ ___ ___
  |  \/  |_ _/ __|     Modernizziamo con l'AI
  | |\/| || | (__      ----------------------
  |_|  |_|___\___|     Da monolite a microservizi,
                       con un agente AI come copilota.
```

> **MIC** strizza l'occhio a **FIC** (FattureInCloud) e all'autore "**Mic**" (Michele
> Mondora). MIC e' un finto SaaS di **fatturazione per PMI italiane**, disegnato come palestra
> didattica per il pattern **Strangler Fig**: estraiamo Bounded Context dal monolite verso
> microservizi Go, un pezzo alla volta, con un agente AI come copilota.

> Ispirato a FattureInCloud, costruito per essere modernizzato.

Tempo stimato: ~1h di attivita' in breakout room, poi una restituzione condivisa.

---

## La tua missione

Hai appena ereditato **MIC**, un monolite PHP legacy di fatturazione. Nessuno nel team ha messo per
iscritto come funziona: cosa fa, com'e' costruito, dove i suoi pezzi sono intrecciati. Prima di
poterlo modernizzare, al team serve proprio quella comprensione condivisa.

**Avvia MIC, esploralo e produci i cinque artefatti qui sotto.** Gli artefatti *sono* il
deliverable di questa fase. Trattali come documenti da consegnare a un collega che entra domani.

Hai un **agente AI di coding**. Usalo come motore di esplorazione: fagli leggere il codice, la GUI
e il database insieme a te. Ma le conclusioni sono tue, e decidi tu cosa finisce negli artefatti.

> 🛠 **Avvialo nella cartella giusta.** Apri il tuo agente AI con `phase-01-monolith/` come
> working directory (o `phase-01-monolith/php-app/` quando ti concentri sul codice applicativo).
> Puntarlo alla cartella della fase, non all'intera repo, e' l'abitudine piu' utile: vede il codice
> che conta e non si distrae con il resto.

> 💡 Questo e' il primo checkpoint di un'estrazione lunga piu' lezioni. Quello che impari a *vedere*
> qui (le schermate, i componenti, il modo in cui i dati sono salvati, gli accoppiamenti) e'
> esattamente cio' su cui agiranno le fasi successive. Migliore la tua mappa, piu' semplice ogni
> fase seguente.

### Deliverable

Producili come file (markdown, o un'immagine di diagramma dove aiuta). Tienili nella tua working
copy; li confronterai con le altre room durante la restituzione.

| # | Artefatto | A cosa deve rispondere |
|---|---|---|
| 1 | **Mappa delle pagine** — `page-map.md` | Ogni schermata esposta dall'app, e a cosa serve ciascuna. Raggruppale come preferisci. |
| 2 | **Diagramma architetturale** — `architecture.md` (o immagine) | I componenti in esecuzione (container, app, database) e come una richiesta li attraversa, dal browser al database e ritorno. |
| 3 | **Guida alla repo** — `repo-guide.md` | Cosa e' e cosa fa ogni file/cartella di primo livello di `phase-01-monolith/`. Abbastanza perche' un nuovo arrivato sappia dove guardare. |
| 4 | **Mappa degli accoppiamenti** — `coupling-map.md` | Come sono salvati davvero i dati di business, e quali parti del dominio leggono o scrivono i dati di un'altra. Nomina gli accoppiamenti che una futura estrazione dovrebbe spezzare. |
| 5 | **Lista dei peggiori** — `worst-antipatterns.md` | Le **3 peggiori scelte architetturali / antipattern** che trovi. Per ognuno: cos'e', perche' fa male, e quanto costerebbe al team continuare a conviverci. |

> Non puntare alla perfezione esaustiva: punta a una mappa che un nuovo arrivato possa davvero
> usare. Catturare la *forma* del sistema e i suoi accoppiamenti *peggiori* vale piu' che elencare
> ogni colonna.

### Domande di uscita (autovalutazione)

Alla fine dovresti saper rispondere a queste, con parole tue e indicando le prove nei tuoi
artefatti:

> *Se volessimo ritagliare per primo un pezzo di MIC nel suo servizio, quale dipendenza tra due
> parti del dominio dovremmo gestire, e perche'?*

> *Quale singola scelta di design rende MIC piu' difficile da cambiare in sicurezza, e cosa faresti
> al riguardo?*

> *Elenca i domini funzionali che compaiono in MIC: le diverse aree di business di cui l'app si
> occupa. Dove vive ciascuno nelle schermate, nel codice e nei dati?*

Se sai rispondere partendo dai tuoi artefatti, sei pronto per la Phase 02.

---

## Prerequisiti

Tutto e' containerizzato: niente da installare "a livello applicativo" sull'host.

> Rancher Desktop avviato con il motore Docker, un browser e un terminale. Gli smoke check Bash
> usano `curl` + `jq`; il percorso PowerShell su Windows usa `Invoke-RestMethod` e non richiede
> `jq`. Go non serve fino alle fasi del servizio Go.

---

## Avvio

### Step 0 — Libera prima le porte

Le cartelle delle fasi riusano le porte host. Se un'altra fase o un vecchio stack MIC e' ancora in
esecuzione, Docker puo' fallire con un errore tipo:

```text
Bind for 0.0.0.0:8088 failed: port is already allocated
```

Da `phase-01-monolith/`, ferma eventuali container MIC precedenti prima di avviare:

```bash
docker compose down
```

Controllo rapido delle porte:

```bash
docker ps --filter "publish=8088" --filter "publish=8082" --filter "publish=3306"
# atteso prima dell'avvio: nessuna riga
```

Eseguilo prima dello Step 1, non dopo che Docker e' gia' fallito.

### Step 1 — Build e avvio

Dalla **radice della repo**:

```bash
cd phase-01-monolith
docker compose up --build
```

### Step 2 — Apri l'app e il browser del database

| URL                                       | Cosa                                              |
|-------------------------------------------|---------------------------------------------------|
| http://localhost:8088                     | GUI di MIC                                         |
| http://localhost:8088/api/dashboard/kpi   | Smoke check dell'API                              |
| http://localhost:8082                     | Adminer (server `mysql`, db `mic`, user/pass `root`/`root`) |
| `localhost:3306`                          | MySQL diretto                                     |

Lo schema viene creato e popolato automaticamente al primo avvio del container `mysql`. Per
ricreare da zero:

```bash
docker compose down -v && docker compose up --build
```

### Step 3 — Conferma che e' attivo

Due chiamate veloci. Se entrambe restituiscono JSON coerente, il monolite e' attivo e puoi iniziare
a esplorare.

```bash
# KPI di business (envelope { data })
curl -s http://localhost:8088/api/dashboard/kpi | jq '.data | keys'

# Una lista paginata (envelope { data, meta })
curl -s "http://localhost:8088/api/articles?limit=5" | jq '{first: .data[0], total: .meta.total}'
```

<details>
<summary><b>Windows PowerShell</b> (niente <code>jq</code>)</summary>

```powershell
$base = "http://localhost:8088"
(Invoke-RestMethod "$base/api/dashboard/kpi").data | Get-Member -MemberType NoteProperty | Select-Object Name
$r = Invoke-RestMethod "$base/api/articles?limit=5"; "total=$($r.meta.total)"
```

Su Windows la via piu' semplice e' **Git Bash** (incluso con Git for Windows): i comandi
`curl + jq` qui sopra funzionano cosi' come sono.
</details>

### Step 4 — Spegnimento

```bash
docker compose down
```

Usa `docker compose down -v` solo quando vuoi intenzionalmente cancellare il volume del database MIC
e ricaricare i dati di seed da zero.

---

## Come esplorare

Tre lenti. Usale tutte e tre; mostrano cose diverse.

1. **La GUI** (http://localhost:8088). Clicca ogni sezione della sidebar. Crea un ordine, genera una
   fattura. Nota cosa l'app permette di *fare* a un utente: da li' vengono la mappa delle pagine e
   parte della mappa degli accoppiamenti.
2. **Il database** (Adminer, http://localhost:8082). Sfoglia le tabelle. Guarda come viene salvato
   davvero un record che hai appena creato nella GUI. Qui vivono le scoperte piu' importanti per la
   tua **mappa degli accoppiamenti** e la **lista dei peggiori**: la forma di archiviazione non e'
   quella che la GUI suggerisce.
3. **Il codice** (`phase-01-monolith/`, con il tuo agente AI). Leggi controller, model, router. Traccia
   una richiesta dall'inizio alla fine. Alimenta **guida alla repo** e **diagramma architetturale**.

> Tratta l'AI come un pair veloce, non come un oracolo. Quando asserisce un accoppiamento o il
> significato di una colonna, chiedi *"dove nel codice o nello schema l'hai visto?"* prima di
> scriverlo in un artefatto.

---

## Criteri di successo

Hai finito la Phase 01 quando:

- [ ] `docker compose up --build` avvia `mic-app`, `mysql` e `adminer`, e `http://localhost:8088` apre la GUI.
- [ ] Le due chiamate "conferma che e' attivo" restituiscono JSON coerente.
- [ ] Hai prodotto tutti e cinque gli artefatti (pagine, architettura, guida alla repo, accoppiamenti, lista dei peggiori).
- [ ] La tua mappa degli accoppiamenti descrive **come sono salvati i dati di business** e nomina almeno un accoppiamento cross-dominio, con evidenze.
- [ ] La tua lista dei peggiori nomina i **3 antipattern peggiori** con una ragione concreta per ognuno.
- [ ] Sai rispondere alle **domande di uscita** partendo dai tuoi artefatti.

---

## Riferimenti

- [`ADR-001`](../docs/adr/ADR-001-strangler-fig-pattern.md) — strategia di migrazione Strangler Fig.
- [`ADR-006`](../docs/adr/ADR-006-php-monolite-implementation.md) — decisione di baseline del monolite PHP MIC.
- [`openapi.yaml`](./openapi.yaml) — contratto API legacy per le rotte di MIC.

---

## Soluzioni

Le versioni di riferimento dei cinque deliverable stanno sotto `solutions/` sul branch
**`lezione-7-soluzione`**: `page-map.md`, `architecture.md`, `repo-guide.md`, `coupling-map.md`,
`worst-antipatterns.md`. Il branch che cloni (`lezione-7`) e' il punto di partenza e non ne ha.

**Passa alla soluzione solo dopo** aver prodotto i tuoi artefatti. Il valore di questa fase sta
nell'esplorare e sbagliare un po', non nel leggere la risposta.

> Per vederla: `git switch lezione-7-soluzione`, oppure sfoglia quel branch sull'host della repo.

---

## Fase successiva

→ [Phase 02](../phase-02-analysis/) — analisi DDD strategica e tattica. Trasforma cio' che hai
scoperto in confini di aggregato puliti e in un piano di estrazione.

---

*MIC v0.1 - by Mic (Michele Mondora) - 2026*
