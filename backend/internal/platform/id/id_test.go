package id

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewGeneratorRejectsLargeWorkerID(t *testing.T) {
	t.Parallel()
	if _, err := NewGenerator(maxWorkerID + 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestNextMonotonicAndUnique(t *testing.T) {
	t.Parallel()
	g, err := NewGenerator(7)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	const n = 5000
	seen := make(map[int64]struct{}, n)
	var prev int64
	for i := 0; i < n; i++ {
		id, err := g.Next(ctx)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if id <= prev {
			t.Fatalf("id %d is not greater than previous %d", id, prev)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = struct{}{}
		prev = id
	}
}

func TestWorkerIDHonored(t *testing.T) {
	t.Parallel()
	const worker uint16 = 42
	g, err := NewGenerator(worker)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		id, err := g.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got := WorkerID(id)
		if got != worker {
			t.Fatalf("WorkerID(%d) = %d, want %d", id, got, worker)
		}
		ts := Timestamp(id)
		if ts.Before(time.UnixMilli(EpochMs)) {
			t.Fatalf("timestamp %s is before epoch", ts)
		}
		if ts.After(time.Now().UTC().Add(2 * time.Second)) {
			t.Fatalf("timestamp %s is in the future", ts)
		}
	}
}

func TestNextConcurrentUnique(t *testing.T) {
	t.Parallel()
	g, err := NewGenerator(3)
	if err != nil {
		t.Fatal(err)
	}
	const goroutines = 8
	const per = 500
	ids := make(chan int64, goroutines*per)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	ctx := context.Background()
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < per; j++ {
				id, err := g.Next(ctx)
				if err != nil {
					t.Errorf("Next: %v", err)
					return
				}
				ids <- id
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := make(map[int64]struct{}, goroutines*per)
	for id := range ids {
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != goroutines*per {
		t.Fatalf("got %d unique ids, want %d", len(seen), goroutines*per)
	}
}

func TestNextClockMovedBack(t *testing.T) {
	t.Parallel()
	g, err := NewGenerator(1)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	var current time.Time
	g.now = func() time.Time { return current }

	current = base
	if _, err := g.Next(context.Background()); err != nil {
		t.Fatalf("first Next: %v", err)
	}

	current = base.Add(-20 * time.Millisecond)
	_, err = g.Next(context.Background())
	if !errors.Is(err, ErrClockMovedBack) {
		t.Fatalf("got %v, want ErrClockMovedBack", err)
	}
}

func TestNextCanceledContext(t *testing.T) {
	t.Parallel()
	g, err := NewGenerator(1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := g.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestNextBeforeEpoch(t *testing.T) {
	t.Parallel()
	g, err := NewGenerator(1)
	if err != nil {
		t.Fatal(err)
	}
	g.now = func() time.Time {
		return time.UnixMilli(EpochMs - 1).UTC()
	}
	if _, err := g.Next(context.Background()); !errors.Is(err, ErrBeforeEpoch) {
		t.Fatalf("got %v, want ErrBeforeEpoch", err)
	}
}

func TestDifferentWorkersDiffer(t *testing.T) {
	t.Parallel()
	a, err := NewGenerator(1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewGenerator(2)
	if err != nil {
		t.Fatal(err)
	}
	idA, err := a.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	idB, err := b.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if WorkerID(idA) == WorkerID(idB) {
		t.Fatalf("workers collided: %d and %d", idA, idB)
	}
}
