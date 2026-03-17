package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"service", "method", "path", "status"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method", "path"})

	natsPublishTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_publish_total",
		Help: "Total NATS events published.",
	}, []string{"service", "subject", "status"})

	natsConsumeTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "nats_consume_total",
		Help: "Total NATS events consumed.",
	}, []string{"service", "subject", "status"})

	natsConsumeDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nats_consume_duration_seconds",
		Help:    "NATS event handler latency.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "subject"})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		natsPublishTotal,
		natsConsumeTotal,
		natsConsumeDuration,
	)
}

func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func HTTPMiddleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath() // normalised path with param placeholders, e.g. /api/v1/orders/:id
		if path == "" {
			path = "unknown"
		}

		duration := time.Since(start).Seconds()
		httpRequestsTotal.WithLabelValues(service, c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(service, c.Request.Method, path).Observe(duration)
	}
}

func RecordPublish(service, subject string, err error) {
	status := "ok"
	if err != nil {
		status = "error"
	}
	natsPublishTotal.WithLabelValues(service, subject, status).Inc()
}

func RecordConsume(service, subject string, duration time.Duration, err error) {
	status := "ok"
	if err != nil {
		status = "error"
	}
	natsConsumeTotal.WithLabelValues(service, subject, status).Inc()
	natsConsumeDuration.WithLabelValues(service, subject).Observe(duration.Seconds())
}
