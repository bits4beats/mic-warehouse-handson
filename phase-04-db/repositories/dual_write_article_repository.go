package repositories

import (
	"context"
	"sync"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ReadMode controls where DualWriteArticleRepository routes reads.
//
// Writes always go to BOTH stores (legacy first, BC second), regardless of mode.
// Reads go to exactly ONE store, chosen by the mode (single read).
type ReadMode int

const (
	// ReadFromLegacy routes all reads to the legacy store. Default during early migration.
	ReadFromLegacy ReadMode = iota
	// ReadFromBC routes all reads to the BC store. Use after cutover.
	ReadFromBC
)

// DualWriteArticleRepository wraps two interfaces.ArticleRepository implementations
// (legacy + BC). It must write to both on every mutation and route reads to a single
// store chosen by mode.
//
// This file is intentionally a starter. In Task 2 you implement the five methods so
// the dual-write decorator works against the two real databases (exercised by the
// seed CLI). Per ADR-013 the decorator is transitional: once cutover completes it is
// deleted.
type DualWriteArticleRepository struct {
	legacy interfaces.ArticleRepository
	bc     interfaces.ArticleRepository

	// mu guards mode so the read-source feature toggle can be flipped at runtime
	// (SetReadMode) while reads are in flight, without recreating the repository.
	mu   sync.RWMutex
	mode ReadMode
}

func NewDualWriteArticleRepository(
	legacy interfaces.ArticleRepository,
	bc interfaces.ArticleRepository,
	mode ReadMode,
) *DualWriteArticleRepository {
	return &DualWriteArticleRepository{legacy: legacy, bc: bc, mode: mode}
}

// compile-time interface check
var _ interfaces.ArticleRepository = (*DualWriteArticleRepository)(nil)

func (r *DualWriteArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	// Legacy first: if it fails, return and do NOT touch BC.
	if err := r.legacy.Save(ctx, a); err != nil {
		return err
	}
	// Then BC: if it fails, return the error (legacy keeps the article — there is
	// no rollback policy in this exercise).
	return r.bc.Save(ctx, a)
}

func (r *DualWriteArticleRepository) Delete(ctx context.Context, id string) error {
	// Same failure rule as Save: legacy first, then BC.
	if err := r.legacy.Delete(ctx, id); err != nil {
		return err
	}
	return r.bc.Delete(ctx, id)
}

func (r *DualWriteArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	return r.readStore().FindByID(ctx, id)
}

func (r *DualWriteArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	return r.readStore().FindBySKU(ctx, skuCode)
}

func (r *DualWriteArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	return r.readStore().List(ctx)
}

// SetReadMode flips the read-source feature toggle at runtime. Writes are
// unaffected (always both stores); only the store reads are routed to changes.
// Safe to call concurrently with reads.
func (r *DualWriteArticleRepository) SetReadMode(mode ReadMode) {
	r.mu.Lock()
	r.mode = mode
	r.mu.Unlock()
}

// ReadMode returns the store reads are currently routed to.
func (r *DualWriteArticleRepository) ReadMode() ReadMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mode
}

// readStore picks the single store reads are routed to, per the configured mode.
// Writes always hit both stores; reads hit exactly one.
func (r *DualWriteArticleRepository) readStore() interfaces.ArticleRepository {
	if r.ReadMode() == ReadFromBC {
		return r.bc
	}
	return r.legacy
}
