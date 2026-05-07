package app

import "testing"

func TestGeminiTableRefAcceptsFullTableName(t *testing.T) {
	cfg := Config{
		GeminiBQProject:    "project-a",
		GeminiBQDataset:    "dataset_a",
		GeminiBillingTable: "billing-project.billing_dataset.gcp_billing_export_v1_ABC",
	}
	if got := cfg.GeminiTableRef(); got != "billing-project.billing_dataset.gcp_billing_export_v1_ABC" {
		t.Fatalf("unexpected table ref %q", got)
	}
}

func TestGeminiTableRefBuildsFromParts(t *testing.T) {
	cfg := Config{
		GeminiBQProject:    "project-a",
		GeminiBQDataset:    "dataset_a",
		GeminiBillingTable: "gcp_billing_export_v1_ABC",
	}
	if got := cfg.GeminiTableRef(); got != "project-a.dataset_a.gcp_billing_export_v1_ABC" {
		t.Fatalf("unexpected table ref %q", got)
	}
}
