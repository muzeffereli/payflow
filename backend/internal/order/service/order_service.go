package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"payment-platform/internal/order/domain"
	"payment-platform/internal/order/port"
	"payment-platform/pkg/catalog"
	"payment-platform/pkg/eventbus"
)

type OrderService struct {
	repo      port.OrderRepository
	publisher port.EventPublisher
	products  port.ProductClient
	log       *slog.Logger
}

var (
	ErrUnauthorized    = errors.New("user does not own this order")
	ErrNotFound        = errors.New("order not found")
	ErrInvalidProduct  = errors.New("invalid product")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrCurrencyMismatch  = errors.New("currency mismatch")
)

func New(repo port.OrderRepository, pub port.EventPublisher, products port.ProductClient, log *slog.Logger) *OrderService {
	return &OrderService{repo: repo, publisher: pub, products: products, log: log}
}

type CreateOrderRequest struct {
	UserID          string
	Currency        string
	PaymentMethod   string
	IdempotencyKey  string
	Items           []OrderItemInput
	ShippingAddress *domain.ShippingAddress
	StoreID         *string // nil = platform order; set for store-specific sub-orders
}

type OrderItemInput struct {
	ProductID string
	VariantID *string
	Quantity  int
}

func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*domain.Order, error) {
	existing, err := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("idempotency check: %w", err)
	}
	if existing != nil {
		s.log.Info("idempotent: returning existing order",
			"order_id", existing.ID,
			"idempotency_key", req.IdempotencyKey,
		)
		return existing, nil
	}

	ids := make([]string, len(req.Items))
	for i, it := range req.Items {
		ids[i] = it.ProductID
	}
	productInfos, err := s.products.GetProducts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch product prices: %w", err)
	}

	byID := make(map[string]port.ProductInfo, len(productInfos))
	for _, p := range productInfos {
		byID[p.ID] = p
	}

	items := make([]domain.OrderItem, len(req.Items))
	for i, input := range req.Items {
		p, ok := byID[input.ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: product %s not found", ErrInvalidProduct, input.ProductID)
		}
		if p.Status != "active" {
			return nil, fmt.Errorf("%w: product %s is %s", ErrInvalidProduct, input.ProductID, p.Status)
		}
		if req.Currency != "" && p.Currency != req.Currency {
			return nil, fmt.Errorf("%w: product %s currency %s does not match order currency %s",
				ErrCurrencyMismatch, input.ProductID, p.Currency, req.Currency)
		}
		items[i] = domain.OrderItem{
			ProductID: input.ProductID,
			Quantity:  input.Quantity,
			Price:     p.Price, // authoritative price from catalog
		}
		if (len(p.Attributes) > 0 || len(p.Variants) > 0) && input.VariantID == nil {
			return nil, fmt.Errorf("%w: product %s requires a variant", ErrInvalidProduct, input.ProductID)
		}
		if input.VariantID != nil {
			variant, ok := catalog.FindVariant(p, input.VariantID)
			if !ok {
				return nil, fmt.Errorf("%w: variant %s not found for product %s", ErrInvalidProduct, *input.VariantID, input.ProductID)
			}
			if variant.Status != "active" {
				return nil, fmt.Errorf("%w: variant %s is %s", ErrInvalidProduct, variant.ID, variant.Status)
			}
			if !catalog.VariantMatchesAttributes(p, variant) {
				return nil, fmt.Errorf("%w: variant %s no longer matches the current product attributes", ErrInvalidProduct, variant.ID)
			}
			if variant.Stock < input.Quantity {
				return nil, fmt.Errorf("%w: variant %s has %d in stock, requested %d",
					ErrInsufficientStock, variant.ID, variant.Stock, input.Quantity)
			}
			items[i].VariantID = input.VariantID
			items[i].VariantSKU = variant.SKU
			items[i].VariantLabel = catalog.FormatVariantLabel(variant.AttributeValues)
			if variant.Price != nil {
				items[i].Price = *variant.Price
			}
			continue
		}
		if p.Stock < input.Quantity {
			return nil, fmt.Errorf("%w: product %s has %d in stock, requested %d",
				ErrInsufficientStock, input.ProductID, p.Stock, input.Quantity)
		}
	}

	order := domain.NewOrder(req.UserID, req.Currency, req.IdempotencyKey, items, req.ShippingAddress)
	order.StoreID = req.StoreID

	eventPayload, err := s.marshalOrderCreated(order, req.PaymentMethod)
	if err != nil {
		return nil, fmt.Errorf("marshal order.created: %w", err)
	}

	if err := s.repo.CreateWithOutbox(ctx, order, eventbus.SubjectOrderCreated, eventPayload); err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	s.log.Info("order created, event queued in outbox", "order_id", order.ID)
	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID, userID string) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	if order.UserID != userID {
		return nil, ErrUnauthorized
	}

	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context, userID string, limit, offset int) ([]domain.Order, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

func (s *OrderService) ListStoreOrders(ctx context.Context, storeID string, limit, offset int) ([]domain.Order, error) {
	return s.repo.ListByStore(ctx, storeID, limit, offset)
}

func (s *OrderService) GetStoreAnalytics(ctx context.Context, storeID string) (*port.StoreAnalytics, error) {
	return s.repo.GetStoreAnalytics(ctx, storeID)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID, userID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if order.UserID != userID {
		return ErrUnauthorized
	}

	if err := order.Transition(domain.StatusCancelled); err != nil {
		return err // returns *InvalidTransitionError if already paid/refunded
	}

	if err := s.repo.UpdateStatus(ctx, order.ID, order.Status); err != nil {
		return fmt.Errorf("cancel order: %w", err)
	}

	if err := s.publishOrderCancelled(ctx, order); err != nil {
		s.log.Error("failed to publish orders.cancelled â€” order is cancelled in DB",
			"order_id", order.ID, "err", err)
	}

	return nil
}

func (s *OrderService) HandlePaymentSucceeded(ctx context.Context, orderID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if order.Status == domain.StatusPaid || order.Status == domain.StatusRefunded {
		s.log.Info("order already paid, skipping duplicate payment.succeeded",
			"order_id", orderID, "status", order.Status)
		return nil
	}

	if order.Status == domain.StatusPending {
		if err := order.Transition(domain.StatusConfirmed); err != nil {
			return fmt.Errorf("transition to confirmed: %w", err)
		}
		if err := s.repo.UpdateStatus(ctx, order.ID, order.Status); err != nil {
			return fmt.Errorf("save confirmed status: %w", err)
		}
		s.publishOrderConfirmed(ctx, order)
		s.log.Info("order confirmed", "order_id", orderID)
	}

	if err := order.Transition(domain.StatusPaid); err != nil {
		return fmt.Errorf("transition to paid: %w", err)
	}
	if err := s.repo.UpdateStatus(ctx, order.ID, order.Status); err != nil {
		return fmt.Errorf("save paid status: %w", err)
	}
	s.log.Info("order paid", "order_id", orderID)
	return nil
}

func (s *OrderService) publishOrderConfirmed(ctx context.Context, order *domain.Order) {
	data, err := json.Marshal(eventbus.OrderConfirmedData{
		OrderID: order.ID,
		UserID:  order.UserID,
	})
	if err != nil {
		s.log.Error("failed to marshal orders.confirmed", "order_id", order.ID, "err", err)
		return
	}
	meta := eventbus.Metadata{CorrelationID: order.ID, UserID: order.UserID}
	event := eventbus.NewEvent("order.confirmed", order.ID, "order", data, meta)
	if err := s.publisher.Publish(ctx, eventbus.SubjectOrderConfirmed, event); err != nil {
		s.log.Error("failed to publish orders.confirmed", "order_id", order.ID, "err", err)
	}
}

func (s *OrderService) HandlePaymentRefunded(ctx context.Context, orderID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if err := order.Transition(domain.StatusRefunded); err != nil {
		var transErr *domain.InvalidTransitionError
		if errors.As(err, &transErr) {
			s.log.Warn("skipping duplicate payment.refunded event",
				"order_id", orderID,
				"current_status", order.Status,
			)
			return nil
		}
		return err
	}

	return s.repo.UpdateStatus(ctx, order.ID, order.Status)
}

func (s *OrderService) HandlePaymentFailed(ctx context.Context, orderID string) error {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get order: %w", err)
	}

	if err := order.Transition(domain.StatusCancelled); err != nil {
		var transErr *domain.InvalidTransitionError
		if errors.As(err, &transErr) {
			s.log.Warn("skipping duplicate payment.failed event", "order_id", orderID)
			return nil
		}
		return err
	}

	return s.repo.UpdateStatus(ctx, order.ID, order.Status)
}


