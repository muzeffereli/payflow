package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"payment-platform/internal/payment/port"
)

var _ port.StoreClient = (*httpStoreClient)(nil)

type httpStoreClient struct {
	base   string
	client *http.Client
}

func NewHTTPStoreClient(storeServiceAddr string) port.StoreClient {
	return &httpStoreClient{
		base:   strings.TrimRight(storeServiceAddr, "/"),
		client: &http.Client{Timeout: 3 * time.Second},
	}
}

type storeOwnerResponse struct {
	ID         string `json:"id"`
	OwnerID    string `json:"owner_id"`
	Commission int    `json:"commission"`
	Status     string `json:"status"`
}

func (c *httpStoreClient) GetStoreOwner(ctx context.Context, storeID string) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.base+"/api/v1/stores/"+storeID, nil)
	if err != nil {
		return "", 0, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("store-service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", 0, nil // store not found â€” treat as platform order
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("store-service returned %d", resp.StatusCode)
	}

	var s storeOwnerResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", 0, fmt.Errorf("decode store response: %w", err)
	}
	return s.OwnerID, s.Commission, nil
}
