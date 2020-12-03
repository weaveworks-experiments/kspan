package events

import (
	"testing"

	"sigs.k8s.io/yaml"
)

func mustParse(t *testing.T, str string, obj interface{}) {
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

