package app

import "testing"

func TestParseCopilotPercent(t *testing.T) {
	html := `<section><h3>Premium requests</h3><div class="Progress"></div><span>76.0%</span></section>`
	got, err := parseCopilotPercent(html)
	if err != nil {
		t.Fatal(err)
	}
	if got != 76.0 {
		t.Fatalf("expected 76.0, got %v", got)
	}
}

func TestIsPremiumRequestUsageItem(t *testing.T) {
	item := CopilotUsageItem{Product: "GitHub Copilot", SKU: "Premium Request", NetQuantity: 3}
	if !isPremiumRequestUsageItem(item) {
		t.Fatal("expected item to be treated as Copilot premium request usage")
	}
}
