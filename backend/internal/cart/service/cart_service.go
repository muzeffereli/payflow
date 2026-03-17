package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/google/uuid"

	"payment-platform/internal/cart/domain"
	"payment-platform/internal/cart/port"
)

type CartService struct {
	repo     port.CartRepository
	products port.ProductClient
	orders   port.OrderClient
	log      *slog.Logger
}

func New(repo port.CartRepository, products port.ProductClient, orders port.OrderClient, log *slog.Logger) *CartService {
	return &CartService{repo: repo, products: products, orders: orders, log: log}
}

var (
	ErrInvalidProduct    = errors.New("invalid product")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrVariantRequired   = errors.New("variant selection is required")
)

type AddItemRequest struct {
	UserID    string
	ProductID string
	VariantID *string
	Quantity  int
}

func (s *CartService) AddItem(ctx context.Context, req AddItemRequest) (*domain.Cart, error) {
	cart, err := s.repo.Get(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	infos, err := s.products.GetProducts(ctx, []string{req.ProductID})
	if err != nil {
		return nil, fmt.Errorf("fetch product: %w", err)
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("%w: %s not found", ErrInvalidProduct, req.ProductID)
	}
	p := infos[0]
	if p.Status != "active" {
		return nil, fmt.Errorf("%w: %s is %s", ErrInvalidProduct, req.ProductID, p.Status)
	}
	if len(p.Variants) > 0 && req.VariantID == nil {
		return nil, fmt.Errorf("%w for product %s", ErrVariantRequired, req.ProductID)
	}

	currentQty := 0
	for _, item := range cart.Items {
		if item.ProductID == req.ProductID && sameVariantID(item.VariantID, req.VariantID) {
			currentQty = item.Quantity
			break
		}
	}

	if err := validateCatalogSelection(p, req.VariantID, currentQty+req.Quantity); err != nil {
		return nil, err
	}

	cart.AddItem(req.ProductID, req.VariantID, req.Quantity)

	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("save cart: %w", err)
	}
	return cart, nil
}

func (s *CartService) RemoveItem(ctx context.Context, userID, productID string, variantID *string) (*domain.Cart, error) {
	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	if err := cart.RemoveItem(productID, variantID); err != nil {
		return nil, err // domain.ErrItemNotFound
	}

	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("save cart: %w", err)
	}
	return cart, nil
}

func (s *CartService) SetQuantity(ctx context.Context, userID, productID string, variantID *string, qty int) (*domain.Cart, error) {
	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	if qty > 0 {
		infos, err := s.products.GetProducts(ctx, []string{productID})
		if err != nil {
			return nil, fmt.Errorf("fetch product: %w", err)
		}
		if len(infos) == 0 {
			return nil, fmt.Errorf("%w: %s not found", ErrInvalidProduct, productID)
		}
		if err := validateCatalogSelection(infos[0], variantID, qty); err != nil {
			return nil, err
		}
	}

	if err := cart.SetQuantity(productID, variantID, qty); err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, cart); err != nil {
		return nil, fmt.Errorf("save cart: %w", err)
	}
	return cart, nil
}

type CartView struct {
	UserID     string         `json:"user_id"`
	Items      []CartViewItem `json:"items"`
	TotalCents int64          `json:"total_cents"`
	Currency   string         `json:"currency"`
}

type CartViewItem struct {
	ProductID    string  `json:"product_id"`
	VariantID    *string `json:"variant_id,omitempty"`
	Name         string  `json:"name"`
	VariantLabel string  `json:"variant_label,omitempty"`
	VariantSKU   string  `json:"variant_sku,omitempty"`
	Quantity     int     `json:"quantity"`
	UnitPrice    int64   `json:"unit_price"`
	LineTotal    int64   `json:"line_total"`
	Currency     string  `json:"currency"`
}

func (s *CartService) GetCart(ctx context.Context, userID string) (*CartView, error) {
	cart, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	if cart.IsEmpty() {
		return &CartView{UserID: userID, Items: []CartViewItem{}}, nil
	}

	ids := uniqueProductIDs(cart.Items)
	infos, err := s.products.GetProducts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch product prices: %w", err)
	}

	byID := make(map[string]port.ProductInfo, len(infos))
	for _, p := range infos {
		byID[p.ID] = p
	}

	view := &CartView{UserID: userID, Items: make([]CartViewItem, 0, len(cart.Items))}
	for _, item := range cart.Items {
		p, ok := byID[item.ProductID]
		if !ok {
			continue // product may have been deleted â€” skip silently
		}
		unitPrice := p.Price
		variantLabel := ""
		variantSKU := ""
		if item.VariantID != nil {
			variant, found := findVariant(p, item.VariantID)
			if !found {
				continue
			}
			unitPrice = effectiveVariantPrice(p, variant)
			variantLabel = formatVariantLabel(variant.AttributeValues)
			variantSKU = variant.SKU
		}
		lineTotal := unitPrice * int64(item.Quantity)
		view.Items = append(view.Items, CartViewItem{
			ProductID:    item.ProductID,
			VariantID:    item.VariantID,
			Name:         p.Name,
			VariantLabel: variantLabel,
			VariantSKU:   variantSKU,
			Quantity:     item.Quantity,
			UnitPrice:    unitPrice,
			LineTotal:    lineTotal,
			Currency:     p.Currency,
		})
		view.TotalCents += lineTotal
		if view.Currency == "" {
			view.Currency = p.Currency
		}
	}

	return view, nil
}

