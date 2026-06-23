# ADR-010: Event Storming as the Method to Find Bounded Contexts

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

Module 2.3 of Tech Track 2 establishes that a Business Capability is *found*, not *given*. The hard part of decomposition is drawing the line between BCs. The repository today documents Strangler Fig (ADR-001) and clean architecture layers (ADR-002), but does not document *how* candidate BCs are identified in the first place.

Without an explicit method, teams default to two failure modes:

- **Naming first.** They write a list of BC names ("Catalog", "Orders", "Returns") and reproduce the structure they already imagine the system has, not the structure the business actually exhibits.
- **Schema first.** They look at database tables and group them. The result is software the database understands, not software the business understands.

The training prescribes Event Storming — a workshop technique born in the DDD community — as the cheapest insurance against bad boundaries. It produces two artifacts (a list of past-tense domain events, and a clustering of those events into BC candidates) before anyone names a BC or designs a table.

## Decision

We adopt **Event Storming** as the canonical method for identifying Bounded Contexts in this exercise and in any future BC extraction work.

The method has exactly three steps, in this order:

1. **Enumerate.** List every meaningful business event from the domain narrative, in the past tense, with no filtering.
2. **Cluster.** Group events that hang together. Each cluster traces an aggregate.
3. **Name.** Each cluster becomes a BC candidate. Naming happens last.

The output is a two-section canvas: domain events on the left, bounded contexts on the right.

In this repository, Phase 02 (`phase-02-analysis/`) hosts the canonical canvas for the Warehouse extraction. Lab 2.A (later labs) hosts a paper exercise built on the ShopRight narrative, used as a teaching artifact.

## Consequences

**Positive:**
- BC boundaries are defended by an artifact (the canvas), not by an opinion.
- The vocabulary used in Event Storming (past-tense events, aggregates) carries directly into the code (ADR-011).
- A new joiner can read the canvas and reconstruct the reasoning without rerunning the workshop.

**Negative:**
- The method requires domain access. If the team running the workshop has no business stakeholders, the canvas is guesswork.
- Step 1 (enumerate without filtering) feels wasteful and is the most-skipped step. The plan must call this out explicitly.

**Neutral:**
- Output is intentionally low-fidelity (sticky notes, spreadsheet rows). It is not a UML diagram and should not be promoted into one.

## Implementation Reference

- `phase-02-analysis/event-storming-canvas.md` — canonical canvas for the Warehouse extraction.
- `phase-02-analysis/ubiquitous-language.md` — the Ubiquitous Language surfaced by the canvas, refined into the canonical Warehouse vocabulary (with AI prompts to validate code against it).
- ADR-011 picks up the DDD vocabulary that Event Storming surfaces.
