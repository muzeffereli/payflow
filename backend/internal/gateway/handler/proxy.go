package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

type ServiceProxy struct {
	proxy *httputil.ReverseProxy
}

func NewServiceProxy(target string) (*ServiceProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"service unavailable"}`))
	}

	return &ServiceProxy{proxy: proxy}, nil
}

func (p *ServiceProxy) Forward(c *gin.Context) {
	if userID := c.GetString("user_id"); userID != "" {
		c.Request.Header.Set("X-User-ID", userID)
	}
	if role := c.GetString("user_role"); role != "" {
		c.Request.Header.Set("X-User-Role", role)
	}

	if reqID := c.GetString("request_id"); reqID != "" {
		c.Request.Header.Set("X-Request-ID", reqID)
	}

	p.proxy.ServeHTTP(c.Writer, c.Request)
}
