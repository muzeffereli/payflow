package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"payment-platform/internal/product/port"
)

var _ port.StoreClient = (*httpStoreClient)(nil)

type httpStoreClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPStoreClient(storeServiceAddr string) port.StoreClient {
	return &httpStoreClient{
		baseURL: storeServiceAddr,
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

type storeResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (c *httpStoreClient) GetStoreIDByOwner(ctx context.Context, ownerID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/stores/me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-User-ID", ownerID)
	req.Header.Set("X-User-Role", "seller")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("store-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // seller has no store yet
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("store-service returned %d", resp.StatusCode)
	}

	var s storeResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", fmt.Errorf("decode store response: %w", err)
	}
	return s.ID, nil
}
