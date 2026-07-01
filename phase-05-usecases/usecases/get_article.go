package usecases

import (
	"context"
	"errors"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ErrGetArticleTODO is returned by the starter until you complete the GetArticle
// slice (Phase 05).
var ErrGetArticleTODO = errors.New("GetArticleUseCase TODO: complete the GetArticle slice")

// GetArticleInput identifies the aggregate to load.
type GetArticleInput struct {
	ID string
}

// GetArticleOutput returns the aggregate. Mapping to a transport DTO is the
// HTTP layer's responsibility.
type GetArticleOutput struct {
	Article *entities.Article
}

// GetArticleUseCase loads an Article by id.
//
// This file is intentionally a starter. Complete Execute following the shape of
// CreateArticleUseCase: validate input, call the repository port, wrap errors.
type GetArticleUseCase struct {
	repo interfaces.ArticleRepository
}

func NewGetArticleUseCase(repo interfaces.ArticleRepository) *GetArticleUseCase {
	return &GetArticleUseCase{repo: repo}
}

func (uc *GetArticleUseCase) Execute(ctx context.Context, in GetArticleInput) (*GetArticleOutput, error) {
	// TODO GetArticle slice:
	// - reject an empty ID;
	// - load with uc.repo.FindByID(ctx, in.ID);
	// - wrap the repository error with fmt.Errorf("GetArticle: %w", err) so the
	//   handler can detect ErrArticleNotFound via errors.Is.
	return nil, ErrGetArticleTODO
}
