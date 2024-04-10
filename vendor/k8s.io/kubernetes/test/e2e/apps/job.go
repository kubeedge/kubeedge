/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/util/retry"
	batchinternal "k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/test/e2e/framework"
	e2ejob "k8s.io/kubernetes/test/e2e/framework/job"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2eresource "k8s.io/kubernetes/test/e2e/framework/resource"
	"k8s.io/kubernetes/test/e2e/scheduling"
	admissionapi "k8s.io/pod-security-admission/api"
	"k8s.io/utils/pointer"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type watchEventConfig struct {
	framework           *framework.Framework
	resourceVersion     string
	w                   *cache.ListWatch
	jobName             string
	watchEvent          watch.EventType
	extJob              *batchv1.Job
	updatedMetadataType string
	updatedKey          string
	updatedValue        string
}

var _ = SIGDescribe("Job", func() {
	f := framework.NewDefaultFramework("job")
	f.NamespacePodSecurityLevel = admissionapi.LevelPrivileged
	parallelism := int32(2)
	completions := int32(4)

	largeParallelism := int32(90)
	largeCompletions := int32(90)

	backoffLimit := int32(6) // default value

	// Simplest case: N pods succeed
	ginkgo.It("should run a job to completion when tasks succeed", func(ctx context.Context) {
		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJob("succeed", "all-succeed", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring job reaches completions")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, completions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring pods for job exist")
		pods, err := e2ejob.GetJobPods(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to get pod list for job in namespace: %s", f.Namespace.Name)
		successes := int32(0)
		for _, pod := range pods.Items {
			if pod.Status.Phase == v1.PodSucceeded {
				successes++
			}
		}
		framework.ExpectEqual(successes, completions, "expected %d successful job pods, but got  %d", completions, successes)
	})

	ginkgo.It("should allow to use the pod failure policy on exit code to fail the job early", func(ctx context.Context) {

		// We fail the Job's pod only once to ensure the backoffLimit is not
		// reached and thus the job is failed due to the pod failure policy
		// with FailJob action.
		// In order to ensure a Job's pod fails once before succeeding we force
		// the Job's Pods to be scheduled to a single Node and use a hostPath
		// volume to persist data across new Pods.
		ginkgo.By("Looking for a node to schedule job pod")
		node, err := e2enode.GetRandomReadySchedulableNode(ctx, f.ClientSet)
		framework.ExpectNoError(err)

		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJobOnNode("failOnce", "pod-failure-failjob", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit, node.Name)
		job.Spec.PodFailurePolicy = &batchv1.PodFailurePolicy{
			Rules: []batchv1.PodFailurePolicyRule{
				{
					Action: batchv1.PodFailurePolicyActionFailJob,
					OnExitCodes: &batchv1.PodFailurePolicyOnExitCodesRequirement{
						Operator: batchv1.PodFailurePolicyOnExitCodesOpIn,
						Values:   []int32{1},
					},
				},
			},
		}
		job, err = e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring job fails")
		err = e2ejob.WaitForJobFailed(f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to ensure job failure in namespace: %s", f.Namespace.Name)
	})

	ginkgo.It("should allow to use the pod failure policy to not count the failure towards the backoffLimit", func(ctx context.Context) {

		// We set the backoffLimit to 0 so that any pod failure would trigger
		// job failure if not for the pod failure policy to ignore the failed
		// pods from counting them towards the backoffLimit. Also, we fail the
		// pod only once so that the job eventually succeeds.
		// In order to ensure a Job's pod fails once before succeeding we force
		// the Job's Pods to be scheduled to a single Node and use a hostPath
		// volume to persist data across new Pods.
		backoffLimit := int32(0)

		ginkgo.By("Looking for a node to schedule job pod")
		node, err := e2enode.GetRandomReadySchedulableNode(ctx, f.ClientSet)
		framework.ExpectNoError(err)

		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJobOnNode("failOnce", "pod-failure-ignore", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit, node.Name)
		job.Spec.PodFailurePolicy = &batchv1.PodFailurePolicy{
			Rules: []batchv1.PodFailurePolicyRule{
				{
					Action: batchv1.PodFailurePolicyActionIgnore,
					OnExitCodes: &batchv1.PodFailurePolicyOnExitCodesRequirement{
						Operator: batchv1.PodFailurePolicyOnExitCodesOpIn,
						Values:   []int32{1},
					},
				},
			},
		}
		job, err = e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring job reaches completions")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, completions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)
	})

	// This test is using an indexed job. The pod corresponding to the 0th index
	// creates a marker file on the host and runs 'forever' until evicted. We use
	// the non-0-indexed pods to determine if the marker file is already
	// created by the 0th indexed pod - the non-0-indexed pods fail and restart
	// until the marker file is created (their potential failures are ignored
	// based on the exit code). Once the marker file is created the 0th indexed
	// pod is evicted (DisruptionTarget condition is added in the process),
	// after restart it runs to successful completion.
	// Steps:
	// 1. Select a node to run all Job's pods to ensure the host marker file is accessible by all pods
	// 2. Create the indexed job
	// 3. Await for all non-0-indexed pods to succeed to ensure the marker file is created by the 0-indexed pod
	// 4. Make sure the 0-indexed pod is running
	// 5. Evict the 0-indexed pod
	// 6. Await for the job to successfully complete
	ginkgo.DescribeTable("Using a pod failure policy to not count some failures towards the backoffLimit",
		func(ctx context.Context, policy *batchv1.PodFailurePolicy) {
			mode := batchv1.IndexedCompletion

			// We set the backoffLimit to 0 so that any pod failure would trigger
			// job failure if not for the pod failure policy to ignore the failed
			// pods from counting them towards the backoffLimit.
			backoffLimit := int32(0)

			ginkgo.By("Looking for a node to schedule job pods")
			node, err := e2enode.GetRandomReadySchedulableNode(ctx, f.ClientSet)
			framework.ExpectNoError(err)

			ginkgo.By("Creating a job")
			job := e2ejob.NewTestJobOnNode("notTerminateOnce", "pod-disruption-failure-ignore", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit, node.Name)
			job.Spec.CompletionMode = &mode
			job.Spec.PodFailurePolicy = policy
			job, err = e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
			framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

			ginkgo.By("Awaiting for all non 0-indexed pods to succeed to ensure the marker file is created")
			err = e2ejob.WaitForJobPodsSucceeded(ctx, f.ClientSet, f.Namespace.Name, job.Name, completions-1)
			framework.ExpectNoError(err, "failed to await for all non 0-indexed pods to succeed for job: %s/%s", job.Name, job.Namespace)

			ginkgo.By("Awaiting for the 0-indexed pod to be running")
			err = e2ejob.WaitForJobPodsRunning(ctx, f.ClientSet, f.Namespace.Name, job.Name, 1)
			framework.ExpectNoError(err, "failed to await for the 0-indexed pod to be running for the job: %s/%s", job.Name, job.Namespace)

			pods, err := e2ejob.GetAllRunningJobPods(ctx, f.ClientSet, f.Namespace.Name, job.Name)
			framework.ExpectNoError(err, "failed to get running pods for the job: %s/%s", job.Name, job.Namespace)
			framework.ExpectEqual(len(pods), 1, "Exactly one running pod is expected")
			pod := pods[0]
			ginkgo.By(fmt.Sprintf("Evicting the running pod: %s/%s", pod.Name, pod.Namespace))
			evictTarget := &policyv1.Eviction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}
			f.ClientSet.CoreV1().Pods(pod.Namespace).EvictV1(context.TODO(), evictTarget)
			framework.ExpectNoError(err, "failed to evict the pod: %s/%s", pod.Name, pod.Namespace)

			ginkgo.By(fmt.Sprintf("Awaiting for the pod: %s/%s to be deleted", pod.Name, pod.Namespace))
			err = e2epod.WaitForPodNotFoundInNamespace(ctx, f.ClientSet, pod.Name, pod.Namespace, f.Timeouts.PodDelete)
			framework.ExpectNoError(err, "failed to await for the pod to be deleted: %s/%s", pod.Name, pod.Namespace)

			ginkgo.By("Ensuring job reaches completions")
			err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, completions)
			framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)
		},
		ginkgo.Entry("Ignore DisruptionTarget condition", &batchv1.PodFailurePolicy{
			Rules: []batchv1.PodFailurePolicyRule{
				{
					// Ignore failures of the non 0-indexed pods which fail until the marker file is created
					Action: batchv1.PodFailurePolicyActionIgnore,
					OnExitCodes: &batchv1.PodFailurePolicyOnExitCodesRequirement{
						Operator: batchv1.PodFailurePolicyOnExitCodesOpIn,
						Values:   []int32{1},
					},
				},
				{
					// Ignore the pod failure caused by the eviction
					Action: batchv1.PodFailurePolicyActionIgnore,
					OnPodConditions: []batchv1.PodFailurePolicyOnPodConditionsPattern{
						{
							Type:   v1.DisruptionTarget,
							Status: v1.ConditionTrue,
						},
					},
				},
			},
		}),
		ginkgo.Entry("Ignore exit code 137", &batchv1.PodFailurePolicy{
			Rules: []batchv1.PodFailurePolicyRule{
				{
					// Ignore failures of the non 0-indexed pods which fail until the marker file is created
					// And the 127 in the 0-indexed pod due to eviction.
					Action: batchv1.PodFailurePolicyActionIgnore,
					OnExitCodes: &batchv1.PodFailurePolicyOnExitCodesRequirement{
						Operator: batchv1.PodFailurePolicyOnExitCodesOpIn,
						Values:   []int32{1, 137},
					},
				},
			},
		}),
	)

	ginkgo.It("should not create pods when created in suspend state", func(ctx context.Context) {
		ginkgo.By("Creating a job with suspend=true")
		job := e2ejob.NewTestJob("succeed", "suspend-true-to-false", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		job.Spec.Suspend = pointer.BoolPtr(true)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring pods aren't created for job")
		framework.ExpectEqual(wait.Poll(framework.Poll, wait.ForeverTestTimeout, func() (bool, error) {
			pods, err := e2ejob.GetJobPods(ctx, f.ClientSet, f.Namespace.Name, job.Name)
			if err != nil {
				return false, err
			}
			return len(pods.Items) > 0, nil
		}), wait.ErrWaitTimeout)

		ginkgo.By("Checking Job status to observe Suspended state")
		job, err = e2ejob.GetJob(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to retrieve latest job object")
		exists := false
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobSuspended {
				exists = true
				break
			}
		}
		if !exists {
			framework.Failf("Job was expected to be completed or failed")
		}

		ginkgo.By("Updating the job with suspend=false")
		job.Spec.Suspend = pointer.BoolPtr(false)
		job, err = e2ejob.UpdateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to update job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Waiting for job to complete")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, completions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)
	})

	ginkgo.It("should delete pods when suspended", func(ctx context.Context) {
		ginkgo.By("Creating a job with suspend=false")
		job := e2ejob.NewTestJob("notTerminate", "suspend-false-to-true", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		job.Spec.Suspend = pointer.Bool(false)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensure pods equal to parallelism count is attached to the job")
		err = e2ejob.WaitForJobPodsRunning(ctx, f.ClientSet, f.Namespace.Name, job.Name, parallelism)
		framework.ExpectNoError(err, "failed to ensure number of pods associated with job %s is equal to parallelism count in namespace: %s", job.Name, f.Namespace.Name)

		ginkgo.By("Updating the job with suspend=true")
		err = wait.PollImmediate(framework.Poll, framework.SingleCallTimeout, func() (bool, error) {
			job, err = e2ejob.GetJob(ctx, f.ClientSet, f.Namespace.Name, job.Name)
			if err != nil {
				return false, err
			}
			job.Spec.Suspend = pointer.Bool(true)
			updatedJob, err := e2ejob.UpdateJob(ctx, f.ClientSet, f.Namespace.Name, job)
			if err == nil {
				job = updatedJob
				return true, nil
			}
			if apierrors.IsConflict(err) {
				return false, nil
			}
			return false, err
		})
		framework.ExpectNoError(err, "failed to update job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring pods are deleted")
		err = e2ejob.WaitForAllJobPodsGone(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to ensure pods are deleted after suspend=true")

		ginkgo.By("Checking Job status to observe Suspended state")
		job, err = e2ejob.GetJob(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to retrieve latest job object")
		exists := false
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobSuspended {
				exists = true
				break
			}
		}
		if !exists {
			framework.Failf("Job was expected to be completed or failed")
		}
	})

	/*
		  Release: v1.24
			Testname: Ensure Pods of an Indexed Job get a unique index.
			Description: Create an Indexed job. Job MUST complete successfully.
			Ensure that created pods have completion index annotation and environment variable.
	*/
	framework.ConformanceIt("should create pods for an Indexed job with completion indexes and specified hostname", func(ctx context.Context) {
		ginkgo.By("Creating Indexed job")
		job := e2ejob.NewTestJob("succeed", "indexed-job", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		mode := batchv1.IndexedCompletion
		job.Spec.CompletionMode = &mode
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create indexed job in namespace %s", f.Namespace.Name)

		ginkgo.By("Ensuring job reaches completions")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, completions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring pods with index for job exist")
		pods, err := e2ejob.GetJobPods(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to get pod list for job in namespace: %s", f.Namespace.Name)
		succeededIndexes := sets.NewInt()
		for _, pod := range pods.Items {
			if pod.Status.Phase == v1.PodSucceeded && pod.Annotations != nil {
				ix, err := strconv.Atoi(pod.Annotations[batchv1.JobCompletionIndexAnnotation])
				framework.ExpectNoError(err, "failed obtaining completion index from pod in namespace: %s", f.Namespace.Name)
				succeededIndexes.Insert(ix)
				expectedName := fmt.Sprintf("%s-%d", job.Name, ix)
				framework.ExpectEqual(pod.Spec.Hostname, expectedName, "expected completed pod with hostname %s, but got %s", expectedName, pod.Spec.Hostname)
			}
		}
		gotIndexes := succeededIndexes.List()
		wantIndexes := []int{0, 1, 2, 3}
		framework.ExpectEqual(gotIndexes, wantIndexes, "expected completed indexes %s, but got %s", wantIndexes, gotIndexes)
	})

	/*
		Testcase: Ensure that the pods associated with the job are removed once the job is deleted
		Description: Create a job and ensure the associated pod count is equal to parallelism count. Delete the
		job and ensure if the pods associated with the job have been removed
	*/
	ginkgo.It("should remove pods when job is deleted", func(ctx context.Context) {
		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJob("notTerminate", "all-pods-removed", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensure pods equal to parallelism count is attached to the job")
		err = e2ejob.WaitForJobPodsRunning(ctx, f.ClientSet, f.Namespace.Name, job.Name, parallelism)
		framework.ExpectNoError(err, "failed to ensure number of pods associated with job %s is equal to parallelism count in namespace: %s", job.Name, f.Namespace.Name)

		ginkgo.By("Delete the job")
		err = e2eresource.DeleteResourceAndWaitForGC(ctx, f.ClientSet, batchinternal.Kind("Job"), f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to delete the job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensure the pods associated with the job are also deleted")
		err = e2ejob.WaitForAllJobPodsGone(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to get PodList for job %s in namespace: %s", job.Name, f.Namespace.Name)
	})

	/*
		Release: v1.16
		Testname: Jobs, completion after task failure
		Description: Explicitly cause the tasks to fail once initially. After restarting, the Job MUST
		execute to completion.
	*/
	framework.ConformanceIt("should run a job to completion when tasks sometimes fail and are locally restarted", func(ctx context.Context) {
		ginkgo.By("Creating a job")
		// One failure, then a success, local restarts.
		// We can't use the random failure approach, because kubelet will
		// throttle frequently failing containers in a given pod, ramping
		// up to 5 minutes between restarts, making test timeout due to
		// successive failures too likely with a reasonable test timeout.
		job := e2ejob.NewTestJob("failOnce", "fail-once-local", v1.RestartPolicyOnFailure, parallelism, completions, nil, backoffLimit)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring job reaches completions")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, completions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)
	})

	// Pods sometimes fail, but eventually succeed, after pod restarts
	ginkgo.It("should run a job to completion when tasks sometimes fail and are not locally restarted", func(ctx context.Context) {
		// One failure, then a success, no local restarts.
		// We can't use the random failure approach, because JobController
		// will throttle frequently failing Pods of a given Job, ramping
		// up to 6 minutes between restarts, making test timeout due to
		// successive failures.
		// Instead, we force the Job's Pods to be scheduled to a single Node
		// and use a hostPath volume to persist data across new Pods.
		ginkgo.By("Looking for a node to schedule job pod")
		node, err := e2enode.GetRandomReadySchedulableNode(ctx, f.ClientSet)
		framework.ExpectNoError(err)

		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJobOnNode("failOnce", "fail-once-non-local", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit, node.Name)
		job, err = e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring job reaches completions")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, *job.Spec.Completions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)
	})

	ginkgo.It("should fail when exceeds active deadline", func(ctx context.Context) {
		ginkgo.By("Creating a job")
		var activeDeadlineSeconds int64 = 1
		job := e2ejob.NewTestJob("notTerminate", "exceed-active-deadline", v1.RestartPolicyNever, parallelism, completions, &activeDeadlineSeconds, backoffLimit)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)
		ginkgo.By("Ensuring job past active deadline")
		err = waitForJobFailure(ctx, f.ClientSet, f.Namespace.Name, job.Name, time.Duration(activeDeadlineSeconds+15)*time.Second, "DeadlineExceeded")
		framework.ExpectNoError(err, "failed to ensure job past active deadline in namespace: %s", f.Namespace.Name)
	})

	/*
		Release: v1.15
		Testname: Jobs, active pods, graceful termination
		Description: Create a job. Ensure the active pods reflect parallelism in the namespace and delete the job. Job MUST be deleted successfully.
	*/
	framework.ConformanceIt("should delete a job", func(ctx context.Context) {
		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJob("notTerminate", "foo", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring active pods == parallelism")
		err = e2ejob.WaitForJobPodsRunning(ctx, f.ClientSet, f.Namespace.Name, job.Name, parallelism)
		framework.ExpectNoError(err, "failed to ensure active pods == parallelism in namespace: %s", f.Namespace.Name)

		ginkgo.By("delete a job")
		framework.ExpectNoError(e2eresource.DeleteResourceAndWaitForGC(ctx, f.ClientSet, batchinternal.Kind("Job"), f.Namespace.Name, job.Name))

		ginkgo.By("Ensuring job was deleted")
		_, err = e2ejob.GetJob(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectError(err, "failed to ensure job %s was deleted in namespace: %s", job.Name, f.Namespace.Name)
		if !apierrors.IsNotFound(err) {
			framework.Failf("failed to ensure job %s was deleted in namespace: %s", job.Name, f.Namespace.Name)
		}
	})

	/*
		Release: v1.16
		Testname: Jobs, orphan pods, re-adoption
		Description: Create a parallel job. The number of Pods MUST equal the level of parallelism.
		Orphan a Pod by modifying its owner reference. The Job MUST re-adopt the orphan pod.
		Modify the labels of one of the Job's Pods. The Job MUST release the Pod.
	*/
	framework.ConformanceIt("should adopt matching orphans and release non-matching pods", func(ctx context.Context) {
		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJob("notTerminate", "adopt-release", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		// Replace job with the one returned from Create() so it has the UID.
		// Save Kind since it won't be populated in the returned job.
		kind := job.Kind
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)
		job.Kind = kind

		ginkgo.By("Ensuring active pods == parallelism")
		err = e2ejob.WaitForJobPodsRunning(ctx, f.ClientSet, f.Namespace.Name, job.Name, parallelism)
		framework.ExpectNoError(err, "failed to ensure active pods == parallelism in namespace: %s", f.Namespace.Name)

		ginkgo.By("Orphaning one of the Job's Pods")
		pods, err := e2ejob.GetJobPods(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to get PodList for job %s in namespace: %s", job.Name, f.Namespace.Name)
		gomega.Expect(pods.Items).To(gomega.HaveLen(int(parallelism)))
		pod := pods.Items[0]
		e2epod.NewPodClient(f).Update(ctx, pod.Name, func(pod *v1.Pod) {
			pod.OwnerReferences = nil
		})

		ginkgo.By("Checking that the Job readopts the Pod")
		gomega.Expect(e2epod.WaitForPodCondition(ctx, f.ClientSet, pod.Namespace, pod.Name, "adopted", e2ejob.JobTimeout,
			func(pod *v1.Pod) (bool, error) {
				controllerRef := metav1.GetControllerOf(pod)
				if controllerRef == nil {
					return false, nil
				}
				if controllerRef.Kind != job.Kind || controllerRef.Name != job.Name || controllerRef.UID != job.UID {
					return false, fmt.Errorf("pod has wrong controllerRef: got %v, want %v", controllerRef, job)
				}
				return true, nil
			},
		)).To(gomega.Succeed(), "wait for pod %q to be readopted", pod.Name)

		ginkgo.By("Removing the labels from the Job's Pod")
		e2epod.NewPodClient(f).Update(ctx, pod.Name, func(pod *v1.Pod) {
			pod.Labels = nil
		})

		ginkgo.By("Checking that the Job releases the Pod")
		gomega.Expect(e2epod.WaitForPodCondition(ctx, f.ClientSet, pod.Namespace, pod.Name, "released", e2ejob.JobTimeout,
			func(pod *v1.Pod) (bool, error) {
				controllerRef := metav1.GetControllerOf(pod)
				if controllerRef != nil {
					return false, nil
				}
				return true, nil
			},
		)).To(gomega.Succeed(), "wait for pod %q to be released", pod.Name)
	})

	ginkgo.It("should fail to exceed backoffLimit", func(ctx context.Context) {
		ginkgo.By("Creating a job")
		backoff := 1
		job := e2ejob.NewTestJob("fail", "backofflimit", v1.RestartPolicyNever, 1, 1, nil, int32(backoff))
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)
		ginkgo.By("Ensuring job exceed backofflimit")

		err = waitForJobFailure(ctx, f.ClientSet, f.Namespace.Name, job.Name, e2ejob.JobTimeout, "BackoffLimitExceeded")
		framework.ExpectNoError(err, "failed to ensure job exceed backofflimit in namespace: %s", f.Namespace.Name)

		ginkgo.By(fmt.Sprintf("Checking that %d pod created and status is failed", backoff+1))
		pods, err := e2ejob.GetJobPods(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to get PodList for job %s in namespace: %s", job.Name, f.Namespace.Name)
		gomega.Expect(pods.Items).To(gomega.HaveLen(backoff + 1))
		for _, pod := range pods.Items {
			framework.ExpectEqual(pod.Status.Phase, v1.PodFailed)
		}
	})

	ginkgo.It("should run a job to completion with CPU requests [Serial]", func(ctx context.Context) {
		ginkgo.By("Creating a job that with CPU requests")

		testNodeName := scheduling.GetNodeThatCanRunPod(ctx, f)
		targetNode, err := f.ClientSet.CoreV1().Nodes().Get(ctx, testNodeName, metav1.GetOptions{})
		framework.ExpectNoError(err, "unable to get node object for node %v", testNodeName)

		cpu, ok := targetNode.Status.Allocatable[v1.ResourceCPU]
		if !ok {
			framework.Failf("Unable to get node's %q cpu", targetNode.Name)
		}

		cpuRequest := fmt.Sprint(int64(0.2 * float64(cpu.Value())))

		backoff := 0
		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJob("succeed", "all-succeed", v1.RestartPolicyNever, largeParallelism, largeCompletions, nil, int32(backoff))
		for i := range job.Spec.Template.Spec.Containers {
			job.Spec.Template.Spec.Containers[i].Resources = v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU: resource.MustParse(cpuRequest),
				},
			}
			job.Spec.Template.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": testNodeName}
		}

		framework.Logf("Creating job %q with a node hostname selector %q with cpu request %q", job.Name, testNodeName, cpuRequest)
		job, err = e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring job reaches completions")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, f.Namespace.Name, job.Name, largeCompletions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensuring pods for job exist")
		pods, err := e2ejob.GetJobPods(ctx, f.ClientSet, f.Namespace.Name, job.Name)
		framework.ExpectNoError(err, "failed to get pod list for job in namespace: %s", f.Namespace.Name)
		successes := int32(0)
		for _, pod := range pods.Items {
			if pod.Status.Phase == v1.PodSucceeded {
				successes++
			}
		}
		framework.ExpectEqual(successes, largeCompletions, "expected %d successful job pods, but got  %d", largeCompletions, successes)
	})

	/*
		Release: v1.24
		Testname: Jobs, apply changes to status
		Description: Attempt to create a running Job which MUST succeed.
		Attempt to patch the Job status to include a new start time which
		MUST succeed. An annotation for the job that was patched MUST be found.
		Attempt to replace the job status with a new start time which MUST
		succeed. Attempt to read its status sub-resource which MUST succeed
	*/
	framework.ConformanceIt("should apply changes to a job status", func(ctx context.Context) {

		ns := f.Namespace.Name
		jClient := f.ClientSet.BatchV1().Jobs(ns)

		ginkgo.By("Creating a job")
		job := e2ejob.NewTestJob("notTerminate", "suspend-false-to-true", v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		job, err := e2ejob.CreateJob(ctx, f.ClientSet, f.Namespace.Name, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", f.Namespace.Name)

		ginkgo.By("Ensure pods equal to parallelism count is attached to the job")
		err = e2ejob.WaitForJobPodsRunning(ctx, f.ClientSet, f.Namespace.Name, job.Name, parallelism)
		framework.ExpectNoError(err, "failed to ensure number of pods associated with job %s is equal to parallelism count in namespace: %s", job.Name, f.Namespace.Name)

		// /status subresource operations
		ginkgo.By("patching /status")
		// we need to use RFC3339 version since conversion over the wire cuts nanoseconds
		now1 := metav1.Now().Rfc3339Copy()
		jStatus := batchv1.JobStatus{
			StartTime: &now1,
		}

		jStatusJSON, err := json.Marshal(jStatus)
		framework.ExpectNoError(err)
		patchedStatus, err := jClient.Patch(ctx, job.Name, types.MergePatchType,
			[]byte(`{"metadata":{"annotations":{"patchedstatus":"true"}},"status":`+string(jStatusJSON)+`}`),
			metav1.PatchOptions{}, "status")
		framework.ExpectNoError(err)
		if !patchedStatus.Status.StartTime.Equal(&now1) {
			framework.Failf("patched object should have the applied StartTime %#v, got %#v instead", jStatus.StartTime, patchedStatus.Status.StartTime)
		}
		framework.ExpectEqual(patchedStatus.Annotations["patchedstatus"], "true", "patched object should have the applied annotation")

		ginkgo.By("updating /status")
		// we need to use RFC3339 version since conversion over the wire cuts nanoseconds
		now2 := metav1.Now().Rfc3339Copy()
		var statusToUpdate, updatedStatus *batchv1.Job
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			statusToUpdate, err = jClient.Get(ctx, job.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			statusToUpdate.Status.StartTime = &now2
			updatedStatus, err = jClient.UpdateStatus(ctx, statusToUpdate, metav1.UpdateOptions{})
			return err
		})
		framework.ExpectNoError(err)
		if !updatedStatus.Status.StartTime.Equal(&now2) {
			framework.Failf("updated object status expected to have updated StartTime %#v, got %#v", statusToUpdate.Status.StartTime, updatedStatus.Status.StartTime)
		}

		ginkgo.By("get /status")
		jResource := schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
		gottenStatus, err := f.DynamicClient.Resource(jResource).Namespace(ns).Get(ctx, job.Name, metav1.GetOptions{}, "status")
		framework.ExpectNoError(err)
		statusUID, _, err := unstructured.NestedFieldCopy(gottenStatus.Object, "metadata", "uid")
		framework.ExpectNoError(err)
		framework.ExpectEqual(string(job.UID), statusUID, fmt.Sprintf("job.UID: %v expected to match statusUID: %v ", job.UID, statusUID))
	})

	/*
		Release: v1.25
		Testname: Jobs, manage lifecycle
		Description: Attempt to create a suspended Job which MUST succeed.
		Attempt to patch the Job to include a new label which MUST succeed.
		The label MUST be found. Attempt to replace the Job to include a
		new annotation which MUST succeed. The annotation MUST be found.
		Attempt to list all namespaces with a label selector which MUST
		succeed. One list MUST be found. It MUST succeed at deleting a
		collection of jobs via a label selector.
	*/
	framework.ConformanceIt("should manage the lifecycle of a job", func(ctx context.Context) {
		jobName := "e2e-" + utilrand.String(5)
		label := map[string]string{"e2e-job-label": jobName}
		labelSelector := labels.SelectorFromSet(label).String()

		ns := f.Namespace.Name
		jobClient := f.ClientSet.BatchV1().Jobs(ns)

		w := &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector
				return jobClient.Watch(ctx, options)
			},
		}
		jobsList, err := jobClient.List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		framework.ExpectNoError(err, "failed to list Job")

		ginkgo.By("Creating a suspended job")
		job := e2ejob.NewTestJob("succeed", jobName, v1.RestartPolicyNever, parallelism, completions, nil, backoffLimit)
		job.Labels = label
		job.Spec.Suspend = pointer.BoolPtr(true)
		job, err = e2ejob.CreateJob(ctx, f.ClientSet, ns, job)
		framework.ExpectNoError(err, "failed to create job in namespace: %s", ns)

		ginkgo.By("Patching the Job")
		payload := "{\"metadata\":{\"labels\":{\"" + jobName + "\":\"patched\"}}}"
		patchedJob, err := f.ClientSet.BatchV1().Jobs(ns).Patch(ctx, jobName, types.StrategicMergePatchType, []byte(payload), metav1.PatchOptions{})
		framework.ExpectNoError(err, "failed to patch Job %s in namespace %s", jobName, ns)

		ginkgo.By("Watching for Job to be patched")
		c := watchEventConfig{
			framework:           f,
			resourceVersion:     jobsList.ResourceVersion,
			w:                   w,
			jobName:             jobName,
			watchEvent:          watch.Modified,
			extJob:              patchedJob,
			updatedMetadataType: "label",
			updatedKey:          jobName,
			updatedValue:        "patched",
		}
		waitForJobEvent(ctx, c)
		framework.ExpectEqual(patchedJob.Labels[jobName], "patched", "Did not find job label for this job. Current labels: %v", patchedJob.Labels)

		ginkgo.By("Updating the job")
		var updatedJob *batchv1.Job

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			patchedJob, err = jobClient.Get(ctx, jobName, metav1.GetOptions{})
			framework.ExpectNoError(err, "Unable to get job %s", jobName)
			patchedJob.Spec.Suspend = pointer.BoolPtr(false)
			if patchedJob.Annotations == nil {
				patchedJob.Annotations = map[string]string{}
			}
			patchedJob.Annotations["updated"] = "true"
			updatedJob, err = e2ejob.UpdateJob(ctx, f.ClientSet, ns, patchedJob)
			return err
		})
		framework.ExpectNoError(err, "failed to update job in namespace: %s", ns)

		ginkgo.By("Watching for Job to be updated")
		c = watchEventConfig{
			framework:           f,
			resourceVersion:     patchedJob.ResourceVersion,
			w:                   w,
			jobName:             jobName,
			watchEvent:          watch.Modified,
			extJob:              updatedJob,
			updatedMetadataType: "annotation",
			updatedKey:          "updated",
			updatedValue:        "true",
		}
		waitForJobEvent(ctx, c)
		framework.ExpectEqual(updatedJob.Annotations["updated"], "true", "updated Job should have the applied annotation")
		framework.Logf("Found Job annotations: %#v", patchedJob.Annotations)

		ginkgo.By("Listing all Jobs with LabelSelector")
		jobs, err := f.ClientSet.BatchV1().Jobs("").List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		framework.ExpectNoError(err, "Failed to list job. %v", err)
		framework.ExpectEqual(len(jobs.Items), 1, "Failed to find job %v", jobName)
		testJob := jobs.Items[0]
		framework.Logf("Job: %v as labels: %v", testJob.Name, testJob.Labels)

		ginkgo.By("Waiting for job to complete")
		err = e2ejob.WaitForJobComplete(ctx, f.ClientSet, ns, jobName, completions)
		framework.ExpectNoError(err, "failed to ensure job completion in namespace: %s", ns)

		ginkgo.By("Delete a job collection with a labelselector")
		propagationPolicy := metav1.DeletePropagationBackground
		err = f.ClientSet.BatchV1().Jobs(ns).DeleteCollection(ctx, metav1.DeleteOptions{PropagationPolicy: &propagationPolicy}, metav1.ListOptions{LabelSelector: labelSelector})
		framework.ExpectNoError(err, "failed to delete job %s in namespace: %s", job.Name, ns)

		ginkgo.By("Watching for Job to be deleted")
		c = watchEventConfig{
			framework:           f,
			resourceVersion:     updatedJob.ResourceVersion,
			w:                   w,
			jobName:             jobName,
			watchEvent:          watch.Deleted,
			extJob:              &testJob,
			updatedMetadataType: "label",
			updatedKey:          "e2e-job-label",
			updatedValue:        jobName,
		}
		waitForJobEvent(ctx, c)

		ginkgo.By("Relist jobs to confirm deletion")
		jobs, err = f.ClientSet.BatchV1().Jobs("").List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		framework.ExpectNoError(err, "Failed to list job. %v", err)
		framework.ExpectEqual(len(jobs.Items), 0, "Found job %v", jobName)
	})

})

// waitForJobEvent is used to track and log Job events.
// As delivery of events is not actually guaranteed we
// will not return an error if we miss the required event.
func waitForJobEvent(ctx context.Context, config watchEventConfig) {
	f := config.framework
	ctx, cancel := context.WithTimeout(ctx, f.Timeouts.PodStartShort)
	defer cancel()
	_, err := watchtools.Until(ctx, config.resourceVersion, config.w, func(event watch.Event) (bool, error) {
		if job, ok := event.Object.(*batchv1.Job); ok {

			var key string
			switch config.updatedMetadataType {
			case "annotation":
				key = job.Annotations[config.updatedKey]
			case "label":
				key = job.Labels[config.updatedKey]
			}

			found := job.ObjectMeta.Name == config.extJob.ObjectMeta.Name &&
				job.ObjectMeta.Namespace == f.Namespace.Name &&
				key == config.updatedValue &&
				event.Type == config.watchEvent
			if !found {
				framework.Logf("Event %v observed for Job %v in namespace %v with labels: %v and annotations: %v", event.Type, job.ObjectMeta.Name, job.ObjectMeta.Namespace, job.Labels, job.Annotations)
				return false, nil
			}
			framework.Logf("Event %v found for Job %v in namespace %v with labels: %v and annotations: %v", event.Type, job.ObjectMeta.Name, job.ObjectMeta.Namespace, job.Labels, job.Annotations)
			return found, nil
		}
		framework.Logf("Observed event: %+v", event.Object)
		return false, nil
	})
	if err != nil {
		j, _ := f.ClientSet.BatchV1().Jobs(f.Namespace.Name).Get(ctx, config.jobName, metav1.GetOptions{})
		framework.Logf("We missed the %v event. Job details: %+v", config.watchEvent, j)
	}
}

// waitForJobFailure uses c to wait for up to timeout for the Job named jobName in namespace ns to fail.
func waitForJobFailure(ctx context.Context, c clientset.Interface, ns, jobName string, timeout time.Duration, reason string) error {
	return wait.Poll(framework.Poll, timeout, func() (bool, error) {
		curr, err := c.BatchV1().Jobs(ns).Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, c := range curr.Status.Conditions {
			if c.Type == batchv1.JobFailed && c.Status == v1.ConditionTrue {
				if reason == "" || reason == c.Reason {
					return true, nil
				}
			}
		}
		return false, nil
	})
}
