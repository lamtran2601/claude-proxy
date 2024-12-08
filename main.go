package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	apiKeys       []string
	currentAPIKey = 0
	mu            sync.Mutex
	connPool      *ConnectionPool
)

func init() {
	apiKeysEnv := os.Getenv("API_KEYS")
	if apiKeysEnv == "" {
		log.Fatalf("API_KEYS environment variable not set")
	}
	apiKeys = strings.Split(apiKeysEnv, ",")
	if len(apiKeys) == 0 {
		log.Fatalf("No API keys found in API_KEYS environment variable")
	}

	// Create a connection pool
	connPool = NewConnectionPool(10, 50, 190*time.Second)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting proxy server on :%s", port)
	server := &http.Server{
		Addr:         ":" + port,
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      http.HandlerFunc(proxyHandler),
	}
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start proxy server: %v", err)
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	var resp *http.Response
	var err error

	for i := 0; i < len(apiKeys); i++ {
		mu.Lock()
		apiKey := apiKeys[currentAPIKey]
		mu.Unlock()

		var req *http.Request
		req, err = http.NewRequest(r.Method, "https://api.anthropic.com"+r.URL.Path, r.Body)
		if err != nil {
			log.Printf("Failed to create request: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
			return
		}

		req.Header = r.Header
		req.Header.Set("x-api-key", apiKey)

		resp, err = connPool.Do(req)
		if err != nil {
			log.Printf("Failed to make request with API key at index %d: %v", currentAPIKey, err)
			http.Error(w, fmt.Sprintf("Failed to make request with API key: %v", err), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			rotateAPIKey()
			log.Printf("Received 'Too Many Requests' response, rotating API key")
			continue
		}

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %v", err)
			http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
			return
		}
		return
	}

	log.Printf("Exhausted all API keys, unable to make successful request")
	http.Error(w, "Unable to make successful request", http.StatusInternalServerError)
}

func rotateAPIKey() {
	mu.Lock()
	defer mu.Unlock()
	prevAPIKeyIndex := currentAPIKey
	currentAPIKey = (currentAPIKey + 1) % len(apiKeys)
	log.Printf("Rotated API key from index %d to index %d", prevAPIKeyIndex, currentAPIKey)
}

type ConnectionPool struct {
	maxIdle     int
	maxActive   int
	idleTimeout time.Duration
	mu          sync.Mutex
	conns       chan *http.Client
}

func NewConnectionPool(maxIdle, maxActive int, idleTimeout time.Duration) *ConnectionPool {
	return &ConnectionPool{
		maxIdle:     maxIdle,
		maxActive:   maxActive,
		idleTimeout: idleTimeout,
		conns:       make(chan *http.Client, maxActive),
	}
}

func (p *ConnectionPool) Get() (*http.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	select {
	case conn := <-p.conns:
		return conn, nil
	default:
		if len(p.conns) >= p.maxActive {
			return nil, fmt.Errorf("connection pool exhausted")
		}
		client := &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        p.maxIdle,
				MaxIdleConnsPerHost: p.maxIdle,
				IdleConnTimeout:     p.idleTimeout,
			},
			Timeout: 200 * time.Second,
		}
		return client, nil
	}
}

func (p *ConnectionPool) Put(conn *http.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns) >= p.maxActive {
		return
	}
	p.conns <- conn
}

func (p *ConnectionPool) Do(req *http.Request) (*http.Response, error) {
	conn, err := p.Get()
	if err != nil {
		return nil, err
	}
	resp, err := conn.Do(req)
	if err != nil {
		return nil, err
	}
	p.Put(conn)
	return resp, nil
}
