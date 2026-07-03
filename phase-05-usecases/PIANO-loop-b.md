# Piano — Loop B: ChangeArticlePrice (`PUT /articles/:id/price`)

## Context

Loop A (GetArticle) è chiuso e verde. Loop B è il secondo giro (opzionale) di Phase 5: il use case di
**cambio prezzo**, portato attorno allo stesso loop `code → Go test → handler → route → API test`. Oggi
`ChangeArticlePriceUseCase.Execute` ritorna lo stub `ErrChangeArticlePriceTODO`, il handler
`ChangeArticlePrice` è TODO (501), e la route `PUT /articles/:id/price` è commentata. I 6 test in
`usecases/change_article_price_test.go` sono RED. Obiettivo: use case + handler + route completi, quei
test verdi, endpoint funzionante da `/docs`.

Extra richiesto: **salvare questo piano come md dedicato nel repo** →
`phase-05-usecases/PIANO-loop-b.md` (primo passo in esecuzione, copia di questo file).

Vincolo di scope (README): non toccare `entities/`, `events/`, `interfaces/`, `dispatcher/`,
`handlers/openapi.go`, né lo scaffolding di test. Le regole di dominio stanno **già** nell'aggregato.

## Design — dove vivono le regole

`Article.ChangePrice(newPrice)` ([entities/article.go:68](phase-05-usecases/entities/article.go#L68)) possiede già tutte le regole:
- prezzo `<= 0` → errore;
- cambio valuta → errore (`currency change requires explicit migration`);
- prezzo uguale → **no-op**: ritorna `nil` e **non** registra evento;
- cambio reale → muta `Price`, aggiorna `UpdatedAt`, registra un evento **privato** (`articlePriceChangedEvent`).

Conseguenza per il use case: l'evento registrato è di tipo non esportato → **non** accessibile da
`usecases`. Quindi, come fa già `CreateArticleUseCase`
([usecases/create_article.go:64](phase-05-usecases/usecases/create_article.go#L64)), il use case cattura
il vecchio prezzo **prima** della mutazione e costruisce a mano il canonico `events.ArticlePriceChanged`
([events/events.go:58](phase-05-usecases/events/events.go#L58)). Il no-op si rileva da
`len(a.PendingEvents()) == 0` dopo `ChangePrice` (l'aggregato non registra nulla su prezzo invariato).

## Step 1 — Use case (`usecases/change_article_price.go`)

Implementare `Execute` sullo stampo di `CreateArticleUseCase.Execute`:

1. valida `ArticleID` non vuoto → `errors.New("ChangeArticlePrice: articleID is required")`;
2. `price, err := entities.NewMoney(in.NewPriceCents, in.Currency)`; su err → `fmt.Errorf("ChangeArticlePrice: %w", err)`;
3. `a, err := uc.repo.FindByID(ctx, in.ArticleID)`; su err → wrap (find failure risale, **nessun dispatch**);
4. `oldCents := a.Price.AmountCents` (cattura prima della mutazione);
5. `if err := a.ChangePrice(*price); err != nil` → wrap (cambio valuta / prezzo ≤0 risale, **nessun dispatch**);
6. **no-op**: `if len(a.PendingEvents()) == 0 { return &ChangeArticlePriceOutput{Article: a}, nil }`
   — niente `Save`, niente `Dispatch`;
7. `if err := uc.repo.Save(ctx, a); err != nil` → wrap (**nessun dispatch**);
8. costruisci canonico:
   ```go
   canonical := events.ArticlePriceChanged{
       ArticleID: a.ID, OldPriceCents: oldCents,
       NewPriceCents: a.Price.AmountCents, Currency: a.Price.Currency, At: a.UpdatedAt,
   }
   ```
9. `if err := uc.dispatcher.Dispatch(ctx, []events.DomainEvent{canonical}); err != nil` → wrap;
10. `a.ClearPendingEvents()`; `return &ChangeArticlePriceOutput{Article: a}, nil`.

Import: aggiungere `fmt` e `events`; `errors` resta. Rimuovere la var stub `ErrChangeArticlePriceTODO`.

**Ordine che i test impongono** (`change_article_price_test.go`): Save-poi-Dispatch; ogni fallimento
(find, save, dispatch, valuta) risale come errore; su fallimento find/save/valuta **zero eventi**
dispatchati; no-op → zero eventi e nessun Save (il test setta `FailOnSave` come sentinella).

## Step 2 — Go test

```bash
go test ./usecases/ -run TestChangeArticlePriceUseCase -v
```

Verdi tutti e 6: changesPriceSavesAndDispatches, samePriceIsNoOp, rejectsCurrencyChange,
findFailure…, saveFailure…, dispatchFailure….

## Step 3 — Handler (`handlers/article_handler.go`)

Implementare `ArticleHandler.ChangeArticlePrice` ([article_handler.go:98](phase-05-usecases/handlers/article_handler.go#L98)):
- `req := new(ChangePriceRequest)`; `if err := c.Bind(req); err != nil` → **400**;
- `out, err := h.changePriceUC.Execute(ctx, usecases.ChangeArticlePriceInput{ArticleID: c.Param("id"), NewPriceCents: req.PriceCents, Currency: req.Currency})`;
- `err != nil` → **400** `{"error": err.Error()}`;
- successo → **200** `toArticleResponse(out.Article)`.

DTO `ChangePriceRequest` e helper `toArticleResponse` già presenti. README specifica **200/400** per Loop B
(nessun 404). Nessun import nuovo.

## Step 4 — Route (`handlers/router.go`)

Scommentare in [router.go](phase-05-usecases/handlers/router.go):
```go
e.PUT("/articles/:id/price", r.articleHandler.ChangeArticlePrice)
```

## Verifica end-to-end

```bash
cd phase-05-usecases
go build ./... && go test ./...              # ora TUTTO verde (Loop A + Loop B)

# API: libera :8081 (phase-04 può occuparla) poi avvia phase-05
docker stop phase-04-db-app-1 2>/dev/null || true
docker compose up --build                     # oppure: docker compose run --rm test
```

Da **http://localhost:8081/docs** (o curl):
1. `POST /articles` `{"id":"a1","sku":"ABC-001","name":"Widget","price_cents":1000,"currency":"EUR"}` → 201;
2. `PUT /articles/a1/price` `{"price_cents":1500,"currency":"EUR"}` → 200 + prezzo 1500;
3. stesso `PUT` con `1500` di nuovo → 200, no-op (nessun evento, stato invariato);
4. `PUT` con `{"price_cents":1500,"currency":"USD"}` → 400 (cambio valuta vietato).

## Done when

- `docker compose run --rm test` verde (Loop A **e** Loop B → intera suite verde), **e**
- `PUT /articles/:id/price` funziona da `/docs`: cambia il prezzo (200), stesso prezzo → 200 no-op.

## Note

- **Save first, dispatch after; failures never dispatch** (README): se `Save` riesce ma `Dispatch`
  fallisce, lo stato cambia ma l'evento è perso — gap reale lasciato aperto di proposito (chiuso poi da
  outbox/Hermes, CP9). Da notare, **non** correggere qui.
- Regola d'architettura: le regole (prezzo>0, no cambio valuta, no-op) restano in `Article.ChangePrice`;
  il use case orchestra soltanto; il handler traduce solo HTTP.
- Diff atteso su 3 file: `usecases/change_article_price.go`, `handlers/article_handler.go`,
  `handlers/router.go` (+ il nuovo `phase-05-usecases/PIANO-loop-b.md`).
- Soluzioni di riferimento in `solutions/` (se presenti) → aprire **solo dopo** il verde.
