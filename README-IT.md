# MIC → Warehouse BC: un'estrazione hands-on (Lezione 7)

> English version: [`README.md`](./README.md)

Un lab hands-on: estrarre la Bounded Context **Warehouse** da **MIC**, un monolite PHP legacy di
fatturazione, dentro un microservizio **Go** pulito, applicando il pattern **Strangler Fig** con un
**agente AI di coding** come motore.

Questa repo e' il lab della **Lezione 7**. Il lavoro lo fai tu: le slide non ti danno la risposta.

## Com'e' organizzata la repo

Ogni lezione e' fatta di **due branch**:

| Branch | Cos'e' |
|---|---|
| `lezione-7` | Il **punto di partenza** da cui lavori, ed e' quello che ottieni quando cloni. Nessuna soluzione. |
| `lezione-7-soluzione` | La **soluzione di riferimento**: un commit sopra al punto di partenza, che aggiunge le soluzioni svolte della Phase 01 e lo skeleton Go sviluppato della Phase 03. |

Costruisci prima il tuo lavoro. Vai alla soluzione solo dopo: `git switch lezione-7-soluzione`,
oppure sfoglia quel branch sull'host della repo.

## I tre checkpoint

| Checkpoint | Cartella | Cosa fai |
|---|---|---|
| CP1 — Capire | [`phase-01-monolith/`](./phase-01-monolith/README-IT.md) | Avvia MIC e mappalo: mappa delle pagine, architettura, guida alla repo, mappa degli accoppiamenti, lista dei peggiori. |
| CP2 — Decidere | [`phase-02-analysis/`](./phase-02-analysis/README-IT.md) | L'analisi DDD (Event Storming, Ubiquitous Language, context mapping, dependency map) che nomina la BC da estrarre: **Warehouse**. |
| CP3 — Costruire | [`phase-03-skeleton/`](./phase-03-skeleton/README-IT.md) | Costruisci tu il **domain layer Go** di Warehouse: aggregate, value object, eventi, repository port. |

**Da dove partire:** apri il `README.md` dentro
[`phase-01-monolith/`](./phase-01-monolith/README-IT.md) e segui i checkpoint in ordine. Ogni
cartella di fase ha la sua guida.

## Usare un agente AI di coding

Gli agenti AI di coding sono parte del metodo, non una scorciatoia per aggirarlo.

- **Avvia l'agente nella cartella giusta.** Aprilo sulla cartella della fase su cui stai lavorando
  (es. `phase-01-monolith/`), non sull'intera repo, cosi' vede il codice che conta.
- **Le conclusioni sono tue.** L'agente legge, abbozza e scrive la sintassi; tu decidi il design,
  gli invarianti e cosa finisce nei tuoi deliverable.
- **Mettilo in discussione.** Quando asserisce un accoppiamento o una regola, chiedi *"dove nel
  codice o nello schema l'hai visto?"* prima di fidarti.

## Riferimenti

Gli Architecture Decision Record dei pattern che questo lab pratica sono in
[`docs/adr/`](./docs/adr/): Strangler Fig (ADR-001), la baseline PHP di MIC (ADR-006), la clean
architecture Go (ADR-007), l'Event Storming (ADR-010) e i building block DDD (ADR-011).
