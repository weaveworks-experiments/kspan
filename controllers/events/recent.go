package events

import (
	"time"

	"go.opentelemetry.io/otel/api/trace"
)

const (
	defaultRecentWindow = time.Second * 5
	defaultExpireAfter  = time.Minute * 5
)

type recentInfoStore struct {
	recentWindow time.Duration // events within this window are considered likely to belong together
	expireAfter  time.Duration // how long to keep events in the recent cache

	info map[parentChild]recentInfo
}

// Info about what happened recently with an object
type recentInfo struct {
	lastUsed time.Time
	trace.SpanContext
}

func newRecentInfoStore() recentInfoStore {
	return recentInfoStore{
		recentWindow: defaultRecentWindow,
		expireAfter:  defaultExpireAfter,
		info:         make(map[parentChild]recentInfo),
	}
}

func (r *recentInfoStore) store(key parentChild, spanContext trace.SpanContext) {
	r.info[key] = recentInfo{
		lastUsed:    time.Now(),
		SpanContext: spanContext,
	}
}

func (r *recentInfoStore) lookupSpanContext(key parentChild) (trace.SpanContext, bool) {
	now := time.Now()
	value, ok := r.info[key]
	if !ok {
		return noTrace, false
	}
	if value.lastUsed.Before(now.Add(-r.recentWindow)) {
		// fmt.Printf("key %v info too old %s, %s\n", key, value.lastUsed, now.Add(-r.recentWindow))
		//return noTrace, false
	}
	value.lastUsed = now
	r.info[key] = value
	return value.SpanContext, ok
}

func (r *recentInfoStore) expire() {
	now := time.Now()
	expiry := now.Add(-r.expireAfter)
	// lock?
	for k, v := range r.info {
		if v.lastUsed.Before(expiry) {
			delete(r.info, k)
		}
	}
}
