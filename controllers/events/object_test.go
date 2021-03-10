package events

import (
	"testing"
	"time"

	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func Test_getUpdateSource(t *testing.T) {
	g := o.NewWithT(t)

	tests := []struct {
		name          string
		objYaml       string
		wantSource    string
		wantOperation string
		wantTs        time.Time
	}{
		{
			name:          "managedFields-kubeadm",
			wantSource:    "kubeadm",
			wantOperation: "Update",
			wantTs:        time.Date(2020, time.November, 24, 16, 52, 40, 0, time.UTC),
			objYaml:       deploymentMFkubeadm,
		},
		{
			name:          "managedFields-other",
			wantSource:    "other",
			wantOperation: "Update",
			wantTs:        time.Date(2020, time.November, 24, 16, 53, 56, 0, time.UTC),
			objYaml:       deploymentMFother,
		},
		{
			name:          "noOwner",
			wantSource:    "unknown",
			wantOperation: "unknown",
			objYaml:       deploymentNoOwner,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var obj unstructured.Unstructured
			g.Expect(yaml.Unmarshal([]byte(tt.objYaml), &obj)).To(o.Succeed())
			gotSource, gotOperation, gotTs := getUpdateSource(&obj, "f:spec")
			g.Expect(gotSource).To(o.BeIdenticalTo(tt.wantSource))
			g.Expect(gotOperation).To(o.BeIdenticalTo(tt.wantOperation))
			g.Expect(gotTs.Equal(tt.wantTs)).To(o.BeTrue())
		})
	}
}
