# ADR-014: Data Products as Versioned Hermes Contracts

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

ADR-004 commits the team to Hermes for event streaming. That ADR treats events as messages — typed payloads emitted by Warehouse and consumed by interested parties. Module 2.2 of the training raises the bar: events on Hermes are not just messages, they are **Data Products** — schema-validated, versioned data definitions contributed to the datalake, with replay guarantees and a published lifecycle.

Today the codebase publishes `ArticleCreated` and `InventoryUpdated` as plain Go structs. There is no schema registry, no versioning policy, no lifecycle for deprecated event versions. A consumer reading the code has no way to know whether a field is required, whether a new version is coming, or whether the producer plans to remove a field next quarter.

## Decision

Every domain event (ADR-011) that crosses a BC boundary on Hermes is treated as a Data Product with the following obligations:

1. **Schema.** Every event has a JSON Schema (Draft 2020-12) checked into `phase-09-events/schemas/`. The schema is the contract. Producers validate before publishing; consumers validate before consuming.
2. **Versioning.** Schemas are versioned (`v1`, `v2`, …). New optional fields are a non-breaking change. Renamed or removed fields require a new major version.
3. **Lifecycle.** Each schema declares its lifecycle stage (`stable`, `deprecated`, `sunset`) in metadata. Deprecated schemas carry a `sunset_date` field.
4. **Replay.** Hermes guarantees historical replay from the event log. Consumers can rebuild state from `t=0` for any data product they subscribe to.

The published artifact is the schema, not the Go struct. Consumers in other languages (PHP, Python) read the schema, not the producer's source code.

## Consequences

**Positive:**
- Cross-team integration becomes a contract negotiation, not a Slack message.
- Consumers detect incompatible producer changes at deploy time (schema validation fails) instead of in production (parsing crashes).
- Late joiners can reconstruct historical state by replaying the event log against the latest schema validator.

**Negative:**
- Producers cannot evolve schemas casually. A new required field is a major version bump.
- The team must maintain a schema registry. For the exercise, that registry is the `schemas/` directory + git history. In production, it would be a managed registry.

**Neutral:**
- ADR-016 (Pact + OpenAPI diff + Deprecation) governs API contracts. ADR-014 governs event contracts. The two ADRs share lifecycle vocabulary on purpose.

## Implementation Reference

- `phase-09-events/schemas/article-dp-v1.json` — Article Data Product v1 schema *(planned — Stream B Plan 3)*.
- `phase-09-events/schemas/inventory-dp-v1.json` — Inventory Data Product v1 schema *(planned — Stream B Plan 3)*.
- `phase-09-events/events/data_product.go` — schema validator middleware *(planned — Stream B Plan 3)*.
- `lab-2B-contracts-decoupling/01-data-products/` — standalone lab variant with consumer + replay demo *(planned — Stream A Plan 5)*.
