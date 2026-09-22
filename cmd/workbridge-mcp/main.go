package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/bridge"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func main() {
	var (
		configPath = flag.String("config", "", "path to WORKBRIDGE_CONFIG_V1 JSON")
		transport = flag.String("transport", "stdio", "MCP transport: stdio or http")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()
	if *showVersion {
		fmt.Println(bridge.Version)
		return
	}
	if *configPath == "" {
		*configPath = os.Getenv("WORKBRIDGE_CONFIG")
	}
	if *configPath == "" {
		log.Fatal("config path is required via --config or WORKBRIDGE_CONFIG")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	rt, err := bridge.New(cfg)
	if err != nil {
		log.Fatalf("initialize WorkBridge: %v", err)
	}
	defer rt.Close()

	switch strings.ToLower(*transport) {
	case "stdio":
		if err := rt.Server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
	case "http":
		runHTTP(rt)
	default:
		log.Fatalf("unsupported transport %q; use stdio or http", *transport)
	}
}

func runHTTP(rt *bridge.Runtime) {
	cfg := rt.Config
	token := ""
	if cfg.HTTP.BearerTokenEnv != "" {
		token = os.Getenv(cfg.HTTP.BearerTokenEnv)
		if token == "" {
			log.Fatalf("HTTP bearer token environment variable %s is empty", cfg.HTTP.BearerTokenEnv)
		}
	}
	mcpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return rt.Server
	}, nil)
	mux := http.NewServeMux()
	mux.Handle(cfg.HTTP.Path, withBearer(token, mcpHandler))
	healthPath := cfg.HTTP.Path + "/healthz"
	mux.Handle(healthPath, withBearer(token, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"version": bridge.Version,
		})
	})))

	server := &http.Server{
		Addr: cfg.HTTP.Listen,
		Handler: mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	log.Printf("WorkBridge MCP listening on http://%s%s", cfg.HTTP.Listen, cfg.HTTP.Path)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func withBearer(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	expected := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if len(got) != len(expected) || subtle.ConstantTimeCompare(got, expected) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