func (s *OrderService) marshalOrderCreated(order *domain.Order, paymentMethod string) ([]byte, error) {
	payloadData, err := json.Marshal(eventbus.OrderCreatedData{
		OrderID:        order.ID,
		UserID:         order.UserID,
		Items:          toEventItems(order.Items),
		TotalAmount:    order.TotalAmount,
		Currency:       order.Currency,
		PaymentMethod:  eventbus.NormalizePaymentMethod(paymentMethod),
		IdempotencyKey: order.IdempotencyKey,
		StoreID:        order.StoreID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal order created data: %w", err)
	}

	meta := eventbus.Metadata{
		CorrelationID: order.ID,
		UserID:        order.UserID,
	}
	event := eventbus.NewEvent("order.created", order.ID, "order", payloadData, meta)

	b, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal event envelope: %w", err)
	}
	return b, nil
}

func (s *OrderService) publishOrderCancelled(ctx context.Context, order *domain.Order) error {
	data, err := json.Marshal(eventbus.OrderCancelledData{
		OrderID: order.ID,
		UserID:  order.UserID,
		Reason:  "user_requested",
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	meta := eventbus.Metadata{
		CorrelationID: order.ID,
		UserID:        order.UserID,
	}

	event := eventbus.NewEvent("order.cancelled", order.ID, "order", data, meta)
	return s.publisher.Publish(ctx, eventbus.SubjectOrderCancelled, event)
}

func toEventItems(items []domain.OrderItem) []eventbus.OrderItem {
	out := make([]eventbus.OrderItem, len(items))
	for i, item := range items {
		out[i] = eventbus.OrderItem{
			ProductID:    item.ProductID,
			VariantID:    item.VariantID,
			VariantSKU:   item.VariantSKU,
			VariantLabel: item.VariantLabel,
			Quantity:     item.Quantity,
			Price:        item.Price,
		}
	}
	return out
}

