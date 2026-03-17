package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"payment-platform/internal/payment/port"
)

var _ port.WalletClient = (*httpWalletClient)(nil)

type httpWalletClient struct {
	base   string
	client *http.Client
}

type debitWalletRequest struct {
	Amount      int64  `json:"amount"`
	ReferenceID string `json:"reference_id"`
}

type debitWalletResponse struct {
	TransactionID string `json:"transaction_id"`
}

func NewHTTPWalletClient(walletServiceAddr string) port.WalletClient {
	return &httpWalletClient{
		base:   strings.TrimRight(walletServiceAddr, "/"),
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *httpWalletClient) Debit(ctx context.Context, userID string, amount int64, referenceID string) (string, error) {
	body, err := json.Marshal(debitWalletRequest{Amount: amount, ReferenceID: referenceID})
	if err != nil {
		return "", fmt.Errorf("marshal wallet debit request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/wallet/payments/debit", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build wallet debit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", userID)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("wallet service request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", fmt.Errorf("wallet not found")
	case http.StatusConflict:
		return "", fmt.Errorf("insufficient funds")
	default:
		return "", fmt.Errorf("wallet debit failed")
	}

	var result debitWalletResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode wallet debit response: %w", err)
	}
	if result.TransactionID == "" {
		return "", fmt.Errorf("wallet debit failed")
	}
	return result.TransactionID, nil
}
