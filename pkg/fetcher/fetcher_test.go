package fetcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetch_OK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}))
	defer ts.Close()

	body, err := Fetch(ts.URL, 5*time.Second)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if body != "hello world" {
		t.Errorf("body = %q, want hello world", body)
	}
}

func TestFetch_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := Fetch(ts.URL, 5*time.Second)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
