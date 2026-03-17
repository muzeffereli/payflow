package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"payment-platform/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

type cachedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`
}

type bodyWriter struct {
	gin.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyWriter) Status() int {
	if w.status == 0 {
		return w.ResponseWriter.Status()
	}
	return w.status
}

func Idempotency(cfg config.RedisConfig) gin.HandlerFunc {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
	})

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		redisKey := "idempotency:" + c.FullPath() + ":" + key

		cached, err := rdb.Get(context.Background(), redisKey).Bytes()
		if err == nil {
			var resp cachedResponse
			if json.Unmarshal(cached, &resp) == nil {
				for k, v := range resp.Headers {
					c.Header(k, v)
				}
				c.Header("X-Idempotency-Replayed", "true")
				c.Data(resp.Status, "application/json", resp.Body)
				c.Abort()
				return
			}
		}

		bw := &bodyWriter{ResponseWriter: c.Writer}
		c.Writer = bw

		c.Next()

		status := bw.Status()
		if status < 200 || status >= 300 {
			return
		}

		headers := map[string]string{
			"Content-Type": c.Writer.Header().Get("Content-Type"),
		}

		resp := cachedResponse{
			Status:  status,
			Headers: headers,
			Body:    bw.buf.Bytes(),
		}

		if data, err := json.Marshal(resp); err == nil {
			rdb.Set(context.Background(), redisKey, data, idempotencyTTL)
		}
	}
}
