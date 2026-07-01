# ADR-013: Dual-Write Pattern for Data Migration During Strangler Fig

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

A Bounded Context owns its data. The monolith does not. Carving a BC out of a monolith means carving its tables out of the shared database into a private store the BC controls. While Module 2.3 prescribes two patterns for this — dual-write and event-based sync — they are not interchangeable.

This ADR commits to **dual-write** as the migration data pattern for Warehouse. Event-based sync is documented in ADR-014 (Data Products) and is used for downstream subscribers, not for the migration itself.

The choice depends on the consumer's consistency requirement, not on the BC. A request that must read its own write (e.g., create article, immediately fetch it for display) cannot tolerate the lag of an asynchronous event pipe. Warehouse's primary integration with Orders is read-after-write within the same request, so eventual consistency is unsafe during migration.

## Decision

During migration, every write to Warehouse-owned data goes to **both** databases:

```
Application
   ↓        ↓
Legacy DB   New BC DB
```

The implementation is a repository decorator (`DualWriteArticleRepository`) that wraps both the legacy and the new repository. On `Save`:

1. Begin transaction in legacy DB. Write. Commit.
2. Begin transaction in new BC DB. Write. Commit.
3. If step 2 fails, log a divergence event and (depending on policy) compensate by rolling back step 1, or accept the divergence and let a reconciler catch it.

Reads route via the facade (ADR-012):

- `legacy` mode: read from legacy DB.
- `canary` / `bc` mode: read from new BC DB.
- `dual-read` mode: read both, compare, log divergence, return legacy's response.

Once 100% of writes are happy on the new BC and divergence rate is zero for one observation window, writes to the legacy DB stop. The new BC is then the system of record.

## Consequences

**Positive:**
- Strong consistency: a read-after-write within the same request sees the just-written row regardless of which DB the read targets.
- Rollback is trivial — cut writes to the new DB; legacy is still complete.
- The application is the only thing that needs to change. No CDC, no triggers, no replication.

**Negative:**
- Double writes increase latency. Real workloads may need a write queue with bounded concurrency.
- Partial-failure handling is non-trivial. The plan must specify the compensation policy explicitly per route.
- The application carries the migration logic, which means tests must cover four states: both ok, legacy fail, new fail, both fail.

**Neutral:**
- Dual-write is *transitional*. Once cutover is complete, the decorator is deleted. Leaving it in production permanently is a smell.

## Implementation Reference

- `phase-04-db/repositories/dual_write_article_repository.go` — decorator implementation *(planned — Stream B Plan 2)*.
- `phase-04-db/repositories/divergence_logger.go` — captures and logs mismatch *(planned — Stream B Plan 2)*.
- `lab-2A-event-storming-ddd/03-strangler-safety/dual-write/` — standalone lab variant *(planned — Stream A Plan 4)*.
