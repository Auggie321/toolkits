package app

import (
	"os"
	"path/filepath"
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
