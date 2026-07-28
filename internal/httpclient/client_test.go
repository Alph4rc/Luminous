package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", 10*time.Second)
	body, err := client.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestGet_WithToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer secret-token" {
			t.Errorf("expected Authorization header 'Bearer secret-token', got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "secret-token", 10*time.Second)
	_, err := client.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet_WithoutToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", 10*time.Second)
	_, err := client.Get(context.Background(), "/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found","details":"sensitive data"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", 10*time.Second)
	_, err := client.Get(context.Background(), "/test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("error should mention HTTP 404, got: %v", err)
	}
	// Sensitive response body must NOT leak into the error message.
	if strings.Contains(err.Error(), "sensitive data") {
		t.Error("error message should NOT contain response body (sensitive data leak)")
	}
}

func TestGet_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", 10*time.Second)
	_, err := client.Get(context.Background(), "/test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("error should mention HTTP 500, got: %v", err)
	}
}

func TestGet_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", 100*time.Millisecond)
	_, err := client.Get(context.Background(), "/test")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestGet_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "", 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.
	_, err := client.Get(ctx, "/test")
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
}

func TestHasToken(t *testing.T) {
	if NewClient("http://x", "", 1).HasToken() {
		t.Error("HasToken should be false when token is empty")
	}
	if !NewClient("http://x", "t", 1).HasToken() {
		t.Error("HasToken should be true when token is set")
	}
}

func TestBaseURL(t *testing.T) {
	c := NewClient("http://example.com/api/", "", 1)
	if c.BaseURL() != "http://example.com/api" {
		t.Errorf("trailing slash should be trimmed, got %q", c.BaseURL())
	}
}
