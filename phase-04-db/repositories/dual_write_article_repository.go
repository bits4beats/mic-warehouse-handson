package repositories

import (
	"context"
	"errors"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ErrDualWriteTODO is returned by the starter implementation until you complete
// the dual-write / single-read decorator in Phase 04 Task 2.
var ErrDualWriteTODO = errors.New("dual-write decorator TODO: complete Phase 04 Task 2")

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
	mode   ReadMode
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
	// TODO Task 2: write to legacy FIRST; if it fails, return the error and do
	// NOT touch BC. Then write to BC; if it fails, return the error (legacy keeps
	// the article — there is no rollback policy in this exercise).
	return ErrDualWriteTODO
}

func (r *DualWriteArticleRepository) Delete(ctx context.Context, id string) error {
	// TODO Task 2: delete from legacy first, then BC (same failure rule as Save).
	return ErrDualWriteTODO
}

func (r *DualWriteArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	// TODO Task 2: route the read to a single store based on r.mode
	// (ReadFromBC -> bc, otherwise legacy).
	return nil, ErrDualWriteTODO
}

func (r *DualWriteArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	// TODO Task 2: route the read to a single store based on r.mode.
	return nil, ErrDualWriteTODO
}

func (r *DualWriteArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	// TODO Task 2: route the read to a single store based on r.mode.
	return nil, ErrDualWriteTODO
}
