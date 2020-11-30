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
	expireAfter  time.Duration // how long to wait for a parent trace to appear

	info map[parentChild]recentInfo
}

// Info about what happened recently with an object
type recentInfo struct {
	trace.SpanContext
}

func newRecentInfoStore() recentInfoStore {
	return recentInfoStore{
		info: make(map[parentChild]recentInfo),
	}
}

func (r *recentInfoStore) store(key parentChild, spanContext trace.SpanContext) {
	r.info[key] = recentInfo{SpanContext: spanContext}
}

func (r *recentInfoStore) lookupSpanContext(key parentChild) (trace.SpanContext, bool) {
	value, ok := r.info[key]
	return value.SpanContext, ok
}
