# ADR-001: Strangler Fig Pattern for Warehouse BC Extraction

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

The PHP monolithic application tightly couples three critical business capabilities: Orders, Invoicing, and Warehouse. These modules share database tables, business logic, and API endpoints, creating strong dependencies that make independent evolution impossible.

The Warehouse module—responsible for inventory management, stock tracking, and article definitions—is a prime candidate for extraction into a separate bounded context. However, a "big-bang" rewrite carries substantial risk:

- **Downtime:** Coordinating a complete cutover requires synchronizing Orders, Invoicing, and Warehouse simultaneously
- **Regression risk:** Migrating years of legacy logic increases the surface area for bugs
- **Team context loss:** Two teams working in parallel without shared deployment authority creates coordination overhead
- **Rollback complexity:** If the new service fails, reverting to the monolith becomes architecturally messy

The business requires:

- **Zero-downtime extraction:** Orders and Invoicing continue operating without interruption
- **Gradual confidence building:** New service behavior validated against legacy service before full cutover
- **Easy rollback:** If the extracted service fails at any phase, we revert to monolith seamlessly
- **Production-grade reliability:** Handling concurrent requests from multiple clients safely

The team lacks practical experience with incremental architectural refactoring at scale. This ADR establishes a proven pattern that teaches both technical execution and organizational change management.

---

## Decision

We will adopt the **Strangler Fig Pattern** to gradually replace the Warehouse module with a new Go-based service while maintaining full backward compatibility and zero-downtime operation.

The Strangler Fig pattern (named after the tropical vine that gradually envelops a host tree) involves:

1. Building the new service alongside the monolith
2. Routing a percentage of traffic to the new service while maintaining the monolith as a fallback
3. Gradually increasing traffic ratio as confidence grows
4. Decommissioning the monolith Warehouse module only after complete cutover

This approach is proven in real-world microservice migrations (Amazon, Netflix, Shopify) and provides:

- **Safety:** Dual-write ensures data consistency while validating new service behavior
- **Confidence building:** Percentile routing lets us validate in production with limited blast radius
- **Rollback simplicity:** Any phase can revert to the monolith immediately
- **Team autonomy:** Orders and Invoicing teams continue deploying independently

---

## Implementation

### Phase 1: Parallel Operation (Weeks 1-3)

**Objective:** New Go service accepts traffic in read-only mode while monolith continues all operations.

**Technical setup:**

- Deploy Warehouse Go service (initially unstable)
- Implement dual-write at the monolith level: writes to both PostgreSQL (monolith) and Go service internal state simultaneously
- Deploy routing proxy layer that:
  - Routes `GET /warehouse/*` requests to Go service (with fallback to monolith on error)
  - Routes `POST/PUT/DELETE /warehouse/*` requests exclusively to monolith
  - Logs all routing decisions for audit trail
- Implement health checks: Go service reports readiness percentage to routing proxy

**Validation:**

- Unit tests on new Go service pass (no external dependencies)
- Integration tests validate data sync between monolith and Go service
- Read-only load tests compare response times (monolith vs. Go service)
- Shadow traffic from staging: send production-like payloads to Go service, discard responses
- Manual QA: developers can enable feature flag to route their traffic to Go service

**Success criteria:**

- 100% of reads can be answered by Go service with <5% latency difference
- Zero data loss during dual-write for 48 hours continuous operation
- All integration tests passing in staging environment

### Phase 2: Percentile Routing (Weeks 4-6)

**Objective:** Route increasing percentages of production traffic to Go service (read + write operations).

**Technical setup:**

- Extend dual-write to include `POST/PUT/DELETE` in Go service (alongside monolith writes)
- Routing proxy routes based on configurable percentile: 10% → 25% → 50% → 75% → 90%
- Implement distributed tracing: correlate monolith and Go service responses for the same logical request
- Metrics dashboard: compare latency, error rates, and consistency metrics between services

**Validation:**

- Load tests at each percentile level (10%, 25%, 50%)
- Chaos engineering: intentionally fail Go service at 10% traffic, verify fallback works
- Data consistency checks: randomly sample operations, verify Go service and monolith reach same final state
- Business user acceptance testing: run sample workflows (receive order, update inventory, invoice) on 10% traffic

**Success criteria:**

- 50% of production traffic routed to Go service with zero data loss
- Error rates within 0.1% of monolith baseline
- Latency p99 within 10% of monolith across all operations
- Fallback to monolith works within 100ms on any Go service error

### Phase 3: Fallback & Monitoring (Weeks 7-8)

**Objective:** Validate that rollback is safe and automatic; finalize monitoring for permanent cutover.

**Technical setup:**

- Implement automatic fallback: if Go service error rate exceeds 1% for 30 seconds, route 100% traffic back to monolith
- Implement consistency checker: background job verifies that Go service state matches monolith for 100% of operations
- Alert on divergence: if any inconsistency detected, page on-call engineer immediately
- Finalize observability: logs, metrics, traces for all critical paths in Go service

