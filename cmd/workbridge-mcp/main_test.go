package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func TestBearerMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := withBearer("secret", next)
	for _, tc := range []struct {
		name string
		header string
		want int
	}{
		{"missing", "", http.StatusUnauthorized},
		{"wrong", "Bearer nope", http.StatusUnauthorized},
		{"correct", "Bearer secret", http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d want=%d", rec.Code, tc.want)
			}
		})
	}
}

func TestLoadHTTPBearerToken(t *testing.T) {
	cfg := &config.Config{HTTP: config.HTTPConfig{BearerTokenEnv: "WORKBRIDGE_TEST_TOKEN"}}
	t.Setenv("WORKBRIDGE_TEST_TOKEN", "short")
	if _, err := loadHTTPBearerToken(cfg); err == nil {
		t.Fatal("short bearer token accepted")
	}
	t.Setenv("WORKBRIDGE_TEST_TOKEN", "0123456789abcdef0123456789abcdef")
	got, err := loadHTTPBearerToken(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected token %q", got)
	}
	t.Setenv("WORKBRIDGE_TEST_TOKEN", "0123456789abcdef0123456789abc def")
	if _, err := loadHTTPBearerToken(cfg); err == nil {
		t.Fatal("whitespace-bearing token accepted")
	}
}
