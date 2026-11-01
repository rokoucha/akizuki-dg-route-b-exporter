package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

func TestJoinRetriesFailedReinitialization(t *testing.T) {
	want := errors.New("reset failed")
	calls := 0
	m := &meter{
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		needsReset:   true,
		reinitialize: func(context.Context) error { calls++; return want },
	}
	for i := 0; i < 2; i++ {
		if err := m.join(context.Background()); !errors.Is(err, want) {
			t.Fatalf("join error = %v", err)
		}
		if !m.needsReset {
			t.Fatal("failed initialization must be retried")
		}
	}
	if calls != 2 {
		t.Fatalf("reset calls = %d", calls)
	}
}
