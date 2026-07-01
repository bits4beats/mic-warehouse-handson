# ADR-002: Clean Architecture Layers for Go Service

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

The new Warehouse Go service will be read and modified by approximately 200 engineers across multiple teams (Warehouse, Orders, Invoicing, Frontend). These engineers bring varying levels of experience:
- Senior architects familiar with distributed systems
- Mid-level engineers from monolith background (procedural PHP)
- Junior engineers and interns learning Go for the first time

The monolith suffers from a common disease: **business logic scattered across layers.** Authorization logic lives in controllers and models. Domain rules are enforced in database triggers and ad-hoc service methods. State mutations happen in repositories with no clear transaction boundaries. This makes the monolith:

- **Hard to test:** Changing a business rule requires mocking 5+ dependencies
- **Hard to understand:** No clear entry points for "how does the system make an authorization decision?"
- **Hard to change safely:** Refactoring one feature often breaks another due to hidden dependencies

The team needs architectural patterns that provide:

1. **Clear boundaries:** Every engineer can answer "where does this logic live?" within 30 seconds
2. **Testability:** Unit tests run without database; integration tests verify data persistence; e2e tests validate HTTP contracts
3. **Team scalability:** Junior engineers can add features without breaking existing ones; architectural decisions enforce safety
4. **Knowledge reuse:** Patterns learned in Warehouse apply to Orders and Invoicing extractions

The challenge: Go's simplicity (no frameworks, no magic) means the team must be **intentional** about layering. Without agreed-upon architecture, engineers will invent local conventions that diverge across the codebase.

---

## Decision

We will implement **Clean Architecture** (as defined by Uncle Bob Robert C. Martin) with four explicit layers:

1. **Entities:** Core domain models and business rules
2. **Use Cases:** Application-level orchestration and authorization
3. **Interface Adapters:** HTTP, database, event bus adapters
4. **Frameworks & Drivers:** Go standard library, external packages

Each layer has clear responsibilities, dependencies flow inward, and business logic is independent of infrastructure.

### Layer Responsibilities

| Layer | Responsibility | Examples | Dependencies |
|-------|-----------------|----------|--------------|
| **Entities** | Core domain models; immutable business rules | `Article`, `Inventory`, `StockReservation` | None (pure Go) |
| **Use Cases** | Orchestrate entities; enforce authorization; publish events | `CreateArticle`, `ReserveStock`, `UpdateInventory` | Entities, external policies |
| **Interface Adapters** | Convert between HTTP/database formats and domain models | HTTP handlers, repository implementations, event publishers | Use Cases, Entities |
| **Frameworks & Drivers** | Framework setup, dependency injection, server initialization | Go `net/http`, postgres driver, OPA client | All layers |

### Architectural Principles

**Dependency Inversion:** Inner layers (business logic) never depend on outer layers (infrastructure). A use case doesn't know about HTTP or databases; adapters call use cases.

**Single Responsibility:** Each file, function, and layer has one reason to change. Changing authorization rules doesn't require changing HTTP routing or database schemas.

**Abstraction:** Each layer communicates via interfaces. Use cases depend on abstract `ArticleRepository`, not concrete PostgreSQL implementation. This enables testing with in-memory repositories.

**Clarity over cleverness:** Explicit is better than implicit. A 20-line straightforward function beats a 5-line function that requires 10 minutes of documentation.

---

## Implementation

### Directory Structure

```
warehouse/
├── entities/                    # Layer 1: Core domain
│   ├── article.go             # Domain model: Article, SKU, pricing
│   ├── inventory.go           # Domain model: Stock levels, reservations
│   ├── business_rules.go      # Pure functions: validation, calculations
│   └── errors.go              # Domain-specific errors
├── usecases/                   # Layer 2: Application logic
│   ├── create_article.go      # Orchestrate entity creation
│   ├── reserve_stock.go       # Orchestrate reservation (with authorization)
│   ├── update_inventory.go    # Orchestrate inventory mutation
│   ├── interfaces.go          # Contracts for repositories, policies, events
│   └── errors.go              # Use case errors (authorization, validation)
├── adapters/                   # Layer 3: Infrastructure conversion
│   ├── http/
│   │   ├── handlers.go        # HTTP handlers (calls use cases)
│   │   ├── serializers.go     # Convert domain models ↔ JSON
│   │   └── middleware.go      # Logging, error handling
│   ├── postgres/
│   │   ├── article_repo.go    # Implements ArticleRepository interface
│   │   ├── inventory_repo.go  # Implements InventoryRepository interface
│   │   ├── migrations/        # SQL migrations
│   │   └── query_helpers.go   # Shared database logic
│   └── events/
│       └── hermes_publisher.go # Implements EventPublisher interface
├── config/                     # Layer 4: Setup
│   ├── database.go            # Database connection, pool config
│   ├── logger.go              # Logger initialization
│   ├── opa.go                 # OPA client setup
│   └── server.go              # HTTP server initialization
├── main.go                     # Entry point
└── tests/                      # Test utilities
    ├── fixtures/              # Reusable test data
    └── mocks/                 # Mock implementations of interfaces
```

