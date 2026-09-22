package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/bridge"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
	"github.com/thebrazenbeard/WorkBridgeMCP/internal/mcpserver"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
)

func main() {
	log.SetOutput(os.Stderr)
	configPath := flag.String("config", "", "path to WorkBridge config JSON")
	transport := flag.String("transport", "stdio", "transport: stdio or http")
	checkConfig := flag.Bool("check-config", false, "validate configuration and exit")
	version := flag.Bool("version", false, "print machine-readable build identity")
	flag.Parse()

	if *version {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{
			"schema":  "WORKBRIDGE_BUILD_V1",
			"version": buildVersion,
			"commit":  buildCommit,
			"goos":    runtime.GOOS,
			"goarch":  runtime.GOARCH,
		})
		return
	}
	if strings.TrimSpace(*configPath) == "" {
		log.Fatal("-config is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	b, err := bridge.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if *checkConfig {
		fmt.Fprintln(os.Stdout, "WORKBRIDGE_CONFIG_OK")
		return
	}
	server := mcpserver.New(b, buildVersion)

	switch *transport {
	case "stdio":
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
	case "http":
		if err := runHTTP(server, cfg); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unsupported transport %q", *transport)
	}
}

func runHTTP(server *mcp.Server, cfg *config.Config) error {
	token, err := loadHTTPBearerToken(cfg)
	if err != nil {
		return err
	}
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)
	protected := requireBearerToken(token, handler)
	mux := http.NewServeMux()
	mux.Handle(cfg.HTTP.Path, exactPath(cfg.HTTP.Path, protected))
	httpServer := &http.Server{
		Addr:              cfg.HTTP.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.ListenAndServe() }()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	}
}

func loadHTTPBearerToken(cfg *config.Config) (string, error) {
	name := cfg.HTTP.BearerTokenEnv
	token, ok := os.LookupEnv(name)
	if !ok || token == "" {
		return "", fmt.Errorf("HTTP bearer token environment variable %s is not set", name)
	}
	if strings.IndexFunc(token, unicode.IsSpace) >= 0 {
		return "", errors.New("HTTP bearer token must not contain whitespace")
	}
	if len(token) < 32 {
		return "", errors.New("HTTP bearer token must be at least 32 bytes")
	}
	return token, nil
}

func requireBearerToken(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.SplitN(strings.TrimSpace(r.Header.Get("Authorization")), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") ||
			len(parts[1]) != len(expected) ||
			subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="WorkBridgeMCP"`)
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, "bearer authorization required", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func exactPath(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expected {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
