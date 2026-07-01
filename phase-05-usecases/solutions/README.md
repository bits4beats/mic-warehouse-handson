# Phase 05 — Solutions

> Italian version: [`README-IT.md`](./README-IT.md)

> ⚠ **Spoiler.** Worked answers for the two vertical slices in [`../README.md`](../README.md).
> Open this **only after your slice is green**. The value is in building the slice yourself.

Reference files (`.txt` so Go does not compile them):

- Slice 1 use case → [`get_article.expected.go.txt`](./get_article.expected.go.txt)
- Slice 2 use case → [`change_article_price.expected.go.txt`](./change_article_price.expected.go.txt)
- Handlers (both) → [`article_handler.expected.go.txt`](./article_handler.expected.go.txt)
- Router (both routes) → [`router.expected.go.txt`](./router.expected.go.txt)

Each block: **🎯 TL;DR · 🧠 Reasoning · 🔬 Verification.**

---

## Slice 1 — GetArticle · `GET /articles/:id`

### 🎯 TL;DR

Use case: validate the id, `repo.FindByID`, **wrap the error** with `%w`. Handler: read the path param,
call the use case, map `ErrArticleNotFound` to **404**, otherwise **200**.

### 🧠 Reasoning

- **Why wrap the error with `%w`?** The repository returns `ErrArticleNotFound`; the use case wraps it
  (`fmt.Errorf("GetArticle: %w", err)`). The `%w` verb keeps the sentinel reachable, so the handler can
  still write `errors.Is(err, usecases.ErrArticleNotFound)` and choose 404. Wrap with `%v` and that link
  is lost — the handler could only ever return 500.
- **The handler chooses the status code, the use case does not.** The use case speaks Go errors; turning
  "not found" into an HTTP 404 is a transport decision that belongs to the handler. That separation is
  what lets the same use case sit behind HTTP today and something else later.

### 🔬 Verification

- `usecases/get_article_test.go` green (found / not-found / empty-id).
- `curl localhost:8081/articles/a1` returns the article; `curl localhost:8081/articles/nope` returns 404.

---

## Slice 2 — ChangeArticlePrice · `PUT /articles/:id/price`

### 🎯 TL;DR

Use case: `FindByID`, `NewMoney`, `Article.ChangePrice`, **no-op if the price is unchanged**, else
`Save` → dispatch `ArticlePriceChanged` → `ClearPendingEvents`. Handler: bind the body, take the id from
the path, call the use case, **200 / 400**.

### 🧠 Reasoning

- **Business rules stay in the aggregate.** The use case does not re-check "price > 0" or "same currency":
  it calls `Article.ChangePrice`, which enforces them and returns an error. The use case surfaces that
  error and dispatches nothing.
- **No-op guard in the use case.** Compare with the old price after `ChangePrice`; if unchanged, return
  without `Save`/`Dispatch`. Announcing an unchanged price would be a false event.
- **Save first, dispatch after; failures never dispatch.** Every failure path returns the error before
  dispatch, so a lost persistence never produces a phantom event. `ClearPendingEvents` drains the fact the
  aggregate recorded so it is not re-emitted later.
- **The handler is thin.** It binds `ChangePriceRequest`, reads `c.Param("id")`, and maps the result.
  Currency lives in the body because the request shape is the BC's clean contract (`price_cents` +
  `currency`), not the legacy decimal.

### 🔬 Verification

- `usecases/change_article_price_test.go` green (happy path, same-price no-op, currency-change rejected,
  find / save / dispatch failures).
- `curl -X PUT localhost:8081/articles/a1/price -d '{"price_cents":1500,"currency":"EUR"}'` returns the
  article at 1500; a second identical call is a no-op (still 200, no new event).

---

## Common AI pitfalls (reject these diffs)

- Putting business rules (price > 0, currency) in the use case or the handler instead of the aggregate.
- Wrapping the repo error with `%v` (breaks the 404), or choosing status codes inside the use case.
- Dispatching before `Save`, on a failure path, or on the same-price no-op.
- Adding auth, list, pagination, MySQL, or middleware — those are CP6, deliberately out of scope here.
- Editing `entities/`, `events/`, `interfaces/`, or `dispatcher/`.
