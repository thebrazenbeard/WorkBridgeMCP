package main

import (
	"context"
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
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)
	mux := http.NewServeMux()
	mux.Handle(cfg.HTTP.Path, exactPath(cfg.HTTP.Path, handler))
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

func exactPath(expected string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expected {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
