package events

import (
	"bytes"
	"context"
	gojson "encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/weaveworks-experiments/kspan/pkg/mtime"
)

type captureDetails struct {
	Timestamp time.Time `json:"time"`
	Style     string    `json:"style"`
	Kind      string    `json:"kind"`
}

// Capture details on an object, so we can play back sequences for testing
func (r *EventWatcher) captureObject(obj runtime.Object, style string) {
	if r.Capture == nil {
		return
	}
	fmt.Fprintln(r.Capture, "---")
	d := captureDetails{Timestamp: mtime.Now(), Style: style, Kind: obj.GetObjectKind().GroupVersionKind().Kind}
	buf, _ := gojson.Marshal(&d)
	fmt.Fprintln(r.Capture, "#", string(buf))
	cf := k8sserializer.NewCodecFactory(r.scheme)
	serializerInfo, _ := runtime.SerializerInfoForMediaType(cf.SupportedMediaTypes(), runtime.ContentTypeYAML)
	encoder := serializerInfo.Serializer
	err := encoder.Encode(obj, r.Capture)
	_ = err // TODO error handling
}

func walkFile(ctx context.Context, filename string, callback func(captureDetails, []byte) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening %q: %v", filename, err)
	}
	defer file.Close()
	fr := json.YAMLFramer.NewFrameReader(file)
	buf := make([]byte, 64*1024) // "should be enough for anyone"
	for {
		n, err := fr.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error: %v", err)
		}

		doc := buf[:n]
		if n < 10 {
			return fmt.Errorf("doc too short: %q", string(doc))
		}
		if bytes.HasPrefix(doc, []byte("---\n")) {
			doc = doc[4:]
		}

		// we expect the top line of each doc to have metadata written out by capture
		line := firstLine(doc)
		var details captureDetails
		err = gojson.Unmarshal(line[2:], &details)
		if err != nil {
			return fmt.Errorf("error parsing first line %q: %v", string(line), err)
		}

		err = callback(details, doc)
		if err != nil {
			return err
		}
	}
	return nil
}

func firstLine(buf []byte) []byte {
	p := bytes.IndexByte(buf, '\n')
	if p == -1 {
		return nil
	}
	return buf[:p]
}

func getInitialObjects(filename string) ([]runtime.Object, time.Time, error) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	//s := serializer.NewSerializer(scheme, nil)
	var maxTimestamp time.Time
	var objects []runtime.Object

	err := walkFile(context.Background(), filename, func(details captureDetails, doc []byte) error {
		if details.Timestamp.After(maxTimestamp) {
			maxTimestamp = details.Timestamp
		}
		switch details.Style {
		case "initial":
			dec := yaml.NewYAMLToJSONDecoder(bytes.NewBuffer(doc))
			var u unstructured.Unstructured
			err := dec.Decode(&u)
			if err != nil {
				return err
			}
			objects = append(objects, &u)
		}
		return nil
	})
	return objects, maxTimestamp, err
}

func playback(ctx context.Context, r *EventWatcher, filename string) error {
	return walkFile(ctx, filename, func(details captureDetails, doc []byte) error {
		switch details.Style {
		case "initial":
			// no-op on playback
		case "event":
			dec := yaml.NewYAMLToJSONDecoder(bytes.NewBuffer(doc))
			var ev v1.Event
			err := dec.Decode(&ev)
			if err != nil {
				return fmt.Errorf("error parsing: %v", err)
			}
			mtime.NowForce(details.Timestamp)
			err = r.handleEvent(ctx, &ev)
			mtime.NowReset()
			return err
		default:
			return fmt.Errorf("style not recognized: %q", string(details.Style))
		}
		return nil
	})
}
