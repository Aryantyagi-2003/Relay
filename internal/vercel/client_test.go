package vercel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchEnv_ParsesAndFiltersByTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("unexpected Authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"envs":[
			{"key":"DATABASE_URL","value":"postgres://prod","type":"encrypted","target":["production"]},
			{"key":"DATABASE_URL","value":"postgres://preview","type":"encrypted","target":["preview"]},
			{"key":"API_SECRET","value":"","type":"sensitive","sensitive":true,"target":["production"]},
			{"key":"DEBUG","value":"","type":"plain","target":["production"]}
		]}`))
	}))
	defer srv.Close()

	c, err := NewClient("test-token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.BaseURL = srv.URL

	values, redacted, err := c.FetchEnv(context.Background(), "my-project", "production")
	if err != nil {
		t.Fatalf("FetchEnv: %v", err)
	}

	if values["DATABASE_URL"] != "postgres://prod" {
		t.Errorf("expected production DATABASE_URL, got %q", values["DATABASE_URL"])
	}
	if _, ok := values["API_SECRET"]; ok {
		t.Error("expected sensitive API_SECRET to be excluded from values")
	}
	if len(redacted) != 1 || redacted[0] != "API_SECRET" {
		t.Errorf("expected API_SECRET in redacted list, got %v", redacted)
	}
	if got, ok := values["DEBUG"]; !ok || got != "" {
		t.Errorf("expected genuinely empty plain value DEBUG=\"\" to be kept, got %q ok=%v", got, ok)
	}
}

func TestFetchEnv_NonOKStatusReturnsClearError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":{"message":"Not authorized"}}`))
	}))
	defer srv.Close()

	c, err := NewClient("bad-token")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.BaseURL = srv.URL

	_, _, err = c.FetchEnv(context.Background(), "my-project", "production")
	if err == nil {
		t.Fatal("expected error for 403 response, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "Not authorized") || !strings.Contains(got, "403") {
		t.Errorf("expected error to mention status and API message, got: %s", got)
	}
}

func TestNewClient_RejectsEmptyToken(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}
