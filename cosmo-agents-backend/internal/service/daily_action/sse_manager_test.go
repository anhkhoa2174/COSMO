package daily_action

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// newOfflineSSEManager points at a port nothing listens on. Local delivery
// and connection bookkeeping never need Redis, and a dead address keeps the
// test from depending on one.
func newOfflineSSEManager(t *testing.T) *SSEManager {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1, DialTimeout: 50 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })
	return NewSSEManager(client)
}

func TestSSEManager_DeliversOnlyToThatUser(t *testing.T) {
	m := newOfflineSSEManager(t)
	defer m.Close()
	ctx := context.Background()
	alice, bob := uuid.New(), uuid.New()

	a1 := m.Register(ctx, alice)
	a2 := m.Register(ctx, alice) // a second tab
	b := m.Register(ctx, bob)

	m.deliverLocal(alice, &SSEEvent{EventType: "action_updated"})

	for i, ch := range []chan *SSEEvent{a1, a2} {
		select {
		case ev := <-ch:
			if ev.EventType != "action_updated" {
				t.Fatalf("tab %d got %q", i, ev.EventType)
			}
		default:
			t.Fatalf("tab %d received nothing", i)
		}
	}
	// One user's events must never reach another user's stream.
	select {
	case ev := <-b:
		t.Fatalf("bob received alice's event: %+v", ev)
	default:
	}
}

func TestSSEManager_FullChannelDropsInsteadOfBlocking(t *testing.T) {
	m := newOfflineSSEManager(t)
	defer m.Close()
	u := uuid.New()
	ch := m.Register(context.Background(), u)

	// A stalled browser tab must not stall delivery for everyone else: once
	// the buffer is full, further events are dropped.
	done := make(chan struct{})
	go func() {
		for i := 0; i < cap(ch)+10; i++ {
			m.deliverLocal(u, &SSEEvent{EventType: "x"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deliverLocal blocked on a full channel")
	}
	if len(ch) != cap(ch) {
		t.Fatalf("buffer holds %d, want %d", len(ch), cap(ch))
	}
}

func TestSSEManager_UnregisterClosesAndCleansUp(t *testing.T) {
	m := newOfflineSSEManager(t)
	defer m.Close()
	ctx := context.Background()
	u := uuid.New()

	first := m.Register(ctx, u)
	second := m.Register(ctx, u)

	m.Unregister(u, first)
	if _, open := <-first; open {
		t.Fatal("unregistered channel should be closed")
	}
	m.mu.RLock()
	stillSubscribed := m.pubsubs[u] != nil
	m.mu.RUnlock()
	if !stillSubscribed {
		t.Fatal("subscription must stay while another tab is open")
	}

	m.Unregister(u, second)
	m.mu.RLock()
	_, conns := m.connections[u]
	_, subs := m.pubsubs[u]
	_, cancels := m.cancelFuncs[u]
	m.mu.RUnlock()
	if conns || subs || cancels {
		t.Fatalf("last tab closed but state remains: conns=%v subs=%v cancels=%v", conns, subs, cancels)
	}

	// Delivering after everyone left, and unregistering twice, must be no-ops
	// rather than a send on (or double close of) a closed channel.
	m.deliverLocal(u, &SSEEvent{EventType: "late"})
	m.Unregister(u, second)
}

// Delivery racing with disconnects is the path that used to panic with "send
// on closed channel". Run under -race to make the lock discipline count.
func TestSSEManager_ConcurrentDeliverAndUnregister(t *testing.T) {
	m := newOfflineSSEManager(t)
	defer m.Close()
	ctx := context.Background()
	u := uuid.New()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		ch := m.Register(ctx, u)
		wg.Add(2)
		go func() { defer wg.Done(); m.deliverLocal(u, &SSEEvent{EventType: "x"}) }()
		go func() { defer wg.Done(); m.Unregister(u, ch) }()
	}
	wg.Wait()
}

func TestSSEManager_CloseClosesEverything(t *testing.T) {
	m := newOfflineSSEManager(t)
	ctx := context.Background()
	a := m.Register(ctx, uuid.New())
	b := m.Register(ctx, uuid.New())

	m.Close()

	for _, ch := range []chan *SSEEvent{a, b} {
		if _, open := <-ch; open {
			t.Fatal("Close should close every stream")
		}
	}
	if len(m.connections) != 0 || len(m.pubsubs) != 0 || len(m.cancelFuncs) != 0 {
		t.Fatal("Close should drop all bookkeeping")
	}
}

func TestSSEManager_PublishFillsDefaultsAndReportsRedisErrors(t *testing.T) {
	m := newOfflineSSEManager(t)
	defer m.Close()
	ev := &SSEEvent{EventType: "generation_complete"}

	// With Redis down the publish must fail loudly: callers use the error to
	// decide whether the client will ever hear about the change.
	if err := m.PublishEvent(context.Background(), uuid.New(), ev); err == nil {
		t.Fatal("expected publish error with Redis unreachable")
	}
	if ev.ID == "" || ev.Timestamp.IsZero() {
		t.Fatalf("id/timestamp should be filled before publishing: %+v", ev)
	}

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ev2 := &SSEEvent{ID: "keep", Timestamp: fixed}
	_ = m.PublishEvent(context.Background(), uuid.New(), ev2)
	if ev2.ID != "keep" || !ev2.Timestamp.Equal(fixed) {
		t.Fatalf("caller-supplied id/timestamp overwritten: %+v", ev2)
	}
}
