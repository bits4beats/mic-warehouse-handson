# MIC → Warehouse BC: adapter e dual-write (Lezione 8 · Fase 4)

> English version: [`README.md`](./README.md)

Un lab pratico che continua l'estrazione del Bounded Context **Warehouse** da **MIC**, un monolite PHP
legacy di fatturazione, verso un microservizio **Go** pulito, con un **agente di coding AI** come motore.
La Lezione 7 ha costruito il dominio (Fasi 1–3); la **Lezione 8** gli dà un vero layer di persistenza.

Questo branch è il lab per la **Lezione 8 — Fase 4** (adapter + dual-write). Il lavoro lo fai tu; le
slide non ti danno la risposta.

## Com'è organizzata la repo

La Lezione 8 è divisa per fase, ogni fase come **due branch** (punto di partenza + soluzione di riferimento):

| Branch | Cos'è |
|---|---|
| `lezione-8-fase-4` | Punto di partenza della Fase 4. Senza soluzioni. |
| `lezione-8-fase-4-soluzione` | Fase 4 con le soluzioni svolte. |
| `lezione-8-fase-5` | Punto di partenza della Fase 5 (include la Fase 4). Senza soluzioni. |
| `lezione-8-fase-5-soluzione` | Fase 5 con le soluzioni svolte. |

Fai prima il tuo lavoro. Vai al branch soluzione solo dopo.

## I checkpoint

| Checkpoint | Cartella | Cosa fai |
|---|---|---|
| CP1 — Capire | [`phase-01-monolith/`](./phase-01-monolith/README.md) | *(Lezione 7)* Avvia MIC e mappalo. |
| CP2 — Decidere | [`phase-02-analysis/`](./phase-02-analysis/README.md) | *(Lezione 7)* Analisi DDD → il BC **Warehouse**. |
| CP3 — Costruire | [`phase-03-skeleton/`](./phase-03-skeleton/README.md) | *(Lezione 7)* Il domain layer Go: aggregate, value object, eventi, porta del repository. |
| **CP4 — Persistere** | [**`phase-04-db/`**](./phase-04-db/README.md) | **Questa lezione:** dai alla porta del repository degli adapter reali (inclusa l'**ACL** legacy) e un decorator **dual-write**. |

**Da dove iniziare:** apri [`phase-04-db/README.md`](./phase-04-db/README.md). Le Fasi 1–3 sono incluse
come contesto dalla Lezione 7.

## Usare un agente di coding AI

Gli agenti di coding AI sono parte del metodo, non una scorciatoia per aggirarlo.

- **Avvia l'agente nella cartella giusta.** Aprilo sulla cartella della fase su cui lavori (es.
  `phase-04-db/`), non sull'intera repo, così vede il codice che conta.
- **Le conclusioni sono tue.** L'agente legge, abbozza e scrive la sintassi; tu decidi il design, gli
  invarianti e cosa finisce nei tuoi deliverable.
- **Contesta.** Quando afferma una regola, chiedi *"dove nel codice l'hai vista?"* prima di fidarti.

## Riferimenti

Gli Architecture Decision Record dei pattern praticati in questo lab sono in [`docs/adr/`](./docs/adr/):
Strangler Fig (ADR-001), baseline PHP di MIC (ADR-006), clean architecture Go (ADR-007), Event Storming
(ADR-010), DDD building blocks (ADR-011), layer di Clean Architecture (ADR-002), Dual-Write (ADR-013).
