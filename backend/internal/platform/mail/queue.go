package mail

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultWorkers     = 8
	defaultSendTimeout = 10 * time.Second
)

// Queue sends mail on a bounded worker pool. Enqueue never blocks the caller:
// if every worker is busy or the queue is shutting down, the message is
// dropped and the failure is logged (no body, recipient, or token).
type Queue struct {
	sender  Sender
	logger  *slog.Logger
	sem     chan struct{}
	wg      sync.WaitGroup
	closed  atomic.Bool
	baseCtx context.Context
	sendTO  time.Duration
}

// NewQueue returns a queue with at most workers in-flight sends. ctx cancels
// in-flight SMTP; Shutdown waits for those goroutines. logger may be nil.
func NewQueue(sender Sender, logger *slog.Logger, workers int, ctx context.Context) *Queue {
	if logger == nil {
		logger = slog.Default()
	}
	if workers < 1 {
		workers = defaultWorkers
	}
	return &Queue{
		sender:  sender,
		logger:  logger,
		sem:     make(chan struct{}, workers),
		baseCtx: ctx,
		sendTO:  defaultSendTimeout,
	}
}

// Enqueue starts a send or drops the message. Safe after Shutdown (no-op).
func (q *Queue) Enqueue(msg Message) {
	if q == nil || q.sender == nil || q.closed.Load() {
		return
	}
	select {
	case q.sem <- struct{}{}:
	default:
		q.logger.Error("mail queue full", slog.Int64("user_id", msg.UserID))
		return
	}
	if q.closed.Load() {
		<-q.sem
		return
	}
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		defer func() { <-q.sem }()
		ctx, cancel := context.WithTimeout(q.baseCtx, q.sendTO)
		defer cancel()
		if err := q.sender.Send(ctx, msg); err != nil {
			q.logger.Error("mail send", slog.String("error", err.Error()), slog.Int64("user_id", msg.UserID))
		}
	}()
}

// Shutdown stops accepting work and waits for in-flight sends or ctx.
func (q *Queue) Shutdown(ctx context.Context) error {
	if q == nil {
		return nil
	}
	q.closed.Store(true)
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
