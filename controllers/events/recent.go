package events

import (
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

const (
	defaultRecentWindow = time.Second * 5
	defaultExpireAfter  = time.Minute * 5
)

type recentInfoStore struct {
	sync.Mutex
	recentWindow time.Duration // events within this window are considered likely to belong together
	expireAfter  time.Duration // how long to keep events in the recent cache

	info map[actionReference]recentInfo
}

// Info about what happened recently with an object
type recentInfo struct {
	lastUsed      time.Time
	spanContext   trace.SpanContext
	parentContext trace.SpanContext
}

func newRecentInfoStore() *recentInfoStore {
	return &recentInfoStore{
		recentWindow: defaultRecentWindow,
		expireAfter:  defaultExpireAfter,
		info:         make(map[actionReference]recentInfo),
	}
}

func (r *recentInfoStore) store(key actionReference, parentContext, spanContext trace.SpanContext) {
	r.Lock()
	defer r.Unlock()
	r.info[key] = recentInfo{
		lastUsed:      time.Now(),
		spanContext:   spanContext,
		parentContext: parentContext,
	}
}

func (r *recentInfoStore) lookupSpanContext(key actionReference) (trace.SpanContext, trace.SpanContext, bool) {
	now := time.Now()
	r.Lock()
	defer r.Unlock()
	value, ok := r.info[key]
	if !ok {
		return noTrace, noTrace, false
	}
	if value.lastUsed.Before(now.Add(-r.recentWindow)) {
		// fmt.Printf("key %v info too old %s, %s\n", key, value.lastUsed, now.Add(-r.recentWindow))
		//return noTrace, false
	}
	value.lastUsed = now
	r.info[key] = value
	return value.spanContext, value.parentContext, ok
}

func (r *recentInfoStore) expire() {
	now := time.Now()
	expiry := now.Add(-r.expireAfter)
	r.Lock()
	defer r.Unlock()
	for k, v := range r.info {
		if v.lastUsed.Before(expiry) {
			delete(r.info, k)
		}
	}
}