### Concrete Example: Create Article Use Case

**Entities layer** (`entities/article.go`):
```go
package entities

// Article represents a warehouse article (domain model)
type Article struct {
    ID          string
    SKU         string
    Name        string
    Price       decimal.Decimal
    CreatedAt   time.Time
}

// NewArticle creates an article with validation (business rule)
func NewArticle(sku, name string, price decimal.Decimal) (*Article, error) {
    if sku == "" {
        return nil, ErrInvalidSKU
    }
    if price.LessThan(decimal.Zero) {
        return nil, ErrNegativePrice
    }
    return &Article{
        ID:        uuid.New().String(),
        SKU:       sku,
        Name:      name,
        Price:     price,
        CreatedAt: time.Now(),
    }, nil
}
```

**Use cases layer** (`usecases/create_article.go`):
```go
package usecases

type CreateArticleRequest struct {
    SKU   string
    Name  string
    Price decimal.Decimal
}

type CreateArticleUseCase struct {
    articleRepo    ArticleRepository
    policyEval     PolicyEvaluator        // OPA client
    eventPublisher EventPublisher         // Hermes
}

// Execute implements the use case (orchestration + authorization)
func (uc *CreateArticleUseCase) Execute(ctx context.Context, 
    req CreateArticleRequest, principal string) (*entities.Article, error) {
    
    // Step 1: Authorization (OPA policy)
    allowed, err := uc.policyEval.Evaluate(ctx, PolicyInput{
        Action:    "create_article",
        Principal: principal,
    })
    if err != nil {
        return nil, ErrPolicyEvaluation
    }
    if !allowed {
        return nil, ErrUnauthorized
    }
    
    // Step 2: Create domain entity (business rules)
    article, err := entities.NewArticle(req.SKU, req.Name, req.Price)
    if err != nil {
        return nil, err
    }
    
    // Step 3: Persist (via abstraction)
    if err := uc.articleRepo.Create(ctx, article); err != nil {
        return nil, ErrPersistence
    }
    
    // Step 4: Publish event (eventual consistency)
    if err := uc.eventPublisher.Publish(ctx, events.ArticleCreated{
        ArticleID: article.ID,
        SKU:       article.SKU,
        CreatedAt: article.CreatedAt,
    }); err != nil {
        // Log but don't fail; event publishing is best-effort
        uc.logger.Warn("failed to publish ArticleCreated event", "article_id", article.ID)
    }
    
    return article, nil
}
```

**Interface adapters layer** (`adapters/http/handlers.go`):
```go
package http

type CreateArticleHandler struct {
    usecase *usecases.CreateArticleUseCase
}

func (h *CreateArticleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Step 1: Extract principal from context (auth middleware set this)
    principal := r.Context().Value("principal").(string)
    
    // Step 2: Deserialize HTTP request
    var req usecases.CreateArticleRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Step 3: Call use case (business logic)
    article, err := h.usecase.Execute(r.Context(), req, principal)
    if err != nil {
        handleError(w, err) // Maps domain errors to HTTP status codes
        return
    }
    
    // Step 4: Serialize response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(article)
}
```

### Data Layer (Repository Pattern)

Each use case depends on abstract repository interfaces, not concrete implementations:

```go
// usecases/interfaces.go
type ArticleRepository interface {
    Create(ctx context.Context, article *entities.Article) error
    GetByID(ctx context.Context, id string) (*entities.Article, error)
    Update(ctx context.Context, article *entities.Article) error
    Delete(ctx context.Context, id string) error
}
```

Concrete implementation lives in adapters:

