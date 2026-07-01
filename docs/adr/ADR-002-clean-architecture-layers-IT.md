# ADR-002: Layer di Clean Architecture per il servizio Go

**Status:** Accepted

**Date:** 2026-05-08

**Authors:** Architecture Team

---

## Context

Il nuovo servizio Go Warehouse sarà letto e modificato da circa 200 ingegneri distribuiti su più team (Warehouse, Orders, Invoicing, Frontend). Questi ingegneri hanno livelli di esperienza diversi:

- architetti senior familiari con sistemi distribuiti;
- engineer mid-level provenienti da un background monolitico (PHP procedurale);
- engineer junior e stagisti che imparano Go per la prima volta.

Il monolite soffre di un problema comune: **la logica di business è dispersa tra i layer**. La logica di autorizzazione vive in controller e model. Le regole di dominio sono enforce nei trigger del database e in metodi di servizio ad hoc. Le mutazioni di stato avvengono nei repository senza confini transazionali chiari. Questo rende il monolite:

- **Difficile da testare:** cambiare una regola di business richiede di mockare 5+ dipendenze.
- **Difficile da capire:** non ci sono punti di ingresso chiari per rispondere a "come prende una decisione di autorizzazione il sistema?".
- **Difficile da cambiare in sicurezza:** refactorare una feature spesso rompe un'altra feature a causa di dipendenze nascoste.

Il team ha bisogno di pattern architetturali che offrano:

1. **Confini chiari:** ogni engineer può rispondere a "dove vive questa logica?" entro 30 secondi.
2. **Testabilità:** gli unit test girano senza database; gli integration test verificano la persistenza dati; gli e2e test validano i contratti HTTP.
3. **Scalabilità dei team:** engineer junior possono aggiungere feature senza rompere quelle esistenti; le decisioni architetturali enforce la safety.
4. **Riuso della conoscenza:** i pattern imparati in Warehouse si applicano anche alle estrazioni di Orders e Invoicing.

La sfida: la semplicità di Go (niente framework, niente magia) significa che il team deve essere **intenzionale** sulla stratificazione. Senza un'architettura condivisa, gli engineer inventeranno convenzioni locali che divergeranno nel codebase.

---

## Decision

Implementeremo la **Clean Architecture** (come definita da Uncle Bob, Robert C. Martin) con quattro layer espliciti:

1. **Entities:** modelli di dominio core e regole di business.
2. **Use Cases:** orchestrazione applicativa e autorizzazione.
3. **Interface Adapters:** adapter HTTP, database ed event bus.
4. **Frameworks & Drivers:** standard library Go e package esterni.

Ogni layer ha responsabilità chiare, le dipendenze puntano verso l'interno e la logica di business resta indipendente dall'infrastruttura.

### Responsabilità dei layer

| Layer | Responsabilità | Esempi | Dipendenze |
|-------|----------------|--------|------------|
| **Entities** | Modelli di dominio core; regole di business immutabili | `Article`, `Inventory`, `StockReservation` | Nessuna (Go puro) |
| **Use Cases** | Orchestrano entity; enforce autorizzazione; pubblicano eventi | `CreateArticle`, `ReserveStock`, `UpdateInventory` | Entities, policy esterne |
| **Interface Adapters** | Convertono tra formati HTTP/database e modelli di dominio | HTTP handler, implementazioni repository, event publisher | Use Cases, Entities |
| **Frameworks & Drivers** | Setup framework, dependency injection, inizializzazione server | Go `net/http`, driver postgres, client OPA | Tutti i layer |

### Principi architetturali

**Dependency Inversion:** i layer interni (logica di business) non dipendono mai dai layer esterni (infrastruttura). Un use case non conosce HTTP o database; sono gli adapter a chiamare gli use case.

**Single Responsibility:** ogni file, funzione e layer ha una sola ragione per cambiare. Cambiare regole di autorizzazione non deve richiedere cambi al routing HTTP o agli schemi database.

**Astrazione:** ogni layer comunica tramite interfacce. Gli use case dipendono da un `ArticleRepository` astratto, non da una specifica implementazione PostgreSQL. Questo abilita test con repository in-memory.

**Chiarezza più che cleverness:** esplicito è meglio di implicito. Una funzione lineare da 20 righe batte una funzione da 5 righe che richiede 10 minuti di documentazione.

---

## Implementation

### Struttura delle directory

