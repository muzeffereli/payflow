package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func main() {
	gatewayAddr := flag.String("gateway", envOrDefault("GATEWAY_ADDR", "http://localhost:8080"), "API gateway address")
	port := flag.String("port", envOrDefault("PORT", "8090"), "server port")
	staticDir := flag.String("static", envOrDefault("STATIC_DIR", "../frontend/dist"), "frontend build directory")
	flag.Parse()

	gatewayURL, err := url.Parse(*gatewayAddr)
	if err != nil {
		log.Fatalf("invalid gateway address: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(gatewayURL)

	orig := proxy.Director
	proxy.Director = func(req *http.Request) {
		orig(req)
		req.Host = gatewayURL.Host
	}

	mux := http.NewServeMux()

	proxyHandler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}))
	mux.Handle("/api/", proxyHandler)
	mux.Handle("/auth/", proxyHandler)
	mux.Handle("/healthz", proxyHandler)

	absStatic, _ := filepath.Abs(*staticDir)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(absStatic, filepath.Clean("/"+r.URL.Path))

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.ServeFile(w, r, path)
			return
		}

		http.ServeFile(w, r, filepath.Join(absStatic, "index.html"))
	})

	addr := ":" + *port
	log.Printf("BFF server listening on %s â€” proxying to %s â€” static: %s", addr, *gatewayAddr, absStatic)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
