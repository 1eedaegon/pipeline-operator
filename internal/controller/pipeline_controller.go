/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
	"github.com/robfig/cron"
	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	scheduledTimeAnnotation = "batch.1eedaegon.github.io/scheduled-at"
)

type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

type Clock interface {
	Now() time.Time
}

// PipelineReconciler reconciles a Pipeline object
type PipelineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=pipelines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=pipelines/finalizers,verbs=update
func (r *PipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var pipeline pipelinev1.Pipeline
	if err := r.Get(ctx, req.NamespacedName, &pipeline); err != nil {
		log.Error(err, "unable to fetch Pipeline")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, err
	}

	// Active 상태인 pipeline 목록을 가져오고 상태 업데이트
	// jobOwnerKey를 이용해서 index를 memory에 cache
	var childJobs kbatchv1.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child jobs")
		return ctrl.Result{}, err
	}

	var activeJobs []*kbatchv1.Job
	var successfulJobs []*kbatchv1.Job
	var failedJobs []*kbatchv1.Job
	var mostRecentTime *time.Time

	isJobFinished := func(job *kbatchv1.Job) (bool, kbatchv1.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == kbatchv1.JobComplete || c.Type == kbatchv1.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}
		return false, ""
	}

	// getScheduledTimeForJob
	getScheduledTimeForJob := func(job *kbatchv1.Job) (*time.Time, error) {
		timeRaw := job.Annotations[scheduledTimeAnnotation]
		if len(timeRaw) == 0 {
			return nil, nil
		}

		timeParsed, err := time.Parse(time.RFC3339, timeRaw)
		if err != nil {
			return nil, err
		}
		return &timeParsed, nil
	}

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatchv1.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatchv1.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}
		// 시작시간을 어노테이션에 담아야 한다. 따라서 active jobs이 스스로 담게한다.
		scheduledTimeForJob, err := getScheduledTimeForJob(&job)
		if err != nil {
			log.Error(err, "unable to parse schedule time for child job", "job", &job)
			continue
		}
		if scheduledTimeForJob != nil {
			if mostRecentTime == nil || mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}
	if mostRecentTime != nil {
		pipeline.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		pipeline.Status.LastScheduleTime = nil
	}
	pipeline.Status.Active = nil

	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		pipeline.Status.Active = append(pipeline.Status.Active, *jobRef)
	}
	log.V(1).Info("job count", "active job", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	if err := r.Status().Update(ctx, &pipeline); err != nil {
		log.Error(err, "unable to update pipeline status")
		return ctrl.Result{}, err
	}

	if err := r.Status().Update(ctx, &pipeline); err != nil {
		log.Error(err, "unable to update pipeline status")
		return ctrl.Result{}, err
	}

	if pipeline.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i] == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if int32(i) >= int32(len(successfulJobs))-*pipeline.Spec.SuccessfulJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete old success job", "job", job)
			} else {
				log.V(0).Info("deleted old success job", "job", job)
			}
		}
	}
	if pipeline.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i] == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if int32(i) >= int32(len(failedJobs))-*pipeline.Spec.FailedJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			} else {
				log.V(0).Info("deleted old failed job", "job", job)
			}
		}
	}
	if pipeline.Spec.Suspend != nil && *pipeline.Spec.Suspend {
		log.V(1).Info("pipeline suspended, skipping")
		return ctrl.Result{}, nil
	}

	getNextSchedule := func(pipeline *pipelinev1.pipeline, now time.Time) (lastMissed time.Time, next time.Time, err error) {
		sched, err := cron.ParseStandard(pipeline.Spec.Schedule)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("Unparseable schedule %q: %v", pipeline.Spec.Schedule, err)
		}

		var earliestTime time.Time
		if pipeline.Status.LastScheduleTime != nil {
			earliestTime = pipeline.Status.LastScheduleTime.Time
		} else {
			earliestTime = pipeline.ObjectMeta.CreationTimestamp.Time
		}
		if pipeline.Spec.StartingDeadlineSeconds != nil {
			schedulingDeadline := now.Add(-time.Second * time.Duration(*pipeline.Spec.StartingDeadlineSeconds))
			if schedulingDeadline.After(earliestTime) {
				earliestTime = schedulingDeadline
			}
		}
		if earliestTime.After(now) {
			return time.Time{}, sched.Next(now), nil
		}
		/*
			주의: 컨트롤러는 여러 번의 시작을 놓칠 수 있다.
			예를 들어, 금요일 오후 5시 1분에 전직원이 퇴근 후 컨트롤러가 멈췄는데
			누군가 화요일 오전에 문제를 발견하고 해결한다음 컨트롤러를 다시 시작하면
			각 시간별 예약된 작업은 추가 개입 없이 모두 실행을 시작해야 한다.
			하지만 어딘가에 버그가 있거나 컨트롤러 서버 또는 API 서버에 잘못된 클락이 있는 경우
			시작 하지 못한 잡이 너무 많이 누적되어 CPU를 모두 소모할 수 있다.
		*/
		starts := 0
		for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
			lastMissed = t
			starts++
			if starts > 100 {
				return time.Time{},
					time.Time{},
					fmt.Errorf("Too many missed start times (> 100). set or decreased .spec.startingDeadlineSeconds or check clock skew.")
			}
		}
		return lastMissed, sched.Next(now), nil
	}
	missedRun, nextRun, err := getNextSchedule(&pipeline, r.Now())
	if err != nil {
		log.Error(err, "unable to figure out pipeline schedule")
		return ctrl.Result{}, nil
	}
	scheduledResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())}
	log = log.WithValues("now", r.Now(), "next run", nextRun)
	if missedRun.IsZero() {
		log.V(1).Info("no upcomming scheduled times, sleeping until next")
		return scheduledResult, nil
	}
	log = log.WithValues("current run", missedRun)
	tooLate := false
	if pipeline.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*pipeline.Spec.StartingDeadlineSeconds) * time.Second).Before(r.Now())
	}
	if tooLate {
		log.V(1).Info("missed starting deadline for last run, sleeping till next")
		return scheduledResult, nil
	}

	if pipeline.Spec.ConcurrencyPolicy == pipelinev1.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return scheduledResult, nil
	}
	if pipeline.Spec.ConcurrencyPolicy == pipelinev1.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			if err := r.Delete(ctx, activeJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return ctrl.Result{}, err
			}
		}
	}
	constructJobForPipeline := func(pipeline *pipelinev1.pipeline, scheduleTime time.Time) (*kbatchv1.Job, error) {
		name := fmt.Sprintf("%s-%d", pipeline.Name, scheduleTime.Unix())

		job := &kbatchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        name,
				Namespace:   pipeline.Namespace,
			},
			Spec: *pipeline.Spec.JobTemplate.Spec.DeepCopy(),
		}

		for k, v := range pipeline.Spec.JobTemplate.Annotations {
			job.Annotations[k] = v
		}
		job.Annotations[scheduledTimeAnnotation] = scheduleTime.Format(time.RFC3339)
		for k, v := range pipeline.Spec.JobTemplate.Labels {
			job.Labels[k] = v
		}
		if err := ctrl.SetControllerReference(pipeline, job, r.Scheme); err != nil {
			return nil, err
		}
		return job, nil
	}
	job, err := constructJobForPipeline(&pipeline, missedRun)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		return scheduledResult, nil
	}

	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for pipeline", "job", job)
		return ctrl.Result{}, err
	}
	log.V(1).Info("created Job for pipeline run", "job", job)
	return scheduledResult, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.Pipeline{}).
		Owns(&kbatchv1.Job{}).
		Complete(r)
}
