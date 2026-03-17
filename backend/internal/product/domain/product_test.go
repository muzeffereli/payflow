package domain

import (
	"errors"
	"testing"
)

func TestNewProduct_Valid(t *testing.T) {
	p, err := NewProduct("Widget", "A fine widget", "WGT-001", 2500, "USD", "gadgets", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
	if p.Status != StatusActive {
		t.Errorf("expected active, got %s", p.Status)
	}
	if p.Price != 2500 {
		t.Errorf("expected 2500, got %d", p.Price)
	}
}

func TestNewProduct_ZeroStock_BecomesOutOfStock(t *testing.T) {
	p, err := NewProduct("Widget", "", "W-002", 1000, "USD", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != StatusOutOfStock {
		t.Errorf("expected out_of_stock, got %s", p.Status)
	}
}

func TestNewProduct_DefaultCurrency(t *testing.T) {
	p, err := NewProduct("Widget", "", "W-003", 1000, "", "", "", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Currency != "USD" {
		t.Errorf("expected default USD, got %s", p.Currency)
	}
}

func TestNewProduct_Validation(t *testing.T) {
	cases := []struct {
		name        string
		productName string
		sku         string
		price       int64
		stock       int
	}{
		{"empty name", "", "S1", 100, 1},
		{"empty SKU", "Name", "", 100, 1},
		{"zero price", "Name", "S2", 0, 1},
		{"negative price", "Name", "S3", -1, 1},
		{"negative stock", "Name", "S4", 100, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewProduct(tc.productName, "", tc.sku, tc.price, "USD", "", "", tc.stock)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestProduct_ReserveStock(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 10)

	if err := p.ReserveStock(3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Stock != 7 {
		t.Errorf("expected 7, got %d", p.Stock)
	}
	if p.Status != StatusActive {
		t.Errorf("expected active, got %s", p.Status)
	}
}

func TestProduct_ReserveStock_ToZero_BecomesOutOfStock(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 5)

	if err := p.ReserveStock(5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Stock != 0 {
		t.Errorf("expected 0, got %d", p.Stock)
	}
	if p.Status != StatusOutOfStock {
		t.Errorf("expected out_of_stock, got %s", p.Status)
	}
}

func TestProduct_ReserveStock_Insufficient(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 3)

	err := p.ReserveStock(5)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
	if p.Stock != 3 {
		t.Errorf("stock should be unchanged at 3, got %d", p.Stock)
	}
}

func TestProduct_ReserveStock_Inactive(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 10)
	p.Deactivate()

	err := p.ReserveStock(1)
	if !errors.Is(err, ErrProductInactive) {
		t.Errorf("expected ErrProductInactive, got %v", err)
	}
}

func TestProduct_ReserveStock_ZeroQty(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 10)
	err := p.ReserveStock(0)
	if err == nil {
		t.Error("expected error for zero quantity")
	}
}

func TestProduct_ReleaseStock(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 5)
	_ = p.ReserveStock(5) // stock=0, status=out_of_stock

	p.ReleaseStock(3)
	if p.Stock != 3 {
		t.Errorf("expected 3, got %d", p.Stock)
	}
	if p.Status != StatusActive {
		t.Errorf("expected active after release, got %s", p.Status)
	}
}

func TestProduct_ReleaseStock_DoesNotChangeInactive(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 5)
	p.Deactivate()
	p.Stock = 0
	p.ReleaseStock(5)
	if p.Status != StatusInactive {
		t.Errorf("expected inactive to remain inactive, got %s", p.Status)
	}
}

func TestProduct_UpdatePrice(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 5)

	if err := p.UpdatePrice(9900); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Price != 9900 {
		t.Errorf("expected 9900, got %d", p.Price)
	}
}

func TestProduct_UpdatePrice_ZeroOrNegative(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 5)

	if err := p.UpdatePrice(0); err == nil {
		t.Error("expected error for zero price")
	}
	if err := p.UpdatePrice(-1); err == nil {
		t.Error("expected error for negative price")
	}
	if p.Price != 100 {
		t.Errorf("price should be unchanged at 100, got %d", p.Price)
	}
}

func TestProduct_Deactivate(t *testing.T) {
	p, _ := NewProduct("W", "", "S", 100, "USD", "", "", 5)
	p.Deactivate()
	if p.Status != StatusInactive {
		t.Errorf("expected inactive, got %s", p.Status)
	}
}
