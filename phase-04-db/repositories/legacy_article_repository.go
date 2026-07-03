package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"warehouse.local/core/entities"
	"warehouse.local/core/interfaces"
)

// legacyDefaultCurrency is the currency the ACL invents on read: legacy_db has
// no currency column, so every legacy row is assumed to be in EUR.
const legacyDefaultCurrency = "EUR"

// LegacyMySQLArticleRepository implements interfaces.ArticleRepository against
// the simplified legacy MySQL schema (legacy-init.sql).
//
// It is an Anti-Corruption Layer: it converts between the legacy
// price DECIMAL(10,2) (no currency) and the domain's Money (integer cents +
// currency), so neither DualWriteArticleRepository nor entities.Article learns a
// legacy detail. Article.Inventories is out of scope for this phase.
type LegacyMySQLArticleRepository struct {
	db *sql.DB
}

func NewLegacyMySQLArticleRepository(db *sql.DB) *LegacyMySQLArticleRepository {
	return &LegacyMySQLArticleRepository{db: db}
}

// compile-time interface check
var _ interfaces.ArticleRepository = (*LegacyMySQLArticleRepository)(nil)

func (r *LegacyMySQLArticleRepository) Save(ctx context.Context, a *entities.Article) error {
	now := time.Now().UTC()
	a.UpdatedAt = now
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}

	// Upsert into the legacy schema: price is a DECIMAL string, currency is dropped.
	const upsert = `
		INSERT INTO articles (id, sku, name, description, price, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		  sku = VALUES(sku),
		  name = VALUES(name),
		  description = VALUES(description),
		  price = VALUES(price),
		  updated_at = VALUES(updated_at)
	`
	_, err := r.db.ExecContext(ctx, upsert,
		a.ID, a.SKU.Code, a.Name, a.Description,
		centsToDecimal(a.Price.AmountCents),
		a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (r *LegacyMySQLArticleRepository) FindByID(ctx context.Context, id string) (*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price, created_at, updated_at FROM articles WHERE id = ?`
	return scanLegacyArticle(r.db.QueryRowContext(ctx, q, id))
}

func (r *LegacyMySQLArticleRepository) FindBySKU(ctx context.Context, skuCode string) (*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price, created_at, updated_at FROM articles WHERE sku = ?`
	return scanLegacyArticle(r.db.QueryRowContext(ctx, q, skuCode))
}

func (r *LegacyMySQLArticleRepository) List(ctx context.Context) ([]*entities.Article, error) {
	const q = `SELECT id, sku, name, description, price, created_at, updated_at FROM articles ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*entities.Article
	for rows.Next() {
		a, err := scanLegacyArticleFromRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *LegacyMySQLArticleRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrArticleNotFound
	}
	return nil
}

// scanner abstracts *sql.Row and *sql.Rows so the two rehydration paths share one
// translation body.
type scanner interface {
	Scan(dest ...any) error
}

// scanLegacyArticle rehydrates a single row fetched via QueryRowContext, mapping
// sql.ErrNoRows to ErrArticleNotFound.
func scanLegacyArticle(row *sql.Row) (*entities.Article, error) {
	a, err := rehydrateLegacyArticle(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrArticleNotFound
	}
	return a, err
}

func scanLegacyArticleFromRows(rows *sql.Rows) (*entities.Article, error) {
	return rehydrateLegacyArticle(rows)
}

// rehydrateLegacyArticle translates one legacy row into the domain aggregate:
// DECIMAL price -> integer cents, invented EUR currency, values rebuilt through
// their domain factories.
func rehydrateLegacyArticle(s scanner) (*entities.Article, error) {
	var (
		id, sku, name, desc  string
		priceDecimal         string
		createdAt, updatedAt time.Time
	)
	if err := s.Scan(&id, &sku, &name, &desc, &priceDecimal, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	cents, err := decimalToCents(priceDecimal)
	if err != nil {
		return nil, fmt.Errorf("convert legacy price %q: %w", priceDecimal, err)
	}
	skuVO, err := entities.NewSKU(sku)
	if err != nil {
		return nil, fmt.Errorf("rehydrate SKU: %w", err)
	}
	priceVO, err := entities.NewMoney(cents, legacyDefaultCurrency)
	if err != nil {
		return nil, fmt.Errorf("rehydrate Money: %w", err)
	}
	return &entities.Article{
		ID:          id,
		SKU:         *skuVO,
		Name:        name,
		Description: desc,
		Price:       *priceVO,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

// centsToDecimal converts integer cents into the legacy DECIMAL string the
// legacy_db.articles.price column expects, e.g. 2999 -> "29.99". Integer/string
// math only (no float64).
func centsToDecimal(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// decimalToCents parses a legacy DECIMAL string (e.g. "29.99") back into integer
// cents. "1" -> 100, "1.5" -> 150, "29.99" -> 2999. No float64.
func decimalToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("decimalToCents: empty string")
	}
	intPart, fracPart, _ := strings.Cut(s, ".")

	whole, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimalToCents: invalid integer part %q: %w", intPart, err)
	}

	// Normalise the fractional part to exactly two digits: pad short parts on the
	// right, truncate longer ones.
	switch {
	case len(fracPart) < 2:
		fracPart = fracPart + strings.Repeat("0", 2-len(fracPart))
	case len(fracPart) > 2:
		fracPart = fracPart[:2]
	}

	frac, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decimalToCents: invalid fractional part %q: %w", fracPart, err)
	}

	return whole*100 + frac, nil
}
