package atime

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestToucher_CoalescesDuplicateBumps(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	dispatch := func(_ context.Context, dir, name string, _ int64) {
		mu.Lock()
		defer mu.Unlock()
		calls[dir+"/"+name]++
	}

	toucher := NewToucherWithOptions(dispatch, 10*time.Millisecond, 16)
	defer toucher.Close()

	for i := 0; i < 100; i++ {
		toucher.Bump(context.Background(), "/bucket", "object")
	}

	time.Sleep(60 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if calls["/bucket/object"] != 1 {
		t.Fatalf("expected coalesced single dispatch, got %d", calls["/bucket/object"])
	}
}

func TestToucher_SystemReadSkipped(t *testing.T) {
	called := false
	toucher := NewToucherWithOptions(func(context.Context, string, string, int64) { called = true }, 10*time.Millisecond, 16)
	defer toucher.Close()

	toucher.Bump(WithSystemRead(context.Background()), "/dir", "file")
	time.Sleep(40 * time.Millisecond)

	if called {
		t.Fatal("system-read bump must be skipped")
	}
}

func TestToucher_DropsAtCapacity(t *testing.T) {
	dispatched := make(map[string]int)
	var mu sync.Mutex
	toucher := NewToucherWithOptions(func(_ context.Context, dir, name string, _ int64) {
		mu.Lock()
		dispatched[dir+"/"+name]++
		mu.Unlock()
	}, 50*time.Millisecond, 2)
	defer toucher.Close()

	toucher.Bump(context.Background(), "/d", "a")
	toucher.Bump(context.Background(), "/d", "b")
	toucher.Bump(context.Background(), "/d", "c") // should be dropped (queue full)

	time.Sleep(80 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if _, ok := dispatched["/d/c"]; ok {
		t.Fatal("third entry should have been dropped at capacity")
	}
	if dispatched["/d/a"] == 0 || dispatched["/d/b"] == 0 {
		t.Fatalf("expected first two entries to dispatch, got %v", dispatched)
	}
}
