# AI Usage Notifier

Go CLI for sending a daily AI usage summary to Bark.

Default mode checks:

- DeepSeek balance
- GitHub Copilot premium requests

Optional mode checks:

- Gemini billing usage from Google Cloud Billing export in BigQuery

## Quick Start

Copy the env template and fill in real values:

```bash
cp .env.example .env
```

Run locally without sending Bark:

```bash
go run ./cmd/ai-usage-notifier --notnotice
```

Run locally with all optional providers, including Gemini:

```bash
go run ./cmd/ai-usage-notifier --notnotice --all
```

Send the Bark notification:

```bash
go run ./cmd/ai-usage-notifier
```

## .env

Real `.env` files are ignored by git. Do not commit API keys, GitHub tokens, Bark keys, Google service account JSON files, or session cookies.

Required for default mode:

```env
DEEPSEEK_API_KEY=sk-your-deepseek-api-key

GITHUB_TOKEN=github_pat_your_token
GITHUB_USERNAME=your-github-login
COPILOT_PREMIUM_LIMIT=300

BARK_BASE_URL=https://your-bark.example.com
BARK_KEY=your-device-key

STATE_PATH=./state.jsonl
HTTP_TIMEOUT_SECONDS=30
```

Optional Gemini config for `--all`:

```env
GEMINI_API_KEY=your-gemini-api-key
GEMINI_BQ_PROJECT=your-gcp-project-id
GEMINI_BQ_DATASET=your_billing_export_dataset
GEMINI_BILLING_TABLE=your-gcp-project-id.your_billing_export_dataset.gcp_billing_export_v1_YOUR_BILLING_ACCOUNT_ID
GOOGLE_APPLICATION_CREDENTIALS=/absolute/path/to/google-service-account.json
TOTAL_BUDGET=10
```

`GEMINI_API_KEY` is kept for completeness, but account-level billing usage is queried through BigQuery. The Google BigQuery client reads `GOOGLE_APPLICATION_CREDENTIALS` automatically after this app loads `.env`.

## Output

Default output:

```text
DeepSeek: CNY 8.1400

GitHub Copilot: PremiumReqs 227 / 300 (Usage: 75.7%)
```

With `--all`, Gemini is included:

```text
Gemini: USD 1.2345 / 10.0000 (Usage: 12.3%)

DeepSeek: CNY 8.1400

GitHub Copilot: PremiumReqs 227 / 300 (Usage: 75.7%)
```

## GitHub Copilot

Create a fine-grained GitHub PAT:

- Repository access: `Public repositories`
- Account permissions: add `Plan`, set it to `Read-only`

The official billing API returns Copilot usage rows with `grossQuantity`, `discountQuantity`, and `netQuantity`. This app uses `grossQuantity` first so the result matches the Copilot settings page usage percentage more closely.

Debug GitHub response parsing:

```bash
go run ./cmd/ai-usage-notifier --notnotice --debug-github
```

## Gemini

Gemini is disabled by default. Use `--all` to query it.

Gemini API keys can call Gemini models, but they do not provide account-level historical billing usage. This app uses Google Cloud Billing export in BigQuery. If there are no matching current-month billing export rows, the Gemini line will say:

```text
Gemini: No current-month billing export rows matched Gemini/Vertex AI.
```

That can mean Gemini has not generated billable usage this month yet, the billing export has not caught up, or the export table/filter does not include the relevant SKU.

## Build

Build for the current OS:

```bash
go build -o ai-usage-notifier ./cmd/ai-usage-notifier
```

Build for Linux:

```bash
GOOS=linux GOARCH=amd64 go build -o ai-usage-notifier ./cmd/ai-usage-notifier
```

Cron example:

```cron
30 9 * * * /opt/ai-usage-notifier/ai-usage-notifier >> /var/log/ai-usage-notifier.log 2>&1
```

Use `--all` in cron only if you want Gemini billing lookup included:

```cron
30 9 * * * /opt/ai-usage-notifier/ai-usage-notifier --all >> /var/log/ai-usage-notifier.log 2>&1
```
