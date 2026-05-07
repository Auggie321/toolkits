package app

import (
	"strings"
	"testing"
)

func TestBuildNotificationBody(t *testing.T) {
	body := BuildNotificationBody(
		UsageReport{
			Gemini:        GeminiUsage{Rows: []GeminiUsageRow{{Currency: "USD", UsageUnit: "seconds", UsageAmount: 1.25, Cost: 2.5}}},
			GeminiBudget:  10,
			GeminiEnabled: true,
			DeepSeek:      []string{"USD 10.0000"},
			Copilot:       CopilotUsage{Used: 30, Limit: 300},
		},
	)

	for _, want := range []string{
		"Gemini: USD 2.5000 / 10.0000 (Usage: 25.0%)",
		"DeepSeek: USD 10.0000",
		"GitHub Copilot: PremiumReqs 30 / 300 (Usage: 10.0%)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestBuildNotificationBodySkipsGeminiWhenDisabled(t *testing.T) {
	body := BuildNotificationBody(UsageReport{
		Gemini:        GeminiUsage{Rows: []GeminiUsageRow{{Currency: "USD", Cost: 2.5}}},
		GeminiBudget:  10,
		GeminiEnabled: false,
		DeepSeek:      []string{"CNY 8.1400"},
		Copilot:       CopilotUsage{Used: 227, Limit: 300},
	})
	if strings.Contains(body, "Gemini") {
		t.Fatalf("expected Gemini to be skipped:\n%s", body)
	}
	for _, want := range []string{"DeepSeek: CNY 8.1400", "GitHub Copilot: PremiumReqs 227 / 300 (Usage: 75.7%)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}
