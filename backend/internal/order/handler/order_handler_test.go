package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"payment-platform/internal/order/domain"
	"payment-platform/internal/order/handler"
	"payment-platform/internal/order/port"
	"payment-platform/internal/order/service"
	"payment-platform/pkg/eventbus"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeRepo struct {
	orders map[string]*domain.Order
	byKey  map[string]*domain.Order
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		orders: make(map[string]*domain.Order),
		byKey:  make(map[string]*domain.Order),
	}
}

func (r *fakeRepo) CreateWithOutbox(_ context.Context, o *domain.Order, _ string, _ []byte) error {
	r.orders[o.ID] = o
	r.byKey[o.IdempotencyKey] = o
	return nil
}

func (r *fakeRepo) Create(_ context.Context, o *domain.Order) error {
	r.orders[o.ID] = o
	r.byKey[o.IdempotencyKey] = o
	return nil
}

func (r *fakeRepo) GetByID(_ context.Context, id string) (*domain.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return nil, service.ErrNotFound
	}
	return o, nil
}

func (r *fakeRepo) GetByIdempotencyKey(_ context.Context, key string) (*domain.Order, error) {
	return r.byKey[key], nil // nil, nil when not found (by design)
}

func (r *fakeRepo) UpdateStatus(_ context.Context, id string, status domain.OrderStatus) error {
	if o, ok := r.orders[id]; ok {
		o.Status = status
	}
	return nil
}

