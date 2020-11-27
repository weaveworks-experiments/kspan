package events

import (
	"testing"

	o "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDeploymentRollout(t *testing.T) {
	g := o.NewWithT(t)

	var (
		deploy1  unstructured.Unstructured
		rs1, rs2 unstructured.Unstructured
		pod1     unstructured.Unstructured
	)
	mustParse(t, deploy1str, &deploy1)
	mustParse(t, replicaSet1str, &rs1)
	mustParse(t, replicaSet2str, &rs2)
	mustParse(t, pod1str, &pod1)

	ctx, r, exporter, log := newTestEventWatcher(&deploy1, &rs1, &rs2, &pod1)

	for _, eventStr := range deploymentUpdateEvents {
		var event corev1.Event
		mustParse(t, eventStr, &event)
		g.Expect(r.handleEvent(ctx, log, &event)).To(o.Succeed())
	}
	expectedTraces := []string{
		"0: kubectl Deployment.Update ",
		"1: deployment-controller Deployment.ScalingReplicaSet (0) Scaled up replica set hello-world-6b9d85fbd6 to 1",
		"2: replicaset-controller ReplicaSet.SuccessfulCreate (1) Created pod: hello-world-6b9d85fbd6-klpv2",
		"3: default-scheduler Pod.Scheduled (2) Successfully assigned default/hello-world-6b9d85fbd6-klpv2 to kind-control-plane",
		"4: kubelet Pod.Pulled (2) Container image \"nginx:1.19.2-alpine\" already present on machine",
		"5: kubelet Pod.Created (2) Created container hello-world",
		"6: kubelet Pod.Started (2) Started container hello-world",
		"7: deployment-controller Deployment.ScalingReplicaSet (0) Scaled down replica set hello-world-7ff854f459 to 0",
		"8: replicaset-controller ReplicaSet.SuccessfulDelete (7) Deleted pod: hello-world-7ff854f459-kl4hq",
	}
	g.Expect(exporter.dump()).To(o.Equal(expectedTraces))

}
