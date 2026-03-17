package domain

import (
	"errors"
	"testing"
)

func TestNewWallet(t *testing.T) {
	w := NewWallet("user-1", "USD")

	if w.ID == "" {
		t.Error("expected non-empty ID")
	}
	if w.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", w.UserID)
	}
	if w.Balance != 0 {
		t.Errorf("expected initial balance 0, got %d", w.Balance)
	}
	if w.Currency != "USD" {
		t.Errorf("expected USD, got %s", w.Currency)
	}
}

func TestWallet_Credit(t *testing.T) {
	w := NewWallet("u", "USD")

	if err := w.Credit(5000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Balance != 5000 {
		t.Errorf("expected 5000, got %d", w.Balance)
	}

	if err := w.Credit(3000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Balance != 8000 {
		t.Errorf("expected 8000, got %d", w.Balance)
	}
}

func TestWallet_Credit_ZeroAmount(t *testing.T) {
	w := NewWallet("u", "USD")
	err := w.Credit(0)
	if !errors.Is(err, ErrNegativeAmount) {
		t.Errorf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestWallet_Credit_NegativeAmount(t *testing.T) {
	w := NewWallet("u", "USD")
	err := w.Credit(-100)
	if !errors.Is(err, ErrNegativeAmount) {
		t.Errorf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestWallet_Debit(t *testing.T) {
	w := NewWallet("u", "USD")
	_ = w.Credit(10000)

	if err := w.Debit(4000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Balance != 6000 {
		t.Errorf("expected 6000, got %d", w.Balance)
	}
}

func TestWallet_Debit_ExactBalance(t *testing.T) {
	w := NewWallet("u", "USD")
	_ = w.Credit(5000)

	if err := w.Debit(5000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Balance != 0 {
		t.Errorf("expected 0, got %d", w.Balance)
	}
}

func TestWallet_Debit_InsufficientFunds(t *testing.T) {
	w := NewWallet("u", "USD")
	_ = w.Credit(1000)

	err := w.Debit(1001)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}
	if w.Balance != 1000 {
		t.Errorf("balance should be unchanged at 1000, got %d", w.Balance)
	}
}

func TestWallet_Debit_ZeroAmount(t *testing.T) {
	w := NewWallet("u", "USD")
	_ = w.Credit(1000)
	err := w.Debit(0)
	if !errors.Is(err, ErrNegativeAmount) {
		t.Errorf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestWallet_Debit_NegativeAmount(t *testing.T) {
	w := NewWallet("u", "USD")
	_ = w.Credit(1000)
	err := w.Debit(-50)
	if !errors.Is(err, ErrNegativeAmount) {
		t.Errorf("expected ErrNegativeAmount, got %v", err)
	}
}

func TestWallet_Debit_EmptyWallet(t *testing.T) {
	w := NewWallet("u", "USD")
	err := w.Debit(1)
	if !errors.Is(err, ErrInsufficientFunds) {
		t.Errorf("expected ErrInsufficientFunds on empty wallet, got %v", err)
	}
}
