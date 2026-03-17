package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"payment-platform/internal/cart/port"
)

var _ port.OrderClient = (*httpOrderClient)(nil)

type httpOrderClient struct {
	base   string
	client *http.Client
}

func NewHTTPOrderClient(addr string) port.OrderClient {
	return &httpOrderClient{
		base:   strings.TrimRight(addr, "/"),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type orderRequest struct {
	Currency      string           `json:"currency"`
	PaymentMethod string           `json:"payment_method,omitempty"`
	Items         []orderItemInput `json:"items"`
	StoreID       *string          `json:"store_id,omitempty"`
}

type orderItemInput struct {
	ProductID string  `json:"product_id"`
	VariantID *string `json:"variant_id,omitempty"`
	Quantity  int     `json:"quantity"`
}

type orderResponse struct {
	ID string `json:"id"`
}

func (c *httpOrderClient) CreateOrder(ctx context.Context, req port.CreateOrderRequest) (string, error) {
	items := make([]orderItemInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = orderItemInput{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity}
	}

	body, err := json.Marshal(orderRequest{
		Currency:      req.Currency,
		PaymentMethod: req.PaymentMethod,
		Items:         items,
		StoreID:       req.StoreID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal order request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.base+"/api/v1/orders", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build order request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)
	httpReq.Header.Set("X-User-ID", req.UserID)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("order service request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("order service returned %d", resp.StatusCode)
	}

	var result orderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode order response: %w", err)
	}
	return result.ID, nil
}
