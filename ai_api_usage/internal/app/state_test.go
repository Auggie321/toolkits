package app

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStateStoreBalanceSnapshots(t *testing.T) {
	store, err := OpenStateStore(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := context.Background()
	first := time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC)
	second := time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC)
	if err := store.InsertBalance(ctx, BalanceSnapshot{Currency: "USD", TotalBalance: 20, CreatedAt: first}); err != nil {
		t.Fatal(err)
	}
	got, err := store.LatestBalanceBefore(ctx, "USD", second)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.TotalBalance != 20 || !got.CreatedAt.Equal(first) {
		t.Fatalf("unexpected snapshot %#v", got)
	}
}
