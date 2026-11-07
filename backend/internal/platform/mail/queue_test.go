package mail

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type stubSender struct {
	mu      sync.Mutex
	msgs    []Message
	err     error
	started chan struct{}
	block   chan struct{}
	sends   atomic.Int32
}

func (s *stubSender) Send(ctx context.Context, msg Message) error {
	s.sends.Add(1)
	if s.started != nil {
		select {
		case s.started <- struct{}{}:
		default:
		}
	}
	if s.block != nil {
		select {
		case <-s.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.mu.Lock()
	s.msgs = append(s.msgs, msg)
	s.mu.Unlock()
	return s.err
}

func TestQueueDelivers(t *testing.T) {
	t.Parallel()
	stub := &stubSender{}
	q := NewQueue(stub, slog.Default(), 2, context.Background())
	q.Enqueue(Message{UserID: 7, To: "a@example.com", Body: "secret-token"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.msgs) != 1 || stub.msgs[0].UserID != 7 {
		t.Fatalf("msgs = %+v", stub.msgs)
	}
}

func TestQueueDropsWhenFull(t *testing.T) {
	t.Parallel()
	started := make(chan struct{}, 1)
	block := make(chan struct{})
	stub := &stubSender{started: started, block: block}
	q := NewQueue(stub, slog.Default(), 1, context.Background())
	q.Enqueue(Message{UserID: 1, Body: "first"})
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first send did not start")
	}
	q.Enqueue(Message{UserID: 2, Body: "dropped-token"})
	close(block)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if stub.sends.Load() != 1 {
		t.Fatalf("sends = %d, want 1", stub.sends.Load())
	}
}

func TestQueueShutdownDropsNewWork(t *testing.T) {
	t.Parallel()
	stub := &stubSender{}
	q := NewQueue(stub, slog.Default(), 1, context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	q.Enqueue(Message{UserID: 9, Body: "after-shutdown"})
	time.Sleep(50 * time.Millisecond)
	if stub.sends.Load() != 0 {
		t.Fatalf("sends after shutdown = %d", stub.sends.Load())
	}
}

func TestQueueSendErrorDoesNotPanic(t *testing.T) {
	t.Parallel()
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	stub := &stubSender{err: context.DeadlineExceeded}
	q := NewQueue(stub, logger, 1, context.Background())
	q.Enqueue(Message{UserID: 3, To: "secret@example.com", Body: "raw-token-abc"})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := q.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	out := logs.String()
	if strings.Contains(out, "raw-token-abc") || strings.Contains(out, "secret@example.com") {
		t.Fatalf("log leaked secrets: %s", out)
	}
	if !strings.Contains(out, "user_id=3") {
		t.Fatalf("expected user_id in log: %s", out)
	}
}
