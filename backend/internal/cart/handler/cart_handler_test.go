package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"payment-platform/internal/cart/domain"
	"payment-platform/internal/cart/handler"
	"payment-platform/internal/cart/port"
	"payment-platform/internal/cart/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type fakeCartRepo struct {
	carts map[string]*domain.Cart
}

func newFakeCartRepo() *fakeCartRepo {
	return &fakeCartRepo{carts: make(map[string]*domain.Cart)}
}

func (r *fakeCartRepo) Get(_ context.Context, userID string) (*domain.Cart, error) {
	if c, ok := r.carts[userID]; ok {
		return c, nil
	}
	return domain.New(userID), nil
}

func (r *fakeCartRepo) Save(_ context.Context, cart *domain.Cart) error {
	r.carts[cart.UserID] = cart
	return nil
}

func (r *fakeCartRepo) Delete(_ context.Context, userID string) error {
	delete(r.carts, userID)
	return nil
}

type fakeProductClient struct {
	products map[string]port.ProductInfo
}

func newFakeProducts(products ...port.ProductInfo) *fakeProductClient {
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

type fakeOrderClient struct {
	orderID string
	err     error
}

func (c *fakeOrderClient) CreateOrder(_ context.Context, _ port.CreateOrderRequest) (string, error) {
	return c.orderID, c.err
}

func buildCartRouter(t *testing.T, products *fakeProductClient, orders *fakeOrderClient) (*gin.Engine, *fakeCartRepo) {
	t.Helper()
	repo := newFakeCartRepo()
	svc := service.New(repo, products, orders, slog.Default())
	h := handler.New(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", "user-abc")
		c.Next()
	})
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1.Group("/cart"))
	return r, repo
}

func mustMarshal(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func TestAddItem_HappyPath(t *testing.T) {
	products := newFakeProducts(port.ProductInfo{
		ID: "prod-1", Name: "Widget", Price: 1500, Currency: "USD", Stock: 5, Status: "active",
	})
	r, _ := buildCartRouter(t, products, &fakeOrderClient{})

	body := map[string]any{"product_id": "prod-1", "quantity": 2}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", mustMarshal(t, body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddItem_InvalidProduct(t *testing.T) {
	r, _ := buildCartRouter(t, newFakeProducts(), &fakeOrderClient{})

	body := map[string]any{"product_id": "nonexistent", "quantity": 1}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", mustMarshal(t, body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAddItem_MissingBody(t *testing.T) {
	r, _ := buildCartRouter(t, newFakeProducts(), &fakeOrderClient{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/items", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRemoveItem_NotInCart(t *testing.T) {
	r, _ := buildCartRouter(t, newFakeProducts(), &fakeOrderClient{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/cart/items/prod-x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetCart_Empty(t *testing.T) {
	r, _ := buildCartRouter(t, newFakeProducts(), &fakeOrderClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	items, ok := result["items"].([]any)
	if !ok || len(items) != 0 {
		t.Error("expected empty items array")
	}
}

func TestCheckout_EmptyCart(t *testing.T) {
	r, _ := buildCartRouter(t, newFakeProducts(), &fakeOrderClient{orderID: "order-1"})

	body := map[string]any{"currency": "USD", "idempotency_key": "key-1"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/checkout", mustMarshal(t, body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCheckout_HappyPath(t *testing.T) {
	products := newFakeProducts(port.ProductInfo{
		ID: "prod-2", Name: "Gadget", Price: 999, Currency: "USD", Stock: 10, Status: "active",
	})
	r, repo := buildCartRouter(t, products, &fakeOrderClient{orderID: "order-xyz"})

	cart := domain.New("user-abc")
	cart.AddItem("prod-2", nil, 3)
	repo.Save(context.Background(), cart)

	body := map[string]any{"currency": "USD", "idempotency_key": "checkout-key"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/checkout", mustMarshal(t, body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	orderIDs, _ := result["order_ids"].([]any)
	if len(orderIDs) == 0 || orderIDs[0] != "order-xyz" {
		t.Errorf("expected order_ids=[order-xyz], got %v", result["order_ids"])
	}

	cartAfter, _ := repo.Get(context.Background(), "user-abc")
	if !cartAfter.IsEmpty() {
		t.Error("expected cart to be empty after checkout")
	}
}

func TestCheckout_OrderServiceError(t *testing.T) {
	products := newFakeProducts(port.ProductInfo{
		ID: "prod-3", Price: 500, Currency: "USD", Stock: 5, Status: "active",
	})
	orderClient := &fakeOrderClient{err: errors.New("order service unavailable")}
	r, repo := buildCartRouter(t, products, orderClient)

	cart := domain.New("user-abc")
	cart.AddItem("prod-3", nil, 1)
	repo.Save(context.Background(), cart)

	body := map[string]any{"currency": "USD"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/checkout", mustMarshal(t, body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