**Validation:**

- Perform controlled failure tests: kill Go service pods, verify automatic fallback, measure recovery time
- Verify consistency checker accuracy: inject intentional inconsistencies, ensure detection within 1 minute
- Chaos at scale: simulate network partitions, database failures, cascading errors
- Long-duration soak test: 72 hours of production traffic at 90% routing to Go service

**Success criteria:**

- Automatic fallback activates and completes within 100ms
- Consistency checker runs continuously with zero false positives
- Monitoring alert accuracy >99%
- No business impact during any failure scenario

### Phase 4: Permanent Cutover (Week 9+)

**Objective:** Complete migration; decommission monolith Warehouse module.

**Technical approach:**

- Route 100% of Warehouse traffic exclusively to Go service
- Deactivate dual-write in monolith (stop writing to internal state)
- Remove routing proxy; Go service becomes canonical Warehouse module
- Archive monolith Warehouse code but retain in version control for historical reference
- Optional: Sunset monolith database Warehouse tables after 30-day retention period

**Rollback strategy:**

- If any critical incident occurs post-cutover, immediately re-enable dual-write and percentile routing
- Revert to 50/50 routing while investigation occurs
- No data loss because Go service remains authoritative

---

## Consequences

### Positive Outcomes

✅ **Zero-downtime evolution:** Orders and Invoicing never experience outages; customers are unaware of internal reorganization.

✅ **Confidence through gradual exposure:** We validate new service behavior at 10%, 25%, 50% traffic before full cutover, catching issues before they affect all customers.

✅ **Rollback simplicity:** Any phase can revert to the monolith in <100ms with automatic fallback. No manual coordination required.

✅ **Real-world practice:** Team experiences production safety patterns used by tier-1 companies (Amazon, Netflix). This pattern applies to future BC extractions.

✅ **Organizational learning:** Teaches incremental change over big-bang rewrites. Builds confidence in distributed systems.

✅ **Data consistency validation:** Dual-write with periodic verification ensures confidence in new service's correctness before monolith is decommissioned.

### Tradeoffs & Challenges

⚠️ **Temporary duplication:** For weeks 1-4, the Warehouse module exists in both monolith and Go service. This doubles storage (temporarily) and complicates deployment.

⚠️ **Dual-write complexity:** Monolith must consistently write to two systems during phases 1-3. If either system fails, we must handle partial writes and recovery.

⚠️ **Operational overhead:** Maintaining routing proxy, consistency checker, and fallback logic adds observability burden (though this becomes reusable infrastructure).

⚠️ **Testing complexity:** Validating dual-write and fallback scenarios requires sophisticated integration tests and load testing expertise.

⚠️ **Timing risk:** If Go service encounters production bugs during percentile routing, we incur blast radius proportional to routing percentage. Mitigation: aggressive monitoring and automatic fallback.

⚠️ **Data modeling mismatches:** If Go service schema diverges from monolith, dual-write may silently fail or introduce inconsistencies. Requires strict schema governance.

---

## Learning Goals

Participants will understand:

1. **Why incremental extraction beats big-bang rewrites:** Psychological safety, rollback guarantees, confidence building
2. **How to implement strangler pattern in practice:** Routing proxy, dual-write, fallback, monitoring
3. **Operational patterns for distributed systems:** Percentile routing, chaos testing, consistency checking
4. **Data migration strategies:** Dual-write, consistency verification, eventual cutover
5. **Organization dynamics:** Managing technical risk, building stakeholder confidence, coordinating across teams

By completing checkpoints 1-4, the team will have hands-on experience with:

- Building a clean architecture service that can accept traffic in parallel with a monolith
- Implementing dual-write without causing data loss
- Setting up monitoring to catch subtle bugs (inconsistencies, latency regressions)
- Executing a production migration safely

This foundation enables safe extraction of additional bounded contexts (Invoicing, Orders) using the same proven pattern.

---

## References

- **Sam Newman, "Building Microservices" (2nd ed.)** - Chapter 3 covers strangler fig pattern with real examples
- **Martin Fowler, "Strangler Fig Application"** (https://martinfowler.com/bliki/StranglerFigApplication.html)
- **Amazon's monolith-to-services transition** - Documented in "All Things Distributed" blog
- **Deployment patterns:** Feature flags, canary releases, blue-green deployments (complementary techniques)

---

## Implementation Reference

This ADR is implemented across exercise checkpoints as follows:

### Phase 02: Analysis & Design (Checkpoint 2)

- **Files:** `phase-02-analysis/DEPENDENCY-MAP.md`
- **Focus:** Understanding current monolith dependencies and designing extraction boundaries
- **Key artifacts:** Dependency map showing Orders → Warehouse, API endpoint inventory
