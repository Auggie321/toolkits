package app

import (
	"strings"
	"testing"
)

func TestGeminiMonthlyUsageSQL(t *testing.T) {
	sql := GeminiMonthlyUsageSQL("project.dataset.table")
	for _, want := range []string{
		"`project.dataset.table`",
		"DATE_TRUNC(CURRENT_DATE(), MONTH)",
		"LOWER(service.description) LIKE '%gemini%'",
		"SUM(CAST(cost AS FLOAT64))",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("SQL missing %q:\n%s", want, sql)
		}
	}
}