func (r *fakeRepo) ListByUser(_ context.Context, userID string, limit, offset int) ([]domain.Order, error) {
	var result []domain.Order
	for _, o := range r.orders {
		if o.UserID == userID {
			result = append(result, *o)
		}
	}
	if offset >= len(result) {
		return []domain.Order{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (r *fakeRepo) ListByStore(_ context.Context, storeID string, limit, offset int) ([]domain.Order, error) {
	var result []domain.Order
	for _, o := range r.orders {
		if o.StoreID != nil && *o.StoreID == storeID {
			result = append(result, *o)
		}
	}
	if offset >= len(result) {
		return []domain.Order{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (r *fakeRepo) GetStoreAnalytics(_ context.Context, storeID string) (*port.StoreAnalytics, error) {
	analytics := &port.StoreAnalytics{}
	for _, o := range r.orders {
		if o.StoreID == nil || *o.StoreID != storeID {
			continue
		}
		analytics.TotalOrders++
		analytics.TotalRevenue += o.TotalAmount
		switch o.Status {
		case domain.StatusPaid:
			analytics.PaidOrders++
		case domain.StatusPending:
			analytics.PendingOrders++
		}
	}
	return analytics, nil
}

type fakePublisher struct{}

func (p *fakePublisher) Publish(_ context.Context, _ string, _ eventbus.Event) error {
	return nil
}

type fakeProductClient struct {
	products map[string]port.ProductInfo
}

func newFakeProductClient(products ...port.ProductInfo) *fakeProductClient {
	m := make(map[string]port.ProductInfo, len(products))
	for _, p := range products {
		m[p.ID] = p
	}
	return &fakeProductClient{products: m}
}

func (c *fakeProductClient) GetProducts(_ context.Context, ids []string) ([]port.ProductInfo, error) {
	var result []port.ProductInfo
	for _, id := range ids {
		if p, ok := c.products[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func buildRouter(t *testing.T, products ...port.ProductInfo) *gin.Engine {
	t.Helper()
	repo := newFakeRepo()
	pub := &fakePublisher{}
	pc := newFakeProductClient(products...)
	log := slog.Default()
	svc := service.New(repo, pub, pc, log)
	h := handler.New(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Next()
	})
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1)
	return r
}

func mustJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewBuffer(b)
}

func TestCreateOrder_HappyPath(t *testing.T) {
	r := buildRouter(t, port.ProductInfo{
		ID:       "prod-1",
		Name:     "Widget",
		Price:    2500,
		Currency: "USD",
		Stock:    10,
		Status:   "active",
	})

	body := map[string]any{
		"currency": "USD",
		"items":    []map[string]any{{"product_id": "prod-1", "quantity": 2}},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", mustJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "key-001")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["id"] == "" {
		t.Error("expected non-empty order id")
	}
}

func TestCreateOrder_MissingIdempotencyKey(t *testing.T) {
	r := buildRouter(t)

	body := map[string]any{"currency": "USD", "items": []map[string]any{{"product_id": "p1", "quantity": 1}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", mustJSON(t, body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOrder_MissingCurrency(t *testing.T) {
	r := buildRouter(t)

	body := map[string]any{"items": []map[string]any{{"product_id": "p1", "quantity": 1}}} // no currency
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", mustJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "key-002")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	r := buildRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetOrder_WrongOwner(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	pc := newFakeProductClient()
	log := slog.Default()
	svc := service.New(repo, pub, pc, log)

	o := domain.NewOrder("user-other", "USD", "key-x", []domain.OrderItem{}, nil)
	_ = repo.Create(context.Background(), o)

	h := handler.New(svc)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-123") // different user
		c.Next()
	})
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+o.ID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestListOrders_Empty(t *testing.T) {
	r := buildRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCancelOrder_NotFound(t *testing.T) {
	r := buildRouter(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/orders/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateOrder_Idempotent(t *testing.T) {
	r := buildRouter(t, port.ProductInfo{
		ID: "prod-2", Name: "Gadget", Price: 1000, Currency: "USD", Stock: 5, Status: "active",
	})

	body := map[string]any{
		"currency": "USD",
		"items":    []map[string]any{{"product_id": "prod-2", "quantity": 1}},
	}

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/orders", mustJSON(t, body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", "idem-key-abc")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("first request: expected 201, got %d: %s", w1.Code, w1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/orders", mustJSON(t, body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", "idem-key-abc")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("second request: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var r1, r2 map[string]any
	json.Unmarshal(w1.Body.Bytes(), &r1)
	json.Unmarshal(w2.Body.Bytes(), &r2)
	if r1["id"] != r2["id"] {
		t.Errorf("idempotency failed: got different IDs %v vs %v", r1["id"], r2["id"])
	}
}

func TestCreateOrder_WithVariantUsesVariantStock(t *testing.T) {
	variantPrice := int64(1800)
	r := buildRouter(t, port.ProductInfo{
		ID:       "prod-variant",
		Name:     "Variant Widget",
		Price:    2500,
		Currency: "USD",
		Stock:    1,
		Status:   "active",
		Variants: []port.VariantInfo{
			{
				ID:     "variant-1",
				SKU:    "VW-1",
				Price:  &variantPrice,
				Stock:  20,
				Status: "active",
				AttributeValues: map[string]string{
					"color": "blue",
					"size":  "L",
				},
			},
		},
	})

	body := map[string]any{
		"currency": "USD",
		"items": []map[string]any{{
			"product_id": "prod-variant",
			"variant_id": "variant-1",
			"quantity":   5,
		}},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", mustJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "variant-key-001")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result domain.Order
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].VariantID == nil || *result.Items[0].VariantID != "variant-1" {
		t.Fatalf("expected variant-1 in response, got %+v", result.Items[0].VariantID)
	}
	if result.TotalAmount != variantPrice*5 {
		t.Fatalf("expected total %d, got %d", variantPrice*5, result.TotalAmount)
	}
}

func TestCreateOrder_ValidationErrorsReturnBadRequest(t *testing.T) {
	r := buildRouter(t, port.ProductInfo{
		ID:       "prod-low-stock",
		Name:     "Scarce Widget",
		Price:    1200,
		Currency: "USD",
		Stock:    1,
		Status:   "active",
	})

	body := map[string]any{
		"currency": "USD",
		"items":    []map[string]any{{"product_id": "prod-low-stock", "quantity": 2}},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", mustJSON(t, body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "stock-key-001")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["error"] == "" {
		t.Fatal("expected error message in response")
	}
}
