package events

import (
	"testing"

	"sigs.k8s.io/yaml"
)

func mustParse(t *testing.T, str string, obj interface{}) {
	t.Helper()
	err := yaml.Unmarshal([]byte(str), obj)
	if err != nil {
		t.Fatal("Failed to parse test object", str, err)
	}
}

// Object definitions used in unit tests.

// Created by exporting yaml from a real Kubernetes system then
// removing non-vital lines to reduce bulk.

// Deployment with managedFields saying kubeadm updated the spec
var deploymentMFkubeadm = `
apiVersion: apps/v1
kind: Deployment
metadata:
  generation: 1
  labels:
    k8s-app: kube-dns
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:labels:
          .: {}
          f:k8s-app: {}
      f:spec:
        f:replicas: {}
        f:template:
          f:spec:
            f:containers:
              k:{"name":"coredns"}:
                .: {}
                f:args: {}
                f:image: {}
            f:dnsPolicy: {}
    manager: kubeadm
    operation: Update
    time: "2020-11-24T16:52:40Z"
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:deployment.kubernetes.io/revision: {}
      f:status:
        f:availableReplicas: {}
        f:conditions:
          .: {}
          k:{"type":"Available"}:
            .: {}
          k:{"type":"Progressing"}:
            .: {}
    manager: kube-controller-manager
    operation: Update
    time: "2020-11-24T16:53:56Z"
  name: coredns
  namespace: kube-system
spec:
  replicas: 2
  selector:
    matchLabels:
      k8s-app: kube-dns
  strategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        k8s-app: kube-dns
    spec:
      containers:
      - name: coredns
        image: k8s.gcr.io/coredns:1.7.0
`

// Deployment with managedFields saying program 'other' was the most recent to update the spec
var deploymentMFother = `
apiVersion: apps/v1
kind: Deployment
metadata:
  generation: 1
  labels:
    k8s-app: kube-dns
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:spec:
        f:replicas: {}
        f:template:
          f:spec:
            f:containers:
              k:{"name":"coredns"}:
                .: {}
                f:args: {}
                f:image: {}
            f:dnsPolicy: {}
    manager: kubeadm
    operation: Update
    time: "2020-11-24T16:52:40Z"
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:deployment.kubernetes.io/revision: {}
      f:spec:
        f:replicas: {}
    manager: other
    operation: Update
    time: "2020-11-24T16:53:56Z"
  name: coredns
  namespace: kube-system
spec:
  replicas: 2
  selector:
    matchLabels:
      k8s-app: kube-dns
  strategy:
    rollingUpdate:
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      labels:
        k8s-app: kube-dns
    spec:
      containers:
      - name: coredns
        image: k8s.gcr.io/coredns:1.7.0
`

// Deployment with no managedFields or other clue as to what created it
var deploymentNoOwner = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coredns
  namespace: kube-system
spec:
  replicas: 2
  selector:
    matchLabels:
      k8s-app: kube-dns
  strategy:
  template:
    metadata:
      labels:
        k8s-app: kube-dns
    spec:
      containers:
      - name: coredns
        image: k8s.gcr.io/coredns:1.7.0
`

// 2 seconds after the last event time
const deploymentUpdateEventsThresholdStr = "2020-11-27T12:04:08Z"

// Events from a Deployment update from one version to another
var deploymentUpdateEvents = []string{`
apiVersion: v1
count: 2
firstTimestamp: "2020-11-24T18:37:54Z"
involvedObject:
  apiVersion: apps/v1
  kind: Deployment
  name: hello-world
  namespace: default
  resourceVersion: "649432"
  uid: 4ecf82fc-0f0a-44e0-9469-cebbb07f7a31
kind: Event
lastTimestamp: "2020-11-27T12:04:05Z"
message: Scaled up replica set hello-world-6b9d85fbd6 to 1
metadata:
  creationTimestamp: "2020-11-27T12:04:05Z"
  name: hello-world.164a8590cbae32b9
  namespace: default
  resourceVersion: "649437"
  selfLink: /api/v1/namespaces/default/events/hello-world.164a8590cbae32b9
  uid: 44f4b5cd-9bce-49da-8d14-4a8334e5eca2
reason: ScalingReplicaSet
source:
  component: deployment-controller
type: Normal
`, `
apiVersion: v1
count: 1
firstTimestamp: "2020-11-27T12:04:05Z"
involvedObject:
  apiVersion: apps/v1
  kind: ReplicaSet
  name: hello-world-6b9d85fbd6
  namespace: default
  resourceVersion: "649435"
  uid: b2fcb2a4-ed25-49dc-87de-db6cf8ec7a00
kind: Event
lastTimestamp: "2020-11-27T12:04:05Z"
message: 'Created pod: hello-world-6b9d85fbd6-klpv2'
metadata:
  creationTimestamp: "2020-11-27T12:04:05Z"
  name: hello-world-6b9d85fbd6.164b5bd11e8d7508
  namespace: default
  resourceVersion: "649440"
  selfLink: /api/v1/namespaces/default/events/hello-world-6b9d85fbd6.164b5bd11e8d7508
  uid: 9f38cba2-eeb3-45e8-8d93-80ef1a278c26
reason: SuccessfulCreate
source:
  component: replicaset-controller
type: Normal
`, `
apiVersion: v1
count: 1
firstTimestamp: "2020-11-27T12:04:05Z"
involvedObject:
  apiVersion: v1
  kind: Pod
  name: hello-world-6b9d85fbd6-klpv2
  namespace: default
  resourceVersion: "649438"
  uid: deb2b4f7-e312-44dd-bd06-7c00d0f5695a
kind: Event
lastTimestamp: "2020-11-27T12:04:05Z"
message: Successfully assigned default/hello-world-6b9d85fbd6-klpv2 to kind-control-plane
metadata:
  creationTimestamp: "2020-11-27T12:04:05Z"
  name: hello-world-6b9d85fbd6-klpv2.164b5bd11eb42666
  namespace: default
  resourceVersion: "649442"
  selfLink: /api/v1/namespaces/default/events/hello-world-6b9d85fbd6-klpv2.164b5bd11eb42666
  uid: cc1216f3-7314-4907-8fc7-3179cb708976
reason: Scheduled
source:
  component: default-scheduler
type: Normal
`, `
apiVersion: v1
count: 1
firstTimestamp: "2020-11-27T12:04:06Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{hello-world}
  kind: Pod
  name: hello-world-6b9d85fbd6-klpv2
  namespace: default
  resourceVersion: "649439"
  uid: deb2b4f7-e312-44dd-bd06-7c00d0f5695a
kind: Event
lastTimestamp: "2020-11-27T12:04:06Z"
message: Container image "nginx:1.19.2-alpine" already present on machine
metadata:
  creationTimestamp: "2020-11-27T12:04:06Z"
  name: hello-world-6b9d85fbd6-klpv2.164b5bd140cc9458
  namespace: default
  resourceVersion: "649446"
  selfLink: /api/v1/namespaces/default/events/hello-world-6b9d85fbd6-klpv2.164b5bd140cc9458
  uid: 9c0f38be-c520-415e-a247-a23f87779296
reason: Pulled
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
firstTimestamp: "2020-11-27T12:04:06Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{hello-world}
  kind: Pod
  name: hello-world-6b9d85fbd6-klpv2
  namespace: default
  resourceVersion: "649439"
  uid: deb2b4f7-e312-44dd-bd06-7c00d0f5695a
kind: Event
lastTimestamp: "2020-11-27T12:04:06Z"
message: Created container hello-world
metadata:
  creationTimestamp: "2020-11-27T12:04:06Z"
  name: hello-world-6b9d85fbd6-klpv2.164b5bd141cc615a
  namespace: default
  resourceVersion: "649447"
  selfLink: /api/v1/namespaces/default/events/hello-world-6b9d85fbd6-klpv2.164b5bd141cc615a
  uid: 03705f73-6e4d-4c9c-a9e3-155bb4befd09
reason: Created
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
firstTimestamp: "2020-11-27T12:04:06Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{hello-world}
  kind: Pod
  name: hello-world-6b9d85fbd6-klpv2
  namespace: default
  resourceVersion: "649439"
  uid: deb2b4f7-e312-44dd-bd06-7c00d0f5695a
kind: Event
lastTimestamp: "2020-11-27T12:04:06Z"
message: Started container hello-world
metadata:
  creationTimestamp: "2020-11-27T12:04:06Z"
  name: hello-world-6b9d85fbd6-klpv2.164b5bd145d6c76d
  namespace: default
  resourceVersion: "649448"
  selfLink: /api/v1/namespaces/default/events/hello-world-6b9d85fbd6-klpv2.164b5bd145d6c76d
  uid: 184a125b-95bf-48a1-8e60-5fec403136b8
reason: Started
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 2
firstTimestamp: "2020-11-24T18:37:59Z"
involvedObject:
  apiVersion: apps/v1
  kind: Deployment
  name: hello-world
  namespace: default
  resourceVersion: "649445"
  uid: 4ecf82fc-0f0a-44e0-9469-cebbb07f7a31
kind: Event
lastTimestamp: "2020-11-27T12:04:06Z"
message: Scaled down replica set hello-world-7ff854f459 to 0
metadata:
  name: hello-world.164a85920cbb7b8b
  namespace: default
  resourceVersion: "649453"
  selfLink: /api/v1/namespaces/default/events/hello-world.164a85920cbb7b8b
  uid: ada9bb44-66ed-4309-870d-da7e47e47c1e
reason: ScalingReplicaSet
source:
  component: deployment-controller
type: Normal
`, `
apiVersion: v1
count: 1
firstTimestamp: "2020-11-27T12:04:06Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{hello-world}
  kind: Pod
  name: hello-world-7ff854f459-kl4hq
  namespace: default
  resourceVersion: "173647"
  uid: add4c490-b2ef-4250-870b-4f0fd222740b
kind: Event
lastTimestamp: "2020-11-27T12:04:06Z"
message: Stopping container hello-world
metadata:
  creationTimestamp: "2020-11-27T12:04:06Z"
  name: hello-world-7ff854f459-kl4hq.164b5bd14885b76c
  namespace: default
  resourceVersion: "649454"
  selfLink: /api/v1/namespaces/default/events/hello-world-7ff854f459-kl4hq.164b5bd14885b76c
  uid: bb189ee2-1d61-4ab5-ad75-bb72f088ec6b
reason: Killing
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
firstTimestamp: "2020-11-27T12:04:06Z"
involvedObject:
  apiVersion: apps/v1
  kind: ReplicaSet
  name: hello-world-7ff854f459
  namespace: default
  resourceVersion: "649451"
  uid: a031073d-040f-4800-aeb7-cc198183b479
kind: Event
lastTimestamp: "2020-11-27T12:04:06Z"
message: 'Deleted pod: hello-world-7ff854f459-kl4hq'
metadata:
  creationTimestamp: "2020-11-27T12:04:06Z"
  name: hello-world-7ff854f459.164b5bd148836510
  namespace: default
  resourceVersion: "649455"
  selfLink: /api/v1/namespaces/default/events/hello-world-7ff854f459.164b5bd148836510
  uid: 18dc842f-7a9b-481d-a248-5281a5b2c5ab
reason: SuccessfulDelete
source:
  component: replicaset-controller
type: Normal
`,
}

var replicaSet1str = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  creationTimestamp: "2020-11-24T18:37:54Z"
  generation: 3
  labels:
    name: hello-world
    pod-template-hash: 6b9d85fbd6
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:labels:
          .: {}
          f:name: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"4ecf82fc-0f0a-44e0-9469-cebbb07f7a31"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:replicas: {}
        f:selector:
          f:matchLabels:
            .: {}
            f:name: {}
            f:pod-template-hash: {}
        f:template:
          f:metadata:
            f:labels:
              .: {}
              f:name: {}
              f:pod-template-hash: {}
          f:spec:
            f:containers:
              k:{"name":"hello-world"}:
                .: {}
                f:image: {}
                f:imagePullPolicy: {}
                f:name: {}
                f:ports:
                  .: {}
                  k:{"containerPort":80,"protocol":"TCP"}:
                    .: {}
                    f:containerPort: {}
                    f:protocol: {}
                f:resources: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
            f:dnsPolicy: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:terminationGracePeriodSeconds: {}
      f:status:
        f:observedGeneration: {}
        f:replicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2020-11-27T12:04:05Z"
  name: hello-world-6b9d85fbd6
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Deployment
    name: hello-world
    uid: 4ecf82fc-0f0a-44e0-9469-cebbb07f7a31
  resourceVersion: "649435"
  selfLink: /apis/apps/v1/namespaces/default/replicasets/hello-world-6b9d85fbd6
  uid: b2fcb2a4-ed25-49dc-87de-db6cf8ec7a00
spec:
  replicas: 1
  selector:
    matchLabels:
      name: hello-world
      pod-template-hash: 6b9d85fbd6
  template:
    metadata:
      labels:
        name: hello-world
        pod-template-hash: 6b9d85fbd6
    spec:
      containers:
      - image: nginx:1.19.2-alpine
        imagePullPolicy: IfNotPresent
        name: hello-world
        ports:
        - containerPort: 80
          protocol: TCP
status:
  observedGeneration: 2
  replicas: 0
`

