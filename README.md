# MIC → Warehouse BC: adapters & dual-write (Lesson 8 · Phase 4)

> Versione italiana: [`README-IT.md`](./README-IT.md)

A hands-on lab continuing the extraction of the **Warehouse** Bounded Context out of **MIC**, a legacy
PHP invoicing monolith, into a clean **Go** microservice, with an **AI coding agent** as your engine.
Lesson 7 built the domain (Phases 1–3); **Lesson 8** gives it a real persistence layer.

This branch is the lab for **Lesson 8 — Phase 4** (adapters + dual-write). You do the work; the slides
do not hand you the answer.

## How this repo is organised

Lesson 8 is split by phase, each phase as **two branches** (starting point + reference solution):

| Branch | What it is |
|---|---|
| `lezione-8-fase-4` | Phase 4 starting point. No solutions. |
| `lezione-8-fase-4-soluzione` | Phase 4 with the worked solutions. |
| `lezione-8-fase-5` | Phase 5 starting point (includes Phase 4). No solutions. |
| `lezione-8-fase-5-soluzione` | Phase 5 with the worked solutions. |

Build your own work first. Reach for the solution branch only afterwards.

## The checkpoints

| Checkpoint | Folder | What you do |
|---|---|---|
| CP1 — Understand | [`phase-01-monolith/`](./phase-01-monolith/README.md) | *(Lesson 7)* Bring MIC up and map it. |
| CP2 — Decide | [`phase-02-analysis/`](./phase-02-analysis/README.md) | *(Lesson 7)* DDD analysis → the **Warehouse** BC. |
| CP3 — Build | [`phase-03-skeleton/`](./phase-03-skeleton/README.md) | *(Lesson 7)* The Go domain layer: aggregate, value objects, events, repository port. |
| **CP4 — Persist** | [**`phase-04-db/`**](./phase-04-db/README.md) | **This lesson:** give the repository port real adapters (incl. the legacy **ACL**) and a **dual-write** decorator. |

**Where to start:** open [`phase-04-db/README.md`](./phase-04-db/README.md). Phases 1–3 are included as
context from Lesson 7.

## Using an AI coding agent

AI coding agents are part of the method here, not a shortcut around it.

- **Start the agent in the right folder.** Open it on the phase folder you are working in (e.g.
  `phase-04-db/`), not the whole repo, so it sees the code that matters.
- **You own the conclusions.** The agent reads, drafts, and writes syntax; you decide the design, the
  invariants, and what goes into your deliverables.
- **Push back.** When it asserts a rule, ask *"where in the code did you see that?"* before you trust it.

## Reference

Architecture Decision Records for the patterns this lab practises live in [`docs/adr/`](./docs/adr/):
Strangler Fig (ADR-001), MIC PHP baseline (ADR-006), Go clean architecture (ADR-007), Event Storming
(ADR-010), DDD building blocks (ADR-011), Clean Architecture layers (ADR-002), Dual-Write (ADR-013).
