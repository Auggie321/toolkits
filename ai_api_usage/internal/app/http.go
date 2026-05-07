package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DeepSeekClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewDeepSeekClient(httpClient *http.Client, baseURL string) DeepSeekClient {
	return DeepSeekClient{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

func (c DeepSeekClient) Balance(ctx context.Context, apiKey string) (DeepSeekBalance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/user/balance", nil)
	if err != nil {
		return DeepSeekBalance{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	var balance DeepSeekBalance
	if err := doJSON(c.httpClient, req, &balance); err != nil {
		return DeepSeekBalance{}, fmt.Errorf("deepseek balance: %w", err)
	}
	return balance, nil
}

type GitHubClient struct {
	httpClient  *http.Client
	baseURL     string
	webBaseURL  string
	DebugWriter io.Writer
}

func NewGitHubClient(httpClient *http.Client, baseURL string) GitHubClient {
	return GitHubClient{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/"), webBaseURL: "https://github.com"}
}

func NewGitHubClientWithWebBaseURL(httpClient *http.Client, baseURL, webBaseURL string) GitHubClient {
	return GitHubClient{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/"), webBaseURL: strings.TrimRight(webBaseURL, "/")}
}

func (c GitHubClient) CopilotUsage(ctx context.Context, cfg Config, now time.Time) (CopilotUsage, error) {
	if cfg.GitHubCopilotSource == "settings_page" || (cfg.GitHubCopilotSource == "auto" && cfg.GitHubSessionCookie != "") {
		usage, err := c.CopilotSettingsPageUsage(ctx, cfg.GitHubSessionCookie)
		if err == nil {
			return usage, nil
		}
		if cfg.GitHubCopilotSource == "settings_page" {
			return CopilotUsage{}, err
		}
	}
	return c.PremiumRequestUsage(ctx, cfg.GitHubToken, cfg.GitHubUsername, now, cfg.CopilotPremiumLimit)
}

func (c GitHubClient) PremiumRequestUsage(ctx context.Context, token, username string, now time.Time, limit float64) (CopilotUsage, error) {
	endpoint := fmt.Sprintf("%s/users/%s/settings/billing/premium_request/usage", c.baseURL, url.PathEscape(username))
	u, err := url.Parse(endpoint)
	if err != nil {
		return CopilotUsage{}, err
	}
	q := u.Query()
	q.Set("year", fmt.Sprintf("%d", now.Year()))
	q.Set("month", fmt.Sprintf("%d", int(now.Month())))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return CopilotUsage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")

	body, err := do(c.httpClient, req)
	if err != nil {
		return CopilotUsage{}, fmt.Errorf("github premium request usage: %w", err)
	}
	if c.DebugWriter != nil {
		fmt.Fprintf(c.DebugWriter, "\n--- GitHub billing API raw JSON ---\n%s\n--- end GitHub billing API raw JSON ---\n\n", strings.TrimSpace(string(body)))
	}

	var payload struct {
		UsageItems []CopilotUsageItem `json:"usageItems"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return CopilotUsage{}, fmt.Errorf("github premium request usage decode response: %w", err)
	}

	var used float64
	for _, item := range payload.UsageItems {
		if isPremiumRequestUsageItem(item) {
			used += premiumRequestsConsumed(item)
		}
	}
	percent := 0.0
	if limit > 0 {
		percent = used / limit * 100
	}
	return CopilotUsage{Used: used, Limit: limit, Percent: percent, Source: "github_billing_api", Items: payload.UsageItems}, nil
}

func (c GitHubClient) CopilotSettingsPageUsage(ctx context.Context, sessionCookie string) (CopilotUsage, error) {
	if strings.TrimSpace(sessionCookie) == "" {
		return CopilotUsage{}, fmt.Errorf("GITHUB_SESSION_COOKIE is required when GITHUB_COPILOT_SOURCE=settings_page")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.webBaseURL+"/settings/copilot", nil)
	if err != nil {
		return CopilotUsage{}, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Cookie", sessionCookie)
	req.Header.Set("User-Agent", "ai-usage-notifier")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return CopilotUsage{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return CopilotUsage{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CopilotUsage{}, fmt.Errorf("github settings page http %d: %s", resp.StatusCode, strings.TrimSpace(string(body[:min(len(body), 300)])))
	}
	percent, err := parseCopilotPercent(string(body))
	if err != nil {
		return CopilotUsage{}, err
	}
	if c.DebugWriter != nil {
		fmt.Fprintf(c.DebugWriter, "\n--- GitHub settings page premium-request snippet ---\n%s\n--- end GitHub settings page snippet ---\n\n", copilotDebugSnippet(string(body)))
	}
	return CopilotUsage{Percent: percent, Source: "github_settings_page", IsPercent: true}, nil
}

func premiumRequestsConsumed(item CopilotUsageItem) float64 {
	if item.GrossQuantity > 0 {
		return item.GrossQuantity
	}
	if item.DiscountQuantity > 0 {
		return item.DiscountQuantity
	}
	return item.NetQuantity
}

func isPremiumRequestUsageItem(item CopilotUsageItem) bool {
	product := strings.ToLower(item.Product)
	if product != "" && !strings.Contains(product, "copilot") {
		return false
	}
	text := strings.ToLower(strings.Join([]string{item.Product, item.SKU, item.Model, item.UnitType}, " "))
	return strings.Contains(text, "premium") && strings.Contains(text, "request")
}

func parseCopilotPercent(html string) (float64, error) {
	lower := strings.ToLower(html)
	index := strings.Index(lower, "premium requests")
	if index >= 0 {
		end := min(len(html), index+5000)
		if percent, ok := firstPercent(html[index:end]); ok {
			return percent, nil
		}
	}
	if percent, ok := firstPercent(html); ok {
		return percent, nil
	}
	return 0, fmt.Errorf("could not find Copilot premium requests percentage in settings page")
}

func firstPercent(text string) (float64, bool) {
	match := regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)%`).FindStringSubmatch(text)
	if len(match) < 2 {
		return 0, false
	}
	percent, err := strconv.ParseFloat(match[1], 64)
	return percent, err == nil
}

func copilotDebugSnippet(html string) string {
	lower := strings.ToLower(html)
	index := strings.Index(lower, "premium requests")
	if index < 0 {
		if len(html) > 1000 {
			return html[:1000]
		}
		return html
	}
	start := index - 500
	if start < 0 {
		start = 0
	}
	end := index + 1500
	if end > len(html) {
		end = len(html)
	}
	return html[start:end]
}

func do(client *http.Client, req *http.Request) ([]byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

type BarkClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewBarkClient(httpClient *http.Client, baseURL string) BarkClient {
	return BarkClient{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

func (c BarkClient) Push(ctx context.Context, key, title, body string) error {
	payload := map[string]string{
		"device_key": key,
		"title":      title,
		"body":       body,
		"group":      "AI Usage",
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/push", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if err := doJSON(c.httpClient, req, nil); err != nil {
		return fmt.Errorf("bark push: %w", err)
	}
	return nil
}

func doJSON(client *http.Client, req *http.Request, dst any) error {
	body, err := do(client, req)
	if err != nil {
		return err
	}
	if dst == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
