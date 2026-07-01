# Fase 05 — Soluzioni

> Versione inglese: [`README.md`](./README.md)

> ⚠ **Spoiler.** Risposte ragionate per le due slice verticali in [`../README-IT.md`](../README-IT.md).
> Apri questo file **solo dopo che la tua slice è verde**. Il valore sta nel costruirla tu.

File di riferimento (`.txt` così Go non li compila):

- Use case Slice 1 → [`get_article.expected.go.txt`](./get_article.expected.go.txt)
- Use case Slice 2 → [`change_article_price.expected.go.txt`](./change_article_price.expected.go.txt)
- Handler (entrambi) → [`article_handler.expected.go.txt`](./article_handler.expected.go.txt)
- Router (entrambe le rotte) → [`router.expected.go.txt`](./router.expected.go.txt)

Ogni blocco: **🎯 TL;DR · 🧠 Ragionamento · 🔬 Verifica.**

---

## Slice 1 — GetArticle · `GET /articles/:id`

### 🎯 TL;DR

Use case: valida l'id, `repo.FindByID`, **avvolgi l'errore** con `%w`. Handler: leggi il path param,
chiama lo use case, mappa `ErrArticleNotFound` a **404**, altrimenti **200**.

### 🧠 Ragionamento

- **Perché avvolgere con `%w`?** Il repository restituisce `ErrArticleNotFound`; lo use case lo avvolge
  (`fmt.Errorf("GetArticle: %w", err)`). Il verbo `%w` mantiene il sentinel raggiungibile, così l'handler
  può scrivere `errors.Is(err, usecases.ErrArticleNotFound)` e scegliere il 404. Con `%v` quel legame si
  perde e l'handler potrebbe solo restituire 500.
- **È l'handler a scegliere lo status code, non lo use case.** Lo use case parla in errori Go; tradurre
  "not found" in un HTTP 404 è una decisione di trasporto che spetta all'handler. Questa separazione è ciò
  che permette allo stesso use case di stare oggi dietro l'HTTP e domani dietro altro.

### 🔬 Verifica

- `usecases/get_article_test.go` verde (trovato / non trovato / id vuoto).
- `curl localhost:8081/articles/a1` restituisce l'articolo; `curl localhost:8081/articles/nope` dà 404.

---

## Slice 2 — ChangeArticlePrice · `PUT /articles/:id/price`

### 🎯 TL;DR

Use case: `FindByID`, `NewMoney`, `Article.ChangePrice`, **no-op se il prezzo non cambia**, altrimenti
`Save` → dispatcha `ArticlePriceChanged` → `ClearPendingEvents`. Handler: bind del body, id dal path,
chiama lo use case, **200 / 400**.

### 🧠 Ragionamento

- **Le regole di business restano nell'aggregate.** Lo use case non ricontrolla "prezzo > 0" o "stessa
  valuta": chiama `Article.ChangePrice`, che le impone e restituisce errore. Lo use case fa emergere
  quell'errore e non dispatcha nulla.
- **Guardia no-op nello use case.** Confronta col prezzo precedente dopo `ChangePrice`; se invariato,
  restituisci senza `Save`/`Dispatch`. Annunciare un prezzo invariato sarebbe un evento falso.
- **Prima Save, poi dispatch; i fallimenti non dispatchano.** Ogni percorso di errore restituisce prima
  del dispatch, così una persistenza persa non produce mai un evento fantasma. `ClearPendingEvents` drena
  il fatto registrato dall'aggregate così non viene riemesso dopo.
- **L'handler è sottile.** Fa il bind di `ChangePriceRequest`, legge `c.Param("id")` e mappa il
  risultato. La valuta sta nel body perché la forma della richiesta è il contratto pulito del BC
  (`price_cents` + `currency`), non il decimale legacy.

### 🔬 Verifica

- `usecases/change_article_price_test.go` verde (happy path, no-op stesso prezzo, cambio valuta
  rifiutato, fallimenti find / save / dispatch).
- `curl -X PUT localhost:8081/articles/a1/price -d '{"price_cents":1500,"currency":"EUR"}'` restituisce
  l'articolo a 1500; una seconda chiamata identica è un no-op (ancora 200, nessun nuovo evento).

---

## Trappole comuni con l'AI (rifiuta questi diff)

- Mettere le regole di business (prezzo > 0, valuta) nello use case o nell'handler invece che
  nell'aggregate.
- Avvolgere l'errore del repo con `%v` (rompe il 404), o scegliere lo status code dentro lo use case.
- Dispatchare prima di `Save`, in un percorso di errore, o nel no-op dello stesso prezzo.
- Aggiungere auth, list, paginazione, MySQL o middleware: sono CP6, fuori scope di proposito.
- Modificare `entities/`, `events/`, `interfaces/` o `dispatcher/`.
