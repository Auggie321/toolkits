package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnvDoesNotOverrideExistingEnv(t *testing.T) {
	t.Setenv("AI_USAGE_TEST_KEY", "from-env")
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("AI_USAGE_TEST_KEY=from-file\nAI_USAGE_TEST_OTHER=\"quoted value\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("AI_USAGE_TEST_KEY"); got != "from-env" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
	if got := os.Getenv("AI_USAGE_TEST_OTHER"); got != "quoted value" {
		t.Fatalf("expected quoted value, got %q", got)
	}
}

func TestLoadConfigWithoutAllDoesNotRequireGemini(t *testing.T) {
	setMinimalRequiredEnv(t)
	t.Setenv("GEMINI_BQ_PROJECT", "")
	t.Setenv("GEMINI_BQ_DATASET", "")
	t.Setenv("GEMINI_BILLING_TABLE", "")

	cfg, err := LoadConfigWithOptions(ConfigOptions{IncludeGemini: false})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DeepSeekAPIKey != "deepseek" || cfg.GitHubUsername != "octo" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigWithAllRequiresGemini(t *testing.T) {
	setMinimalRequiredEnv(t)
	t.Setenv("GEMINI_BQ_PROJECT", "")
	t.Setenv("GEMINI_BQ_DATASET", "")
	t.Setenv("GEMINI_BILLING_TABLE", "")

	_, err := LoadConfigWithOptions(ConfigOptions{IncludeGemini: true})
	if err == nil {
		t.Fatal("expected missing Gemini config error")
	}
	if got := err.Error(); !strings.Contains(got, "missing Gemini environment variables for --all") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func setMinimalRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DEEPSEEK_API_KEY", "deepseek")
	t.Setenv("GITHUB_TOKEN", "github")
	t.Setenv("GITHUB_USERNAME", "octo")
	t.Setenv("COPILOT_PREMIUM_LIMIT", "300")
	t.Setenv("BARK_BASE_URL", "https://bark.example.com")
	t.Setenv("BARK_KEY", "bark")
	t.Setenv("GITHUB_COPILOT_SOURCE", "auto")
}