var replicaSet2str = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  annotations:
    deployment.kubernetes.io/desired-replicas: "1"
    deployment.kubernetes.io/max-replicas: "2"
    deployment.kubernetes.io/revision: "3"
    deployment.kubernetes.io/revision-history: "1"
  creationTimestamp: "2020-11-24T18:06:07Z"
  generation: 4
  labels:
    name: hello-world
    pod-template-hash: 7ff854f459
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:deployment.kubernetes.io/desired-replicas: {}
          f:deployment.kubernetes.io/max-replicas: {}
          f:deployment.kubernetes.io/revision: {}
          f:deployment.kubernetes.io/revision-history: {}
        f:labels:
          .: {}
          f:name: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"4ecf82fc-0f0a-44e0-9469-cebbb07f7a31"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:replicas: {}
        f:selector:
          f:matchLabels:
            .: {}
            f:name: {}
            f:pod-template-hash: {}
        f:template:
          f:metadata:
            f:labels:
              .: {}
              f:name: {}
              f:pod-template-hash: {}
          f:spec:
            f:containers:
              k:{"name":"hello-world"}:
                .: {}
                f:image: {}
                f:imagePullPolicy: {}
                f:name: {}
                f:ports:
                  .: {}
                  k:{"containerPort":80,"protocol":"TCP"}:
                    .: {}
                    f:containerPort: {}
                    f:protocol: {}
                f:resources: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
            f:dnsPolicy: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:terminationGracePeriodSeconds: {}
      f:status:
        f:availableReplicas: {}
        f:fullyLabeledReplicas: {}
        f:observedGeneration: {}
        f:readyReplicas: {}
        f:replicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2020-11-27T12:04:06Z"
  name: hello-world-7ff854f459
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Deployment
    name: hello-world
    uid: 4ecf82fc-0f0a-44e0-9469-cebbb07f7a31
  resourceVersion: "649451"
  selfLink: /apis/apps/v1/namespaces/default/replicasets/hello-world-7ff854f459
  uid: a031073d-040f-4800-aeb7-cc198183b479
spec:
  replicas: 0
  selector:
    matchLabels:
      name: hello-world
      pod-template-hash: 7ff854f459
  template:
    metadata:
      creationTimestamp: null
      labels:
        name: hello-world
        pod-template-hash: 7ff854f459
    spec:
      containers:
      - image: nginx:1.19.3-alpine
        imagePullPolicy: IfNotPresent
        name: hello-world
        ports:
        - containerPort: 80
          protocol: TCP
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  fullyLabeledReplicas: 1
  observedGeneration: 3
  readyReplicas: 1
  replicas: 1
