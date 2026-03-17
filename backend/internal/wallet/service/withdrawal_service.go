package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"payment-platform/internal/wallet/domain"
	"payment-platform/internal/wallet/port"
	"payment-platform/pkg/eventbus"
)

type WithdrawalService struct {
	withdrawals port.WithdrawalRepository
	wallets     port.WalletRepository
	publisher   port.EventPublisher
	log         *slog.Logger
}

func NewWithdrawalService(
	withdrawals port.WithdrawalRepository,
	wallets port.WalletRepository,
	pub port.EventPublisher,
	log *slog.Logger,
) *WithdrawalService {
	return &WithdrawalService{
		withdrawals: withdrawals,
		wallets:     wallets,
		publisher:   pub,
		log:         log,
	}
}

var ErrInsufficientBalance = errors.New("wallet balance is insufficient for this withdrawal")

func (s *WithdrawalService) RequestWithdrawal(
	ctx context.Context,
	userID, storeID, currency, method string,
	amount int64,
) (*domain.Withdrawal, error) {
	wallet, err := s.wallets.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	if wallet.Balance < amount {
		return nil, ErrInsufficientBalance
	}

	w, err := domain.NewWithdrawal(userID, storeID, currency, method, amount)
	if err != nil {
		return nil, err
	}

	if err := s.withdrawals.Create(ctx, w); err != nil {
		return nil, fmt.Errorf("save withdrawal: %w", err)
	}

	s.log.Info("withdrawal requested",
		"withdrawal_id", w.ID, "user_id", userID, "amount", amount)

	s.publishWithdrawalRequested(ctx, w)
	return w, nil
}

func (s *WithdrawalService) ListMyWithdrawals(ctx context.Context, userID string, limit, offset int) ([]*domain.Withdrawal, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.withdrawals.ListByUser(ctx, userID, limit, offset)
}

func (s *WithdrawalService) ListPendingWithdrawals(ctx context.Context, limit, offset int) ([]*domain.Withdrawal, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.withdrawals.ListByStatus(ctx, domain.WithdrawalPending, limit, offset)
}

func (s *WithdrawalService) ApproveWithdrawal(ctx context.Context, withdrawalID string) (*domain.Withdrawal, error) {
	w, err := s.withdrawals.GetByID(ctx, withdrawalID)
	if err != nil {
		return nil, err
	}

	if err := w.Approve(); err != nil {
		return nil, err // already approved/rejected
	}

	tx, err := s.wallets.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	wallet, err := s.wallets.GetByUserIDForUpdate(ctx, tx, w.UserID)
	if err != nil {
		return nil, fmt.Errorf("lock wallet: %w", err)
	}
	if wallet.Balance < w.Amount {
		return nil, ErrInsufficientBalance
	}
	balanceBefore := wallet.Balance
	if err := wallet.Debit(w.Amount); err != nil {
		return nil, err
	}
	if err := s.wallets.UpdateBalance(ctx, tx, wallet); err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}
	txRecord := domain.NewTransaction(wallet.ID, "debit", "withdrawal", w.ID, w.Amount, balanceBefore, wallet.Balance)
	if err := s.wallets.SaveTransaction(ctx, tx, txRecord); err != nil {
		return nil, fmt.Errorf("save transaction: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	if err := s.withdrawals.UpdateStatus(ctx, w); err != nil {
		s.log.Error("wallet debited but withdrawal status update failed â€” manual fix needed",
			"withdrawal_id", w.ID, "err", err)
		return nil, fmt.Errorf("update withdrawal status: %w", err)
	}

	s.log.Info("withdrawal approved and wallet debited",
		"withdrawal_id", w.ID, "user_id", w.UserID, "amount", w.Amount)

	s.publishWithdrawalApproved(ctx, w)
	return w, nil
}

func (s *WithdrawalService) RejectWithdrawal(ctx context.Context, withdrawalID, reason string) (*domain.Withdrawal, error) {
	w, err := s.withdrawals.GetByID(ctx, withdrawalID)
	if err != nil {
		return nil, err
	}

	if err := w.Reject(reason); err != nil {
		return nil, err
	}

	if err := s.withdrawals.UpdateStatus(ctx, w); err != nil {
		return nil, fmt.Errorf("update withdrawal status: %w", err)
	}

	s.log.Info("withdrawal rejected",
		"withdrawal_id", w.ID, "user_id", w.UserID, "reason", reason)

	s.publishWithdrawalRejected(ctx, w)
	return w, nil
}

func (s *WithdrawalService) publishWithdrawalRequested(ctx context.Context, w *domain.Withdrawal) {
	data, _ := json.Marshal(eventbus.WithdrawalRequestedData{
		WithdrawalID: w.ID,
		UserID:       w.UserID,
		StoreID:      w.StoreID,
		Amount:       w.Amount,
		Currency:     w.Currency,
		Method:       w.Method,
	})
	event := eventbus.NewEvent("withdrawal.requested", w.ID, "withdrawal", data,
		eventbus.Metadata{UserID: w.UserID})
	if err := s.publisher.Publish(ctx, eventbus.SubjectWithdrawalRequested, event); err != nil {
		s.log.Error("failed to publish withdrawal.requested", "withdrawal_id", w.ID, "err", err)
	}
}

func (s *WithdrawalService) publishWithdrawalApproved(ctx context.Context, w *domain.Withdrawal) {
	data, _ := json.Marshal(eventbus.WithdrawalApprovedData{
		WithdrawalID: w.ID,
		UserID:       w.UserID,
		Amount:       w.Amount,
		Currency:     w.Currency,
	})
	event := eventbus.NewEvent("withdrawal.approved", w.ID, "withdrawal", data,
		eventbus.Metadata{UserID: w.UserID})
	if err := s.publisher.Publish(ctx, eventbus.SubjectWithdrawalApproved, event); err != nil {
		s.log.Error("failed to publish withdrawal.approved", "withdrawal_id", w.ID, "err", err)
	}
}

func (s *WithdrawalService) publishWithdrawalRejected(ctx context.Context, w *domain.Withdrawal) {
	data, _ := json.Marshal(eventbus.WithdrawalRejectedData{
		WithdrawalID: w.ID,
		UserID:       w.UserID,
		Amount:       w.Amount,
		Reason:       w.Notes,
	})
	event := eventbus.NewEvent("withdrawal.rejected", w.ID, "withdrawal", data,
		eventbus.Metadata{UserID: w.UserID})
	if err := s.publisher.Publish(ctx, eventbus.SubjectWithdrawalRejected, event); err != nil {
		s.log.Error("failed to publish withdrawal.rejected", "withdrawal_id", w.ID, "err", err)
	}
}
