package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeepSeekBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/balance" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[{"currency":"USD","total_balance":"12.3400"}]}`))
	}))
	defer server.Close()

	client := NewDeepSeekClient(server.Client(), server.URL)
	balance, err := client.Balance(context.Background(), "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if !balance.IsAvailable || len(balance.BalanceInfo) != 1 || balance.BalanceInfo[0].TotalBalance != "12.3400" {
		t.Fatalf("unexpected balance %#v", balance)
	}
}

func TestGitHubPremiumRequestUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/octo/settings/billing/premium_request/usage" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("year") != "2026" || r.URL.Query().Get("month") != "5" {
			t.Fatalf("unexpected query %s", r.URL.RawQuery)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer gh-token" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"usageItems": [
				{"product":"Copilot","sku":"Copilot Premium Request","grossQuantity":10,"netQuantity":0},
				{"product":"Copilot","sku":"Other","netQuantity":99},
				{"product":"Actions","sku":"Copilot Premium Request","netQuantity":99}
			]
		}`))
	}))
	defer server.Close()

	client := NewGitHubClient(server.Client(), server.URL)
	usage, err := client.PremiumRequestUsage(context.Background(), "gh-token", "octo", time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC), 300)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Used != 10 || usage.Limit != 300 {
		t.Fatalf("unexpected usage %#v", usage)
	}
}

func TestPremiumRequestsConsumedPrefersGrossQuantity(t *testing.T) {
	got := premiumRequestsConsumed(CopilotUsageItem{GrossQuantity: 72, DiscountQuantity: 72, NetQuantity: 0})
	if got != 72 {
		t.Fatalf("expected gross quantity, got %v", got)
	}
}

func TestBarkPush(t *testing.T) {
	var payload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/push" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200}`))
	}))
	defer server.Close()

	client := NewBarkClient(server.Client(), server.URL)
	if err := client.Push(context.Background(), "key", "AI Usage Daily", "body"); err != nil {
		t.Fatal(err)
	}
	if payload["device_key"] != "key" || payload["title"] != "AI Usage Daily" || payload["body"] != "body" {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestDoJSONReturnsHTTPErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad token", http.StatusUnauthorized)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = doJSON(server.Client(), req, nil)
	if err == nil || !strings.Contains(err.Error(), "bad token") {
		t.Fatalf("expected response body in error, got %v", err)
	}
}