`

var deploy1str = `
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "3"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"name":"hello-world","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"name":"hello-world"}},"template":{"metadata":{"labels":{"name":"hello-world"}},"spec":{"containers":[{"image":"nginx:1.19.2-alpine","name":"hello-world","ports":[{"containerPort":80}]}]}}}}
  creationTimestamp: "2020-11-24T18:06:07Z"
  generation: 4
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          f:deployment.kubernetes.io/revision: {}
      f:status:
        f:availableReplicas: {}
        f:conditions:
          .: {}
          k:{"type":"Available"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
          k:{"type":"Progressing"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
        f:observedGeneration: {}
        f:readyReplicas: {}
        f:replicas: {}
        f:updatedReplicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2020-11-25T10:48:13Z"
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        f:progressDeadlineSeconds: {}
        f:replicas: {}
        f:revisionHistoryLimit: {}
        f:selector:
          f:matchLabels:
            .: {}
            f:name: {}
        f:strategy:
          f:rollingUpdate:
            .: {}
            f:maxSurge: {}
            f:maxUnavailable: {}
          f:type: {}
        f:template:
          f:metadata:
            f:labels:
              .: {}
              f:name: {}
          f:spec:
            f:containers:
              k:{"name":"hello-world"}:
                .: {}
                f:image: {}
                f:imagePullPolicy: {}
                f:name: {}
                f:ports:
                  .: {}
                  k:{"containerPort":80,"protocol":"TCP"}:
                    .: {}
                    f:containerPort: {}
                    f:protocol: {}
                f:resources: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
            f:dnsPolicy: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:terminationGracePeriodSeconds: {}
    manager: kubectl
    operation: Update
    time: "2020-11-27T12:04:05Z"
  name: hello-world
  namespace: default
  resourceVersion: "649432"
  selfLink: /apis/apps/v1/namespaces/default/deployments/hello-world
  uid: 4ecf82fc-0f0a-44e0-9469-cebbb07f7a31
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      name: hello-world
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        name: hello-world
    spec:
      containers:
      - image: nginx:1.19.2-alpine
        imagePullPolicy: IfNotPresent
        name: hello-world
        ports:
        - containerPort: 80
          protocol: TCP
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2020-11-24T18:06:13Z"
    lastUpdateTime: "2020-11-24T18:06:13Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2020-11-24T18:06:07Z"
    lastUpdateTime: "2020-11-25T10:48:13Z"
    message: ReplicaSet "hello-world-7ff854f459" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 3
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1
`

var pod1str = `
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: "2020-11-27T12:04:05Z"
  generateName: hello-world-6b9d85fbd6-
  labels:
    name: hello-world
    pod-template-hash: 6b9d85fbd6
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:generateName: {}
        f:labels:
          .: {}
          f:name: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"b2fcb2a4-ed25-49dc-87de-db6cf8ec7a00"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:containers:
          k:{"name":"hello-world"}:
            .: {}
            f:image: {}
            f:imagePullPolicy: {}
            f:name: {}
            f:ports:
              .: {}
              k:{"containerPort":80,"protocol":"TCP"}:
                .: {}
                f:containerPort: {}
                f:protocol: {}
            f:resources: {}
            f:terminationMessagePath: {}
            f:terminationMessagePolicy: {}
        f:dnsPolicy: {}
        f:enableServiceLinks: {}
        f:restartPolicy: {}
        f:schedulerName: {}
        f:securityContext: {}
        f:terminationGracePeriodSeconds: {}
    manager: kube-controller-manager
    operation: Update
    time: "2020-11-27T12:04:05Z"
  name: hello-world-6b9d85fbd6-klpv2
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: hello-world-6b9d85fbd6
    uid: b2fcb2a4-ed25-49dc-87de-db6cf8ec7a00
  resourceVersion: "649438"
  selfLink: /api/v1/namespaces/default/pods/hello-world-6b9d85fbd6-klpv2
  uid: deb2b4f7-e312-44dd-bd06-7c00d0f5695a
spec:
  containers:
  - image: nginx:1.19.2-alpine
    imagePullPolicy: IfNotPresent
    name: hello-world
    ports:
    - containerPort: 80
      protocol: TCP
    resources: {}
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-d2h5p
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: default-token-d2h5p
    secret:
      defaultMode: 420
      secretName: default-token-d2h5p
status:
  phase: Pending
  qosClass: BestEffort
`

var pod0str = `
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: "2020-11-27T10:04:05Z"
  generateName: hello-world-7ff854f459-
  labels:
    name: hello-world
    pod-template-hash: 7ff854f459
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:generateName: {}
        f:labels:
          .: {}
          f:name: {}
          f:pod-template-hash: {}
      f:spec:
        f:containers:
          k:{"name":"hello-world"}:
            .: {}
    manager: kube-controller-manager
    operation: Update
    time: "2020-11-27T10:04:05Z"
  name: hello-world-7ff854f459-kl4hq
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: hello-world-7ff854f459
    uid: a031073d-040f-4800-aeb7-cc198183b479
  resourceVersion: "649438"
  selfLink: /api/v1/namespaces/default/pods/hello-world-7ff854f459-kl4hq
  uid: deb2b4f7-e312-44dd-bd06-7c00d0f5695b
spec:
  containers:
  - image: nginx:1.19.1-alpine
    imagePullPolicy: IfNotPresent
    name: hello-world
    ports:
    - containerPort: 80
      protocol: TCP
status:
  phase: Pending
  qosClass: BestEffort
`

// 2 seconds after the last event time
const fluxDeploymentUpdateEventsThresholdStr = "2020-12-03T17:53:59Z"

// Events from a Deployment update triggered by Flux; objects have no managedFields
var fluxDeploymentUpdateEvents = []string{`
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:51Z"
  involvedObject:
    apiVersion: apps/v1
    kind: Deployment
    name: hello-world
    namespace: default
    resourceVersion: "10025"
    uid: 06790b30-23ce-411c-bb6d-4059ec328e1a
  kind: Event
  lastTimestamp: "2020-12-03T17:53:51Z"
  message: Scaled up replica set hello-world-f77b4f6c8 to 1
  metadata:
    creationTimestamp: "2020-12-03T17:53:51Z"
    name: hello-world.164d4662bb4dce0e
    namespace: default
    resourceVersion: "10028"
    selfLink: /api/v1/namespaces/default/events/hello-world.164d4662bb4dce0e
    uid: 41ef9f7e-b621-4a72-820c-eabbd142990c
  reason: ScalingReplicaSet
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: deployment-controller
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:51Z"
  involvedObject:
    apiVersion: apps/v1
    kind: ReplicaSet
    name: hello-world-f77b4f6c8
    namespace: default
    resourceVersion: "10026"
    uid: 01482a68-bdd2-423e-8cda-d0421f418758
  kind: Event
  lastTimestamp: "2020-12-03T17:53:51Z"
  message: 'Created pod: hello-world-f77b4f6c8-6tcj2'
  metadata:
    creationTimestamp: "2020-12-03T17:53:51Z"
    name: hello-world-f77b4f6c8.164d4662bba34b8b
    namespace: default
    resourceVersion: "10030"
    selfLink: /api/v1/namespaces/default/events/hello-world-f77b4f6c8.164d4662bba34b8b
    uid: 2b648f02-d0c3-4c5d-9d3b-372630376e43
  reason: SuccessfulCreate
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: replicaset-controller
  type: Normal
`, `
  action: Binding
  apiVersion: v1
  eventTime: "2020-12-03T17:53:51.917155Z"
  firstTimestamp: null
  involvedObject:
    apiVersion: v1
    kind: Pod
    name: hello-world-f77b4f6c8-6tcj2
    namespace: default
    resourceVersion: "10027"
    uid: 95c52c1f-c369-44da-8075-1b82b92f0e46
  kind: Event
  lastTimestamp: null
  message: Successfully assigned default/hello-world-f77b4f6c8-6tcj2 to node2
  metadata:
    creationTimestamp: "2020-12-03T17:53:51Z"
    name: hello-world-f77b4f6c8-6tcj2.164d4662bc11d4d7
    namespace: default
    resourceVersion: "10033"
    selfLink: /api/v1/namespaces/default/events/hello-world-f77b4f6c8-6tcj2.164d4662bc11d4d7
    uid: e60e3be1-9c3e-41ec-805b-2ddfdf4f2b42
  reason: Scheduled
  reportingComponent: default-scheduler
  reportingInstance: default-scheduler-node0
  source:
    component: default-scheduler
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:52Z"
  involvedObject:
    apiVersion: apps/v1
    kind: deployment
    name: hello-world
    namespace: default
  kind: Event
  lastTimestamp: "2020-12-03T17:53:52Z"
  message: 'Commit e332e7bac962: Update nginx'
  metadata:
    annotations:
      topLevelSpan: 'true'
      syncMetadata: '{"commits":[{"revision":"e332e7bac962bc5601d64fc9c382e3b5a09f96f0","message":"Update nginx"}],"includes":{"other":true}}'
    creationTimestamp: "2020-12-03T17:53:52Z"
    name: hello-world.164d4662cc294a22
    namespace: default
    resourceVersion: "10041"
    selfLink: /api/v1/namespaces/default/events/hello-world.164d4662cc294a22
    uid: 3bf5fb14-c2c7-4bce-ae04-41f754dd7227
  reason: Sync
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: flux
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:52Z"
  involvedObject:
    apiVersion: v1
    fieldPath: spec.containers{hello-world}
    kind: Pod
    name: hello-world-f77b4f6c8-6tcj2
    namespace: default
    resourceVersion: "10029"
    uid: 95c52c1f-c369-44da-8075-1b82b92f0e46
  kind: Event
  lastTimestamp: "2020-12-03T17:53:52Z"
  message: Pulling image "nginx:1.19.3-alpine"
  metadata:
    creationTimestamp: "2020-12-03T17:53:52Z"
    name: hello-world-f77b4f6c8-6tcj2.164d4662e5ba3bc2
    namespace: default
    resourceVersion: "10042"
    selfLink: /api/v1/namespaces/default/events/hello-world-f77b4f6c8-6tcj2.164d4662e5ba3bc2
    uid: cac34aa6-135b-49ed-a5f3-283ea80a3d2d
  reason: Pulling
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: kubelet
    host: node2
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:56Z"
  involvedObject:
    apiVersion: v1
    fieldPath: spec.containers{hello-world}
    kind: Pod
    name: hello-world-f77b4f6c8-6tcj2
    namespace: default
    resourceVersion: "10029"
    uid: 95c52c1f-c369-44da-8075-1b82b92f0e46
  kind: Event
  lastTimestamp: "2020-12-03T17:53:56Z"
  message: Successfully pulled image "nginx:1.19.3-alpine"
  metadata:
    creationTimestamp: "2020-12-03T17:53:56Z"
    name: hello-world-f77b4f6c8-6tcj2.164d4663d2e191b4
    namespace: default
    resourceVersion: "10049"
    selfLink: /api/v1/namespaces/default/events/hello-world-f77b4f6c8-6tcj2.164d4663d2e191b4
    uid: 2aef3d61-3581-430f-8a17-88022dffac35
  reason: Pulled
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: kubelet
    host: node2
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:56Z"
  involvedObject:
    apiVersion: v1
    fieldPath: spec.containers{hello-world}
    kind: Pod
    name: hello-world-f77b4f6c8-6tcj2
    namespace: default
    resourceVersion: "10029"
    uid: 95c52c1f-c369-44da-8075-1b82b92f0e46
  kind: Event
  lastTimestamp: "2020-12-03T17:53:56Z"
  message: Created container hello-world
  metadata:
    creationTimestamp: "2020-12-03T17:53:56Z"
    name: hello-world-f77b4f6c8-6tcj2.164d4663d94eb5da
    namespace: default
    resourceVersion: "10050"
    selfLink: /api/v1/namespaces/default/events/hello-world-f77b4f6c8-6tcj2.164d4663d94eb5da
    uid: a1a1e9fc-2479-42b9-ae04-f3add70c4efa
  reason: Created
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: kubelet
    host: node2
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:56Z"
  involvedObject:
    apiVersion: v1
    fieldPath: spec.containers{hello-world}
    kind: Pod
    name: hello-world-f77b4f6c8-6tcj2
    namespace: default
    resourceVersion: "10029"
    uid: 95c52c1f-c369-44da-8075-1b82b92f0e46
  kind: Event
  lastTimestamp: "2020-12-03T17:53:56Z"
  message: Started container hello-world
  metadata:
    creationTimestamp: "2020-12-03T17:53:56Z"
    name: hello-world-f77b4f6c8-6tcj2.164d4663df67b68b
    namespace: default
    resourceVersion: "10051"
    selfLink: /api/v1/namespaces/default/events/hello-world-f77b4f6c8-6tcj2.164d4663df67b68b
    uid: 36976fcb-5f5b-47c7-a634-30163c73811d
  reason: Started
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: kubelet
    host: node2
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:57Z"
  involvedObject:
    apiVersion: apps/v1
    kind: Deployment
    name: hello-world
    namespace: default
    resourceVersion: "10036"
    uid: 06790b30-23ce-411c-bb6d-4059ec328e1a
  kind: Event
  lastTimestamp: "2020-12-03T17:53:57Z"
  message: Scaled down replica set hello-world-779cbf9f67 to 0
  metadata:
    creationTimestamp: "2020-12-03T17:53:57Z"
    name: hello-world.164d46640e52ccf5
    namespace: default
    resourceVersion: "10057"
    selfLink: /api/v1/namespaces/default/events/hello-world.164d46640e52ccf5
    uid: e2308244-f074-41dc-a281-7378a2063b74
  reason: ScalingReplicaSet
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: deployment-controller
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:57Z"
  involvedObject:
    apiVersion: v1
    fieldPath: spec.containers{hello-world}
    kind: Pod
    name: hello-world-779cbf9f67-nbwfm
    namespace: default
    resourceVersion: "1746"
    uid: 097da021-9b68-447d-8bf0-c41aae8592dc
  kind: Event
  lastTimestamp: "2020-12-03T17:53:57Z"
  message: Stopping container hello-world
  metadata:
    creationTimestamp: "2020-12-03T17:53:57Z"
    name: hello-world-779cbf9f67-nbwfm.164d46640ec14f6a
    namespace: default
    resourceVersion: "10059"
    selfLink: /api/v1/namespaces/default/events/hello-world-779cbf9f67-nbwfm.164d46640ec14f6a
    uid: 5f0b590e-ba00-4dd2-9cd0-00537763a0c4
  reason: Killing
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: kubelet
    host: node2
  type: Normal
`, `
  apiVersion: v1
  count: 1
  eventTime: null
  firstTimestamp: "2020-12-03T17:53:57Z"
  involvedObject:
    apiVersion: apps/v1
    kind: ReplicaSet
    name: hello-world-779cbf9f67
    namespace: default
    resourceVersion: "10055"
    uid: 223916cb-845a-4ac3-b589-91fb02a6ea2e
  kind: Event
  lastTimestamp: "2020-12-03T17:53:57Z"
  message: 'Deleted pod: hello-world-779cbf9f67-nbwfm'
  metadata:
    creationTimestamp: "2020-12-03T17:53:57Z"
    name: hello-world-779cbf9f67.164d46640f0468d7
    namespace: default
    resourceVersion: "10060"
    selfLink: /api/v1/namespaces/default/events/hello-world-779cbf9f67.164d46640f0468d7
    uid: b41275a2-7715-46a0-a145-96f1ef187410
  reason: SuccessfulDelete
  reportingComponent: ""
  reportingInstance: ""
  source:
    component: replicaset-controller
  type: Normal
`,
}

var fluxReplicaSet1astr = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  annotations:
    deployment.kubernetes.io/desired-replicas: "1"
    deployment.kubernetes.io/max-replicas: "2"
    deployment.kubernetes.io/revision: "1"
    fluxcd.io/sync-checksum: f74e2e342655750a0146bf9bf983a57a44ec92cf
  creationTimestamp: "2020-12-03T16:26:59Z"
  generation: 2
  labels:
    name: hello-world
    pod-template-hash: 779cbf9f67
  name: hello-world-779cbf9f67
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Deployment
    name: hello-world
    uid: 06790b30-23ce-411c-bb6d-4059ec328e1a
  resourceVersion: "10062"
  selfLink: /apis/apps/v1/namespaces/default/replicasets/hello-world-779cbf9f67
  uid: 223916cb-845a-4ac3-b589-91fb02a6ea2e
spec:
  replicas: 0
  selector:
    matchLabels:
      name: hello-world
      pod-template-hash: 779cbf9f67
  template:
    metadata:
      creationTimestamp: null
      labels:
        name: hello-world
        pod-template-hash: 779cbf9f67
    spec:
      containers:
      - image: nginx:1.19.2-alpine
        imagePullPolicy: IfNotPresent
        name: hello-world
        ports:
        - containerPort: 80
          protocol: TCP
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  observedGeneration: 2
  replicas: 0`

var fluxReplicaSet1bstr = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  annotations:
    deployment.kubernetes.io/desired-replicas: "1"
    deployment.kubernetes.io/max-replicas: "2"
    deployment.kubernetes.io/revision: "2"
    fluxcd.io/sync-checksum: cd2ade7e07de56b6a686c0c46c3f344c75953070
  creationTimestamp: "2020-12-03T17:53:51Z"
  generation: 1
  labels:
    name: hello-world
    pod-template-hash: f77b4f6c8
  name: hello-world-f77b4f6c8
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Deployment
    name: hello-world
    uid: 06790b30-23ce-411c-bb6d-4059ec328e1a
  resourceVersion: "10054"
  selfLink: /apis/apps/v1/namespaces/default/replicasets/hello-world-f77b4f6c8
  uid: 01482a68-bdd2-423e-8cda-d0421f418758
spec:
  replicas: 1
  selector:
    matchLabels:
      name: hello-world
      pod-template-hash: f77b4f6c8
  template:
    metadata:
      creationTimestamp: null
      labels:
        name: hello-world
        pod-template-hash: f77b4f6c8
    spec:
      containers:
      - image: nginx:1.19.3-alpine
        imagePullPolicy: IfNotPresent
        name: hello-world
        ports:
        - containerPort: 80
          protocol: TCP
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  fullyLabeledReplicas: 1
  observedGeneration: 1
  readyReplicas: 1
`

var fluxDeploymentStr = `
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "2"
    fluxcd.io/sync-checksum: cd2ade7e07de56b6a686c0c46c3f344c75953070
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{"fluxcd.io/sync-checksum":"cd2ade7e07de56b6a686c0c46c3f344c75953070"},"labels":{"fluxcd.io/sync-gc-mark":"sha256.3OUfC7OWDQ3jHPMCtb8Xm0EwoWiKMAgAMEwEJeYLpOA"},"name":"hello-world","namespace":"default"},"spec":{"replicas":1,"selector":{"matchLabels":{"name":"hello-world"}},"template":{"metadata":{"labels":{"name":"hello-world"}},"spec":{"containers":[{"image":"nginx:1.19.3-alpine","name":"hello-world","ports":[{"containerPort":80}]}]}}}}
  creationTimestamp: "2020-12-03T16:26:59Z"
  generation: 2
  labels:
    fluxcd.io/sync-gc-mark: sha256.3OUfC7OWDQ3jHPMCtb8Xm0EwoWiKMAgAMEwEJeYLpOA
  name: hello-world
  namespace: default
  resourceVersion: "10063"
  selfLink: /apis/apps/v1/namespaces/default/deployments/hello-world
  uid: 06790b30-23ce-411c-bb6d-4059ec328e1a
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      name: hello-world
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      creationTimestamp: null
      labels:
        name: hello-world
    spec:
      containers:
      - image: nginx:1.19.3-alpine
        imagePullPolicy: IfNotPresent
        name: hello-world
        ports:
        - containerPort: 80
          protocol: TCP
        resources: {}
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2020-12-03T16:27:05Z"
    lastUpdateTime: "2020-12-03T16:27:05Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2020-12-03T16:26:59Z"
    lastUpdateTime: "2020-12-03T17:53:57Z"
    message: ReplicaSet "hello-world-f77b4f6c8" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 2
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1`

var fluxPod1astr = `
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: "2020-12-03T17:53:51Z"
  generateName: hello-world-f77b4f6c8-
  labels:
    name: hello-world
    pod-template-hash: f77b4f6c8
  name: hello-world-f77b4f6c8-6tcj2
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: hello-world-f77b4f6c8
    uid: 01482a68-bdd2-423e-8cda-d0421f418758
  resourceVersion: "10053"
  selfLink: /api/v1/namespaces/default/pods/hello-world-f77b4f6c8-6tcj2
  uid: 95c52c1f-c369-44da-8075-1b82b92f0e46
spec:
  containers:
  - image: nginx:1.19.3-alpine
    imagePullPolicy: IfNotPresent
    name: hello-world
    ports:
    - containerPort: 80
      protocol: TCP
    resources: {}
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-jmjmm
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: node2
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: default-token-jmjmm
    secret:
      defaultMode: 420
      secretName: default-token-jmjmm
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2020-12-03T17:53:51Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2020-12-03T17:53:57Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2020-12-03T17:53:57Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2020-12-03T17:53:51Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: docker://9eca0bda241afbb87c1e64e9ca9fb3a331346b835d82b8b3f0208ab130b52686
    image: nginx:1.19.3-alpine
    imageID: docker-pullable://nginx@sha256:5aa44b407756b274a600c7399418bdfb1d02c33317ae27fd5e8a333afb115db1
    lastState: {}
    name: hello-world
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2020-12-03T17:53:56Z"
  hostIP: 172.17.0.7
  phase: Running
  podIP: 172.30.192.3
  podIPs:
  - ip: 172.30.192.3
  qosClass: BestEffort
  startTime: "2020-12-03T17:53:51Z"
  `

//----------------------------------------------------------------------------------
// Events from a two-pod deployment rolling update
var p2deploymentUpdateEvents = []string{`
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: apps/v1
  kind: Deployment
  name: bryan-podinfo
  namespace: default
  resourceVersion: "1611421"
  uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: Scaled up replica set bryan-podinfo-5c5df9754b to 1
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo.165ebba1f2678aa8
  namespace: default
  resourceVersion: "1611424"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo.165ebba1f2678aa8
  uid: 3e558875-b9e8-4fa7-9818-c70e231eda83
