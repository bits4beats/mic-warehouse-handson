# ADR-007: Go Clean Architecture Implementation - Skeleton to Complete Service

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

The Strangler Fig pattern (ADR-001) requires extracting the Warehouse capability from the PHP monolith into a separate Go service. The Go service must grow phase by phase while keeping the exercise runnable and understandable at every checkpoint.

The service must:

1. Run independently from Phase 03 onward.
2. Preserve the Warehouse API and business semantics introduced by the monolith baseline.
3. Keep domain rules isolated from HTTP, database, authentication, authorization, and event transport details.
4. Teach Clean Architecture incrementally rather than presenting a complete service all at once.
5. Make every layer testable through explicit boundaries.

Go is chosen because it supports a small deployable runtime, interface-based dependency inversion, straightforward testing, and production-grade service patterns without requiring a large framework.

The current repository uses a **flat phase structure** rather than an `internal/` tree. This is intentional for the training exercise: participants can see the layers directly in each phase directory.

---

## Decision

We will implement the Warehouse service as a Clean Architecture Go application, built incrementally across Phase 03-10.

The key architectural decisions are:

1. `Article` is the Warehouse Aggregate root.
2. `InventoryLevel` is owned by the `Article` aggregate and is changed through aggregate-level behavior.
3. `ArticleRepository` is the repository port for persisting and loading the aggregate.
4. Use cases orchestrate aggregate construction, mutation, persistence, and event dispatch.
5. HTTP handlers are adapters: they translate requests into use case inputs and use case outputs into responses.
6. Authentication, authorization, observability, and event publishing are edge concerns wired outside the domain.
7. The service remains phase-runnable even when later production concerns are represented by local adapters or explicit scope gaps.

### Phase Progression

```text
Phase 03: Domain foundation
  entities/, events/, interfaces/, main.go

Phase 04: Persistence adapters
  repositories/

Phase 05: Use cases and event dispatcher boundary
  usecases/, dispatcher/

Phase 06: HTTP API adapters
  handlers/

Phase 07: Authentication and request context
  middleware/, iam-mock/

Phase 08: Authorization and OPA/Rego policy decisions
  policies/, middleware/

Phase 09: Hermes/Data Product event publishing
  events/, schemas/

Phase 10: Integrated service
  handlers/, middleware/, policies/, events/, observability/, tests/, backoffice/
```

---

## Architecture Shape

The layers are not implemented as independent deployables. They are packages inside the same Go service, with dependencies pointing inward.

```text
HTTP / Auth / Policy / Observability / Events
                 |
              Use cases
                 |
          Repository ports
                 |
       Entities and Value Objects
```

The outer packages may depend on inner packages. Inner packages must not depend on outer packages.

### Package Responsibilities

| Package | Responsibility | Introduced |
|---|---|---|
| `entities/` | Aggregate root, entities, value objects, local invariants | Phase 03 |
| `events/` | Domain events and, later, Hermes/Data Product adapter | Phase 03, extended in Phase 09 |
| `interfaces/` | Repository port for aggregate persistence | Phase 03 |
| `repositories/` | MySQL adapter, in-memory adapter, dual-write decorator, divergence logger | Phase 04 |
| `dispatcher/` | Event dispatch port used by use cases | Phase 05 |
| `usecases/` | Application workflows and business orchestration | Phase 05 |
| `handlers/` | Echo HTTP adapter and route registration | Phase 06 |
| `middleware/` | Auth, M2M, correlation, request context | Phase 07 |
| `policies/` | In-memory policy enforcer and OPA client adapter | Phase 08 |
| `schemas/` | Local Data Product schemas used by the Hermes adapter | Phase 09 |
| `observability/` | Request logging and trace/correlation support | Phase 10 |
| `tests/` | E2E-style tests for the integrated service | Phase 10 |

---

## Implementation Map

### Phase 03: Domain Foundation

**Objective:** establish the domain model and the first dependency boundary.

Key files:

- `phase-03-skeleton/entities/article.go`
- `phase-03-skeleton/entities/inventory.go`
- `phase-03-skeleton/entities/money.go`
- `phase-03-skeleton/entities/sku.go`
- `phase-03-skeleton/events/events.go`
- `phase-03-skeleton/interfaces/repository.go`
- `phase-03-skeleton/main.go`

Important choices:

- `Article` is the Aggregate root.
- `Money` and `SKU` are Value Objects with local validation.
- `InventoryLevel` is not saved through an independent repository.
- `ArticleRepository` exposes aggregate-level methods: `Save`, `FindByID`, `FindBySKU`, `List`, `Delete`.
- Aggregates record pending events; they do not publish them.
- `main.go` only provides a minimal Echo health endpoint so the skeleton is runnable.

## Testing Strategy

The exercise uses tests to validate each boundary at the level where it matters:

| Area | Test style |
|---|---|
| Entities and Value Objects | Unit tests for invariants and mutation behavior |
| Repository port | Compile-time interface checks and focused repository tests |
| Use cases | Unit tests with in-memory repositories and dispatchers |
| Handlers | HTTP handler tests with in-memory use case wiring |
| Middleware | Token, M2M, correlation, and skip-path tests |
| Policies | In-memory policy tests and OPA client decision tests |
| Events | Schema validation and CloudEvents envelope tests |
| Integration | `httptest` E2E flow in Phase 10 |

Tests should preserve the Clean Architecture rule: inner layers are tested without outer adapters.

---

## Consequences

### Positive Outcomes

- Participants can inspect one new architectural concern per phase.
- The domain model stays independent from database, HTTP, auth, policy, and event transport details.
- Use cases remain testable because repositories, dispatchers, and policy decisions are represented by explicit boundaries.
- The aggregate-root rule is reinforced by making inventory changes flow through `Article`.
- Later platform concerns can be taught through local adapters without changing the domain model.

### Tradeoffs

- The flat package structure is more didactic than a conventional production `internal/` layout.
- Some production integrations are represented by local adapters or mocks.
- Dual-write is not complete until a real legacy translation adapter exists.
- Participants need to distinguish Clean Architecture layers from cross-cutting adapters such as auth, policy, observability, and events.

---

## References

**Related Exercise ADRs**

- ADR-001: Strangler Fig Pattern
- ADR-002: Clean Architecture Layers
- ADR-006: MIC PHP Monolith Baseline
- ADR-008: MySQL Repository Pattern
- ADR-009: HTTP Handlers + Echo
- ADR-011: DDD Building Blocks
- ADR-013: Dual-Write Pattern
- ADR-014: Data Products on Hermes
- ADR-016: Pact Testing and API Deprecation
- ADR-017: Observability

**Key Phase References**

- `phase-03-skeleton/README.md`

---

## Success Criteria

By the end of the exercise:

1. A participant can explain why `Article` is the Aggregate root.
2. A participant can point to the repository port and its MySQL implementation.
3. A participant can explain why handlers call use cases rather than repositories.
4. A participant can explain why OPA, Hermes, auth, and observability stay at the edge.
5. A participant can run tests for each layer and interpret what boundary each test protects.
6. A participant can identify the current dual-write scope gap and the missing legacy adapter needed to close it.