```
warehouse/
├── entities/                    # Layer 1: dominio core
│   ├── article.go             # Modello di dominio: Article, SKU, pricing
│   ├── inventory.go           # Modello di dominio: livelli stock, prenotazioni
│   ├── business_rules.go      # Funzioni pure: validazione, calcoli
│   └── errors.go              # Errori specifici del dominio
├── usecases/                   # Layer 2: logica applicativa
│   ├── create_article.go      # Orchestra la creazione di entity
│   ├── reserve_stock.go       # Orchestra la prenotazione (con autorizzazione)
│   ├── update_inventory.go    # Orchestra la mutazione inventory
│   ├── interfaces.go          # Contratti per repository, policy, eventi
│   └── errors.go              # Errori use case (autorizzazione, validazione)
├── adapters/                   # Layer 3: conversione infrastrutturale
│   ├── http/
│   │   ├── handlers.go        # HTTP handler (chiamano gli use case)
│   │   ├── serializers.go     # Converti modelli dominio ↔ JSON
│   │   └── middleware.go      # Logging, gestione errori
│   ├── postgres/
│   │   ├── article_repo.go    # Implementa l'interfaccia ArticleRepository
│   │   ├── inventory_repo.go  # Implementa l'interfaccia InventoryRepository
│   │   ├── migrations/        # Migrazioni SQL
│   │   └── query_helpers.go   # Logica database condivisa
│   └── events/
│       └── hermes_publisher.go # Implementa l'interfaccia EventPublisher
├── config/                     # Layer 4: setup
│   ├── database.go            # Connessione DB, configurazione pool
│   ├── logger.go              # Inizializzazione logger
│   ├── opa.go                 # Setup client OPA
│   └── server.go              # Inizializzazione server HTTP
├── main.go                     # Entry point
└── tests/                      # Utility di test
    ├── fixtures/              # Dati di test riusabili
    └── mocks/                 # Implementazioni mock delle interfacce
```

### Esempio concreto: use case Create Article

**Layer Entities** (`entities/article.go`):

```go
package entities

// Article rappresenta un articolo di magazzino (modello di dominio)
type Article struct {
    ID          string
    SKU         string
    Name        string
    Price       decimal.Decimal
    CreatedAt   time.Time
}

// NewArticle crea un articolo con validazione (regola di business)
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

**Layer Use cases** (`usecases/create_article.go`):

```go
package usecases

type CreateArticleRequest struct {
    SKU   string
    Name  string
    Price decimal.Decimal
}

type CreateArticleUseCase struct {
    articleRepo    ArticleRepository
    policyEval     PolicyEvaluator        // client OPA
    eventPublisher EventPublisher         // Hermes
}

// Execute implementa lo use case (orchestrazione + autorizzazione)
func (uc *CreateArticleUseCase) Execute(ctx context.Context, 
    req CreateArticleRequest, principal string) (*entities.Article, error) {
    
    // Step 1: autorizzazione (policy OPA)
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
    
    // Step 2: crea l'entity di dominio (regole di business)
    article, err := entities.NewArticle(req.SKU, req.Name, req.Price)
    if err != nil {
        return nil, err
    }
    
    // Step 3: persiste (tramite astrazione)
    if err := uc.articleRepo.Create(ctx, article); err != nil {
        return nil, ErrPersistence
    }
    
    // Step 4: pubblica evento (consistenza eventuale)
    if err := uc.eventPublisher.Publish(ctx, events.ArticleCreated{
        ArticleID: article.ID,
        SKU:       article.SKU,
        CreatedAt: article.CreatedAt,
    }); err != nil {
        // Logga ma non fallisce; la pubblicazione eventi è best-effort
        uc.logger.Warn("failed to publish ArticleCreated event", "article_id", article.ID)
    }
    
    return article, nil
}
```

**Layer Interface adapters** (`adapters/http/handlers.go`):

```go
package http

type CreateArticleHandler struct {
    usecase *usecases.CreateArticleUseCase
}

