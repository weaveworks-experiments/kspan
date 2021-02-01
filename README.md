# kspan
Turning Kubernetes Events into spans

Most Kubernetes components produce Events when something interesting happens.
This program turns those Events into OpenTelemetry Spans, joining them up
by causality and grouping them together into Traces.

Example: rollout of a Deployment of two Pods:

![image](example-2pod.png)

We start with this concrete information:
 * Each Event has an Involved Object, e.g. when Kubelet sends a "Started" event,
   the Involved Object is a Pod.
 * Every Kubernetes object can have one or more Owner References. So for instance
   we can walk from the Pod up to a Deployment that caused it to be created.

Complications:
 * We cannot expect events to arrive in the ideal order; we need to delay handling some until their "parent" arrives to make sense.
