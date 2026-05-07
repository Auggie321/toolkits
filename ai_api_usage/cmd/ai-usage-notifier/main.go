package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ai-usage-notifier/internal/app"
)

func main() {
	notNotice := flag.Bool("notnotice", false, "print usage report locally without sending Bark notification")
	all := flag.Bool("all", false, "include optional providers such as Gemini billing usage")
	debugGitHub := flag.Bool("debug-github", false, "print GitHub usage response details for local debugging")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	runner := app.Runner{
		Config:         cfg,
		GeminiReporter: app.NewBigQueryGeminiReporter(cfg),
		DeepSeekClient: app.NewDeepSeekClient(httpClient, cfg.DeepSeekBaseURL),
		GitHubClient:   app.NewGitHubClientWithWebBaseURL(httpClient, cfg.GitHubBaseURL, cfg.GitHubWebBaseURL),
		BarkClient:     app.NewBarkClient(httpClient, cfg.BarkBaseURL),
		Clock:          time.Now,
		NotNotice:      *notNotice,
		All:            *all,
		DebugGitHub:    *debugGitHub,
		Output:         os.Stdout,
	}

	if err := runner.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
