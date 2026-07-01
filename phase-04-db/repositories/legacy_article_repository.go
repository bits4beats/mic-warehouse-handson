package repositories

import (
	"context"
	"database/sql"
	"errors"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// ErrLegacyRepositoryTODO is returned by the starter implementation until the
// participant completes the legacy ACL adapter in Phase 04 Task 1.
var ErrLegacyRepositoryTODO = errors.New("legacy mysql repository TODO: complete Phase 04 Task 1")

// LegacyMySQLArticleRepository implements interfaces.ArticleRepository against
// the simplified legacy MySQL schema (legacy-init.sql).
//
// This file is intentionally a starter. In Task 1 you will complete it with AI
// support so the dual-write decorator can write to:
//
//   - legacy_db.articles       (price DECIMAL(10,2), no currency column)
//   - warehouse_db.articles    (price_cents BIGINT + currency CHAR(3))
//
// The important design point: this adapter is an Anti-Corruption Layer. It
// translates the legacy schema into the Warehouse domain model without making
// DualWriteArticleRepository or entities.Article know legacy details.
type LegacyMySQLArticleRepository struct {
	db *sql.DB
}

func NewLegacyMySQLArticleRepository(db *sql.DB) *LegacyMySQLArticleRepository {
	return &LegacyMySQLArticleRepository{db: db}
}

// compile-time interface check
var _ interfaces.ArticleRepository = (*LegacyMySQLArticleRepository)(nil)

func (r *LegacyMySQLArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	// TODO Task 1:
	// - upsert into legacy_db.articles
	// - convert Money.AmountCents into DECIMAL string, e.g. 2999 -> "29.99"
	// - drop Currency because legacy_db has no currency column
	// - leave Article.Inventories out of scope for this phase
	return ErrLegacyRepositoryTODO
}

func (r *LegacyMySQLArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	// TODO Task 1:
	// - SELECT id, sku, name, description, price, created_at, updated_at
	// - convert DECIMAL price back to cents
	// - rehydrate SKU and Money through their factories
	// - default Currency to "EUR"
	return nil, ErrLegacyRepositoryTODO
}

func (r *LegacyMySQLArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	// TODO Task 1: same translation as FindByID, filtered by sku.
	return nil, ErrLegacyRepositoryTODO
}

func (r *LegacyMySQLArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	// TODO Task 1: list legacy rows and rehydrate each as entities.Article.
	return nil, ErrLegacyRepositoryTODO
}

func (r *LegacyMySQLArticleRepository) Delete(ctx context.Context, id string) error {
	// TODO Task 1: DELETE FROM articles WHERE id = ? and return ErrArticleNotFound
	// when no row was deleted.
	return ErrLegacyRepositoryTODO
}

// centsToDecimal converts integer cents into the legacy DECIMAL string the
// legacy_db.articles.price column expects, e.g. 2999 -> "29.99".
//
// TODO Task 1: implement WITHOUT float64 (integer/string math only). The
// conversion tests in legacy_conversions_test.go pin the expected behaviour.
func centsToDecimal(cents int64) string {
	return "" // TODO Task 1
}

// decimalToCents parses a legacy DECIMAL string (e.g. "29.99") back into integer
// cents. "1" -> 100, "1.5" -> 150, "29.99" -> 2999.
//
// TODO Task 1: implement WITHOUT float64. The conversion tests pin the behaviour.
func decimalToCents(s string) (int64, error) {
	return 0, ErrLegacyRepositoryTODO
}