func (h *CreateArticleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Step 1: estrae il principal dal context (impostato dal middleware auth)
    principal := r.Context().Value("principal").(string)
    
    // Step 2: deserializza la richiesta HTTP
    var req usecases.CreateArticleRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }
    
    // Step 3: chiama lo use case (logica di business)
    article, err := h.usecase.Execute(r.Context(), req, principal)
    if err != nil {
        handleError(w, err) // Mappa errori di dominio su status code HTTP
        return
    }
    
    // Step 4: serializza la risposta
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(article)
}
```

### Data Layer (Repository Pattern)

Ogni use case dipende da interfacce repository astratte, non da implementazioni concrete:

```go
// usecases/interfaces.go
type ArticleRepository interface {
    Create(ctx context.Context, article *entities.Article) error
    GetByID(ctx context.Context, id string) (*entities.Article, error)
    Update(ctx context.Context, article *entities.Article) error
    Delete(ctx context.Context, id string) error
}
```

L'implementazione concreta vive negli adapter:

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

### Strategia di test

**Unit test (senza database):**

```go
// usecases/create_article_test.go
func TestCreateArticle_UnauthorizedPrincipal(t *testing.T) {
    mockRepo := &MockArticleRepository{}
    mockPolicy := &MockPolicyEvaluator{
        ShouldAllow: false, // Simula un diniego
    }
    mockEvents := &MockEventPublisher{}
    
    uc := &CreateArticleUseCase{
        articleRepo: mockRepo,
        policyEval: mockPolicy,
        eventPublisher: mockEvents,
    }
    
    _, err := uc.Execute(context.Background(), CreateArticleRequest{...}, "user123")
    
    require.Equal(t, ErrUnauthorized, err)
    assert.Equal(t, 0, mockRepo.CreateCallCount) // Verifica nessuna persistenza
    assert.Equal(t, 0, mockEvents.PublishCallCount) // Verifica nessun evento
}
```

**Integration test (con database):**

```go
// adapters/postgres/article_repo_test.go
func TestCreateArticle_Integration(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    repo := &PostgresArticleRepository{db: db}
    article, _ := entities.NewArticle("SKU123", "Test Article", decimal.NewFromFloat(99.99))
    
    err := repo.Create(context.Background(), article)
    
    require.NoError(t, err)
    
    // Verifica persistenza
    retrieved, err := repo.GetByID(context.Background(), article.ID)
    require.NoError(t, err)
    assert.Equal(t, article.SKU, retrieved.SKU)
}
```

**End-to-end test (HTTP completo):**

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

### Esiti positivi

✅ **Confini chiari:** i nuovi engineer possono orientarsi nel codebase con sicurezza. "Dove viene controllata l'autorizzazione?" Risposta: nel layer use cases.

✅ **Testabilità:** gli unit test girano in millisecondi senza database. Gli integration test verificano la persistenza dati. Gli E2E test validano i contratti HTTP. Ogni tipo di test ha uno scopo chiaro.

✅ **Sviluppo parallelo:** i team possono lavorare su use case diversi senza aspettare l'infrastruttura (schema database, routing HTTP). I mock repository permettono a frontend e backend di sviluppare in parallelo.

✅ **Refactoring semplice:** spostare logica di business da uno use case a una regola di dominio condivisa è a basso rischio perché i layer sono isolati.

✅ **Indipendenza dal framework:** se migriamo da PostgreSQL a MongoDB, cambia solo il layer adapter. Use case ed entity restano intatti.

✅ **Riuso della conoscenza:** i pattern architetturali sono espliciti e insegnabili. Pattern simili si applicano ai servizi Orders e Invoicing.

### Tradeoff e sfide

⚠️ **Più boilerplate:** Clean Architecture richiede interfacce, adapter e gestione esplicita delle dipendenze. Una semplice operazione CRUD attraversa 4 layer invece di 1. Gli engineer junior potrebbero percepirla inizialmente come pesante.

⚠️ **Richiede disciplina:** l'architettura è forte solo quanto l'impegno del team. Se un engineer bypassa i layer (es. handler che chiama direttamente il repository), il pattern si rompe. Le code review devono enforce i confini.

⚠️ **Overhead di test:** test completi (unit + integration + e2e) richiedono più codice di test dell'implementazione. È intenzionale, ma inizialmente sembra più lento.

⚠️ **Complessità di debug:** tracciare una feature attraverso 4 layer richiede di capire i contratti di ciascun layer. Il distributed tracing diventa essenziale.

⚠️ **Considerazioni di performance:** le astrazioni della Clean Architecture (interfacce, layer) aggiungono indirezione. Nei path latency-critical va profilato. (In Go di solito è trascurabile, ma non è zero.)

---

## Learning Goals

I partecipanti capiranno:

1. **Perché l'architettura a layer conta:** disaccoppiare la logica di business dall'infrastruttura abilita test, refactoring e scalabilità dei team.
2. **Come strutturare un servizio Go:** layout directory, responsabilità dei layer, direzione delle dipendenze.
3. **Repository pattern e astrazione:** scrivere interfacce che abilitano test senza database.
4. **Strategia di test:** unit (veloci, isolati), integration (con DB), e2e (HTTP completo) e quando usare ciascuno.
5. **Dependency inversion:** come i layer interni restano indipendenti dai layer esterni (le decisioni di framework non entrano nella logica di business).

Completando i checkpoint 1-3, gli engineer sapranno:

- scrivere una domain entity con regole di business;
- creare use case che orchestrano entity ed enforce autorizzazione;
- implementare HTTP handler e repository;
- testare ogni layer in modo indipendente;
- riconoscere violazioni architetturali e proporre miglioramenti.

Questa base assicura che, mentre il servizio cresce (più use case, più adapter), resti comprensibile e manutenibile.

---

## References

- **Robert C. Martin, "Clean Architecture" (2017)** - guida definitiva all'architettura a layer.
- **"The Clean Code Blog"** - Martin Fowler su Clean Architecture (https://blog.cleancoder.com).
- **"Hexagonal Architecture"** (Alistair Cockburn) - pattern simile, enfatizza port e adapter.
- **"Go Package Naming"** (pratiche standard Go) - convenzioni di naming per organizzare il codice.

---

## Implementation Reference

Questa ADR è implementata lungo le phase dell'esercizio (Phase 03-10) usando l'attuale struttura piatta del repository. Le directory delle phase evitano intenzionalmente un albero `internal/`, così i partecipanti possono navigare direttamente i layer.

### Phase 03: Skeleton (Checkpoint 3)

- **File:** `phase-03-skeleton/entities/`, `phase-03-skeleton/events/`, `phase-03-skeleton/interfaces/repository.go`, `phase-03-skeleton/main.go`.
- **Focus:** stabilire i primi confini di Clean Architecture: entità di dominio, domain event, port repository e bootstrap applicativo.
- **Artefatti chiave:** `Article`, `InventoryLevel`, Value Object, Domain Event e port `ArticleRepository`.

### Phase 04-06: Core Layers (Checkpoint 4-6)

- **Layer Entities:** `phase-04-db/entities/{article.go,inventory.go,money.go,sku.go}`.
- **Repository adapter:** `phase-04-db/repositories/`.
- **Use case:** `phase-05-usecases/usecases/{create_article.go,get_article.go,adjust_inventory.go}`.
- **Confine event dispatcher:** `phase-05-usecases/dispatcher/`.
- **Handler HTTP:** `phase-06-http-api/handlers/{article_handler.go,inventory_handler.go,router.go}`.

### Phase 07-09: Integration (Checkpoint 7-9)

- **Middleware auth:** `phase-07-auth/middleware/auth.go` - adapter locale per validazione JWT user e M2M in stile TSID.
- **Policy enforcer:** `phase-08-policies/policies/enforcer.go`, `phase-08-policies/policies/warehouse.rego` - adapter per decisioni OPA/Rego.
- **Event publisher:** `phase-09-events/events/hermes_publisher.go`, `phase-09-events/schemas/` - adapter locale Hermes/Data Product.

### Phase 10: Implementazione completa (Checkpoint 10)

- **Servizio completo:** tutti i layer di Clean Architecture più gli adapter cross-cutting lavorano insieme in codice production-grade.
- **Struttura test:** test per package (`*_test.go`) più copertura E2E in `phase-10-integration/tests/`.
- **File:** `phase-10-integration/entities/`, `phase-10-integration/interfaces/`, `phase-10-integration/repositories/`, `phase-10-integration/usecases/`, `phase-10-integration/handlers/`, `phase-10-integration/middleware/`, `phase-10-integration/policies/`, `phase-10-integration/events/`, `phase-10-integration/observability/`, `phase-10-integration/tests/`.

**Riferimenti concreti ai file per layer:**

- **Entities:** `phase-04-db/entities/article.go` - Aggregate root con regole di business.
- **Repository port:** `phase-03-skeleton/interfaces/repository.go` - confine di dependency inversion.
- **Repositories:** `phase-04-db/repositories/article_repository.go` - implementazione MySQL del port.
- **Use Cases:** `phase-05-usecases/usecases/create_article.go` - orchestrazione business.
- **Handlers:** `phase-06-http-api/handlers/article_handler.go` - adapter HTTP.
- **Middleware auth:** `phase-07-auth/middleware/auth.go` - adapter di autenticazione.
- **Policies:** `phase-08-policies/policies/enforcer.go` - adapter di autorizzazione.
- **Events:** `phase-09-events/events/hermes_publisher.go` - adapter di pubblicazione eventi.
- **Server:** `phase-03-skeleton/main.go` e `phase-10-integration/main.go` - bootstrap applicativo.
