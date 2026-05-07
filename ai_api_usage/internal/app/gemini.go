package app

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type GeminiReporter interface {
	MonthlyUsage(context.Context) (GeminiUsage, error)
}

type BigQueryGeminiReporter struct {
	projectID string
	querySQL  string
}

func NewBigQueryGeminiReporter(cfg Config) BigQueryGeminiReporter {
	return BigQueryGeminiReporter{
		projectID: cfg.GeminiBQProject,
		querySQL:  GeminiMonthlyUsageSQL(cfg.GeminiTableRef()),
	}
}

func GeminiMonthlyUsageSQL(table string) string {
	return fmt.Sprintf(`
SELECT
  COALESCE(currency, '') AS currency,
  COALESCE(usage.unit, '') AS usage_unit,
  SUM(CAST(usage.amount AS FLOAT64)) AS usage_amount,
  SUM(CAST(cost AS FLOAT64)) AS cost
FROM %s
WHERE DATE(usage_start_time) >= DATE_TRUNC(CURRENT_DATE(), MONTH)
  AND DATE(usage_start_time) < DATE_ADD(DATE_TRUNC(CURRENT_DATE(), MONTH), INTERVAL 1 MONTH)
  AND (
    LOWER(service.description) LIKE '%%gemini%%'
    OR LOWER(sku.description) LIKE '%%gemini%%'
    OR LOWER(sku.description) LIKE '%%generative ai%%'
    OR LOWER(service.description) LIKE '%%vertex ai%%'
  )
GROUP BY currency, usage_unit
ORDER BY cost DESC`, quoteBQTable(table))
}

func quoteBQTable(table string) string {
	return "`" + table + "`"
}

func (r BigQueryGeminiReporter) MonthlyUsage(ctx context.Context) (GeminiUsage, error) {
	client, err := bigquery.NewClient(ctx, r.projectID)
	if err != nil {
		return GeminiUsage{}, fmt.Errorf("create BigQuery client from GOOGLE_APPLICATION_CREDENTIALS: %w", err)
	}
	defer client.Close()

	query := client.Query(r.querySQL)
	it, err := query.Read(ctx)
	if err != nil {
		return GeminiUsage{}, fmt.Errorf("query Gemini billing export: %w", err)
	}

	var usage GeminiUsage
	for {
		var row struct {
			Currency    string  `bigquery:"currency"`
			UsageUnit   string  `bigquery:"usage_unit"`
			UsageAmount float64 `bigquery:"usage_amount"`
			Cost        float64 `bigquery:"cost"`
		}
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return GeminiUsage{}, err
		}
		usage.Rows = append(usage.Rows, GeminiUsageRow(row))
	}
	return usage, nil
}
