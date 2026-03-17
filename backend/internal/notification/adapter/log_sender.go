package adapter

import (
	"context"
	"log/slog"
)

type LogSender struct {
	log *slog.Logger
}

func NewLogSender(log *slog.Logger) *LogSender {
	return &LogSender{log: log}
}

func (s *LogSender) SendEmail(_ context.Context, to, subject, body string) error {
	s.log.Info("ðŸ“§ [DEV] sending email",
		"to", to,
		"subject", subject,
		"body", body,
	)
	return nil
}

func (s *LogSender) SendWebhook(_ context.Context, url string, payload []byte) error {
	s.log.Info("ðŸ”” [DEV] sending webhook",
		"url", url,
		"payload", string(payload),
	)
	return nil
}
