package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"payment-platform/internal/wallet/domain"
	"payment-platform/internal/wallet/port"
	"payment-platform/pkg/eventbus"
)

type WalletService struct {
	repo                 port.WalletRepository
	publisher            port.EventPublisher
	platformWalletUserID string
	log                  *slog.Logger
}

var ErrNotFound = errors.New("wallet not found")

func New(repo port.WalletRepository, pub port.EventPublisher, log *slog.Logger, platformWalletUserID string) *WalletService {
	return &WalletService{
		repo:                 repo,
		publisher:            pub,
		platformWalletUserID: platformWalletUserID,
		log:                  log,
	}
}

func (s *WalletService) CreateWallet(ctx context.Context, userID, currency string) (*domain.Wallet, error) {
	wallet := domain.NewWallet(userID, currency)
	if err := s.repo.Create(ctx, wallet); err != nil {
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	return wallet, nil
}

func (s *WalletService) GetWallet(ctx context.Context, userID string) (*domain.Wallet, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *WalletService) Credit(ctx context.Context, userID string, amount int64, source, referenceID string) error {
	if err := s.ensureWalletExists(ctx, userID, "USD"); err != nil {
		return err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer rollbackTx(tx)

	wallet, err := s.repo.GetByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("lock wallet: %w", err)
	}

	balanceBefore := wallet.Balance
	if err := wallet.Credit(amount); err != nil {
		return err
	}

	if err := s.repo.UpdateBalance(ctx, tx, wallet); err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	txRecord := domain.NewTransaction(wallet.ID, "credit", source, referenceID, amount, balanceBefore, wallet.Balance)
	if err := s.repo.SaveTransaction(ctx, tx, txRecord); err != nil {
		return fmt.Errorf("save transaction: %w", err)
	}

	if err := commitTx(tx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	s.log.Info("wallet credited",
		"user_id", userID,
		"amount", amount,
		"balance_after", wallet.Balance,
		"source", source,
	)

	s.publishCredited(ctx, wallet, txRecord)
	return nil
}

func (s *WalletService) ensureWalletExists(ctx context.Context, userID, currency string) error {
	_, err := s.repo.GetByUserID(ctx, userID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("get wallet: %w", err)
	}

	wallet := domain.NewWallet(userID, currency)
	if err := s.repo.Create(ctx, wallet); err != nil {
		if _, retryErr := s.repo.GetByUserID(ctx, userID); retryErr == nil {
			return nil
		}
		return fmt.Errorf("create wallet: %w", err)
	}

	s.log.Info("wallet auto-provisioned",
		"user_id", userID,
		"currency", currency,
	)
	return nil
}

func (s *WalletService) Debit(ctx context.Context, userID string, amount int64, source, referenceID string) error {
	_, err := s.debit(ctx, userID, amount, source, referenceID)
	return err
}

func (s *WalletService) DebitForPayment(ctx context.Context, userID string, amount int64, referenceID string) (*domain.Transaction, error) {
	return s.debit(ctx, userID, amount, "payment", referenceID)
}

func (s *WalletService) debit(ctx context.Context, userID string, amount int64, source, referenceID string) (*domain.Transaction, error) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer rollbackTx(tx)

	wallet, err := s.repo.GetByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		return nil, fmt.Errorf("lock wallet: %w", err)
	}

	balanceBefore := wallet.Balance
	if err := wallet.Debit(amount); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateBalance(ctx, tx, wallet); err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	txRecord := domain.NewTransaction(wallet.ID, "debit", source, referenceID, amount, balanceBefore, wallet.Balance)
	if err := s.repo.SaveTransaction(ctx, tx, txRecord); err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}

	if err := commitTx(tx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return txRecord, nil
}

func (s *WalletService) HandlePaymentSucceeded(ctx context.Context, data eventbus.PaymentSucceededData) error {
	if data.StoreOwnerID == "" || data.StoreID == nil {
		return nil
	}

	sellerAmount := data.Amount * int64(100-data.Commission) / 100
	if sellerAmount <= 0 {
		s.log.Warn("seller amount is zero after commission - skipping settlement",
			"store_owner_id", data.StoreOwnerID, "commission", data.Commission)
		return nil
	}

	s.log.Info("crediting seller wallet (settlement)",
		"store_owner_id", data.StoreOwnerID,
		"store_id", *data.StoreID,
		"gross_amount", data.Amount,
		"commission_pct", data.Commission,
		"seller_amount", sellerAmount,
		"payment_id", data.PaymentID,
	)

	if err := s.Credit(ctx, data.StoreOwnerID, sellerAmount, "settlement", data.PaymentID); err != nil {
		s.log.Error("seller wallet credit failed - settlement pending manual review",
			"store_owner_id", data.StoreOwnerID,
			"payment_id", data.PaymentID,
			"seller_amount", sellerAmount,
			"err", err,
		)
		return err
	}

	if err := s.creditPlatformCommission(ctx, data, sellerAmount); err != nil {
		return err
	}

	s.publishSettlementCompleted(ctx, data, sellerAmount)
	return nil
}

