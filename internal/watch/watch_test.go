package watch

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFingerprint_StableAcrossCalls(t *testing.T) {
	items := []map[string]any{
		{"uuid": "a", "name": "A"},
		{"uuid": "b", "name": "B"},
	}
	first := Fingerprint(items)
	for i := 0; i < 50; i++ {
		if got := Fingerprint(items); got != first {
			t.Fatalf("fingerprint is not stable: %q then %q", first, got)
		}
	}
}

func TestFingerprint_IgnoresItemOrder(t *testing.T) {
	// The API does not guarantee list order. An order-sensitive hash would report
	// a change on every poll.
	a := []map[string]any{{"uuid": "a"}, {"uuid": "b"}, {"uuid": "c"}}
	b := []map[string]any{{"uuid": "c"}, {"uuid": "a"}, {"uuid": "b"}}

	if Fingerprint(a) != Fingerprint(b) {
		t.Error("reordering the same items changed the fingerprint")
	}
}

func TestFingerprint_DetectsContentChange(t *testing.T) {
	a := []map[string]any{{"uuid": "a", "name": "A"}}
	b := []map[string]any{{"uuid": "a", "name": "CHANGED"}}

	if Fingerprint(a) == Fingerprint(b) {
		t.Error("a changed field did not change the fingerprint")
	}
}

func TestFingerprint_DetectsAdditionAndRemoval(t *testing.T) {
	one := []map[string]any{{"uuid": "a"}}
	two := []map[string]any{{"uuid": "a"}, {"uuid": "b"}}

	if Fingerprint(one) == Fingerprint(two) {
		t.Error("adding an item did not change the fingerprint")
	}
}

func TestFingerprint_EmptyAndNilAgree(t *testing.T) {
	if Fingerprint(nil) != Fingerprint([]map[string]any{}) {
		t.Error("nil and empty collections should fingerprint alike")
	}
}

func TestOptions_IntervalClamping(t *testing.T) {
	tests := []struct {
		name  string
		given time.Duration
		want  time.Duration
	}{
		{name: "zero uses the default", given: 0, want: DefaultInterval},
		{name: "below the minimum uses the default", given: time.Millisecond, want: DefaultInterval},
		{name: "at the minimum is honored", given: MinInterval, want: MinInterval},
		{name: "above the minimum is honored", given: time.Minute, want: time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Options{Interval: tt.given}).interval(); got != tt.want {
				t.Errorf("interval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWatch_FirstEventAlwaysReportsChanged(t *testing.T) {
	// The caller needs an initial render; there is nothing to compare against.
	list := func(ctx context.Context) ([]map[string]any, error) {
		return []map[string]any{{"uuid": "a"}}, nil
	}

	events := make(chan Event)
	go Watch(context.Background(), list, Options{Interval: MinInterval, MaxIterations: 1}, events)

	ev, ok := <-events
	if !ok {
		t.Fatal("channel closed with no event")
	}
	if !ev.Changed {
		t.Error("Changed = false on the first event")
	}
	if ev.Iteration != 1 {
		t.Errorf("Iteration = %d, want 1", ev.Iteration)
	}
}

func TestWatch_UnchangedPollsReportNotChanged(t *testing.T) {
	list := func(ctx context.Context) ([]map[string]any, error) {
		return []map[string]any{{"uuid": "a"}}, nil
	}

	events := make(chan Event)
	go Watch(context.Background(), list, Options{Interval: MinInterval, MaxIterations: 3}, events)

	var changedCount int
	for ev := range events {
		if ev.Changed {
			changedCount++
		}
	}
	if changedCount != 1 {
		t.Errorf("%d events reported Changed, want only the first", changedCount)
	}
}

func TestWatch_ReportsChangeWhenContentChanges(t *testing.T) {
	call := 0
	list := func(ctx context.Context) ([]map[string]any, error) {
		call++
		if call == 1 {
			return []map[string]any{{"uuid": "a"}}, nil
		}
		return []map[string]any{{"uuid": "a"}, {"uuid": "b"}}, nil
	}

	events := make(chan Event)
	go Watch(context.Background(), list, Options{Interval: MinInterval, MaxIterations: 2}, events)

	var changed []int
	for ev := range events {
		if ev.Changed {
			changed = append(changed, ev.Iteration)
		}
	}
	if len(changed) != 2 {
		t.Fatalf("Changed on iterations %v, want both 1 and 2", changed)
	}
}

func TestWatch_PollErrorIsReportedAndDoesNotStopTheWatch(t *testing.T) {
	// A transient API error should not kill a session the user left running.
	call := 0
	wantErr := errors.New("boom")
	list := func(ctx context.Context) ([]map[string]any, error) {
		call++
		if call == 1 {
			return nil, wantErr
		}
		return []map[string]any{{"uuid": "a"}}, nil
	}

	events := make(chan Event)
	go Watch(context.Background(), list, Options{Interval: MinInterval, MaxIterations: 2}, events)

	var sawErr, sawSuccess bool
	for ev := range events {
		if ev.Err != nil {
			sawErr = true
		} else {
			sawSuccess = true
		}
	}
	if !sawErr {
		t.Error("the failing poll was not reported")
	}
	if !sawSuccess {
		t.Error("the watch stopped after an error instead of continuing")
	}
}

func TestWatch_ClosesChannelOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	list := func(ctx context.Context) ([]map[string]any, error) {
		return []map[string]any{{"uuid": "a"}}, nil
	}

	events := make(chan Event)
	go Watch(ctx, list, Options{Interval: MinInterval}, events)

	<-events // consume the first event
	cancel()

	// Draining must terminate, which only happens if the channel is closed.
	done := make(chan struct{})
	go func() {
		for range events {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("channel was not closed after the context was cancelled")
	}
}

func TestWatch_StopsAtMaxIterations(t *testing.T) {
	calls := 0
	list := func(ctx context.Context) ([]map[string]any, error) {
		calls++
		return []map[string]any{{"uuid": "a"}}, nil
	}

	events := make(chan Event)
	go Watch(context.Background(), list, Options{Interval: MinInterval, MaxIterations: 3}, events)

	count := 0
	for range events {
		count++
	}
	if count != 3 {
		t.Errorf("got %d events, want 3", count)
	}
	if calls != 3 {
		t.Errorf("list was called %d times, want 3", calls)
	}
}
