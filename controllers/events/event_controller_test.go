package events

import (
	"testing"
	"time"

	o "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDeploymentRolloutWithManagedFields(t *testing.T) {
	g := o.NewWithT(t)

	var (
		deploy1    unstructured.Unstructured
		rs1, rs2   unstructured.Unstructured
		pod0, pod1 unstructured.Unstructured
	)
	mustParse(t, deploy1str, &deploy1)
	mustParse(t, replicaSet1str, &rs1)
	mustParse(t, replicaSet2str, &rs2)
	mustParse(t, pod0str, &pod0)
	mustParse(t, pod1str, &pod1)

	// There is no 'top-level' event here; the controller must synthesise one from the managed fields of the Deployment.

	tests := []struct {
		name       string
		perm       []int
		wantTraces []string
	}{
		{
			name: "scaledown-later",
			perm: []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
			wantTraces: []string{
				"0: kubectl Deployment.Update ",
				"1: deployment-controller Deployment.ScalingReplicaSet (0) Scaled up replica set hello-world-6b9d85fbd6 to 1",
				"2: replicaset-controller ReplicaSet.SuccessfulCreate (1) Created pod: hello-world-6b9d85fbd6-klpv2",
				"3: default-scheduler Pod.Scheduled (2) Successfully assigned default/hello-world-6b9d85fbd6-klpv2 to kind-control-plane",
				"4: kubelet Pod.Pulled (2) Container image \"nginx:1.19.2-alpine\" already present on machine",
				"5: kubelet Pod.Created (2) Created container hello-world",
				"6: kubelet Pod.Started (2) Started container hello-world",
				"7: deployment-controller Deployment.ScalingReplicaSet (0) Scaled down replica set hello-world-7ff854f459 to 0",
				"8: kubelet Pod.Killing (7) Stopping container hello-world",
				"9: replicaset-controller ReplicaSet.SuccessfulDelete (7) Deleted pod: hello-world-7ff854f459-kl4hq",
			},
		},
		{
			name: "scaledown-earlier",
			perm: []int{0, 6, 1, 2, 3, 4, 5, 7, 8},
			wantTraces: []string{
				"0: kubectl Deployment.Update ",
				"1: deployment-controller Deployment.ScalingReplicaSet (0) Scaled up replica set hello-world-6b9d85fbd6 to 1",
				"2: deployment-controller Deployment.ScalingReplicaSet (0) Scaled down replica set hello-world-7ff854f459 to 0",
				"3: replicaset-controller ReplicaSet.SuccessfulCreate (1) Created pod: hello-world-6b9d85fbd6-klpv2",
				"4: default-scheduler Pod.Scheduled (3) Successfully assigned default/hello-world-6b9d85fbd6-klpv2 to kind-control-plane",
				"5: kubelet Pod.Pulled (3) Container image \"nginx:1.19.2-alpine\" already present on machine",
				"6: kubelet Pod.Created (3) Created container hello-world",
				"7: kubelet Pod.Started (3) Started container hello-world",
				"8: kubelet Pod.Killing (2) Stopping container hello-world",
				"9: replicaset-controller ReplicaSet.SuccessfulDelete (2) Deleted pod: hello-world-7ff854f459-kl4hq",
			},
		},
	}

	threshold, err := time.Parse(time.RFC3339, deploymentUpdateEventsThresholdStr)
	g.Expect(err).NotTo(o.HaveOccurred())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, r, exporter, log := newTestEventWatcher(&deploy1, &rs1, &rs2, &pod0, &pod1)
			defer r.stop()
			for _, index := range tt.perm {
				var event corev1.Event
				mustParse(t, deploymentUpdateEvents[index], &event)
				g.Expect(r.handleEvent(ctx, log, &event)).To(o.Succeed())
			}
			g.Expect(r.checkOlderPending(ctx, threshold)).To(o.Succeed())
			g.Expect(exporter.dump()).To(o.Equal(tt.wantTraces))
		})
	}
}

func TestDeploymentRolloutFromFlux(t *testing.T) {
	g := o.NewWithT(t)

	var (
		deploy1  unstructured.Unstructured
		rs1, rs2 unstructured.Unstructured
		pod1     unstructured.Unstructured
	)
	mustParse(t, fluxDeploymentStr, &deploy1)
	mustParse(t, fluxReplicaSet1astr, &rs1)
	mustParse(t, fluxReplicaSet1bstr, &rs2)
	mustParse(t, fluxPod1astr, &pod1)

	tests := []struct {
		name       string
		perm       []int
		wantTraces []string
	}{
		{
			name: "flux-event-later",
			perm: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			wantTraces: []string{
				"0: flux deployment.Sync Commit e332e7bac962: Update nginx",
				"1: deployment-controller Deployment.ScalingReplicaSet (0) Scaled up replica set hello-world-f77b4f6c8 to 1",
				"2: replicaset-controller ReplicaSet.SuccessfulCreate (1) Created pod: hello-world-f77b4f6c8-6tcj2",
				"3: default-scheduler Pod.Scheduled (2) Successfully assigned default/hello-world-f77b4f6c8-6tcj2 to node2",
				"4: kubelet Pod.Pulling (2) Pulling image \"nginx:1.19.3-alpine\"",
				"5: kubelet Pod.Pulled (2) Successfully pulled image \"nginx:1.19.3-alpine\"",
				"6: kubelet Pod.Created (2) Created container hello-world",
				"7: kubelet Pod.Started (2) Started container hello-world",
				"8: deployment-controller Deployment.ScalingReplicaSet (0) Scaled down replica set hello-world-779cbf9f67 to 0",
				"9: replicaset-controller ReplicaSet.SuccessfulDelete (8) Deleted pod: hello-world-779cbf9f67-nbwfm",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, r, exporter, log := newTestEventWatcher(&deploy1, &rs1, &rs2, &pod1)
			defer r.stop()
			for _, index := range tt.perm {
				var event corev1.Event
				mustParse(t, fluxDeploymentUpdateEvents[index], &event)
				g.Expect(r.handleEvent(ctx, log, &event)).To(o.Succeed())
			}
			g.Expect(exporter.dump()).To(o.Equal(tt.wantTraces))
		})
	}
}