type CheckoutRequest struct {
	UserID         string
	Currency       string
	PaymentMethod  string
	IdempotencyKey string
}

type CheckoutResult struct {
	OrderIDs []string // one entry per store (or one platform order if no store items)
}

func (s *CartService) Checkout(ctx context.Context, req CheckoutRequest) (*CheckoutResult, error) {
	cart, err := s.repo.Get(ctx, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}
	if cart.IsEmpty() {
		return nil, domain.ErrCartEmpty
	}

	ids := uniqueProductIDs(cart.Items)
	infos, err := s.products.GetProducts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("fetch products for checkout: %w", err)
	}
	byID := make(map[string]port.ProductInfo, len(infos))
	for _, p := range infos {
		byID[p.ID] = p
	}

	groups := make(map[string][]port.OrderItemInput) // storeKey â†’ items
	storeIDByKey := make(map[string]*string)         // storeKey â†’ *string store ID

	for _, item := range cart.Items {
		p, ok := byID[item.ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: product %s not found", ErrInvalidProduct, item.ProductID)
		}
		if p.Status != "active" {
			return nil, fmt.Errorf("%w: product %s is %s", ErrInvalidProduct, item.ProductID, p.Status)
		}
		if err := validateCatalogSelection(p, item.VariantID, item.Quantity); err != nil {
			return nil, err
		}

		storeKey := ""
		if p.StoreID != nil {
			storeKey = *p.StoreID
		}
		groups[storeKey] = append(groups[storeKey], port.OrderItemInput{
			ProductID: item.ProductID,
			VariantID: item.VariantID,
			Quantity:  item.Quantity,
		})
		storeIDByKey[storeKey] = p.StoreID
	}

	baseKey := req.IdempotencyKey
	if baseKey == "" {
		baseKey = uuid.New().String()
	}

	var orderIDs []string
	groupIdx := 0
	for storeKey, items := range groups {
		idemKey := fmt.Sprintf("%s-g%d", baseKey, groupIdx)
		groupIdx++

		orderID, err := s.orders.CreateOrder(ctx, port.CreateOrderRequest{
			UserID:         req.UserID,
			Currency:       req.Currency,
			PaymentMethod:  normalizeCheckoutPaymentMethod(req.PaymentMethod),
			IdempotencyKey: idemKey,
			Items:          items,
			StoreID:        storeIDByKey[storeKey],
		})
		if err != nil {
			return nil, fmt.Errorf("create order for store %q: %w", storeKey, err)
		}
		orderIDs = append(orderIDs, orderID)
	}

	if err := s.repo.Delete(ctx, req.UserID); err != nil {
		s.log.Warn("failed to clear cart after checkout", "user_id", req.UserID, "err", err)
	}

	s.log.Info("checkout successful", "user_id", req.UserID, "orders", len(orderIDs))
	return &CheckoutResult{OrderIDs: orderIDs}, nil
}

func validateCatalogSelection(product port.ProductInfo, variantID *string, qty int) error {
	if len(product.Variants) > 0 && variantID == nil {
		return fmt.Errorf("%w for product %s", ErrVariantRequired, product.ID)
	}
	if variantID != nil {
		variant, ok := findVariant(product, variantID)
		if !ok {
			return fmt.Errorf("%w: variant %s not found for product %s", ErrInvalidProduct, *variantID, product.ID)
		}
		if variant.Status != "active" {
			return fmt.Errorf("%w: variant %s is %s", ErrInvalidProduct, variant.ID, variant.Status)
		}
		if variant.Stock < qty {
			return fmt.Errorf("%w: variant %s has %d in stock, need %d",
				ErrInsufficientStock, variant.ID, variant.Stock, qty)
		}
		return nil
	}

	if product.Stock < qty {
		return fmt.Errorf("%w: product %s has %d in stock, need %d",
			ErrInsufficientStock, product.ID, product.Stock, qty)
	}
	return nil
}

func findVariant(product port.ProductInfo, variantID *string) (port.VariantInfo, bool) {
	if variantID == nil {
		return port.VariantInfo{}, false
	}
	for _, variant := range product.Variants {
		if variant.ID == *variantID {
			return variant, true
		}
	}
	return port.VariantInfo{}, false
}

func effectiveVariantPrice(product port.ProductInfo, variant port.VariantInfo) int64 {
	if variant.Price != nil {
		return *variant.Price
	}
	return product.Price
}

func sameVariantID(a, b *string) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

func uniqueProductIDs(items []domain.CartItem) []string {
	seen := make(map[string]struct{}, len(items))
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.ProductID]; ok {
			continue
		}
		seen[item.ProductID] = struct{}{}
		ids = append(ids, item.ProductID)
	}
	return ids
}

func formatVariantLabel(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", key, values[key]))
	}
	return strings.Join(parts, " / ")
}

func normalizeCheckoutPaymentMethod(method string) string {
	if method == "wallet" {
		return "wallet"
	}
	return "card"
}
