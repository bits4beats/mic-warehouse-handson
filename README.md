# MIC → Warehouse BC: a hands-on extraction (Lesson 7)

> Versione italiana: [`README-IT.md`](./README-IT.md)

A hands-on lab: extract the **Warehouse** Bounded Context out of **MIC**, a legacy PHP invoicing
monolith, into a clean **Go** microservice, applying the **Strangler Fig** pattern with an **AI
coding agent** as your engine.

This repository is the lab for **Lesson 7**. You do the work; the slides do not hand you the answer.

## How this repo is organised

Each lesson is **two branches**:

| Branch | What it is |
|---|---|
| `lezione-7` | The **starting point** you work from, and what you get when you clone. No solutions. |
| `lezione-7-soluzione` | The **reference solution**: one commit on top of the starting point, adding the worked solutions for Phase 01 and the developed Go skeleton for Phase 03. |

Build your own work first. Reach for the solution only afterwards: `git switch lezione-7-soluzione`,
or browse that branch on the repository host.

## The three checkpoints

| Checkpoint | Folder | What you do |
|---|---|---|
| CP1 — Understand | [`phase-01-monolith/`](./phase-01-monolith/README.md) | Bring MIC up and map it: page map, architecture, repo guide, coupling map, worst-of list. |
| CP2 — Decide | [`phase-02-analysis/`](./phase-02-analysis/README.md) | The DDD analysis (Event Storming, Ubiquitous Language, context mapping, dependency map) that names the BC to extract: **Warehouse**. |
| CP3 — Build | [`phase-03-skeleton/`](./phase-03-skeleton/README.md) | Build the Warehouse **Go domain layer** yourself: aggregate, value objects, events, repository port. |

**Where to start:** open the `README.md` inside [`phase-01-monolith/`](./phase-01-monolith/README.md)
and follow the checkpoints in order. Each phase folder has its own guide.

## Using an AI coding agent

AI coding agents are part of the method here, not a shortcut around it.

- **Start the agent in the right folder.** Open it on the phase folder you are working in (e.g.
  `phase-01-monolith/`), not the whole repo, so it sees the code that matters.
- **You own the conclusions.** The agent reads, drafts, and writes syntax; you decide the design,
  the invariants, and what goes into your deliverables.
- **Push back.** When it asserts a coupling or a rule, ask *"where in the code or schema did you see
  that?"* before you trust it.

## Reference

Architecture Decision Records for the patterns this lab practises live in
[`docs/adr/`](./docs/adr/): Strangler Fig (ADR-001), the MIC PHP baseline (ADR-006), Go clean
architecture (ADR-007), Event Storming (ADR-010), and the DDD building blocks (ADR-011).