reason: ScalingReplicaSet
reportingComponent: ""
reportingInstance: ""
source:
  component: deployment-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: apps/v1
  kind: ReplicaSet
  name: bryan-podinfo-5c5df9754b
  namespace: default
  resourceVersion: "1611422"
  uid: 05072034-2449-457f-9110-70517854f927
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: 'Created pod: bryan-podinfo-5c5df9754b-4w2hj'
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo-5c5df9754b.165ebba1f2eeaad9
  namespace: default
  resourceVersion: "1611426"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b.165ebba1f2eeaad9
  uid: 7c984095-ba82-4220-ac52-0a543243fd7c
reason: SuccessfulCreate
reportingComponent: ""
reportingInstance: ""
source:
  component: replicaset-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: v1
  kind: Pod
  name: bryan-podinfo-5c5df9754b-4w2hj
  namespace: default
  resourceVersion: "1611425"
  uid: 958b118d-9e36-466b-8ca5-40d2a04e04d5
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: Successfully assigned default/bryan-podinfo-5c5df9754b-4w2hj to kind-control-plane
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo-5c5df9754b-4w2hj.165ebba1f35ef17d
  namespace: default
  resourceVersion: "1611430"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-4w2hj.165ebba1f35ef17d
  uid: 27f40b15-f116-4691-ad69-d699e83800fa
reason: Scheduled
reportingComponent: ""
reportingInstance: ""
source:
  component: default-scheduler
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: apps/v1
  kind: ReplicaSet
  name: bryan-podinfo-787c9986b5
  namespace: default
  resourceVersion: "1611427"
  uid: 548a11d4-3f6f-43eb-baad-07c7c17b71c6
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: 'Deleted pod: bryan-podinfo-787c9986b5-tkd9p'
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo-787c9986b5.165ebba1f4a2c54c
  namespace: default
  resourceVersion: "1611433"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-787c9986b5.165ebba1f4a2c54c
  uid: 98c81744-e539-4ac6-9d7c-f60447036869
reason: SuccessfulDelete
reportingComponent: ""
reportingInstance: ""
source:
  component: replicaset-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-787c9986b5-tkd9p
  namespace: default
  resourceVersion: "1412655"
  uid: d932d275-e404-4a9b-b560-9258d8834a90
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: Stopping container podinfod
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo-787c9986b5-tkd9p.165ebba1f4af5de8
  namespace: default
  resourceVersion: "1611435"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-787c9986b5-tkd9p.165ebba1f4af5de8
  uid: 3cd55e29-4ce1-49f2-9ad2-6183c2e18b98
reason: Killing
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 3
eventTime: null
firstTimestamp: "2021-01-28T16:59:06Z"
involvedObject:
  apiVersion: apps/v1
  kind: Deployment
  name: bryan-podinfo
  namespace: default
  resourceVersion: "1611421"
  uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: Scaled down replica set bryan-podinfo-787c9986b5 to 1
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo.165e73e581c6c81c
  namespace: default
  resourceVersion: "1611436"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo.165e73e581c6c81c
  uid: a9422298-c49c-4fd9-ad61-4a8e2241773d
reason: ScalingReplicaSet
reportingComponent: ""
reportingInstance: ""
source:
  component: deployment-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: apps/v1
  kind: Deployment
  name: bryan-podinfo
  namespace: default
  resourceVersion: "1611423"
  uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: Scaled up replica set bryan-podinfo-5c5df9754b to 2
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo.165ebba1f65798a6
  namespace: default
  resourceVersion: "1611440"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo.165ebba1f65798a6
  uid: 53b624a3-54fe-4bbf-9708-0449024546e3
reason: ScalingReplicaSet
reportingComponent: ""
reportingInstance: ""
source:
  component: deployment-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: apps/v1
  kind: ReplicaSet
  name: bryan-podinfo-5c5df9754b
  namespace: default
  resourceVersion: "1611439"
  uid: 05072034-2449-457f-9110-70517854f927
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: 'Created pod: bryan-podinfo-5c5df9754b-bhj4w'
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo-5c5df9754b.165ebba1f6bfff5c
  namespace: default
  resourceVersion: "1611444"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b.165ebba1f6bfff5c
  uid: 27ba545e-f4fd-4ac8-8d28-d958fceb9c1d
reason: SuccessfulCreate
reportingComponent: ""
reportingInstance: ""
source:
  component: replicaset-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:40Z"
involvedObject:
  apiVersion: v1
  kind: Pod
  name: bryan-podinfo-5c5df9754b-bhj4w
  namespace: default
  resourceVersion: "1611442"
  uid: b17ddbbe-b2f2-459e-a391-0c5efe4b76d4
kind: Event
lastTimestamp: "2021-01-29T14:53:40Z"
message: Successfully assigned default/bryan-podinfo-5c5df9754b-bhj4w to kind-control-plane
metadata:
  creationTimestamp: "2021-01-29T14:53:40Z"
  name: bryan-podinfo-5c5df9754b-bhj4w.165ebba1f7236fe9
  namespace: default
  resourceVersion: "1611447"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-bhj4w.165ebba1f7236fe9
  uid: f72ad0ce-2f14-410f-8f69-006d5b195ba9
reason: Scheduled
reportingComponent: ""
reportingInstance: ""
source:
  component: default-scheduler
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:41Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-4w2hj
  namespace: default
  resourceVersion: "1611428"
  uid: 958b118d-9e36-466b-8ca5-40d2a04e04d5
kind: Event
lastTimestamp: "2021-01-29T14:53:41Z"
message: Pulling image "ghcr.io/stefanprodan/podinfo:5.0.3"
metadata:
  creationTimestamp: "2021-01-29T14:53:41Z"
  name: bryan-podinfo-5c5df9754b-4w2hj.165ebba211681482
  namespace: default
  resourceVersion: "1611451"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-4w2hj.165ebba211681482
  uid: bb755e32-cf06-4c66-99f4-ede8c91df6fd
reason: Pulling
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:41Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-bhj4w
  namespace: default
  resourceVersion: "1611445"
  uid: b17ddbbe-b2f2-459e-a391-0c5efe4b76d4
kind: Event
lastTimestamp: "2021-01-29T14:53:41Z"
message: Pulling image "ghcr.io/stefanprodan/podinfo:5.0.3"
metadata:
  creationTimestamp: "2021-01-29T14:53:41Z"
  name: bryan-podinfo-5c5df9754b-bhj4w.165ebba214657257
  namespace: default
  resourceVersion: "1611452"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-bhj4w.165ebba214657257
  uid: fd343dd5-90e0-4fca-8d37-7a070ac0952a
reason: Pulling
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:48Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-4w2hj
  namespace: default
  resourceVersion: "1611428"
  uid: 958b118d-9e36-466b-8ca5-40d2a04e04d5
kind: Event
lastTimestamp: "2021-01-29T14:53:48Z"
message: Successfully pulled image "ghcr.io/stefanprodan/podinfo:5.0.3" in 7.556422631s
metadata:
  creationTimestamp: "2021-01-29T14:53:48Z"
  name: bryan-podinfo-5c5df9754b-4w2hj.165ebba3d3ce1b9c
  namespace: default
  resourceVersion: "1611476"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-4w2hj.165ebba3d3ce1b9c
  uid: 8caf0547-a31d-4c71-bc0c-ec145178161a
reason: Pulled
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:49Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-4w2hj
  namespace: default
  resourceVersion: "1611428"
  uid: 958b118d-9e36-466b-8ca5-40d2a04e04d5
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: Created container podinfod
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b-4w2hj.165ebba3d777efc8
  namespace: default
  resourceVersion: "1611477"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-4w2hj.165ebba3d777efc8
  uid: a0cfbcbf-6fd0-4d9e-a8c1-ea71e595f25f
reason: Created
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:49Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-4w2hj
  namespace: default
  resourceVersion: "1611428"
  uid: 958b118d-9e36-466b-8ca5-40d2a04e04d5
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: Started container podinfod
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b-4w2hj.165ebba3dbc2684e
  namespace: default
  resourceVersion: "1611478"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-4w2hj.165ebba3dbc2684e
  uid: c7d2f53b-9bb5-4d85-98e3-c91517a3e947
reason: Started
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:49Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-bhj4w
  namespace: default
  resourceVersion: "1611445"
  uid: b17ddbbe-b2f2-459e-a391-0c5efe4b76d4
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: Successfully pulled image "ghcr.io/stefanprodan/podinfo:5.0.3" in 8.129591184s
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b-bhj4w.165ebba3f8f55017
  namespace: default
  resourceVersion: "1611479"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-bhj4w.165ebba3f8f55017
  uid: 49737621-f199-49ee-a46e-44e0a1c277e2
reason: Pulled
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:49Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-bhj4w
  namespace: default
  resourceVersion: "1611445"
  uid: b17ddbbe-b2f2-459e-a391-0c5efe4b76d4
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: Created container podinfod
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b-bhj4w.165ebba3fa321d97
  namespace: default
  resourceVersion: "1611480"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-bhj4w.165ebba3fa321d97
  uid: 765e2d59-ff78-4b81-9327-7239f7549f8d
reason: Created
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:49Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-5c5df9754b-bhj4w
  namespace: default
  resourceVersion: "1611445"
  uid: b17ddbbe-b2f2-459e-a391-0c5efe4b76d4
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: Started container podinfod
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b-bhj4w.165ebba3ff64a8d1
  namespace: default
  resourceVersion: "1611481"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-5c5df9754b-bhj4w.165ebba3ff64a8d1
  uid: 04607530-2579-4e1e-9b5b-918faf2ce8ba
reason: Started
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 2
eventTime: null
firstTimestamp: "2021-01-28T18:06:03Z"
involvedObject:
  apiVersion: apps/v1
  kind: Deployment
  name: bryan-podinfo
  namespace: default
  resourceVersion: "1611450"
  uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: Scaled down replica set bryan-podinfo-787c9986b5 to 0
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo.165e778cd8e469b5
  namespace: default
  resourceVersion: "1611488"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo.165e778cd8e469b5
  uid: 5a6c866d-851f-4c36-b410-aefb3559b211
reason: ScalingReplicaSet
reportingComponent: ""
reportingInstance: ""
source:
  component: deployment-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:49Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{podinfod}
  kind: Pod
  name: bryan-podinfo-787c9986b5-fws9t
  namespace: default
  resourceVersion: "1412667"
  uid: 02eb41fb-4201-41e7-9db3-b903b8689e08
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: Stopping container podinfod
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-787c9986b5-fws9t.165ebba40cc3d8d8
  namespace: default
  resourceVersion: "1611492"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-787c9986b5-fws9t.165ebba40cc3d8d8
  uid: 0ac1096e-03d6-4d58-b144-ad0e3b5684a1
reason: Killing
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: kind-control-plane
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-01-29T14:53:49Z"
involvedObject:
  apiVersion: apps/v1
  kind: ReplicaSet
  name: bryan-podinfo-787c9986b5
  namespace: default
  resourceVersion: "1611484"
  uid: 548a11d4-3f6f-43eb-baad-07c7c17b71c6
kind: Event
lastTimestamp: "2021-01-29T14:53:49Z"
message: 'Deleted pod: bryan-podinfo-787c9986b5-fws9t'
metadata:
  creationTimestamp: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-787c9986b5.165ebba40ccf1289
  namespace: default
  resourceVersion: "1611493"
  selfLink: /api/v1/namespaces/default/events/bryan-podinfo-787c9986b5.165ebba40ccf1289
  uid: 2054b1b3-1c85-42e5-ae23-93d9ce29122e
reason: SuccessfulDelete
reportingComponent: ""
reportingInstance: ""
source:
  component: replicaset-controller
