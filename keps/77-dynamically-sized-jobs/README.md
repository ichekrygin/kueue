# KEP-77: Dynamically Sized Jobs

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 - RayCluster w/ autoscaling](#story-1---raycluster-w-autoscaling)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
- [Design Details](#design-details)
  - [Workload Slices](#workload-slices)
  - [Creating Workload Slices](#creating-workload-slices)
  - [Pod Scheduling Gates](#pod-scheduling-gates)
  - [Garbage Collecting Workload Slices](#garbage-collecting-workload-slices)
- [Phases for MVP (alpha)](#phases-for-mvp-alpha)
  - [Phase 1 - Scale Down](#phase-1---scale-down)
    - [Job controller](#job-controller)
  - [Phase 2 - Aggregating Workload Slices](#phase-2---aggregating-workload-slices)
  - [Phase 3 - Scale up with Workload Slices and Scheduling Gates](#phase-3---scale-up-with-workload-slices-and-scheduling-gates)
    - [Scheduler](#scheduler)
- [Additional Details](#additional-details)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [Integration tests](#integration-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Ignore Resize from Kuberay](#ignore-resize-from-kuberay)
<!-- /toc -->

## Summary

Enable dynamic resizing of Kueue-managed batch/v1.Job workloads by supporting in-place horizontal scaling of job parallelism. 
The primary goal is to allow jobs to scale their pod count without requiring suspension and requeueing, improving scheduling efficiency and reducing disruption during resource fluctuations.

As part of the MVP, this feature will also support dynamic resizing of autoscaling-enabled RayCluster workloads.
While the initial focus is on these two job types, the long-term goal is to generalize support for dynamically sized (elastic) workloads across other Kueue-integrated resource types.

## Motivation

Kueue currently lacks native support for resizing jobs. Any change in job size leads to the recreation of the associated Workload, resulting in job suspension and requeueing. 
This disrupts execution and hinders usability for elastic workloads like RayCluster, which rely on in-place autoscaling. 

To support such scenarios, Kueue must gracefully handle horizontal scale-up and scale-down operations without disrupting admitted jobs or re-acquiring quota.

### Goals

- Gracefully handle resize operations for Kueue-managed jobs by updating quota usage without suspending the job or terminating running pods.
  - Support dynamic updates to `batch/v1.Job` parallelism, allowing changes to pod count without deleting existing pods.
  - Enable seamless integration with autoscaling-enabled RayClusters, ensuring scale events are reflected in Kueue’s scheduling and quota system without disrupting job execution. This forms the core of the MVP due to strong demand and existing autoscaler integration.

### Non-Goals

- Vertical scaling of workloads – Kueue will only handle resize operations that scale Pods horizontally. 
- Support resize for other Kueue jobs such as Job, JobSet, etc (future).
- Resizing of the Workload parent objects (`batchv1/Job`, RayJobs, etc.)
- Partial Preemption.

## Proposal

Update the Job framework reconciler and introduce new controllers to orchestrate dynamic resizing of jobs. We are only interested in horizontal scaling of jobs (e.g. scaling more replicas). At a high level, this will be accomplished by:
- Creating Workload Slice objects that represent incremental scale up of jobs. This gives us per-replica control of workload admission. Workload Slices will be garbage collected and consolidated with their parent workloads after successful admission.
- Adding default scheduling gates to control the scheduling of new pods based on their admission status.
- Dynamically adjust quotas in ClusterQueue based on scaling events.

For the MVP, dynamic resizing will be available through an opt-in mechanism for selected, supported Kueue frameworks—starting with `batch/v1.Job` and, optionally, RayCluster workloads with autoscaling enabled.

### User Stories

#### Story 1 - batchv1/Job scale-down

1. The user creates a `batch/v1.Job` with workload-slice enablement explicitly opted in.
2. Kueue admits the Job based on its requested resources, creating the corresponding Workload and Job's Pods. The Pods begin running as usual.
3. The user updates the Job by reducing its parallelism, triggering a scale-down event:
   1. The associated Workload is updated to reflect the new, lower pod count in its spec. 
   2. The local queue’s capacity is updated accordingly to reflect the reduced resource usage. 
   3. One running Job pod is terminated, while the remaining Pods continue running uninterrupted.

#### Store 2 - batchv1/Job scale-up
1. The user creates a `batch/v1.Job` with workload-slice enablement explicitly opted in.
2. Kueue admits the Job based on its requested resources, creating the corresponding Workload and Job’s Pods. The Pods begin running as usual.
3. The user updates the Job by increasing its parallelism, triggering a scale-up event:
   1. A new Workload Slice is created to represent the additional pod capacity requested.
   2. The new Pods are created in a gated state (via PodSchedulingGates) and held from scheduling until the slice is admitted. 
   3. Kueue evaluates available resources; if sufficient, the slice is admitted and the scheduling gates on the new Pods are removed. 
   4. The newly admitted Pods begin scheduling and running, resulting in the Job operating with the increased parallelism level.

### Notes/Constraints/Caveats (Optional)

If Kueue needs to preempt a resized Job, it will preempt the entire Job as a single unit—regardless of whether the Job has undergone a scale-up operation.

## Design Details

To support horizontal scaling of jobs, we will introduce the concept of a "Workload Slice". A Workload Slice is a Workload object with an owner reference to the original Workload for a job. 
Workload Slices represent per-replica changes to a job that were not initially accounted for when the job was created. **It's an internal state, not an extra CRD.**

The benefit of Workload Slices is that Kueue can evaluate admission on a per-replica basis without changing the existing semantics of the Workload API. 
Once a Workload Slice is admitted, it will be garbage collected and its resources will be aggregated into the admission status of the parent workload.
- Workload Slices will be submitted to the same LocalQueue that's referenced by the top-level Workload. 
- In MultiKueue, Workload Slices would go into both clusters (management and workload) in a multi-cluster environment.
- Workload Slices will be created by Kueue and use identical PodTemplates (which is already enforced by Kuberay in the case for RayCluster).
- Workload Slices will belong to the same resource flavor as the top-level Workload that was initially admitted.

The parent Workload should have a condition that reflects the scaling progression status. 

### Enablement

WorkloadSlices in Kueue are enabled through a combination of a Kubernetes feature gate and an opt-in annotation on individual Workload objects. 
At the cluster level, the WorkloadSlices feature must be enabled via the corresponding Kubernetes feature gate, which controls whether the controller logic for slicing is active. 
Once the feature gate is enabled, individual Workload objects can opt into slicing by including the kueue.x-k8s.io/enable-workload-slices: "true" annotation. 
When both conditions are met, Kueue treats the Workload as eligible for partitioning into one or more WorkloadSlice objects, enabling fine-grained scheduling and 
execution across multiple clusters or within a single cluster. If the feature gate is disabled or the annotation is omitted (or set to "false"), 
the system defaults to the traditional single-Workload scheduling path for full backward compatibility.

#### Features
```golang
// owner: @ichekrygin
// kep: https://github.com/kubernetes-sigs/kueue/tree/main/keps/77-dynamically-sized-jobs
//
// WorkloadSlices enables workload-slices support.
WorkloadSlices featuregate.Feature = "WorkloadSlices"
```

#### WorkloadSliceAnnotation
```golang
// Workload slicing can be enabled for a specific integration job instance,
// provided that the integration supports workload slicing.
const (
  // EnabledAnnotationKey refers to the annotation key present on Job's that support
  // workload slicing.
  EnabledAnnotationKey = "kueue.x-k8s.io/workload-slicing-enabled"
  // EnabledAnnotationValue refers to the annotation value. To enable
  // workload slicing for a given job, we match both annotation key and value.
  EnabledAnnotationValue = "true"
)
```

### Processing of Workload Slices

#### Creation
The creation of WorkloadSlices in Kueue is initiated when a Kueue Job with slicing enabled—via both annotation and feature gate—is admitted for execution. 
Upon admission, the `jobframework.JobReconciler` evaluates the Job’s PodSets and resource requirements, then creates a new or additional (slice) Workload object 
that represents a partition of the original workload. Each slice contains a `PodSet.Count` identical to the Job’s definition, along with scheduling metadata needed for handling preemption of the old workload (slice).

At most two active workload slices can exist for any single Job in Kueue: the current admitted slice and a newly created slice during a scaling operation. 
This ensures a controlled transition between slices while maintaining consistency and avoiding resource contention and race conditions.

To facilitate precise resource scheduling and effective workload preemption, every new job configured with slice enablement will be appropriately annotated using `EnabledAnnotationKey/Value` mentioned above.
Furthermore, each individual workload slice will record the identity of any preempting slice it encounters.

```golang

const (
	// WorkloadPreemptibleSliceNameKey is the annotation key used to capture an "old" workload slice
	// that will be preempted by the "new" workload slice, i.e., this annotates a "new" workload slice.
	WorkloadPreemptibleSliceNameKey = "kueue.x-k8s.io/workload-preemptible-slice"
)
```

##### Naming
Currently, Kueue uses a deterministic Workload naming strategy by generating consistent names for the same Job instance via GetWorkloadNameForOwnerWithGVK.
As a result, Workload names are not unique across different revisions of the same owner object (e.g., a Job), which makes this approach unsuitable for representing Workload slices.
To address this, a new naming strategy will be introduced specifically for Workload slices, incorporating the owner object's generation value to ensure name uniqueness across revisions.

#### Scheduling and Preemption
Kueue’s scheduler and preemptor components will be augmented to support workload slice processing, specifically by enforcing preemption of the old workload slice regardless of queue capacity to make room for the new slice. 

Scheduling assignment is based on pod count capacity, accounting for the existing workload slice's allocated pods. 
For example, scaling a job up from `3` pods to `10` pods results in the creation of a new Workload representing all `10` pods; 
however, the scheduling assignment will reflect only the `7` additional pods needed to admit the new slice, since `3` pods are already accounted for by the existing slice.

A Workload that is preempted by a Workload Slice will be correctly marked with a Preempted status condition, including an appropriate reason and message.
```golang
// WorkloadSlicePreemptionReason indicates the Workload was preempted due to
// the workload slice succession (roll-up/aggregation).
WorkloadSlicePreemptionReason string = "WorkloadSlicePreemption"
...
// WorkloadEvictedByWorkloadSliceAggregation indicates that the workload was
// deactivated as a result of workload slice aggregation.
WorkloadEvictedByWorkloadSliceAggregation = "WorkloadSliceAggregation"	
```

#### Garbage Collection of Preempted Workload Slices
Currently, a Workload can become outdated, marked by Kueue as "out-of-sync", and subsequently transitioned to the "Finished" state.
This also applies to outdated Workload slices: when multiple successive updates are issued for a given Job, any superseded slices are marked as "Finished" to reflect their obsolescence.

Preempted (i.e., aggregated) Workload slices are first marked for Deactivation and then transitioned to the Finished state.
Under the current design proposal, all deactivated slices are retained indefinitely and are not garbage collected.
To address potential resource buildup, a `PreemptedWorkloadSliceHistory` mechanism whether as a configuration option or an API field on the Workload—could be introduced to limit the number of retained inactive slices, 
similar to how `revisionHistoryLimit` is used in Kubernetes Deployments to manage ReplicaSet history.

Preempted Workload slices that fail admission requirements are not marked as Finished, as they were never actually started (i.e., never admitted or executed). 

Similar to current functionality, all Workloads associated with a Job (including slices) will be deleted when the parent Job is removed.

### Pod Scheduling Gates

Workload slices operate independently of the `spec.suspend` mechanism where applicable, relying instead on pod scheduling gates for admission control. 
This approach is automatically enabled for supported Job types through defaulting. When a Workload (either initial or a slice) is created, all associated pods are initialized with scheduling gates—effectively "gated" from scheduling. 
These gates are removed only upon the admission of the corresponding Workload, ensuring controlled and deliberate execution.

```golang
const (
...
  // WorkloadSliceSchedulingGate is the name of the scheduling gate applied to Pods
  // to delay their scheduling until the associated workload slice has been admitted.
  // This gate ensures that Pods do not begin scheduling prematurely, maintaining
  // proper sequencing in workload processing.
  WorkloadSliceSchedulingGate = "kueue.x-k8s.io/workload-slice"
)
```

## Phases for MVP (alpha)

### Phase 1 - batchv1/Job WorkloadSlices Support in Single-Cluster Configuration.

Scaling up and down for `batch/v1.Job` will be the initial phase of the MVP, as it builds on stable and well-understood Kubernetes components without requiring additional integration work.

#### Scale Down

1. Create a Job with workload-slice enablement and a parallelism value greater than 1. 
2. Observe that a corresponding Workload is created and admitted, and that the Job's pods are in the Running state, matching the specified parallelism. 
3. Update the Job by reducing its parallelism value. 
4. Confirm that the existing Workload is updated accordingly, with WorkloadSpec.PodSets[0].Count reflecting the new parallelism value. 
5. Observe that the number of running pods is reduced, while the remaining pods continue to run uninterrupted.

#### Scale Up

1. Create a `batch/v1.Job` with workload-slice enablement and an initial `parallelism` value greater than 0.
2. Verify that a corresponding `Workload` is created and admitted, and that the Job’s pods are in the `Running` state, matching the specified parallelism.
3. Increase the Job’s `parallelism` value to trigger a scale-up event.
4. Confirm that a new `Workload` slice is created and admitted, reflecting the increased pod count, while the previous slice is deactivated and marked as `Finished`.
5. Observe that the total number of running pods increases to match the updated parallelism, with the original pods continuing to run without disruption.

### Phase 2 – RayCluster WorkloadSlice Support in Single-Cluster Configuration
This phase mirrors the steps from Phase 1 but is adapted specifically for RayCluster workloads, taking into account their autoscaling behavior and internal lifecycle management.

#### Phase 3 – Enabling Workload Slicing for batch/v1.Job in Multi-Cluster Configuration
In this phase, Workload Slicing support for batch/v1.Job will be extended to multi-cluster environments using Kueue’s MultiKueue architecture. 
When a job is scaled in the management cluster, its corresponding WorkloadSlice will be propagated to the appropriate workload cluster(s), respecting existing cluster assignment and resource flavor constraints. 
Each WorkloadSlice will be subject to independent admission in the target cluster, and only after successful admission will scheduling gates be lifted to allow pod execution. 
Slice preemption, quota accounting, and garbage collection must be coordinated across clusters to ensure consistency and avoid orphaned resources. 
This phase will validate correctness and stability of the slicing mechanism in a federated deployment model.

## Additional Details

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Unit Tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->
The code will adhere to regular best practices for unit tests and coverage.



#### Integration tests
Integration tests will be executed against mocked clients for Jobs 
that will provide predefined responses and allow to test various scenarios, 
including situations like:

* Job has a scale up, workload slices get created, pods are gated
* Job has a scale up, gated pods are admitted, pods get ungated and assigned same flavors as parent workload
* Workload slices are correctly folded and deleted
* Job has a scale down, Workload spec reflects podset count values
* When Kueue preempt a resized Job, it should preempt it as a whole

### Graduation Criteria
<!--

Clearly define what it means for the feature to be implemented and
considered stable.

If the feature you are introducing has high complexity, consider adding graduation
milestones with these graduation criteria:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/
-->

The feature starts at the alpha level, with a feature gate.

In the Alpha version, Dynamically Sized Jobs will support:
- `batchv1/Job` and `RayCluster` resizing in single-cluster and multi-cluster configurations.


## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->


## Alternatives

Initial Design Assumptions/Proposals
### Creating Workload Slices

The [GenericJob interface (7e778f5)](https://github.com/kubernetes-sigs/kueue/blob/main/pkg/controller/jobframework/interface.go#L30-L55) will be updated to handle resize operations of jobs.

```golang
type GenericJob interface {
  ...
  ...
  ResizeJob(wl *kueue.Workload) error
}
```
Jobs implementing the ResizeJob method will create a Workload Slice for every new replica of a job.

On scale down to M we will change the original Workload's resources and then on scale up to N we will create N-M WorkloadSlice objects that go through admission and scheduling gate checks, i.e. the Workload can "lose" the original (first time admission) quota when scaling down before scaling up.

### Pod Scheduling Gates

Inside the job's webhook, implement schedulingGate injection for pods on creation time.
The Pods will be ungated following a similar behavior as to how a job is suspended and then unsuspended in the when admitted.
When the job scales up, the new pods will be gated due to the schedulingGates injection in the webhook.

After the creation of each individual Workload Slice and admission of a Workload Slice, the **workload_scheduling_gates_controller** should be in charge of removing the scheduling gates from each pod. All worker pods from the same worker group share the same pod template, so we only need to ungate the number of pods to match the number of admitted pods, this should be a counter. We don’t want to accidentally ungate too many pods since race conditions could happen and we also don’t want to double count. It's worth mentioning that for the case of recreated pods (i.e. machine failure for example), these pods will go through the admission/scheduling check again, Kueue is responsible fo removing the scheduling gates when there's available quota and resources to spend on the Job.

### Phase 3 - Scale up with Workload Slices and Scheduling Gates

In Phase 3, scale up will be implemented by introducing Workload Slices and adding Pod scheduling gates as part of Kueue’s mutating admission for the job.

When a job scales, its webhook would be modified to intercept and "gate" all pods. Every time there’s a resize, you create a dependable (child) workload slice and once it's admitted, you increase the count in the original workload, delete the old workload and remove the schedulingGates.
- Pros: You are able to hold the pods added by Kuberay
- Cons: The fact of having schedulingGates, means we need an API call for every pod, because all pods that are created by the job are going to have schedulingGates. We need to remove those gates and for every pod you need to make API calls.

### Ignore Resize from Kuberay

- Idea: Ignoring the scale up or rejecting the scale up from the auto scaler and storing the value in an annotation so that Kueue takes a decision based on the annotation. We’d need a signal from Kueue to hold this in the raycluster_webhook and identify when the resize comes from Kueue, it has to be accepted.
- Pros: No need to intercept the pods and no need of using schedulingGates
- Cons: There would be a permanent race bewteen Kueue and Kuberay autoscaler to change the counter in the RayCluster replica number for the worker group. 
- Exploration: See if autoscaler would indicate a desired size in the spec without altering the number of replicas directly. 
- Discarded: Higher complexity than gating/ungating pods via SchedulingGates

### Garbage Collecting Workload Slices

The logic of folding and deleting would be isolated in this controller. We don’t necessarily have to say that this Workload Slice belongs to a specific pod. This controller will look at the Workload objects and check whether they have the Admitted condition or not.

1. The controller increments the `.status.admission.podSetAssignments.count` and passes the UID of the workload you are folding to the parent workload.
   If we still don’t see the workload being deleted, we at least know it has been counted towards the parent workload and the controller shouldn't fold it again.
