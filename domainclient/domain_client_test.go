package domainclient

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerifyDomainRetriesWithSameIdempotencyKey(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer authorization")
		}
		if r.Header.Get("Idempotency-Key") != "media.example:domain-onboard" {
			t.Fatal("idempotency key changed")
		}
		body, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(body)) != `{"domain":"media.example","idempotency_key":"media.example:domain-onboard"}` {
			t.Fatalf("body = %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			io.WriteString(w, `{"ok":false,"error":{"message":"retry later"}}`)
			return
		}
		io.WriteString(w, `{"ok":true,"data":{"verification":{"status":"pending"}},"metadata":{}}`)
	}))
	defer server.Close()

	client, err := New("test-key")
	if err != nil {
		t.Fatal(err)
	}
	client.baseURL = server.URL
	client.httpClient = server.Client()
	client.sleep = func(context.Context, time.Duration) error { return nil }

	domain, err := client.VerifyDomain(context.Background(), "media.example", "media.example:domain-onboard")
	if err != nil {
		t.Fatal(err)
	}
	if domain.Verification.Status != "pending" || calls != 2 {
		t.Fatalf("status = %q, calls = %d", domain.Verification.Status, calls)
	}
}
