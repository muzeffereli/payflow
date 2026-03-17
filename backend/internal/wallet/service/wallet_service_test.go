package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"payment-platform/internal/wallet/domain"
	"payment-platform/internal/wallet/port"
	"payment-platform/pkg/eventbus"
)

type fakeWalletRepo struct {
	wallets      map[string]*domain.Wallet
	transactions []*domain.Transaction
	createCalls  int
}

func newFakeWalletRepo() *fakeWalletRepo {
	return &fakeWalletRepo{
		wallets: make(map[string]*domain.Wallet),
	}
}

func (f *fakeWalletRepo) Create(_ context.Context, wallet *domain.Wallet) error {
	f.createCalls++
	if _, exists := f.wallets[wallet.UserID]; exists {
		return errors.New("duplicate wallet")
	}
	copy := *wallet
	f.wallets[wallet.UserID] = &copy
	return nil
}

func (f *fakeWalletRepo) GetByUserID(_ context.Context, userID string) (*domain.Wallet, error) {
	wallet, ok := f.wallets[userID]
	if !ok {
		return nil, ErrNotFound
	}
	copy := *wallet
	return &copy, nil
}

func (f *fakeWalletRepo) GetByUserIDForUpdate(_ context.Context, _ *sql.Tx, userID string) (*domain.Wallet, error) {
	return f.GetByUserID(context.Background(), userID)
}

func (f *fakeWalletRepo) UpdateBalance(_ context.Context, _ *sql.Tx, wallet *domain.Wallet) error {
	copy := *wallet
	f.wallets[wallet.UserID] = &copy
	return nil
}

func (f *fakeWalletRepo) SaveTransaction(_ context.Context, _ *sql.Tx, tx *domain.Transaction) error {
	copy := *tx
	f.transactions = append(f.transactions, &copy)
	return nil
}

func (f *fakeWalletRepo) BeginTx(_ context.Context) (*sql.Tx, error) {
	return nil, nil
}

func (f *fakeWalletRepo) ListTransactions(_ context.Context, _ string, _ int, _ int) ([]*domain.Transaction, int, error) {
	return nil, 0, nil
}

var _ port.WalletRepository = (*fakeWalletRepo)(nil)

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, string, eventbus.Event) error { return nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEnsureWalletExistsCreatesMissingWallet(t *testing.T) {
	repo := newFakeWalletRepo()
	svc := New(repo, noopPublisher{}, testLogger(), "")

	if err := svc.ensureWalletExists(context.Background(), "seller-1", "USD"); err != nil {
		t.Fatalf("ensureWalletExists returned error: %v", err)
	}

	wallet, err := repo.GetByUserID(context.Background(), "seller-1")
	if err != nil {
		t.Fatalf("expected wallet to exist, got error: %v", err)
	}
	if wallet.Balance != 0 {
		t.Fatalf("expected initial balance 0, got %d", wallet.Balance)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected exactly one wallet creation, got %d", repo.createCalls)
	}
}

func TestEnsureWalletExistsDoesNotRecreateExistingWallet(t *testing.T) {
	repo := newFakeWalletRepo()
	repo.wallets["seller-1"] = domain.NewWallet("seller-1", "USD")
	svc := New(repo, noopPublisher{}, testLogger(), "")

	if err := svc.ensureWalletExists(context.Background(), "seller-1", "USD"); err != nil {
		t.Fatalf("ensureWalletExists returned error: %v", err)
	}

	if repo.createCalls != 0 {
		t.Fatalf("expected no wallet creation, got %d", repo.createCalls)
	}
}

func TestHandlePaymentSucceededCreditsSellerAndPlatformWallet(t *testing.T) {
	repo := newFakeWalletRepo()
	svc := New(repo, noopPublisher{}, testLogger(), "admin-1")
	storeID := "store-1"

	err := svc.HandlePaymentSucceeded(context.Background(), eventbus.PaymentSucceededData{
		PaymentID:    "payment-1",
		OrderID:      "order-1",
		UserID:       "buyer-1",
		Amount:       1000,
		Currency:     "USD",
		Method:       "card",
		StoreID:      &storeID,
		StoreOwnerID: "seller-1",
		Commission:   10,
	})
	if err != nil {
		t.Fatalf("HandlePaymentSucceeded returned error: %v", err)
	}

	sellerWallet, err := repo.GetByUserID(context.Background(), "seller-1")
	if err != nil {
		t.Fatalf("expected seller wallet to exist, got error: %v", err)
	}
	if sellerWallet.Balance != 900 {
		t.Fatalf("expected seller balance 900, got %d", sellerWallet.Balance)
	}

	adminWallet, err := repo.GetByUserID(context.Background(), "admin-1")
	if err != nil {
		t.Fatalf("expected platform wallet to exist, got error: %v", err)
	}
	if adminWallet.Balance != 100 {
		t.Fatalf("expected platform balance 100, got %d", adminWallet.Balance)
	}

	if len(repo.transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(repo.transactions))
	}
	if repo.transactions[0].Source != "settlement" {
		t.Fatalf("expected first transaction source settlement, got %s", repo.transactions[0].Source)
	}
	if repo.transactions[1].Source != "commission" {
		t.Fatalf("expected second transaction source commission, got %s", repo.transactions[1].Source)
	}
}
