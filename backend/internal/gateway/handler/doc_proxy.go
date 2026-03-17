package handler

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type DocProxy struct {
	proxy *httputil.ReverseProxy
}

func NewDocProxy(targetAddr, pathPrefix string) (*DocProxy, error) {
	u, err := url.Parse(targetAddr)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	origDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		origDirector(req) // sets host + scheme

		tail := strings.TrimPrefix(req.URL.Path, pathPrefix)
		if tail == "" || tail == "/" {
			tail = "/index.html"
		}
		req.URL.Path = "/swagger" + tail
		req.URL.RawPath = "" // clear so http.Request re-encodes cleanly
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"swagger service unavailable"}`))
	}

	return &DocProxy{proxy: proxy}, nil
}

func (d *DocProxy) Forward(c *gin.Context) {
	d.proxy.ServeHTTP(c.Writer, c.Request)
}
