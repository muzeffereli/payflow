package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sony/gobreaker"

	"payment-platform/internal/order/port"
)

var _ port.ProductClient = (*httpProductClient)(nil)

type httpProductClient struct {
	baseURL string
	client  *http.Client
	cb      *gobreaker.CircuitBreaker
}

func NewHTTPProductClient(baseURL string) port.ProductClient {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "product-service",
		MaxRequests: 3,                // half-open: allow 3 probes before closing again
		Interval:    60 * time.Second, // rolling window for counting failures
		Timeout:     30 * time.Second, // open â†’ half-open after 30 s
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	})
	return &httpProductClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Second},
		cb:      cb,
	}
}

type productResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Price    int64             `json:"price"`
	Currency string            `json:"currency"`
	Stock    int               `json:"stock"`
	Status   string            `json:"status"`
	Variants []variantResponse `json:"variants,omitempty"`
}

type variantResponse struct {
	ID              string            `json:"id"`
	SKU             string            `json:"sku"`
	Price           *int64            `json:"price"`
	Stock           int               `json:"stock"`
	Status          string            `json:"status"`
	AttributeValues map[string]string `json:"attribute_values"`
}

type listProductsResponse struct {
	Products []productResponse `json:"products"`
}

func (c *httpProductClient) GetProducts(ctx context.Context, ids []string) ([]port.ProductInfo, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.doGetProducts(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	return result.([]port.ProductInfo), nil
}

func (c *httpProductClient) doGetProducts(ctx context.Context, ids []string) ([]port.ProductInfo, error) {
	url := fmt.Sprintf("%s/api/v1/products?ids=%s&limit=100",
		c.baseURL, strings.Join(ids, ","))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call product service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product service returned %d", resp.StatusCode)
	}

	var body listProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode product response: %w", err)
	}

	result := make([]port.ProductInfo, len(body.Products))
	for i, p := range body.Products {
		variants := make([]port.VariantInfo, len(p.Variants))
		for j, variant := range p.Variants {
			variants[j] = port.VariantInfo{
				ID:              variant.ID,
				SKU:             variant.SKU,
				Price:           variant.Price,
				Stock:           variant.Stock,
				Status:          variant.Status,
				AttributeValues: variant.AttributeValues,
			}
		}
		result[i] = port.ProductInfo{
			ID:       p.ID,
			Name:     p.Name,
			Price:    p.Price,
			Currency: p.Currency,
			Stock:    p.Stock,
			Status:   p.Status,
			Variants: variants,
		}
	}
	return result, nil
}