type: Normal
`}

// 2 seconds after the last event time
const p2deploymentUpdateEventsThresholdStr = "2021-01-29T14:53:51Z"

var p2pod1str = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    prometheus.io/port: "9797"
    prometheus.io/scrape: "true"
  creationTimestamp: "2021-01-28T18:18:20Z"
  generateName: bryan-podinfo-787c9986b5-
  labels:
    app: podinfo
    pod-template-hash: 787c9986b5
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:prometheus.io/port: {}
          f:prometheus.io/scrape: {}
        f:generateName: {}
        f:labels:
          .: {}
          f:app: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"548a11d4-3f6f-43eb-baad-07c7c17b71c6"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:containers:
          k:{"name":"podinfod"}:
            .: {}
            f:command: {}
            f:env:
              .: {}
              k:{"name":"GOGC"}:
                .: {}
                f:name: {}
                f:value: {}
            f:image: {}
            f:imagePullPolicy: {}
            f:name: {}
            f:resources:
              .: {}
              f:limits:
                .: {}
                f:cpu: {}
                f:memory: {}
              f:requests:
                .: {}
                f:cpu: {}
                f:memory: {}
            f:terminationMessagePath: {}
            f:terminationMessagePolicy: {}
        f:dnsPolicy: {}
        f:enableServiceLinks: {}
        f:restartPolicy: {}
        f:schedulerName: {}
        f:securityContext: {}
        f:terminationGracePeriodSeconds: {}
        f:tolerations: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-28T18:18:20Z"
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        f:conditions:
          k:{"type":"ContainersReady"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Initialized"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Ready"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
        f:containerStatuses: {}
        f:hostIP: {}
        f:phase: {}
        f:podIP: {}
        f:podIPs:
          .: {}
          k:{"ip":"10.244.0.13"}:
            .: {}
            f:ip: {}
        f:startTime: {}
    manager: kubelet
    operation: Update
    time: "2021-01-28T18:18:21Z"
  name: bryan-podinfo-787c9986b5-fws9t
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: bryan-podinfo-787c9986b5
    uid: 548a11d4-3f6f-43eb-baad-07c7c17b71c6
  resourceVersion: "1412681"
  selfLink: /api/v1/namespaces/default/pods/bryan-podinfo-787c9986b5-fws9t
  uid: 02eb41fb-4201-41e7-9db3-b903b8689e08
spec:
  containers:
  - command:
    - ./podinfo
    - --port-metrics=9797
    env:
    - name: GOGC
      value: "100"
    image: ghcr.io/stefanprodan/podinfo:5.1.2
    imagePullPolicy: IfNotPresent
    name: podinfod
    resources:
      limits:
        cpu: "2"
        memory: 127Mi
      requests:
        cpu: 200m
        memory: 100Mi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-tkz6n
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: kind-control-plane
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoSchedule
    operator: Exists
  - effect: NoExecute
    operator: Exists
  volumes:
  - name: default-token-tkz6n
    secret:
      defaultMode: 420
      secretName: default-token-tkz6n
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:20Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:21Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:21Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:20Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: containerd://caf42259e0f766e0f509c42e775fc02aece37d55b0a0864f392142d7f1c094ca
    image: ghcr.io/stefanprodan/podinfo:5.1.2
    imageID: ghcr.io/stefanprodan/podinfo@sha256:fbdde7bc6ae32ddac92131165ea46ce688e1d0be98a808c02fb78393d85750a8
    lastState: {}
    name: podinfod
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2021-01-28T18:18:21Z"
  hostIP: 172.18.0.2
  phase: Running
  podIP: 10.244.0.13
  podIPs:
  - ip: 10.244.0.13
  qosClass: Burstable
  startTime: "2021-01-28T18:18:20Z"
  `

