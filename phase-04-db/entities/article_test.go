package entities

import "testing"

func TestNewArticle_validInputs(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(2999, "EUR")

	a, err := NewArticle("article-1", *sku, "Widget", "A useful widget", *price)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if a.ID != "article-1" {
		t.Errorf("expected ID article-1, got %q", a.ID)
	}
	if !a.SKU.Equals(*sku) {
		t.Errorf("expected SKU ABC-001, got %v", a.SKU)
	}
	if !a.Price.Equals(*price) {
		t.Errorf("expected Price 2999 EUR, got %v", a.Price)
	}
	if a.Name != "Widget" {
		t.Errorf("expected Name Widget, got %q", a.Name)
	}
}

func TestNewArticle_rejectsEmptyID(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(100, "EUR")
	if _, err := NewArticle("", *sku, "X", "", *price); err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestNewArticle_rejectsEmptyName(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	price, _ := NewMoney(100, "EUR")
	if _, err := NewArticle("id", *sku, "", "", *price); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := NewArticle("id", *sku, "   ", "", *price); err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestNewArticle_rejectsZeroPrice(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	zero, _ := NewMoney(0, "EUR")
	if _, err := NewArticle("id", *sku, "X", "", *zero); err == nil {
		t.Fatal("expected error for price=0 (invariant: price > 0)")
	}
}

func TestArticle_ChangePrice_recordsEvent(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	old, _ := NewMoney(100, "EUR")
	a, _ := NewArticle("id", *sku, "X", "", *old)

	newPrice, _ := NewMoney(200, "EUR")
	if err := a.ChangePrice(*newPrice); err != nil {
		t.Fatalf("expected ChangePrice to succeed, got %v", err)
	}
	if !a.Price.Equals(*newPrice) {
		t.Errorf("expected price=200 after ChangePrice, got %v", a.Price)
	}
	if len(a.PendingEvents()) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(a.PendingEvents()))
	}
}

func TestArticle_ChangePrice_rejectsZero(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	old, _ := NewMoney(100, "EUR")
	a, _ := NewArticle("id", *sku, "X", "", *old)

	zero, _ := NewMoney(0, "EUR")
	if err := a.ChangePrice(*zero); err == nil {
		t.Fatal("expected error when changing price to 0")
	}
}

func TestArticle_ChangePrice_rejectsCurrencyMismatch(t *testing.T) {
	sku, _ := NewSKU("ABC-001")
	old, _ := NewMoney(100, "EUR")
	a, _ := NewArticle("id", *sku, "X", "", *old)

	usd, _ := NewMoney(100, "USD")
	if err := a.ChangePrice(*usd); err == nil {
		t.Fatal("expected error when changing currency via ChangePrice")
	}
}
