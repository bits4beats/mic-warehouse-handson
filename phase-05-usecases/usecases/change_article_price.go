package usecases

import (
	"context"
	"errors"

	"warehouse.local/core/dispatcher"
	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ErrChangeArticlePriceTODO is returned by the starter implementation until
// the participant completes the Phase 05 task.
var ErrChangeArticlePriceTODO = errors.New("ChangeArticlePriceUseCase TODO: complete the Phase 05 task")

// ChangeArticlePriceInput carries the primitive request shape for a price
// change application workflow.
type ChangeArticlePriceInput struct {
	ArticleID     string
	NewPriceCents int64
	Currency      string
}

// ChangeArticlePriceOutput exposes the mutated aggregate to the caller.
type ChangeArticlePriceOutput struct {
	Article *entities.Article
}

// ChangeArticlePriceUseCase loads an Article, asks the aggregate to change its
// price, persists the result, and dispatches the canonical price-changed event.
//
// This file is intentionally a starter. Complete Execute using the same
// workflow pattern already used by CreateArticleUseCase (the given example).
type ChangeArticlePriceUseCase struct {
	repo       interfaces.ArticleRepository
	dispatcher dispatcher.EventDispatcher
}

func NewChangeArticlePriceUseCase(repo interfaces.ArticleRepository, d dispatcher.EventDispatcher) *ChangeArticlePriceUseCase {
	return &ChangeArticlePriceUseCase{repo: repo, dispatcher: d}
}

func (uc *ChangeArticlePriceUseCase) Execute(ctx context.Context, in ChangeArticlePriceInput) (*ChangeArticlePriceOutput, error) {
	return nil, ErrChangeArticlePriceTODO
}
