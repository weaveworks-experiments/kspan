package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"

	"github.com/weaveworks-experiments/kspan/pkg/mtime"
)

// Take out watches on individual objects, and notify changes in Conditions back as synthetic Events
type watchManager struct {
	sync.Mutex
	client dynamic.Interface
	mapper meta.RESTMapper

	watches map[objectReference]*watchInfo
}

type watchInfo struct {
	watch     watch.Interface
	lastEvent time.Time
	serial    int
}

func newWatchManager(kubeClient dynamic.Interface, mapper meta.RESTMapper) *watchManager {
	return &watchManager{
		client:  kubeClient,
		mapper:  mapper,
		watches: make(map[objectReference]*watchInfo),
	}
}

// local interface to insulate from EventWatcher type
type eventNotifier interface {
	handleEvent(ctx context.Context, event *corev1.Event) error
}

func (m *watchManager) watch(ctx context.Context, obj runtime.Object, ew eventNotifier) error {
	ma, _ := meta.Accessor(obj)
	var wi *watchInfo
	{
		ref := refFromObject(ma)
		m.Lock()
		if _, exists := m.watches[ref]; exists {
			m.Unlock()
			return nil
		}
		wi = &watchInfo{
			lastEvent: mtime.Now().Add(-defaultRecentWindow), // TODO: maybe this can be done more cleanly
		}
		m.watches[ref] = wi
		m.Unlock()
	}

	gvk := obj.GetObjectKind().GroupVersionKind()
	r, err := m.getResourceInterface(gvk, ma.GetNamespace())
	if err != nil {
		return err
	}
	listOptions := v1.ListOptions{
		FieldSelector:   fields.OneTermEqualSelector("metadata.name", ma.GetName()).String(),
		ResourceVersion: ma.GetResourceVersion(),
	}
	wi.watch, err = r.Watch(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("object watch failed: %w", err)
	}

	go wi.run(ew)

	return nil
}

func (m *watchManager) getResourceInterface(gvk schema.GroupVersionKind, ns string) (dynamic.ResourceInterface, error) {
	mapping, err := m.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("resource mapping failed: %w", err)
	}
	if mapping.Scope.Name() == meta.RESTScopeNameRoot {
		return m.client.Resource(mapping.Resource), nil
	}
	return m.client.Resource(mapping.Resource).Namespace(ns), nil
}

func (w *watchInfo) run(ew eventNotifier) {
	for e := range w.watch.ResultChan() {
		obj, ok := e.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}
		_ = w.checkConditionUpdates(obj, ew)
	}
}

// Given an object, walk through all its conditions and notify any new ones as Events
func (w *watchInfo) checkConditionUpdates(obj *unstructured.Unstructured, ew eventNotifier) error {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	var latest time.Time
	for _, conditionUncast := range conditions {
		condition, ok := conditionUncast.(map[string]interface{})
		if !ok {
			continue
		}
		// Check if this condition changed since last time we looked
		lastTransitionStr, found, err := unstructured.NestedString(condition, "lastTransitionTime")
		if !found || err != nil {
			continue
		}
		lastTransitionTime, err := time.Parse(time.RFC3339, lastTransitionStr)
		if err != nil {
			continue
		}
		if !lastTransitionTime.After(w.lastEvent) {
			continue
		}

		w.serial++

		name, _, _ := unstructured.NestedString(condition, "type")
		status, _, _ := unstructured.NestedString(condition, "status")
		message, _, _ := unstructured.NestedString(condition, "message")
		reason, _, _ := unstructured.NestedString(condition, "reason")

		// See if we can find a managedFields entry for this condition
		source, operation, ts := getUpdateSource(obj, "f:status", "f:conditions", `k:{"type":"`+name+`"}`)
		if ts.IsZero() {
			// Look for a manager of 'conditions' as a whole
			source, operation, ts = getUpdateSource(obj, "f:status", "f:conditions")
		}

		if reason == "" {
			if name != "" {
				reason = name
			} else if operation != "" {
				reason = operation
			} else {
				reason = "Unknown"
			}
		}

		if message == "" {
			message = name + " " + status
		}

		ma, _ := meta.Accessor(obj)
		gvk := obj.GetObjectKind().GroupVersionKind()
		ref := corev1.ObjectReference{
			Kind:       gvk.Kind,
			Namespace:  ma.GetNamespace(),
			Name:       ma.GetName(),
			APIVersion: gvk.Group + "/" + gvk.Version,
		}
		// synthesise an Event which we will use to generate a Span with all relevant information
		event := corev1.Event{
			ObjectMeta: v1.ObjectMeta{
				Namespace: ma.GetNamespace(),
				Name:      fmt.Sprintf("%s-%d", ma.GetName(), w.serial),
				UID:       types.UID(fmt.Sprintf("%s/%s-%d", ma.GetNamespace(), ma.GetName(), w.serial)),
			},
			Source: corev1.EventSource{
				Component: source,
			},
			EventTime:      v1.NewMicroTime(lastTransitionTime),
			Type:           corev1.EventTypeNormal,
			InvolvedObject: ref,
			Message:        message,
			Reason:         reason,
		}
		adjustEventTime(&event, mtime.Now())

		err = ew.handleEvent(context.TODO(), &event)
		if err != nil {
			continue
		}

		if lastTransitionTime.After(latest) {
			latest = lastTransitionTime
		}
	}
	if !latest.IsZero() {
		w.lastEvent = latest
	}
	return nil
}
