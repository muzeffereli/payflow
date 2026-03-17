package service

import (
	"context"
	"errors"
	"log/slog"

	"payment-platform/internal/store/domain"
	"payment-platform/internal/store/port"
	"payment-platform/pkg/eventbus"
)

type StoreService struct {
	repo port.StoreRepository
	pub  port.EventPublisher
	log  *slog.Logger
}

func New(repo port.StoreRepository, pub port.EventPublisher, log *slog.Logger) *StoreService {
	return &StoreService{repo: repo, pub: pub, log: log}
}

func (s *StoreService) CreateStore(ctx context.Context, ownerID, name, description, email string, commission int) (*domain.Store, error) {
	store, err := domain.NewStore(ownerID, name, description, email, commission)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, store); err != nil {
		return nil, err
	}

	s.publishAsync(ctx, eventbus.SubjectStoreCreated, eventbus.StoreCreatedData{
		StoreID: store.ID,
		OwnerID: store.OwnerID,
		Name:    store.Name,
	})

	s.log.Info("store created", "store_id", store.ID, "owner_id", ownerID)
	return store, nil
}

func (s *StoreService) GetStore(ctx context.Context, id string) (*domain.Store, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *StoreService) GetMyStore(ctx context.Context, ownerID string) (*domain.Store, error) {
	return s.repo.GetByOwnerID(ctx, ownerID)
}

func (s *StoreService) ListStores(ctx context.Context, status string, limit, offset int) ([]*domain.Store, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.List(ctx, port.ListFilter{Status: status, Limit: limit, Offset: offset})
}

func (s *StoreService) UpdateStore(ctx context.Context, storeID, callerID, name, description, email string) (*domain.Store, error) {
	store, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if store.OwnerID != callerID {
		return nil, domain.ErrNotStoreOwner
	}
	if err := store.UpdateDetails(name, description, email); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, store); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *StoreService) ApproveStore(ctx context.Context, storeID string) (*domain.Store, error) {
	store, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if err := store.Approve(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, store); err != nil {
		return nil, err
	}
	s.publishAsync(ctx, eventbus.SubjectStoreApproved, eventbus.StoreStatusChangedData{
		StoreID: store.ID,
		OwnerID: store.OwnerID,
		Status:  string(store.Status),
	})
	s.log.Info("store approved", "store_id", storeID)
	return store, nil
}

func (s *StoreService) SuspendStore(ctx context.Context, storeID string) (*domain.Store, error) {
	store, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if err := store.Suspend(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, store); err != nil {
		return nil, err
	}
	s.publishAsync(ctx, eventbus.SubjectStoreSuspended, eventbus.StoreStatusChangedData{
		StoreID: store.ID,
		OwnerID: store.OwnerID,
		Status:  string(store.Status),
	})
	s.log.Info("store suspended", "store_id", storeID)
	return store, nil
}

func (s *StoreService) ReactivateStore(ctx context.Context, storeID string) (*domain.Store, error) {
	store, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if err := store.Reactivate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, store); err != nil {
		return nil, err
	}
	s.publishAsync(ctx, eventbus.SubjectStoreApproved, eventbus.StoreStatusChangedData{
		StoreID: store.ID,
		OwnerID: store.OwnerID,
		Status:  string(store.Status),
	})
	return store, nil
}

func (s *StoreService) CloseStore(ctx context.Context, storeID, callerID, callerRole string) (*domain.Store, error) {
	store, err := s.repo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if callerRole != "admin" && store.OwnerID != callerID {
		return nil, domain.ErrNotStoreOwner
	}
	if err := store.Close(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, store); err != nil {
		return nil, err
	}
	s.log.Info("store closed", "store_id", storeID)
	return store, nil
}

func (s *StoreService) publishAsync(ctx context.Context, subject string, data interface{}) {
	go func() {
		if err := s.pub.Publish(ctx, subject, data); err != nil {
			if !errors.Is(err, context.Canceled) {
				s.log.Error("failed to publish event", "subject", subject, "err", err)
			}
		}
	}()
}