var p2pod2str = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    prometheus.io/port: "9797"
    prometheus.io/scrape: "true"
  creationTimestamp: "2021-01-28T18:18:20Z"
  generateName: bryan-podinfo-787c9986b5-
  labels:
    app: podinfo
    pod-template-hash: 787c9986b5
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:prometheus.io/port: {}
          f:prometheus.io/scrape: {}
        f:generateName: {}
        f:labels:
          .: {}
          f:app: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"548a11d4-3f6f-43eb-baad-07c7c17b71c6"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:containers:
          k:{"name":"podinfod"}:
            .: {}
            f:command: {}
            f:env:
              .: {}
              k:{"name":"GOGC"}:
                .: {}
                f:name: {}
                f:value: {}
            f:image: {}
            f:imagePullPolicy: {}
            f:name: {}
            f:resources:
              .: {}
              f:limits:
                .: {}
                f:cpu: {}
                f:memory: {}
              f:requests:
                .: {}
                f:cpu: {}
                f:memory: {}
            f:terminationMessagePath: {}
            f:terminationMessagePolicy: {}
        f:dnsPolicy: {}
        f:enableServiceLinks: {}
        f:restartPolicy: {}
        f:schedulerName: {}
        f:securityContext: {}
        f:terminationGracePeriodSeconds: {}
        f:tolerations: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-28T18:18:20Z"
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        f:conditions:
          k:{"type":"ContainersReady"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Initialized"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Ready"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
        f:containerStatuses: {}
        f:hostIP: {}
        f:phase: {}
        f:podIP: {}
        f:podIPs:
          .: {}
          k:{"ip":"10.244.0.12"}:
            .: {}
            f:ip: {}
        f:startTime: {}
    manager: kubelet
    operation: Update
    time: "2021-01-28T18:18:21Z"
  name: bryan-podinfo-787c9986b5-tkd9p
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: bryan-podinfo-787c9986b5
    uid: 548a11d4-3f6f-43eb-baad-07c7c17b71c6
  resourceVersion: "1412683"
  selfLink: /api/v1/namespaces/default/pods/bryan-podinfo-787c9986b5-tkd9p
  uid: d932d275-e404-4a9b-b560-9258d8834a90
spec:
  containers:
  - command:
    - ./podinfo
    - --port-metrics=9797
    env:
    - name: GOGC
      value: "100"
    image: ghcr.io/stefanprodan/podinfo:5.1.2
    imagePullPolicy: IfNotPresent
    name: podinfod
    resources:
      limits:
        cpu: "2"
        memory: 127Mi
      requests:
        cpu: 200m
        memory: 100Mi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-tkz6n
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: kind-control-plane
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoSchedule
    operator: Exists
  - effect: NoExecute
    operator: Exists
  volumes:
  - name: default-token-tkz6n
    secret:
      defaultMode: 420
      secretName: default-token-tkz6n
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:20Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:21Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:21Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2021-01-28T18:18:20Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: containerd://4f29ed67f451255545dc9d5a56ddd8e1cda65c32ea92d13f4bd54c530531c6ed
    image: ghcr.io/stefanprodan/podinfo:5.1.2
    imageID: ghcr.io/stefanprodan/podinfo@sha256:fbdde7bc6ae32ddac92131165ea46ce688e1d0be98a808c02fb78393d85750a8
    lastState: {}
    name: podinfod
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2021-01-28T18:18:21Z"
  hostIP: 172.18.0.2
  phase: Running
  podIP: 10.244.0.12
  podIPs:
  - ip: 10.244.0.12
  qosClass: Burstable
  startTime: "2021-01-28T18:18:20Z"
`

var p2replicaSet1str = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  annotations:
    deployment.kubernetes.io/desired-replicas: "2"
    deployment.kubernetes.io/max-replicas: "3"
    deployment.kubernetes.io/revision: "5"
    deployment.kubernetes.io/revision-history: "2"
  creationTimestamp: "2021-01-28T16:58:40Z"
  generation: 8
  labels:
    app: podinfo
    pod-template-hash: 787c9986b5
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:deployment.kubernetes.io/desired-replicas: {}
          f:deployment.kubernetes.io/max-replicas: {}
          f:deployment.kubernetes.io/revision: {}
          f:deployment.kubernetes.io/revision-history: {}
        f:labels:
          .: {}
          f:app: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"0bf707bd-fb6f-4682-8a0d-d06ea200633f"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:replicas: {}
        f:selector:
          f:matchLabels:
            .: {}
            f:app: {}
            f:pod-template-hash: {}
        f:template:
          f:metadata:
            f:annotations:
              .: {}
              f:prometheus.io/port: {}
              f:prometheus.io/scrape: {}
            f:labels:
              .: {}
              f:app: {}
              f:pod-template-hash: {}
          f:spec:
            f:containers:
              k:{"name":"podinfod"}:
                .: {}
                f:command: {}
                f:env:
                  .: {}
                  k:{"name":"GOGC"}:
                    .: {}
                    f:name: {}
                    f:value: {}
                f:image: {}
                f:imagePullPolicy: {}
                f:name: {}
                f:resources:
                  .: {}
                  f:limits:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                  f:requests:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
            f:dnsPolicy: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:terminationGracePeriodSeconds: {}
            f:tolerations: {}
      f:status:
        f:availableReplicas: {}
        f:fullyLabeledReplicas: {}
        f:observedGeneration: {}
        f:readyReplicas: {}
        f:replicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-28T18:18:21Z"
  name: bryan-podinfo-787c9986b5
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Deployment
    name: bryan-podinfo
    uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
  resourceVersion: "1412685"
  selfLink: /apis/apps/v1/namespaces/default/replicasets/bryan-podinfo-787c9986b5
  uid: 548a11d4-3f6f-43eb-baad-07c7c17b71c6
spec:
  replicas: 2
  selector:
    matchLabels:
      app: podinfo
      pod-template-hash: 787c9986b5
  template:
    metadata:
      annotations:
        prometheus.io/port: "9797"
        prometheus.io/scrape: "true"
      creationTimestamp: null
      labels:
        app: podinfo
        pod-template-hash: 787c9986b5
    spec:
      containers:
      - command:
        - ./podinfo
        - --port-metrics=9797
        env:
        - name: GOGC
          value: "100"
        image: ghcr.io/stefanprodan/podinfo:5.1.2
        imagePullPolicy: IfNotPresent
        name: podinfod
        resources:
          limits:
            cpu: "2"
            memory: 127Mi
          requests:
            cpu: 200m
            memory: 100Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
status:
  availableReplicas: 2
  fullyLabeledReplicas: 2
  observedGeneration: 8
  readyReplicas: 2
  replicas: 2
`

var p2deployment1 = `
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "5"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"name":"bryan-podinfo","namespace":"default"},"spec":{"replicas":2,"selector":{"matchLabels":{"app":"podinfo"}},"strategy":{"rollingUpdate":{"maxSurge":1,"maxUnavailable":1}},"template":{"metadata":{"annotations":{"prometheus.io/port":"9797","prometheus.io/scrape":"true"},"labels":{"app":"podinfo"}},"spec":{"containers":[{"command":["./podinfo","--port-metrics=9797"],"env":[{"name":"GOGC","value":"100"}],"image":"ghcr.io/stefanprodan/podinfo:5.1.2","name":"podinfod","resources":{"limits":{"cpu":"2","memory":"127Mi"},"requests":{"cpu":"200m","memory":"100Mi"}}}],"tolerations":[{"effect":"NoSchedule","operator":"Exists"},{"effect":"NoExecute","operator":"Exists"}]}}}}
  creationTimestamp: "2021-01-28T16:57:00Z"
  generation: 8
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        f:progressDeadlineSeconds: {}
        f:replicas: {}
        f:revisionHistoryLimit: {}
        f:selector:
          f:matchLabels:
            .: {}
            f:app: {}
        f:strategy:
          f:rollingUpdate:
            .: {}
            f:maxSurge: {}
            f:maxUnavailable: {}
          f:type: {}
        f:template:
          f:metadata:
            f:annotations:
              .: {}
              f:prometheus.io/port: {}
              f:prometheus.io/scrape: {}
            f:labels:
              .: {}
              f:app: {}
          f:spec:
            f:containers:
              k:{"name":"podinfod"}:
                .: {}
                f:command: {}
                f:env:
                  .: {}
                  k:{"name":"GOGC"}:
                    .: {}
                    f:name: {}
                    f:value: {}
                f:image: {}
                f:imagePullPolicy: {}
                f:name: {}
                f:resources:
                  .: {}
                  f:limits:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                  f:requests:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
            f:dnsPolicy: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:terminationGracePeriodSeconds: {}
            f:tolerations: {}
    manager: kubectl-client-side-apply
    operation: Update
    time: "2021-01-28T18:18:20Z"
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          f:deployment.kubernetes.io/revision: {}
      f:status:
        f:availableReplicas: {}
        f:conditions:
          .: {}
          k:{"type":"Available"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
          k:{"type":"Progressing"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
        f:observedGeneration: {}
        f:readyReplicas: {}
        f:replicas: {}
        f:updatedReplicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-28T18:18:21Z"
  name: bryan-podinfo
  namespace: default
  resourceVersion: "1412694"
  selfLink: /apis/apps/v1/namespaces/default/deployments/bryan-podinfo
  uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
spec:
  progressDeadlineSeconds: 600
  replicas: 2
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: podinfo
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      annotations:
        prometheus.io/port: "9797"
        prometheus.io/scrape: "true"
      creationTimestamp: null
      labels:
        app: podinfo
    spec:
      containers:
      - command:
        - ./podinfo
        - --port-metrics=9797
        env:
        - name: GOGC
          value: "100"
        image: ghcr.io/stefanprodan/podinfo:5.1.2
        imagePullPolicy: IfNotPresent
        name: podinfod
        resources:
          limits:
            cpu: "2"
            memory: 127Mi
          requests:
            cpu: 200m
            memory: 100Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
status:
  availableReplicas: 2
  conditions:
  - lastTransitionTime: "2021-01-28T16:57:00Z"
    lastUpdateTime: "2021-01-28T16:57:00Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2021-01-28T16:57:00Z"
    lastUpdateTime: "2021-01-28T18:18:21Z"
    message: ReplicaSet "bryan-podinfo-787c9986b5" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 8
  readyReplicas: 2
  replicas: 2
  updatedReplicas: 2
`

var p2pod3str = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    prometheus.io/port: "9797"
    prometheus.io/scrape: "true"
  creationTimestamp: "2021-01-29T14:53:40Z"
  generateName: bryan-podinfo-5c5df9754b-
  labels:
    app: podinfo
    pod-template-hash: 5c5df9754b
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:prometheus.io/port: {}
          f:prometheus.io/scrape: {}
        f:generateName: {}
        f:labels:
          .: {}
          f:app: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"05072034-2449-457f-9110-70517854f927"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:containers:
          k:{"name":"podinfod"}:
            .: {}
            f:command: {}
            f:env:
              .: {}
              k:{"name":"GOGC"}:
                .: {}
                f:name: {}
                f:value: {}
            f:image: {}
            f:imagePullPolicy: {}
            f:name: {}
            f:resources:
              .: {}
              f:limits:
                .: {}
                f:cpu: {}
                f:memory: {}
              f:requests:
                .: {}
                f:cpu: {}
                f:memory: {}
            f:terminationMessagePath: {}
            f:terminationMessagePolicy: {}
        f:dnsPolicy: {}
        f:enableServiceLinks: {}
        f:restartPolicy: {}
        f:schedulerName: {}
        f:securityContext: {}
        f:terminationGracePeriodSeconds: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-29T14:53:40Z"
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        f:conditions:
          k:{"type":"ContainersReady"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Initialized"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Ready"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
        f:containerStatuses: {}
        f:hostIP: {}
        f:phase: {}
        f:podIP: {}
        f:podIPs:
          .: {}
          k:{"ip":"10.244.0.14"}:
            .: {}
            f:ip: {}
        f:startTime: {}
    manager: kubelet
    operation: Update
    time: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b-4w2hj
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: bryan-podinfo-5c5df9754b
    uid: 05072034-2449-457f-9110-70517854f927
  resourceVersion: "1611482"
  selfLink: /api/v1/namespaces/default/pods/bryan-podinfo-5c5df9754b-4w2hj
  uid: 958b118d-9e36-466b-8ca5-40d2a04e04d5
spec:
  containers:
  - command:
    - ./podinfo
    - --port-metrics=9797
    env:
    - name: GOGC
      value: "100"
    image: ghcr.io/stefanprodan/podinfo:5.0.3
    imagePullPolicy: IfNotPresent
    name: podinfod
    resources:
      limits:
        cpu: "2"
        memory: 127Mi
      requests:
        cpu: 200m
        memory: 100Mi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-tkz6n
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: kind-control-plane
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: default-token-tkz6n
    secret:
      defaultMode: 420
      secretName: default-token-tkz6n
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:40Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:49Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:49Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:40Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: containerd://3fe4c96ea25b28261f5e4ca9defff5dfe8b484c6984f1291c96dccc4b5a75909
    image: ghcr.io/stefanprodan/podinfo:5.0.3
    imageID: ghcr.io/stefanprodan/podinfo@sha256:8704da90172710d422af855049175c1a8295731cbe2ad3b9a1c1074feecf8c10
    lastState: {}
    name: podinfod
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2021-01-29T14:53:49Z"
  hostIP: 172.18.0.2
  phase: Running
  podIP: 10.244.0.14
  podIPs:
  - ip: 10.244.0.14
  qosClass: Burstable
  startTime: "2021-01-29T14:53:40Z"
`

var p2pod4str = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    prometheus.io/port: "9797"
    prometheus.io/scrape: "true"
  creationTimestamp: "2021-01-29T14:53:40Z"
  generateName: bryan-podinfo-5c5df9754b-
  labels:
    app: podinfo
    pod-template-hash: 5c5df9754b
  managedFields:
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:prometheus.io/port: {}
          f:prometheus.io/scrape: {}
        f:generateName: {}
        f:labels:
          .: {}
          f:app: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"05072034-2449-457f-9110-70517854f927"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:containers:
          k:{"name":"podinfod"}:
            .: {}
            f:command: {}
            f:env:
              .: {}
              k:{"name":"GOGC"}:
                .: {}
                f:name: {}
                f:value: {}
            f:image: {}
            f:imagePullPolicy: {}
            f:name: {}
            f:resources:
              .: {}
              f:limits:
                .: {}
                f:cpu: {}
                f:memory: {}
              f:requests:
                .: {}
                f:cpu: {}
                f:memory: {}
            f:terminationMessagePath: {}
            f:terminationMessagePolicy: {}
        f:dnsPolicy: {}
        f:enableServiceLinks: {}
        f:restartPolicy: {}
        f:schedulerName: {}
        f:securityContext: {}
        f:terminationGracePeriodSeconds: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-29T14:53:40Z"
  - apiVersion: v1
    fieldsType: FieldsV1
    fieldsV1:
      f:status:
        f:conditions:
          k:{"type":"ContainersReady"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Initialized"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
          k:{"type":"Ready"}:
            .: {}
            f:lastProbeTime: {}
            f:lastTransitionTime: {}
            f:status: {}
            f:type: {}
        f:containerStatuses: {}
        f:hostIP: {}
        f:phase: {}
        f:podIP: {}
        f:podIPs:
          .: {}
          k:{"ip":"10.244.0.15"}:
            .: {}
            f:ip: {}
        f:startTime: {}
    manager: kubelet
    operation: Update
    time: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b-bhj4w
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: bryan-podinfo-5c5df9754b
    uid: 05072034-2449-457f-9110-70517854f927
  resourceVersion: "1611485"
  selfLink: /api/v1/namespaces/default/pods/bryan-podinfo-5c5df9754b-bhj4w
  uid: b17ddbbe-b2f2-459e-a391-0c5efe4b76d4
spec:
  containers:
  - command:
    - ./podinfo
    - --port-metrics=9797
    env:
    - name: GOGC
      value: "100"
    image: ghcr.io/stefanprodan/podinfo:5.0.3
    imagePullPolicy: IfNotPresent
    name: podinfod
    resources:
      limits:
        cpu: "2"
        memory: 127Mi
      requests:
        cpu: 200m
        memory: 100Mi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-tkz6n
      readOnly: true
  dnsPolicy: ClusterFirst
  enableServiceLinks: true
  nodeName: kind-control-plane
  preemptionPolicy: PreemptLowerPriority
  priority: 0
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext: {}
  serviceAccount: default
  serviceAccountName: default
  terminationGracePeriodSeconds: 30
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: default-token-tkz6n
    secret:
      defaultMode: 420
      secretName: default-token-tkz6n
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:40Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:49Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:49Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2021-01-29T14:53:40Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: containerd://c140cbbdf6efc8aa60f1002077ea6a956abb5e63f4085e8044e5352aa2af336f
    image: ghcr.io/stefanprodan/podinfo:5.0.3
    imageID: ghcr.io/stefanprodan/podinfo@sha256:8704da90172710d422af855049175c1a8295731cbe2ad3b9a1c1074feecf8c10
    lastState: {}
    name: podinfod
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2021-01-29T14:53:49Z"
  hostIP: 172.18.0.2
  phase: Running
  podIP: 10.244.0.15
  podIPs:
  - ip: 10.244.0.15
  qosClass: Burstable
  startTime: "2021-01-29T14:53:40Z"
`

var p2replicaSet2str = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  annotations:
    deployment.kubernetes.io/desired-replicas: "2"
    deployment.kubernetes.io/max-replicas: "3"
    deployment.kubernetes.io/revision: "6"
  creationTimestamp: "2021-01-29T14:53:40Z"
  generation: 2
  labels:
    app: podinfo
    pod-template-hash: 5c5df9754b
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:deployment.kubernetes.io/desired-replicas: {}
          f:deployment.kubernetes.io/max-replicas: {}
          f:deployment.kubernetes.io/revision: {}
        f:labels:
          .: {}
          f:app: {}
          f:pod-template-hash: {}
        f:ownerReferences:
          .: {}
          k:{"uid":"0bf707bd-fb6f-4682-8a0d-d06ea200633f"}:
            .: {}
            f:apiVersion: {}
            f:blockOwnerDeletion: {}
            f:controller: {}
            f:kind: {}
            f:name: {}
            f:uid: {}
      f:spec:
        f:replicas: {}
        f:selector:
          f:matchLabels:
            .: {}
            f:app: {}
            f:pod-template-hash: {}
        f:template:
          f:metadata:
            f:annotations:
              .: {}
              f:prometheus.io/port: {}
              f:prometheus.io/scrape: {}
            f:labels:
              .: {}
              f:app: {}
              f:pod-template-hash: {}
          f:spec:
            f:containers:
              k:{"name":"podinfod"}:
                .: {}
                f:command: {}
                f:env:
                  .: {}
                  k:{"name":"GOGC"}:
                    .: {}
                    f:name: {}
                    f:value: {}
                f:image: {}
                f:imagePullPolicy: {}
                f:name: {}
                f:resources:
                  .: {}
                  f:limits:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                  f:requests:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
            f:dnsPolicy: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:terminationGracePeriodSeconds: {}
      f:status:
        f:availableReplicas: {}
        f:fullyLabeledReplicas: {}
        f:observedGeneration: {}
        f:readyReplicas: {}
        f:replicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-29T14:53:49Z"
  name: bryan-podinfo-5c5df9754b
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Deployment
    name: bryan-podinfo
    uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
  resourceVersion: "1611486"
  selfLink: /apis/apps/v1/namespaces/default/replicasets/bryan-podinfo-5c5df9754b
  uid: 05072034-2449-457f-9110-70517854f927
spec:
  replicas: 2
  selector:
    matchLabels:
      app: podinfo
      pod-template-hash: 5c5df9754b
  template:
    metadata:
      annotations:
        prometheus.io/port: "9797"
        prometheus.io/scrape: "true"
      creationTimestamp: null
      labels:
        app: podinfo
        pod-template-hash: 5c5df9754b
    spec:
      containers:
      - command:
        - ./podinfo
        - --port-metrics=9797
        env:
        - name: GOGC
          value: "100"
        image: ghcr.io/stefanprodan/podinfo:5.0.3
        imagePullPolicy: IfNotPresent
        name: podinfod
        resources:
          limits:
            cpu: "2"
            memory: 127Mi
          requests:
            cpu: 200m
            memory: 100Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 2
  fullyLabeledReplicas: 2
  observedGeneration: 2
  readyReplicas: 2
  replicas: 2
`

var p2deployment2 = `
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "6"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"name":"bryan-podinfo","namespace":"default"},"spec":{"replicas":2,"selector":{"matchLabels":{"app":"podinfo"}},"strategy":{"rollingUpdate":{"maxSurge":1,"maxUnavailable":1}},"template":{"metadata":{"annotations":{"prometheus.io/port":"9797","prometheus.io/scrape":"true"},"labels":{"app":"podinfo"}},"spec":{"containers":[{"command":["./podinfo","--port-metrics=9797"],"env":[{"name":"GOGC","value":"100"}],"image":"ghcr.io/stefanprodan/podinfo:5.0.3","name":"podinfod","resources":{"limits":{"cpu":"2","memory":"127Mi"},"requests":{"cpu":"200m","memory":"100Mi"}}}]}}}}
  creationTimestamp: "2021-01-28T16:57:00Z"
  generation: 9
  managedFields:
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .: {}
          f:kubectl.kubernetes.io/last-applied-configuration: {}
      f:spec:
        f:progressDeadlineSeconds: {}
        f:replicas: {}
        f:revisionHistoryLimit: {}
        f:selector:
          f:matchLabels:
            .: {}
            f:app: {}
        f:strategy:
          f:rollingUpdate:
            .: {}
            f:maxSurge: {}
            f:maxUnavailable: {}
          f:type: {}
        f:template:
          f:metadata:
            f:annotations:
              .: {}
              f:prometheus.io/port: {}
              f:prometheus.io/scrape: {}
            f:labels:
              .: {}
              f:app: {}
          f:spec:
            f:containers:
              k:{"name":"podinfod"}:
                .: {}
                f:command: {}
                f:env:
                  .: {}
                  k:{"name":"GOGC"}:
                    .: {}
                    f:name: {}
                    f:value: {}
                f:image: {}
                f:imagePullPolicy: {}
                f:name: {}
                f:resources:
                  .: {}
                  f:limits:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                  f:requests:
                    .: {}
                    f:cpu: {}
                    f:memory: {}
                f:terminationMessagePath: {}
                f:terminationMessagePolicy: {}
            f:dnsPolicy: {}
            f:restartPolicy: {}
            f:schedulerName: {}
            f:securityContext: {}
            f:terminationGracePeriodSeconds: {}
    manager: kubectl-client-side-apply
    operation: Update
    time: "2021-01-29T14:53:40Z"
  - apiVersion: apps/v1
    fieldsType: FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          f:deployment.kubernetes.io/revision: {}
      f:status:
        f:availableReplicas: {}
        f:conditions:
          .: {}
          k:{"type":"Available"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
          k:{"type":"Progressing"}:
            .: {}
            f:lastTransitionTime: {}
            f:lastUpdateTime: {}
            f:message: {}
            f:reason: {}
            f:status: {}
            f:type: {}
        f:observedGeneration: {}
        f:readyReplicas: {}
        f:replicas: {}
        f:updatedReplicas: {}
    manager: kube-controller-manager
    operation: Update
    time: "2021-01-29T14:53:49Z"
  name: bryan-podinfo
  namespace: default
  resourceVersion: "1611495"
  selfLink: /apis/apps/v1/namespaces/default/deployments/bryan-podinfo
  uid: 0bf707bd-fb6f-4682-8a0d-d06ea200633f
spec:
  progressDeadlineSeconds: 600
  replicas: 2
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app: podinfo
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      annotations:
        prometheus.io/port: "9797"
        prometheus.io/scrape: "true"
      creationTimestamp: null
      labels:
        app: podinfo
    spec:
      containers:
      - command:
        - ./podinfo
        - --port-metrics=9797
        env:
        - name: GOGC
          value: "100"
        image: ghcr.io/stefanprodan/podinfo:5.0.3
        imagePullPolicy: IfNotPresent
        name: podinfod
        resources:
          limits:
            cpu: "2"
            memory: 127Mi
          requests:
            cpu: 200m
            memory: 100Mi
        terminationMessagePath: /dev/termination-log
        terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 2
  conditions:
  - lastTransitionTime: "2021-01-28T16:57:00Z"
    lastUpdateTime: "2021-01-28T16:57:00Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2021-01-28T16:57:00Z"
    lastUpdateTime: "2021-01-29T14:53:49Z"
    message: ReplicaSet "bryan-podinfo-5c5df9754b" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 9
  readyReplicas: 2
  replicas: 2
  updatedReplicas: 2
`

// 2 seconds after the last event time
const stsUpdateEventsThresholdStr = "2021-02-17T14:24:53Z"

// Events from a Flux StatefulSet update from one version to another
var stsUpdateEvents = []string{`
apiVersion: v1
count: 7
eventTime: null
firstTimestamp: "2021-02-05T10:59:21Z"
involvedObject:
  apiVersion: apps/v1
  kind: StatefulSet
  name: ingester
  namespace: cortex
  resourceVersion: "291777939"
  uid: ded43c86-942a-4d5d-94ed-4505b4c02eb5
kind: Event
lastTimestamp: "2021-02-17T14:20:29Z"
message: delete Pod ingester-3 in StatefulSet ingester successful
metadata:
  creationTimestamp: "2021-02-17T14:20:29Z"
  name: ingester.1660d4e87d1b450d
  namespace: cortex
  resourceVersion: "291777943"
  selfLink: /api/v1/namespaces/cortex/events/ingester.1660d4e87d1b450d
  uid: 67902e38-cbac-4814-a781-b5e56421fbbd
reason: SuccessfulDelete
reportingComponent: ""
reportingInstance: ""
source:
  component: statefulset-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:20:29Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-3
  namespace: cortex
  resourceVersion: "291595271"
  uid: c17e806e-e9c8-49b1-acea-73ec2f33e522
kind: Event
lastTimestamp: "2021-02-17T14:20:29Z"
message: Stopping container ingester
metadata:
  creationTimestamp: "2021-02-17T14:20:29Z"
  name: ingester-3.16648ed91d347a99
  namespace: cortex
  resourceVersion: "291777944"
  selfLink: /api/v1/namespaces/cortex/events/ingester-3.16648ed91d347a99
  uid: 5731fb33-d61e-41ad-b473-bd2f30b1df45
reason: Killing
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-25.ec2.internal
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:20:30Z"
involvedObject:
  apiVersion: apps/v1
  kind: statefulset
  name: ingester
  namespace: cortex
kind: Event
lastTimestamp: "2021-02-17T14:20:30Z"
message: 'Commit fc4e825b46ac: Update ingester to latest, in dev'
metadata:
  annotations:
    topLevelSpan: 'true'
    syncMetadata: '{"commits":[{"revision":"fc4e825b46ac659138e227c6110f615ab89a5ecb","message":"Update
      ingester to latest, in dev"}],"includes":{"other":true}}'
  creationTimestamp: "2021-02-17T14:20:30Z"
  name: ingester.16648ed93a40dc0b
  namespace: cortex
  resourceVersion: "291777946"
  selfLink: /api/v1/namespaces/cortex/events/ingester.16648ed93a40dc0b
  uid: f9d551be-c466-4fe3-b197-5ed049cc8e51
reason: Sync
reportingComponent: ""
reportingInstance: ""
source:
  component: flux
type: Normal
`, `
apiVersion: v1
count: 15
eventTime: null
firstTimestamp: "2021-02-05T08:48:34Z"
involvedObject:
  apiVersion: apps/v1
  kind: StatefulSet
  name: ingester
  namespace: cortex
  resourceVersion: "291777997"
  uid: ded43c86-942a-4d5d-94ed-4505b4c02eb5
kind: Event
lastTimestamp: "2021-02-17T14:21:02Z"
message: create Pod ingester-3 in StatefulSet ingester successful
metadata:
  creationTimestamp: "2021-02-17T14:21:02Z"
  name: ingester.1660cdc576ee5fac
  namespace: cortex
  resourceVersion: "291778030"
  selfLink: /api/v1/namespaces/cortex/events/ingester.1660cdc576ee5fac
  uid: 4354e75b-57a9-4c33-afa6-aa280d865ae7
reason: SuccessfulCreate
reportingComponent: ""
reportingInstance: ""
source:
  component: statefulset-controller
type: Normal
`, `
action: Binding
apiVersion: v1
eventTime: "2021-02-17T14:21:02.722817Z"
firstTimestamp: null
involvedObject:
  apiVersion: v1
  kind: Pod
  name: ingester-3
  namespace: cortex
  resourceVersion: "291778027"
  uid: 921485ae-aaf1-46f8-b749-d9d23ba338e5
kind: Event
lastTimestamp: null
message: Successfully assigned cortex/ingester-3 to ip-172-20-2-25.ec2.internal
metadata:
  creationTimestamp: "2021-02-17T14:21:02Z"
  name: ingester-3.16648ee0d03aff4f
  namespace: cortex
  resourceVersion: "291778031"
  selfLink: /api/v1/namespaces/cortex/events/ingester-3.16648ee0d03aff4f
  uid: 373c1184-6f5b-4ef4-9665-2bf49fc8c4e4
reason: Scheduled
reportingComponent: default-scheduler
reportingInstance: default-scheduler-ip-172-20-3-89
source:
  component: default-scheduler
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:21:06Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-3
  namespace: cortex
  resourceVersion: "291778029"
  uid: 921485ae-aaf1-46f8-b749-d9d23ba338e5
kind: Event
lastTimestamp: "2021-02-17T14:21:06Z"
message: Container image "cortexproject/cortex:master-a2be3d8" already present on
  machine
metadata:
  creationTimestamp: "2021-02-17T14:21:06Z"
  name: ingester-3.16648ee1abf439c4
  namespace: cortex
  resourceVersion: "291778042"
  selfLink: /api/v1/namespaces/cortex/events/ingester-3.16648ee1abf439c4
  uid: 3ced3409-8bcd-4711-82d9-96913b5d62af
reason: Pulled
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-25.ec2.internal
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:21:06Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-3
  namespace: cortex
  resourceVersion: "291778029"
  uid: 921485ae-aaf1-46f8-b749-d9d23ba338e5
kind: Event
lastTimestamp: "2021-02-17T14:21:06Z"
message: Created container ingester
metadata:
  creationTimestamp: "2021-02-17T14:21:06Z"
  name: ingester-3.16648ee1aff8055e
  namespace: cortex
  resourceVersion: "291778044"
  selfLink: /api/v1/namespaces/cortex/events/ingester-3.16648ee1aff8055e
  uid: 6fa699ec-51e7-4079-80ec-8be65cd81fa7
reason: Created
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-25.ec2.internal
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:21:06Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-3
  namespace: cortex
  resourceVersion: "291778029"
  uid: 921485ae-aaf1-46f8-b749-d9d23ba338e5
kind: Event
lastTimestamp: "2021-02-17T14:21:06Z"
message: Started container ingester
metadata:
  creationTimestamp: "2021-02-17T14:21:06Z"
  name: ingester-3.16648ee1bafe09a9
  namespace: cortex
  resourceVersion: "291778045"
  selfLink: /api/v1/namespaces/cortex/events/ingester-3.16648ee1bafe09a9
  uid: 41ebd886-c8d9-4359-a84e-81e44d36b02d
reason: Started
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-25.ec2.internal
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:24:10Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-2
  namespace: cortex
  resourceVersion: "291595539"
  uid: 953d4e89-0b3e-419b-a860-72a9c2bc9d1c
kind: Event
lastTimestamp: "2021-02-17T14:24:10Z"
message: Stopping container ingester
metadata:
  creationTimestamp: "2021-02-17T14:24:10Z"
  name: ingester-2.16648f0c8744a7b0
  namespace: cortex
  resourceVersion: "291778475"
  selfLink: /api/v1/namespaces/cortex/events/ingester-2.16648f0c8744a7b0
  uid: 7c05b7c7-c943-4963-9a52-9ba916429fb2
reason: Killing
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-22.ec2.internal
type: Normal
`, `
apiVersion: v1
count: 7
eventTime: null
firstTimestamp: "2021-02-05T11:12:42Z"
involvedObject:
  apiVersion: apps/v1
  kind: StatefulSet
  name: ingester
  namespace: cortex
  resourceVersion: "291778028"
  uid: ded43c86-942a-4d5d-94ed-4505b4c02eb5
kind: Event
lastTimestamp: "2021-02-17T14:24:10Z"
message: delete Pod ingester-2 in StatefulSet ingester successful
metadata:
  creationTimestamp: "2021-02-17T14:24:10Z"
  name: ingester.1660d5a3006faa13
  namespace: cortex
  resourceVersion: "291778476"
  selfLink: /api/v1/namespaces/cortex/events/ingester.1660d5a3006faa13
  uid: 6d750ec9-7543-45f4-98e6-92cd3a90fb3a
reason: SuccessfulDelete
reportingComponent: ""
reportingInstance: ""
source:
  component: statefulset-controller
type: Normal
`, `
action: Binding
apiVersion: v1
eventTime: "2021-02-17T14:24:45.413475Z"
firstTimestamp: null
involvedObject:
  apiVersion: v1
  kind: Pod
  name: ingester-2
  namespace: cortex
  resourceVersion: "291778561"
  uid: 07cb0e28-8791-4cd7-aafa-28edfb8f4388
kind: Event
lastTimestamp: null
message: Successfully assigned cortex/ingester-2 to ip-172-20-2-22.ec2.internal
metadata:
  creationTimestamp: "2021-02-17T14:24:45Z"
  name: ingester-2.16648f14a9a0bec8
  namespace: cortex
  resourceVersion: "291778564"
  selfLink: /api/v1/namespaces/cortex/events/ingester-2.16648f14a9a0bec8
  uid: dfdc554c-fcda-4fbc-a40e-bf0bae2062f3
reason: Scheduled
reportingComponent: default-scheduler
reportingInstance: default-scheduler-ip-172-20-3-89
source:
  component: default-scheduler
type: Normal
`, `
apiVersion: v1
count: 8
eventTime: null
firstTimestamp: "2021-02-04T17:55:29Z"
involvedObject:
  apiVersion: apps/v1
  kind: StatefulSet
  name: ingester
  namespace: cortex
  resourceVersion: "291778533"
  uid: ded43c86-942a-4d5d-94ed-4505b4c02eb5
kind: Event
lastTimestamp: "2021-02-17T14:24:45Z"
message: create Pod ingester-2 in StatefulSet ingester successful
metadata:
  creationTimestamp: "2021-02-17T14:24:45Z"
  name: ingester.16609d0927697813
  namespace: cortex
  resourceVersion: "291778565"
  selfLink: /api/v1/namespaces/cortex/events/ingester.16609d0927697813
  uid: a6e02b56-1f56-46b9-b310-e62bf18431be
reason: SuccessfulCreate
reportingComponent: ""
reportingInstance: ""
source:
  component: statefulset-controller
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:24:51Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-2
  namespace: cortex
  resourceVersion: "291778562"
  uid: 07cb0e28-8791-4cd7-aafa-28edfb8f4388
kind: Event
lastTimestamp: "2021-02-17T14:24:51Z"
message: Container image "cortexproject/cortex:master-a2be3d8" already present on
  machine
metadata:
  creationTimestamp: "2021-02-17T14:24:51Z"
  name: ingester-2.16648f160a2154e2
  namespace: cortex
  resourceVersion: "291778582"
  selfLink: /api/v1/namespaces/cortex/events/ingester-2.16648f160a2154e2
  uid: d2b5ed78-5012-4701-9a3e-a99355bfb42f
reason: Pulled
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-22.ec2.internal
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:24:51Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-2
  namespace: cortex
  resourceVersion: "291778562"
  uid: 07cb0e28-8791-4cd7-aafa-28edfb8f4388
kind: Event
lastTimestamp: "2021-02-17T14:24:51Z"
message: Created container ingester
metadata:
  creationTimestamp: "2021-02-17T14:24:51Z"
  name: ingester-2.16648f16100ae6f8
  namespace: cortex
  resourceVersion: "291778583"
  selfLink: /api/v1/namespaces/cortex/events/ingester-2.16648f16100ae6f8
  uid: a017b612-9a88-4a49-a265-e50e67448d7f
reason: Created
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-22.ec2.internal
type: Normal
`, `
apiVersion: v1
count: 1
eventTime: null
firstTimestamp: "2021-02-17T14:24:51Z"
involvedObject:
  apiVersion: v1
  fieldPath: spec.containers{ingester}
  kind: Pod
  name: ingester-2
  namespace: cortex
  resourceVersion: "291778562"
  uid: 07cb0e28-8791-4cd7-aafa-28edfb8f4388
kind: Event
lastTimestamp: "2021-02-17T14:24:51Z"
message: Started container ingester
metadata:
  creationTimestamp: "2021-02-17T14:24:51Z"
  name: ingester-2.16648f162a53f9e5
  namespace: cortex
  resourceVersion: "291778584"
  selfLink: /api/v1/namespaces/cortex/events/ingester-2.16648f162a53f9e5
  uid: 76776584-5e9d-492d-83d4-5b75b4494846
reason: Started
reportingComponent: ""
reportingInstance: ""
source:
  component: kubelet
  host: ip-172-20-2-22.ec2.internal
type: Normal
`}

var sts1str = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  annotations:
    fluxcd.io/sync-checksum: 0b3fbf30ce8e60a021858681d80a0cb2da979394
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"StatefulSet","metadata":{"annotations":{"fluxcd.io/sync-checksum":"0b3fbf30ce8e60a021858681d80a0cb2da979394"},"labels":{"fluxcd.io/sync-gc-mark":"sha256.Ysq-2PNfBo9cDgfOAFO1z5RG34IJB07VIzqR_wzuy1k"},"name":"ingester","namespace":"cortex"},"spec":{"replicas":4,"selector":{"matchLabels":{"name":"ingester"}},"serviceName":"ingester","template":{"metadata":{"annotations":{"iam.amazonaws.com/role":"cortex-blocks-writer"},"labels":{"name":"ingester"}},"spec":{"affinity":{"podAntiAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":[{"labelSelector":{"matchLabels":{"name":"ingester"}},"topologyKey":"kubernetes.io/hostname"}]}},"containers":[{"args":["-target=ingester","--log.level=info","-store.engine=blocks","-blocks-storage.backend=s3","-blocks-storage.bucket-store.ignore-deletion-marks-delay=1h","-blocks-storage.bucket-store.sync-dir=/data/tsdb","-blocks-storage.s3.bucket-name=cortex-blocks-dev","-blocks-storage.s3.endpoint=s3.dualstack.us-east-1.amazonaws.com","-blocks-storage.tsdb.block-ranges-period=2h","-blocks-storage.tsdb.dir=/data/tsdb","-blocks-storage.tsdb.retention-period=24h","-blocks-storage.tsdb.ship-interval=1m","-consul.hostname=consul.cortex.svc.cluster.local.:8500","-ingester.join-after=0s","-ingester-client.expected-timeseries=1000","-ingester-client.expected-samples-per-series=1","-ingester.max-series-per-user=300000","-ingester.max-samples-per-query=2000000","-ingester.max-series-per-query=300000","-ingester.num-tokens=128","-ingester.tokens-file-path=/data/tokens","-limits.per-user-override-config=/etc/config/user-overrides.yaml"],"env":[{"name":"GOMAXPROCS","value":"4"},{"name":"GOGC","value":"40"},{"name":"JAEGER_AGENT_HOST","valueFrom":{"fieldRef":{"fieldPath":"status.hostIP"}}},{"name":"JAEGER_SAMPLER_MANAGER_HOST_PORT","value":"http://jaeger-agent.tracing.svc.cluster.local./sampling"}],"image":"cortexproject/cortex:master-813e156","imagePullPolicy":"IfNotPresent","name":"ingester","ports":[{"containerPort":80,"name":"http-metrics"},{"containerPort":9095,"name":"grpc-noscrape"}],"readinessProbe":{"httpGet":{"path":"/ready","port":80},"initialDelaySeconds":15,"timeoutSeconds":1},"resources":{"limits":{"memory":"5Gi"},"requests":{"cpu":"400m","memory":"5Gi"}},"volumeMounts":[{"mountPath":"/data","name":"ingester-pvc"},{"mountPath":"/etc/config","name":"overrides"}]}],"priorityClassName":"ingester","securityContext":{"runAsUser":0},"terminationGracePeriodSeconds":4800,"volumes":[{"name":"ingester-pvc","persistentVolumeClaim":{"claimName":"ingester-pvc"}},{"configMap":{"name":"overrides-config"},"name":"overrides"}]}},"updateStrategy":{"type":"RollingUpdate"},"volumeClaimTemplates":[{"metadata":{"name":"ingester-pvc"},"spec":{"accessModes":["ReadWriteOnce"],"resources":{"requests":{"storage":"50Gi"}},"storageClassName":"gp2"}}]}}
  creationTimestamp: "2021-02-04T17:08:25Z"
  generation: 10
  labels:
    fluxcd.io/sync-gc-mark: sha256.Ysq-2PNfBo9cDgfOAFO1z5RG34IJB07VIzqR_wzuy1k
  name: ingester
  namespace: cortex
  resourceVersion: "291596474"
  selfLink: /apis/apps/v1/namespaces/cortex/statefulsets/ingester
  uid: ded43c86-942a-4d5d-94ed-4505b4c02eb5
spec:
  podManagementPolicy: OrderedReady
  replicas: 4
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      name: ingester
  serviceName: ingester
  template:
    metadata:
      labels:
        name: ingester
    spec:
      containers:
      - args:
        - -target=ingester
        image: cortexproject/cortex:master-813e156
        name: ingester
        volumeMounts:
        - mountPath: /data
          name: ingester-pvc
        - mountPath: /etc/config
          name: overrides
      restartPolicy: Always
      volumes:
      - name: ingester-pvc
        persistentVolumeClaim:
          claimName: ingester-pvc
      - configMap:
          defaultMode: 420
          name: overrides-config
        name: overrides
  updateStrategy:
    type: RollingUpdate
  volumeClaimTemplates:
  - metadata:
      creationTimestamp: null
      name: ingester-pvc
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 50Gi
      storageClassName: gp2
      volumeMode: Filesystem
    status:
      phase: Pending
status:
  collisionCount: 0
  currentReplicas: 4
  currentRevision: ingester-77fff6c776
  observedGeneration: 10
  readyReplicas: 4
  replicas: 4
  updateRevision: ingester-77fff6c776
  updatedReplicas: 4
`

const stsPod2str = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    iam.amazonaws.com/role: cortex-blocks-writer
  creationTimestamp: "2021-02-17T14:24:45Z"
  generateName: ingester-
  labels:
    controller-revision-hash: ingester-7cf8dbb565
    name: ingester
    statefulset.kubernetes.io/pod-name: ingester-2
  name: ingester-2
  namespace: cortex
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: StatefulSet
    name: ingester
    uid: ded43c86-942a-4d5d-94ed-4505b4c02eb5
  resourceVersion: "291779005"
  selfLink: /api/v1/namespaces/cortex/pods/ingester-2
  uid: 07cb0e28-8791-4cd7-aafa-28edfb8f4388
spec:
containers:
- args:
  - -target=ingester
  image: cortexproject/cortex:master-a2be3d8
  imagePullPolicy: IfNotPresent
  name: ingester
  ports:
  - containerPort: 80
    name: http-metrics
    protocol: TCP
  - containerPort: 9095
    name: grpc-noscrape
    protocol: TCP
  volumeMounts:
  - mountPath: /data
    name: ingester-pvc
  - mountPath: /etc/config
    name: overrides
  - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
    name: default-token-e3ov9
    readOnly: true
  hostname: ingester-2
  nodeName: ip-172-20-2-22.ec2.internal
  priority: 1000
  priorityClassName: ingester
  restartPolicy: Always
  schedulerName: default-scheduler
  securityContext:
    runAsUser: 0
  serviceAccount: default
  serviceAccountName: default
  subdomain: ingester
  terminationGracePeriodSeconds: 4800
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: ingester-pvc
    persistentVolumeClaim:
      claimName: ingester-pvc-ingester-2
  - configMap:
      defaultMode: 420
      name: overrides-config
    name: overrides
  - name: default-token-e3ov9
    secret:
      defaultMode: 420
      secretName: default-token-e3ov9
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:24:45Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:27:58Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:27:58Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:24:45Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: docker://ba937b5bc0b855b8fac7fef27638ee1ebe77a37591a9e4e933775bc687a3bcfb
    image: cortexproject/cortex:master-a2be3d8
    imageID: docker-pullable://cortexproject/cortex@sha256:71d6c4e1c1e5efcdfac1c4ae1266ace450d5c29d02568bfd65514336c117fb83
    lastState: {}
    name: ingester
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2021-02-17T14:24:51Z"
  hostIP: 172.20.2.22
  phase: Running
  podIP: 10.244.228.33
  podIPs:
  - ip: 10.244.228.33
  qosClass: Burstable
  startTime: "2021-02-17T14:24:45Z"
`

const stsPod3str = `
apiVersion: v1
kind: Pod
metadata:
  annotations:
    iam.amazonaws.com/role: cortex-blocks-writer
  creationTimestamp: "2021-02-17T14:21:02Z"
  generateName: ingester-
  labels:
    controller-revision-hash: ingester-7cf8dbb565
    name: ingester
    statefulset.kubernetes.io/pod-name: ingester-3
  name: ingester-3
  namespace: cortex
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: StatefulSet
    name: ingester
    uid: ded43c86-942a-4d5d-94ed-4505b4c02eb5
  resourceVersion: "291778472"
  selfLink: /api/v1/namespaces/cortex/pods/ingester-3
  uid: 921485ae-aaf1-46f8-b749-d9d23ba338e5
spec:
  containers:
  - args:
    - -target=ingester
    image: cortexproject/cortex:master-a2be3d8
    imagePullPolicy: IfNotPresent
    name: ingester
    ports:
    - containerPort: 80
      name: http-metrics
      protocol: TCP
    - containerPort: 9095
      name: grpc-noscrape
      protocol: TCP
    volumeMounts:
    - mountPath: /data
      name: ingester-pvc
    - mountPath: /etc/config
      name: overrides
    - mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      name: default-token-e3ov9
      readOnly: true
  hostname: ingester-3
  nodeName: ip-172-20-2-25.ec2.internal
  priority: 1000
  priorityClassName: ingester
  restartPolicy: Always
  subdomain: ingester
  terminationGracePeriodSeconds: 4800
  tolerations:
  - effect: NoExecute
    key: node.kubernetes.io/not-ready
    operator: Exists
    tolerationSeconds: 300
  - effect: NoExecute
    key: node.kubernetes.io/unreachable
    operator: Exists
    tolerationSeconds: 300
  volumes:
  - name: ingester-pvc
    persistentVolumeClaim:
      claimName: ingester-pvc-ingester-3
  - configMap:
      defaultMode: 420
      name: overrides-config
    name: overrides
  - name: default-token-e3ov9
    secret:
      defaultMode: 420
      secretName: default-token-e3ov9
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:21:02Z"
    status: "True"
    type: Initialized
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:24:10Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:24:10Z"
    status: "True"
    type: ContainersReady
  - lastProbeTime: null
    lastTransitionTime: "2021-02-17T14:21:02Z"
    status: "True"
    type: PodScheduled
  containerStatuses:
  - containerID: docker://327d16ab1bda5d9f599735916b34cf6d3b2616cfd74c624a8368d6742ced1e6d
    image: cortexproject/cortex:master-a2be3d8
    imageID: docker-pullable://cortexproject/cortex@sha256:71d6c4e1c1e5efcdfac1c4ae1266ace450d5c29d02568bfd65514336c117fb83
    lastState: {}
    name: ingester
    ready: true
    restartCount: 0
    started: true
    state:
      running:
        startedAt: "2021-02-17T14:21:06Z"
  hostIP: 172.20.2.25
  phase: Running
  podIP: 10.244.226.98
  podIPs:
  - ip: 10.244.226.98
  qosClass: Burstable
  startTime: "2021-02-17T14:21:02Z"
`
