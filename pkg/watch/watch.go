// Package watch polls a list function and reports when the result changes.
package watch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// DefaultInterval is the polling interval when none is given.
//
// IAM resources change on human timescales — someone edits a group, a policy
// binding is added — so a short interval spends API quota for no benefit. Ten
// seconds is responsive enough to watch a change land while staying well inside
// rate limits.
const DefaultInterval = 10 * time.Second

// MinInterval is the shortest interval allowed, to keep a typo from hammering
// the API.
const MinInterval = 2 * time.Second

// ListFunc fetches the current state of a resource collection.
type ListFunc func(ctx context.Context) ([]map[string]any, error)

// Event describes one poll result.
type Event struct {
	// Items is the current collection.
	Items []map[string]any

	// Changed is true when Items differs from the previous poll. The first poll
	// always reports true, since there is nothing to compare against and the
	// caller needs an initial render.
	Changed bool

	// Err is set when the poll failed. A failed poll is reported rather than
	// ending the watch: a transient API error should not terminate a session the
	// user left running.
	Err error

	// Iteration counts polls, starting at 1.
	Iteration int
}

// Options configures a watch.
type Options struct {
	// Interval between polls. Values below MinInterval are raised to it.
	Interval time.Duration

	// MaxIterations stops the watch after this many polls. Zero means run until
	// the context is cancelled; it exists so tests do not need to cancel.
	MaxIterations int
}

// interval returns the effective interval.
func (o Options) interval() time.Duration {
	if o.Interval < MinInterval {
		return DefaultInterval
	}
	return o.Interval
}

// Watch polls list and sends an Event for every poll.
//
// The channel is closed when the context is cancelled or MaxIterations is
// reached. The first event always has Changed set so the caller can render an
// initial state.
func Watch(ctx context.Context, list ListFunc, opts Options, events chan<- Event) {
	defer close(events)

	var lastFingerprint string
	ticker := time.NewTicker(opts.interval())
	defer ticker.Stop()

	for iteration := 1; ; iteration++ {
		items, err := list(ctx)

		ev := Event{Items: items, Err: err, Iteration: iteration}
		if err == nil {
			fp := Fingerprint(items)
			ev.Changed = iteration == 1 || fp != lastFingerprint
			lastFingerprint = fp
		}

		select {
		case events <- ev:
		case <-ctx.Done():
			return
		}

		if opts.MaxIterations > 0 && iteration >= opts.MaxIterations {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Fingerprint returns a stable hash of a collection.
//
// Each item is JSON-encoded and the encodings are sorted before hashing. Sorting
// the encodings rather than the items means no ID field is needed and a reordered
// response is never mistaken for a change -- the API does not guarantee list
// order, so an order-sensitive hash would report a change on every poll.
// Marshalling a Go map already sorts keys, so field order is stable too.
func Fingerprint(items []map[string]any) string {
	encoded := make([]string, 0, len(items))
	for _, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			// An unmarshalable item still has to contribute something, or two
			// different collections could hash alike.
			encoded = append(encoded, fmt.Sprintf("%v", item))
			continue
		}
		encoded = append(encoded, string(b))
	}
	sort.Strings(encoded)

	h := sha256.New()
	for _, e := range encoded {
		h.Write([]byte(e))
		h.Write([]byte("\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}
