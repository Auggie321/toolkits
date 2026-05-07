package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type StateStore struct {
	path string
}

type BalanceSnapshot struct {
	Currency     string    `json:"currency"`
	TotalBalance float64   `json:"total_balance"`
	CreatedAt    time.Time `json:"created_at"`
}

func OpenStateStore(path string) (*StateStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return &StateStore{path: path}, nil
}

func (s *StateStore) Close() error {
	return nil
}

func (s *StateStore) LatestBalanceBefore(ctx context.Context, currency string, before time.Time) (*BalanceSnapshot, error) {
	file, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var latest *BalanceSnapshot
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		var snapshot BalanceSnapshot
		if err := json.Unmarshal(scanner.Bytes(), &snapshot); err != nil {
			return nil, fmt.Errorf("parse state snapshot: %w", err)
		}
		if snapshot.Currency != currency || !snapshot.CreatedAt.Before(before) {
			continue
		}
		if latest == nil || snapshot.CreatedAt.After(latest.CreatedAt) {
			copy := snapshot
			latest = &copy
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return latest, nil
}

func (s *StateStore) InsertBalance(ctx context.Context, snapshot BalanceSnapshot) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	line, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}
