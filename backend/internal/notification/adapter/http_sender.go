package adapter

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
)

type HTTPSender struct {
	client *http.Client
}

func NewHTTPSender() *HTTPSender {
	return &HTTPSender{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *HTTPSender) SendEmail(_ context.Context, to, subject, body string) error {

	_ = to
	_ = subject
	_ = body
	return nil
}

func (s *HTTPSender) SendWebhook(ctx context.Context, url string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}
