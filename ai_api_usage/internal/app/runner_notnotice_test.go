package app

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubGeminiReporter struct{}

func (stubGeminiReporter) MonthlyUsage(context.Context) (GeminiUsage, error) {
	return GeminiUsage{Rows: []GeminiUsageRow{{Currency: "USD", UsageUnit: "seconds", UsageAmount: 1, Cost: 2}}}, nil
}

type failingGeminiReporter struct{}

func (failingGeminiReporter) MonthlyUsage(context.Context) (GeminiUsage, error) {
	return GeminiUsage{}, errors.New("no gcloud token")
}

type trackingGeminiReporter struct {
	called *bool
}

func (r trackingGeminiReporter) MonthlyUsage(context.Context) (GeminiUsage, error) {
	*r.called = true
	return GeminiUsage{}, nil
}

func TestRunnerNotNoticeSkipsBark(t *testing.T) {
	barkCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/balance":
			_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[{"currency":"USD","total_balance":"10"}]}`))
		case "/users/octo/settings/billing/premium_request/usage":
			_, _ = w.Write([]byte(`{"usageItems":[{"product":"Copilot","sku":"Copilot Premium Request","netQuantity":3}]}`))
		case "/push":
			barkCalled = true
			_, _ = w.Write([]byte(`{"code":200}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	runner := Runner{
		Config: Config{
			DeepSeekAPIKey:      "deepseek",
			GitHubToken:         "github",
			GitHubUsername:      "octo",
			CopilotPremiumLimit: 300,
			BarkKey:             "bark",
			StatePath:           filepath.Join(t.TempDir(), "state.jsonl"),
		},
		GeminiReporter: stubGeminiReporter{},
		DeepSeekClient: NewDeepSeekClient(server.Client(), server.URL),
		GitHubClient:   NewGitHubClient(server.Client(), server.URL),
		BarkClient:     NewBarkClient(server.Client(), server.URL),
		Clock:          func() time.Time { return time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC) },
		NotNotice:      true,
		Output:         &out,
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if barkCalled {
		t.Fatal("expected Bark not to be called")
	}
	if got := out.String(); !strings.Contains(got, "Bark notification skipped") || !strings.Contains(got, "PremiumReqs 3 / 300 (Usage: 1.0%)") {
		t.Fatalf("unexpected output:\n%s", got)
	}
}

func TestRunnerWithoutAllSkipsGeminiReporter(t *testing.T) {
	geminiCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/balance":
			_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[{"currency":"USD","total_balance":"10"}]}`))
		case "/users/octo/settings/billing/premium_request/usage":
			_, _ = w.Write([]byte(`{"usageItems":[{"product":"Copilot","sku":"Copilot Premium Request","netQuantity":3}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	runner := Runner{
		Config: Config{
			DeepSeekAPIKey:      "deepseek",
			GitHubToken:         "github",
			GitHubUsername:      "octo",
			CopilotPremiumLimit: 300,
			BarkKey:             "bark",
			StatePath:           filepath.Join(t.TempDir(), "state.jsonl"),
		},
		GeminiReporter: trackingGeminiReporter{called: &geminiCalled},
		DeepSeekClient: NewDeepSeekClient(server.Client(), server.URL),
		GitHubClient:   NewGitHubClient(server.Client(), server.URL),
		BarkClient:     NewBarkClient(server.Client(), server.URL),
		Clock:          func() time.Time { return time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC) },
		NotNotice:      true,
		Output:         &out,
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if geminiCalled {
		t.Fatal("expected Gemini reporter to be skipped without --all")
	}
	if got := out.String(); strings.Contains(got, "Gemini:") {
		t.Fatalf("expected Gemini to be absent from output:\n%s", got)
	}
}

func TestRunnerContinuesWhenGeminiFails(t *testing.T) {
	var deepseekCalled bool
	var githubCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/balance":
			deepseekCalled = true
			_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[{"currency":"USD","total_balance":"10"}]}`))
		case "/users/octo/settings/billing/premium_request/usage":
			githubCalled = true
			_, _ = w.Write([]byte(`{"usageItems":[{"product":"Copilot","sku":"Copilot Premium Request","netQuantity":3}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out bytes.Buffer
	runner := Runner{
		Config: Config{
			DeepSeekAPIKey:      "deepseek",
			GitHubToken:         "github",
			GitHubUsername:      "octo",
			CopilotPremiumLimit: 300,
			BarkKey:             "bark",
			StatePath:           filepath.Join(t.TempDir(), "state.jsonl"),
		},
		GeminiReporter: failingGeminiReporter{},
		DeepSeekClient: NewDeepSeekClient(server.Client(), server.URL),
		GitHubClient:   NewGitHubClient(server.Client(), server.URL),
		BarkClient:     NewBarkClient(server.Client(), server.URL),
		Clock:          func() time.Time { return time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC) },
		NotNotice:      true,
		All:            true,
		Output:         &out,
	}

	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !deepseekCalled || !githubCalled {
		t.Fatalf("expected deepseek and github to continue, deepseek=%v github=%v", deepseekCalled, githubCalled)
	}
	if got := out.String(); !strings.Contains(got, "Gemini usage failed") || !strings.Contains(got, "PremiumReqs 3 / 300 (Usage: 1.0%)") {
		t.Fatalf("unexpected output:\n%s", got)
	}
}