```go
// adapters/postgres/article_repo.go
type PostgresArticleRepository struct {
    db *sql.DB
}

func (r *PostgresArticleRepository) Create(ctx context.Context, article *entities.Article) error {
    _, err := r.db.ExecContext(ctx, `
        INSERT INTO articles (id, sku, name, price, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `, article.ID, article.SKU, article.Name, article.Price, article.CreatedAt)
    return err
}
```

### Testing Strategy

**Unit tests (no database):**
```go
// usecases/create_article_test.go
func TestCreateArticle_UnauthorizedPrincipal(t *testing.T) {
    mockRepo := &MockArticleRepository{}
    mockPolicy := &MockPolicyEvaluator{
        ShouldAllow: false, // Simulate denial
    }
    mockEvents := &MockEventPublisher{}
    
    uc := &CreateArticleUseCase{
        articleRepo: mockRepo,
        policyEval: mockPolicy,
        eventPublisher: mockEvents,
    }
    
    _, err := uc.Execute(context.Background(), CreateArticleRequest{...}, "user123")
    
    require.Equal(t, ErrUnauthorized, err)
    assert.Equal(t, 0, mockRepo.CreateCallCount) // Verify no persistence
    assert.Equal(t, 0, mockEvents.PublishCallCount) // Verify no event
}
```

**Integration tests (with database):**
```go
// adapters/postgres/article_repo_test.go
func TestCreateArticle_Integration(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    repo := &PostgresArticleRepository{db: db}
    article, _ := entities.NewArticle("SKU123", "Test Article", decimal.NewFromFloat(99.99))
    
    err := repo.Create(context.Background(), article)
    
    require.NoError(t, err)
    
    // Verify persistence
    retrieved, err := repo.GetByID(context.Background(), article.ID)
    require.NoError(t, err)
    assert.Equal(t, article.SKU, retrieved.SKU)
}
```

**End-to-end tests (full HTTP):**
```go
// tests/e2e/create_article_test.go
func TestCreateArticleE2E(t *testing.T) {
    server := startTestServer(t)
    defer server.Close()
    
    body := `{"sku": "SKU123", "name": "Test", "price": 99.99}`
    resp, _ := http.Post(server.URL+"/articles", "application/json", 
        strings.NewReader(body))
    
    require.Equal(t, http.StatusCreated, resp.StatusCode)
    
    var article entities.Article
    json.NewDecoder(resp.Body).Decode(&article)
    assert.Equal(t, "SKU123", article.SKU)
}
```

---

## Consequences

### Positive Outcomes

✅ **Clear boundaries:** New engineers can navigate the codebase confidently. "Where is authorization checked?" Answer: use cases layer.

✅ **Testability:** Unit tests run in milliseconds without database. Integration tests verify data persistence. E2E tests validate HTTP contracts. Each test type has a clear purpose.

✅ **Parallel development:** Teams can work on different use cases without waiting for infrastructure (database schema, HTTP routing). Mock repositories enable frontend and backend to develop in parallel.

✅ **Easy refactoring:** Moving business logic from one use case to a shared domain rule is low-risk because layers are isolated.

✅ **Framework independence:** If we migrate from PostgreSQL to MongoDB, only the adapters layer changes. Use cases and entities remain untouched.

✅ **Knowledge reuse:** Architecture patterns are explicit and teachable. Similar patterns apply to Orders and Invoicing services.

### Tradeoffs & Challenges

⚠️ **More boilerplate:** Clean architecture requires interfaces, adapters, and explicit dependency management. A simple CRUD operation spans 4 layers instead of 1. Junior engineers may find this heavy-handed initially.

⚠️ **Requires discipline:** The architecture is only as strong as team commitment. If one engineer bypasses layers (e.g., handler calls repository directly), the pattern breaks. Code reviews must enforce boundaries.

⚠️ **Testing overhead:** Comprehensive testing (unit + integration + e2e) requires more test code than the implementation. This is intentional but feels slower initially.

⚠️ **Debugging complexity:** Tracing a feature through 4 layers requires understanding each layer's contracts. Distributed tracing tooling becomes essential.

⚠️ **Performance considerations:** Clean architecture abstractions (interfaces, layers) add indirection. In latency-critical paths, this must be profiled. (Typically negligible in Go, but not zero.)

---

## Learning Goals

Participants will understand:

1. **Why layered architecture matters:** Decoupling business logic from infrastructure enables testing, refactoring, and team scalability
2. **How to structure a Go service:** Directory layout, layer responsibilities, dependency direction
3. **Repository pattern and abstraction:** Writing interfaces that enable testing without databases
4. **Testing strategy:** Unit (fast, isolated), integration (with DB), e2e (full HTTP) and when to use each
5. **Dependency inversion:** How inner layers remain independent of outer layers (framework decisions don't leak into business logic)

By completing checkpoints 1-3, engineers will:
- Write a domain entity with business rules
- Create use cases that orchestrate entities and enforce authorization
- Implement HTTP handlers and repositories
- Test each layer independently
- Recognize architectural violations and suggest improvements

This foundation ensures that as the service grows (more use cases, more adapters), it remains understandable and maintainable.

---

## References

- **Robert C. Martin, "Clean Architecture" (2017)** - Definitive guide to layered architecture
- **"The Clean Code Blog"** - Martin Fowler on Clean Architecture (https://blog.cleancoder.com)
- **"Hexagonal Architecture"** (Alistair Cockburn) - Similar pattern, emphasizes ports and adapters
- **"Go Package Naming"** (standard Go practices) - Naming conventions for organizing code

---

## Implementation Reference

This ADR is implemented throughout the exercise phases (Phase 03-10) using the repository's current flat phase structure. The phase directories intentionally avoid an `internal/` tree so participants can navigate the layers directly.

### Phase 03: Skeleton (Checkpoint 3)
- **Files:** `phase-03-skeleton/entities/`, `phase-03-skeleton/events/`, `phase-03-skeleton/interfaces/repository.go`, `phase-03-skeleton/main.go`
- **Focus:** Establish the first Clean Architecture boundaries: domain entities, domain events, repository ports, and application bootstrap
- **Key artifacts:** `Article`, `InventoryLevel`, Value Objects, Domain Events, and the `ArticleRepository` port

### Phase 04-06: Core Layers (Checkpoints 4-6)
- **Entities layer:** `phase-04-db/entities/{article.go,inventory.go,money.go,sku.go}`
- **Repository adapters:** `phase-04-db/repositories/`
- **Use cases:** `phase-05-usecases/usecases/{create_article.go,get_article.go,adjust_inventory.go}`
- **Event dispatcher boundary:** `phase-05-usecases/dispatcher/`
- **HTTP handlers:** `phase-06-http-api/handlers/{article_handler.go,inventory_handler.go,router.go}`

### Phase 07-09: Integration (Checkpoints 7-9)
- **Auth middleware:** `phase-07-auth/middleware/auth.go` - local adapter for TSID-like user and M2M JWT validation
- **Policy enforcer:** `phase-08-policies/policies/enforcer.go`, `phase-08-policies/policies/warehouse.rego` - adapter for OPA/Rego policy decisions
- **Event publisher:** `phase-09-events/events/hermes_publisher.go`, `phase-09-events/schemas/` - local Hermes/Data Product adapter

### Phase 10: Complete Implementation (Checkpoint 10)
- **Full service:** all Clean Architecture layers plus cross-cutting adapters working together in production-grade code
- **Test structure:** package-level tests (`*_test.go`) plus E2E coverage in `phase-10-integration/tests/`
- **Files:** `phase-10-integration/entities/`, `phase-10-integration/interfaces/`, `phase-10-integration/repositories/`, `phase-10-integration/usecases/`, `phase-10-integration/handlers/`, `phase-10-integration/middleware/`, `phase-10-integration/policies/`, `phase-10-integration/events/`, `phase-10-integration/observability/`, `phase-10-integration/tests/`

**Concrete file references by layer:**
- **Entities:** `phase-04-db/entities/article.go` - Aggregate root with business rules
- **Repository port:** `phase-03-skeleton/interfaces/repository.go` - dependency-inversion boundary
- **Repositories:** `phase-04-db/repositories/article_repository.go` - MySQL implementation of the port
- **Use Cases:** `phase-05-usecases/usecases/create_article.go` - business orchestration
- **Handlers:** `phase-06-http-api/handlers/article_handler.go` - HTTP adapter
- **Auth Middleware:** `phase-07-auth/middleware/auth.go` - authentication adapter
- **Policies:** `phase-08-policies/policies/enforcer.go` - authorization adapter
- **Events:** `phase-09-events/events/hermes_publisher.go` - event publication adapter
- **Server:** `phase-03-skeleton/main.go` and `phase-10-integration/main.go` - application bootstrap
