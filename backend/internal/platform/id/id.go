// Package id generates 64-bit Snowflake IDs (ADR 0006).
//
// Bit layout (Twitter-style):
//
//	1 unused sign bit | 41-bit ms since epoch | 10-bit worker ID | 12-bit sequence
package id

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	// Epoch is 2026-01-01T00:00:00Z, the start of Friendfold time.
	EpochMs = 1767225600000

	workerBits   = 10
	sequenceBits = 12

	maxWorkerID = (1 << workerBits) - 1   // 1023
	maxSequence = (1 << sequenceBits) - 1 // 4095

	workerShift    = sequenceBits
	timestampShift = sequenceBits + workerBits

	// maxSkewMs is how far the wall clock may jump backwards before Next fails.
	maxSkewMs = 5
)

// ErrClockMovedBack is returned when the wall clock is more than maxSkewMs
// behind the timestamp of the last issued ID.
var ErrClockMovedBack = errors.New("id: clock moved backwards")

// ErrBeforeEpoch is returned when the clock is earlier than the Snowflake epoch.
var ErrBeforeEpoch = errors.New("id: timestamp before epoch")

// Generator issues process-local Snowflake IDs. It is safe for concurrent use.
type Generator struct {
	mu       sync.Mutex
	workerID int64
	lastMs   int64
	seq      int64
	now      func() time.Time
}

// NewGenerator returns a generator bound to workerID (0–1023).
func NewGenerator(workerID uint16) (*Generator, error) {
	if int(workerID) > maxWorkerID {
		return nil, fmt.Errorf("id: worker ID %d exceeds max %d", workerID, maxWorkerID)
	}
	return &Generator{
		workerID: int64(workerID),
		now:      time.Now,
	}, nil
}

// Next returns a new ID. IDs from a single generator are strictly increasing
// while the clock does not jump backwards beyond the allowed skew.
func (g *Generator) Next(ctx context.Context) (int64, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}

		g.mu.Lock()
		ts := g.now().UTC().UnixMilli()
		if ts < EpochMs {
			g.mu.Unlock()
			return 0, fmt.Errorf("%w (%d)", ErrBeforeEpoch, ts)
		}

		if ts < g.lastMs {
			skew := g.lastMs - ts
			g.mu.Unlock()
			if skew > maxSkewMs {
				return 0, fmt.Errorf("%w by %d ms", ErrClockMovedBack, skew)
			}
			if err := sleep(ctx, time.Duration(skew)*time.Millisecond); err != nil {
				return 0, fmt.Errorf("id: wait for clock: %w", err)
			}
			continue
		}

		if ts == g.lastMs {
			if g.seq >= maxSequence {
				g.mu.Unlock()
				if err := sleep(ctx, time.Millisecond); err != nil {
					return 0, fmt.Errorf("id: wait for next millisecond: %w", err)
				}
				continue
			}
			g.seq++
		} else {
			g.seq = 0
			g.lastMs = ts
		}

		id := pack(ts, g.workerID, g.seq)
		g.mu.Unlock()
		return id, nil
	}
}

// WorkerID returns the worker ID encoded in id.
func WorkerID(id int64) uint16 {
	return uint16((id >> workerShift) & maxWorkerID)
}

// Timestamp returns the wall time encoded in id.
func Timestamp(id int64) time.Time {
	ms := (id >> timestampShift) + EpochMs
	return time.UnixMilli(ms).UTC()
}

func pack(unixMs, workerID, seq int64) int64 {
	ts := unixMs - EpochMs
	return (ts << timestampShift) | (workerID << workerShift) | seq
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
