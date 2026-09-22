package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thebrazenbeard/WorkBridgeMCP/internal/config"
)

func TestRequireBearerToken(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := requireBearerToken(token, next)

	for _, tc := range []struct {
		name   string
		header string
		want   int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "wrong", header: "Bearer nope", want: http.StatusUnauthorized},
		{name: "correct", header: "Bearer " + token, want: http.StatusNoContent},
		{name: "case-insensitive-scheme", header: "bearer " + token, want: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/mcp", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%q", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestLoadHTTPBearerTokenRequiresEntropy(t *testing.T) {
	cfg := &config.Config{HTTP: config.HTTPConfig{BearerTokenEnv: "WORKBRIDGE_TEST_TOKEN"}}
	t.Setenv("WORKBRIDGE_TEST_TOKEN", "short")
	if _, err := loadHTTPBearerToken(cfg); err == nil || !strings.Contains(err.Error(), "32") {
		t.Fatalf("short token accepted: %v", err)
	}
	want := "0123456789abcdef0123456789abcdef"
	t.Setenv("WORKBRIDGE_TEST_TOKEN", want)
	got, err := loadHTTPBearerToken(cfg)
	if err != nil || got != want {
		t.Fatalf("token load got=%q err=%v", got, err)
	}
}
