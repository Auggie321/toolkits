package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	GeminiBQProject    string
	GeminiBQDataset    string
	GeminiBillingTable string
	BigQueryBaseURL    string
	GeminiAPIKey       string
	GoogleCredentials  string
	TotalBudget        float64

	DeepSeekAPIKey  string
	DeepSeekBaseURL string

	GitHubToken         string
	GitHubUsername      string
	GitHubBaseURL       string
	GitHubWebBaseURL    string
	GitHubSessionCookie string
	GitHubCopilotSource string
	CopilotPremiumLimit float64

	BarkBaseURL string
	BarkKey     string

	StatePath   string
	HTTPTimeout time.Duration
}

func LoadConfig() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		GeminiBQProject:     strings.TrimSpace(os.Getenv("GEMINI_BQ_PROJECT")),
		GeminiBQDataset:     strings.TrimSpace(os.Getenv("GEMINI_BQ_DATASET")),
		GeminiBillingTable:  strings.TrimSpace(os.Getenv("GEMINI_BILLING_TABLE")),
		BigQueryBaseURL:     envOrDefault("BIGQUERY_BASE_URL", "https://bigquery.googleapis.com"),
		GeminiAPIKey:        strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
		GoogleCredentials:   strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")),
		DeepSeekAPIKey:      strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")),
		DeepSeekBaseURL:     envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		GitHubToken:         strings.TrimSpace(os.Getenv("GITHUB_TOKEN")),
		GitHubUsername:      strings.TrimSpace(os.Getenv("GITHUB_USERNAME")),
		GitHubBaseURL:       envOrDefault("GITHUB_BASE_URL", "https://api.github.com"),
		GitHubWebBaseURL:    envOrDefault("GITHUB_WEB_BASE_URL", "https://github.com"),
		GitHubSessionCookie: strings.TrimSpace(os.Getenv("GITHUB_SESSION_COOKIE")),
		GitHubCopilotSource: strings.ToLower(envOrDefault("GITHUB_COPILOT_SOURCE", "auto")),
		BarkBaseURL:         strings.TrimRight(strings.TrimSpace(os.Getenv("BARK_BASE_URL")), "/"),
		BarkKey:             strings.TrimSpace(os.Getenv("BARK_KEY")),
		StatePath:           envOrDefault("STATE_PATH", defaultStatePath()),
		HTTPTimeout:         30 * time.Second,
	}

	if raw := strings.TrimSpace(os.Getenv("COPILOT_PREMIUM_LIMIT")); raw != "" {
		limit, err := strconv.ParseFloat(raw, 64)
		if err != nil || limit <= 0 {
			return Config{}, fmt.Errorf("COPILOT_PREMIUM_LIMIT must be a positive number")
		}
		cfg.CopilotPremiumLimit = limit
	}
	if raw := strings.TrimSpace(os.Getenv("TOTAL_BUDGET")); raw != "" {
		budget, err := strconv.ParseFloat(raw, 64)
		if err != nil || budget <= 0 {
			return Config{}, fmt.Errorf("TOTAL_BUDGET must be a positive number")
		}
		cfg.TotalBudget = budget
	}

	if raw := strings.TrimSpace(os.Getenv("HTTP_TIMEOUT_SECONDS")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return Config{}, fmt.Errorf("HTTP_TIMEOUT_SECONDS must be a positive integer")
		}
		cfg.HTTPTimeout = time.Duration(seconds) * time.Second
	}

	var missing []string
	for name, value := range map[string]string{
		"GEMINI_BQ_PROJECT":     cfg.GeminiBQProject,
		"GEMINI_BQ_DATASET":     cfg.GeminiBQDataset,
		"GEMINI_BILLING_TABLE":  cfg.GeminiBillingTable,
		"DEEPSEEK_API_KEY":      cfg.DeepSeekAPIKey,
		"GITHUB_TOKEN":          cfg.GitHubToken,
		"GITHUB_USERNAME":       cfg.GitHubUsername,
		"COPILOT_PREMIUM_LIMIT": fmt.Sprintf("%v", cfg.CopilotPremiumLimit),
		"BARK_BASE_URL":         cfg.BarkBaseURL,
		"BARK_KEY":              cfg.BarkKey,
	} {
		if value == "" || value == "0" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	if err := validateBQIdentifier(cfg.GeminiBQProject); err != nil {
		return Config{}, fmt.Errorf("invalid GEMINI_BQ_PROJECT: %w", err)
	}
	if err := validateBQIdentifier(cfg.GeminiBQDataset); err != nil {
		return Config{}, fmt.Errorf("invalid GEMINI_BQ_DATASET: %w", err)
	}
	if err := validateBQTable(cfg.GeminiBillingTable); err != nil {
		return Config{}, fmt.Errorf("invalid GEMINI_BILLING_TABLE: %w", err)
	}
	if cfg.GitHubCopilotSource != "auto" && cfg.GitHubCopilotSource != "api" && cfg.GitHubCopilotSource != "settings_page" {
		return Config{}, fmt.Errorf("GITHUB_COPILOT_SOURCE must be one of auto, api, settings_page")
	}
	return cfg, nil
}

func (cfg Config) GeminiTableRef() string {
	table := strings.Trim(strings.TrimSpace(cfg.GeminiBillingTable), "`")
	if strings.Count(table, ".") >= 2 {
		return table
	}
	return fmt.Sprintf("%s.%s.%s", cfg.GeminiBQProject, cfg.GeminiBQDataset, table)
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func defaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "ai-usage-notifier.sqlite"
	}
	return filepath.Join(home, ".local", "state", "ai-usage-notifier", "state.jsonl")
}

func validateBQIdentifier(value string) error {
	if value == "" {
		return errors.New("empty identifier")
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return fmt.Errorf("contains unsupported character %q", r)
	}
	return nil
}

func validateBQTable(value string) error {
	value = strings.Trim(strings.TrimSpace(value), "`")
	if value == "" {
		return errors.New("empty table")
	}
	for _, part := range strings.Split(value, ".") {
		if err := validateBQIdentifier(part); err != nil {
			return err
		}
	}
	return nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d invalid .env line: expected KEY=VALUE", path, lineNumber)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("%s:%d invalid .env line: empty key", path, lineNumber)
		}
		value = trimEnvQuotes(value)
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func trimEnvQuotes(value string) string {
	if len(value) < 2 {
		return value
	}
	if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}