func (s *WalletService) ListTransactions(ctx context.Context, userID string, limit, offset int) ([]*domain.Transaction, int, error) {
	wallet, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListTransactions(ctx, wallet.ID, limit, offset)
}

func (s *WalletService) HandlePaymentRefunded(ctx context.Context, data eventbus.PaymentRefundedData) error {
	s.log.Info("processing refund credit",
		"user_id", data.UserID,
		"order_id", data.OrderID,
		"amount", data.Amount,
		"refund_id", data.RefundID,
	)
	return s.Credit(ctx, data.UserID, data.Amount, "refund", data.RefundID)
}

// HandleOrderCancelled is intentionally a no-op.
// Wallet debits only occur when a payment succeeds (via processPayment in the payment service).
// If the wallet debit fails the payment is marked failed — no debit happened.
// If the order is cancelled while still pending/confirmed, no debit has occurred yet.
// Paid orders go through the refund flow (HandlePaymentRefunded) which already credits the buyer.
func (s *WalletService) HandleOrderCancelled(_ context.Context, data eventbus.OrderCancelledData) error {
	s.log.Debug("order cancelled — no wallet action required", "order_id", data.OrderID)
	return nil
}

func (s *WalletService) publishSettlementCompleted(ctx context.Context, data eventbus.PaymentSucceededData, sellerAmount int64) {
	payload, err := json.Marshal(eventbus.SettlementCompletedData{
		PaymentID:    data.PaymentID,
		OrderID:      data.OrderID,
		StoreID:      *data.StoreID,
		StoreOwnerID: data.StoreOwnerID,
		GrossAmount:  data.Amount,
		Commission:   data.Commission,
		SellerAmount: sellerAmount,
		Currency:     data.Currency,
	})
	if err != nil {
		s.log.Error("failed to marshal settlement.completed", "payment_id", data.PaymentID, "err", err)
		return
	}
	meta := eventbus.Metadata{CorrelationID: data.OrderID, UserID: data.StoreOwnerID}
	event := eventbus.NewEvent("settlement.completed", data.PaymentID, "payment", payload, meta)
	if err := s.publisher.Publish(ctx, eventbus.SubjectSettlementCompleted, event); err != nil {
		s.log.Error("failed to publish settlement.completed", "payment_id", data.PaymentID, "err", err)
	}
}

func (s *WalletService) publishCredited(ctx context.Context, w *domain.Wallet, tx *domain.Transaction) {
	data, err := json.Marshal(eventbus.WalletCreditedData{
		WalletID:      w.ID,
		UserID:        w.UserID,
		Amount:        tx.Amount,
		TransactionID: tx.ID,
		Source:        tx.Source,
	})
	if err != nil {
		s.log.Error("failed to marshal wallet.credited", "wallet_id", w.ID, "err", err)
		return
	}

	meta := eventbus.Metadata{CorrelationID: tx.ReferenceID, UserID: w.UserID}
	event := eventbus.NewEvent("wallet.credited", w.ID, "wallet", data, meta)

	if err := s.publisher.Publish(ctx, eventbus.SubjectWalletCredited, event); err != nil {
		s.log.Error("failed to publish wallet.credited", "err", err)
	}
}

func (s *WalletService) creditPlatformCommission(ctx context.Context, data eventbus.PaymentSucceededData, sellerAmount int64) error {
	commissionAmount := data.Amount - sellerAmount
	if commissionAmount <= 0 {
		return nil
	}
	if s.platformWalletUserID == "" {
		s.log.Warn("platform wallet user id not configured - commission not credited",
			"payment_id", data.PaymentID,
			"commission_amount", commissionAmount,
		)
		return nil
	}

	s.log.Info("crediting platform wallet (commission)",
		"platform_wallet_user_id", s.platformWalletUserID,
		"payment_id", data.PaymentID,
		"commission_amount", commissionAmount,
		"commission_pct", data.Commission,
	)

	if err := s.Credit(ctx, s.platformWalletUserID, commissionAmount, "commission", data.PaymentID); err != nil {
		s.log.Error("platform wallet credit failed - commission pending manual review",
			"platform_wallet_user_id", s.platformWalletUserID,
			"payment_id", data.PaymentID,
			"commission_amount", commissionAmount,
			"err", err,
		)
		return err
	}
	return nil
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func commitTx(tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	return tx.Commit()
}
