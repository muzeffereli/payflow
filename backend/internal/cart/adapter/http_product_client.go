package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"payment-platform/internal/cart/port"
)

var _ port.ProductClient = (*httpProductClient)(nil)

type httpProductClient struct {
	base   string
	client *http.Client
}

func NewHTTPProductClient(addr string) port.ProductClient {
	return &httpProductClient{
		base:   strings.TrimRight(addr, "/"),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type listProductsResponse struct {
	Products []productInfoResponse `json:"products"`
}

type productInfoResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Price    int64             `json:"price"`
	Currency string            `json:"currency"`
	Stock    int               `json:"stock"`
	Status   string            `json:"status"`
	StoreID  *string           `json:"store_id,omitempty"`
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

func (c *httpProductClient) GetProducts(ctx context.Context, ids []string) ([]port.ProductInfo, error) {
	url := c.base + "/api/v1/products?ids=" + strings.Join(ids, ",")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("product service request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product service returned %d", resp.StatusCode)
	}

	var result listProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode product response: %w", err)
	}

	out := make([]port.ProductInfo, len(result.Products))
	for i, p := range result.Products {
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
		out[i] = port.ProductInfo{
			ID:       p.ID,
			Name:     p.Name,
			Price:    p.Price,
			Currency: p.Currency,
			Stock:    p.Stock,
			Status:   p.Status,
			StoreID:  p.StoreID,
			Variants: variants,
		}
	}
	return out, nil
}
