package atime

import (
	"context"
	"sync"
	"time"
)

const (
	defaultFlushInterval = time.Second
	defaultCapacity      = 4096
)

type sysReadKey struct{}

func WithSystemRead(ctx context.Context) context.Context {
	return context.WithValue(ctx, sysReadKey{}, true)
}

func IsSystemRead(ctx context.Context) bool {
	v, _ := ctx.Value(sysReadKey{}).(bool)
	return v
}

// DispatchFn sends a TouchAccessTime RPC for a single entry. It must be safe
// for concurrent use and tolerant of transient errors (the toucher swallows
// them: atime updates are best-effort).
type DispatchFn func(ctx context.Context, dir, name string, atimeNs int64)

type Toucher struct {
	dispatch     DispatchFn
	flushInterval time.Duration
	capacity     int

	mu      sync.Mutex
	pending map[entryKey]int64

	stop chan struct{}
	done chan struct{}
}

type entryKey struct {
	dir  string
	name string
}

func NewToucher(dispatch DispatchFn) *Toucher {
	return NewToucherWithOptions(dispatch, defaultFlushInterval, defaultCapacity)
}

func NewToucherWithOptions(dispatch DispatchFn, flushInterval time.Duration, capacity int) *Toucher {
	if flushInterval <= 0 {
		flushInterval = defaultFlushInterval
	}
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	t := &Toucher{
		dispatch:      dispatch,
		flushInterval: flushInterval,
		capacity:      capacity,
		pending:       make(map[entryKey]int64),
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go t.run()
	return t
}

// Bump enqueues an atime update. Returns immediately. Drops the update when
// the pending queue is at capacity or when ctx is marked SystemRead.
func (t *Toucher) Bump(ctx context.Context, dir, name string) {
	if t == nil || t.dispatch == nil {
		return
	}
	if IsSystemRead(ctx) {
		return
	}
	if dir == "" || name == "" {
		return
	}
	now := time.Now().UnixNano()
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.pending) >= t.capacity {
		if _, exists := t.pending[entryKey{dir, name}]; !exists {
			return
		}
	}
	t.pending[entryKey{dir, name}] = now
}

func (t *Toucher) Close() {
	if t == nil {
		return
	}
	select {
	case <-t.stop:
		return
	default:
	}
	close(t.stop)
	<-t.done
}

func (t *Toucher) run() {
	defer close(t.done)
	ticker := time.NewTicker(t.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			t.flush()
			return
		case <-ticker.C:
			t.flush()
		}
	}
}

func (t *Toucher) flush() {
	t.mu.Lock()
	if len(t.pending) == 0 {
		t.mu.Unlock()
		return
	}
	batch := t.pending
	t.pending = make(map[entryKey]int64, len(batch))
	t.mu.Unlock()

	ctx := context.Background()
	for key, ns := range batch {
		t.dispatch(ctx, key.dir, key.name, ns)
	}
}
