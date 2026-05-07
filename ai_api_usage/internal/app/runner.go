package app

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Runner struct {
	Config         Config
	GeminiReporter GeminiReporter
	DeepSeekClient DeepSeekClient
	GitHubClient   GitHubClient
	BarkClient     BarkClient
	Clock          func() time.Time
	NotNotice      bool
	All            bool
	DebugGitHub    bool
	Output         io.Writer
}

func (r Runner) Run(ctx context.Context) error {
	now := r.Clock()
	out := r.output()

	var gemini GeminiUsage
	var geminiErr error
	var geminiEnabled bool
	if r.All {
		geminiEnabled = true
		fmt.Fprintln(out, "Fetching Gemini usage from BigQuery billing export...")
		var err error
		gemini, err = r.GeminiReporter.MonthlyUsage(ctx)
		if err != nil {
			geminiErr = err
			fmt.Fprintf(out, "Gemini usage failed: %v\n", err)
		}
	}

	fmt.Fprintln(out, "Fetching DeepSeek balance...")
	deepseek, err := r.DeepSeekClient.Balance(ctx, r.Config.DeepSeekAPIKey)
	var deepseekErr error
	if err != nil {
		deepseekErr = err
		fmt.Fprintf(out, "DeepSeek balance failed: %v\n", err)
	}

	fmt.Fprintln(out, "Fetching GitHub Copilot premium request usage...")
	r.GitHubClient.DebugWriter = nil
	if r.DebugGitHub {
		r.GitHubClient.DebugWriter = out
	}
	copilot, err := r.GitHubClient.CopilotUsage(ctx, r.Config, now)
	var copilotErr error
	if err != nil {
		copilotErr = err
		fmt.Fprintf(out, "GitHub Copilot usage failed: %v\n", err)
	}

	var deepseekLines []string
	fmt.Fprintln(out, "Recording DeepSeek balance snapshot...")
	if deepseekErr == nil {
		store, err := OpenStateStore(r.Config.StatePath)
		if err != nil {
			deepseekErr = err
			fmt.Fprintf(out, "DeepSeek snapshot failed: %v\n", err)
		} else {
			defer store.Close()
			deepseekLines, err = r.recordDeepSeekBalances(ctx, store, deepseek, now)
			if err != nil {
				deepseekErr = err
				fmt.Fprintf(out, "DeepSeek snapshot failed: %v\n", err)
			}
		}
	} else {
		fmt.Fprintln(out, "DeepSeek snapshot skipped because balance fetch failed.")
	}

	body := BuildNotificationBody(UsageReport{
		Gemini:        gemini,
		GeminiErr:     geminiErr,
		GeminiBudget:  r.Config.TotalBudget,
		GeminiEnabled: geminiEnabled,
		DeepSeek:      deepseekLines,
		DeepSeekErr:   deepseekErr,
		Copilot:       copilot,
		CopilotErr:    copilotErr,
	})
	if r.NotNotice {
		fmt.Fprintln(out, "Bark notification skipped because --notnotice is set.")
		fmt.Fprintln(out, "\n--- AI Usage Daily ---")
		fmt.Fprintln(out, body)
		return nil
	}
	fmt.Fprintln(out, "Sending Bark notification...")
	return r.BarkClient.Push(ctx, r.Config.BarkKey, "AI Usage Daily", body)
}

func (r Runner) output() io.Writer {
	if r.Output != nil {
		return r.Output
	}
	return io.Discard
}

func (r Runner) recordDeepSeekBalances(ctx context.Context, store *StateStore, balance DeepSeekBalance, now time.Time) ([]string, error) {
	var lines []string
	for _, info := range balance.BalanceInfo {
		total, err := strconv.ParseFloat(info.TotalBalance, 64)
		if err != nil {
			return nil, fmt.Errorf("parse deepseek %s total_balance: %w", info.Currency, err)
		}
		lines = append(lines, fmt.Sprintf("%s %.4f", info.Currency, total))
		if err := store.InsertBalance(ctx, BalanceSnapshot{
			Currency:     info.Currency,
			TotalBalance: total,
			CreatedAt:    now,
		}); err != nil {
			return nil, err
		}
	}
	if len(lines) == 0 {
		lines = append(lines, fmt.Sprintf("no balance info; available=%v", balance.IsAvailable))
	}
	return lines, nil
}

type UsageReport struct {
	Gemini        GeminiUsage
	GeminiErr     error
	GeminiBudget  float64
	GeminiEnabled bool
	DeepSeek      []string
	DeepSeekErr   error
	Copilot       CopilotUsage
	CopilotErr    error
}

func BuildNotificationBody(report UsageReport) string {
	var b strings.Builder
	if report.GeminiEnabled {
		b.WriteString("Gemini: ")
		if report.GeminiErr != nil {
			b.WriteString(fmt.Sprintf("Failed: %v\n\n", report.GeminiErr))
		} else if len(report.Gemini.Rows) == 0 {
			b.WriteString("No current-month billing export rows matched Gemini/Vertex AI.\n\n")
		} else {
			currency, cost := report.Gemini.TotalCost()
			if report.GeminiBudget > 0 {
				b.WriteString(fmt.Sprintf("%s %.4f / %.4f (Usage: %.1f%%)\n\n", valueOr(currency, "currency?"), cost, report.GeminiBudget, cost/report.GeminiBudget*100))
			} else {
				b.WriteString(fmt.Sprintf("%s %.4f\n\n", valueOr(currency, "currency?"), cost))
			}
		}
	}

	b.WriteString("DeepSeek: ")
	if report.DeepSeekErr != nil {
		b.WriteString(fmt.Sprintf("Failed: %v\n", report.DeepSeekErr))
	} else {
		b.WriteString(strings.Join(report.DeepSeek, "; "))
		b.WriteString("\n")
	}

	percent := 0.0
	if report.Copilot.IsPercent {
		percent = report.Copilot.Percent
	} else if report.Copilot.Limit > 0 {
		percent = report.Copilot.Used / report.Copilot.Limit * 100
	}
	b.WriteString("\nGitHub Copilot: ")
	if report.CopilotErr != nil {
		b.WriteString(fmt.Sprintf("Failed: %v", report.CopilotErr))
	} else if report.Copilot.IsPercent {
		b.WriteString(fmt.Sprintf("PremiumReqs (Usage: %.1f%%)", percent))
	} else {
		b.WriteString(fmt.Sprintf("PremiumReqs %.0f / %.0f (Usage: %.1f%%)", report.Copilot.Used, report.Copilot.Limit, percent))
	}
	return strings.TrimSpace(b.String())
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
