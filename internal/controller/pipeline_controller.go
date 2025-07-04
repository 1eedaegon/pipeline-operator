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
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
	"github.com/r3labs/diff/v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	apiGVStr = pipelinev1.GroupVersion.String()
)

const (
	runOwnerKey = ".metadata.labels" + pipelinev1.PipelineNameLabel
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
// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=runs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=runs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=runs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
func (r *PipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	pipeline := &pipelinev1.Pipeline{}

	// Checking pipeline CRD
	log.Info("Reconciling pipeline.")
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		log.V(1).Error(err, "unable to fetch pipeline")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update pipeline with status(with sum)
	if err := r.updatePipelineStatus(ctx, pipeline); err != nil {
		log.V(1).Error(err, "unable to update pipeline status")
		return ctrl.Result{}, err
	}

	// Ensure pipeline annotations and labels
	if err := r.ensurePipelineMetadata(ctx, pipeline); err != nil {
		log.V(1).Error(err, "Unable to ensure pipeline")
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		log.V(1).Error(err, "unable to fetch pipeline")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check the list of runs and create one if there are changes or none
	if err := r.ensureRunExists(ctx, pipeline); err != nil {
		log.V(1).Error(err, "Unable to ensure run exists for pipeline")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// TODO: Must be implement constraints in pipeline controller
func (r *PipelineReconciler) ensurePipelineMetadata(ctx context.Context, pipeline *pipelinev1.Pipeline) error {
	objKey := client.ObjectKey{
		Name:      pipeline.ObjectMeta.Name,
		Namespace: pipeline.ObjectMeta.Namespace,
	}

	// Retry backoff
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pipeline = &pipelinev1.Pipeline{}
		err := r.Get(ctx, objKey, pipeline)
		if err != nil {
			return err
		}

		if pipeline.ObjectMeta.Namespace == "" {
			pipeline.ObjectMeta.Namespace = "pipeline"
		}

		objectMeta := pipeline.ObjectMeta
		if objectMeta.Annotations == nil {
			objectMeta.Annotations = make(map[string]string)
		}
		if objectMeta.Labels == nil {
			objectMeta.Labels = make(map[string]string)
		}
		objectMeta.Annotations[pipelinev1.TriggerAnnotation] = pipeline.Spec.Trigger.String()
		objectMeta.Labels[pipelinev1.PipelineNameLabel] = pipeline.ObjectMeta.Name

		if pipeline.Spec.Schedule != nil && pipeline.Spec.Schedule.ScheduleDate != "" {
			objectMeta.Annotations[pipelinev1.ScheduleDateAnnotation] = string(pipeline.Spec.Schedule.ScheduleDate)
		}

		if !reflect.DeepEqual(pipeline.ObjectMeta, objectMeta) {
			pipeline.ObjectMeta = objectMeta
		}

		return r.Update(ctx, pipeline)
	}); err != nil {
		return err
	}
	return nil
}

func (r *PipelineReconciler) ensureRunExists(ctx context.Context, pipeline *pipelinev1.Pipeline) error {
	if pipeline.Spec.Schedule.ScheduleDate != "" {
		return nil
	}

	log := log.FromContext(ctx)
	run := &pipelinev1.Run{}

	if err := pipelinev1.ConstructRunFromPipeline(ctx, pipeline, run); err != nil {
		log.V(1).Error(err, "Unable to parse run from pipeline")
		return err
	}

	objKey := client.ObjectKey{
		Name:      run.ObjectMeta.Name,
		Namespace: run.ObjectMeta.Namespace,
	}
	if err := r.Get(ctx, objKey, run); err != nil {
		if !apierrors.IsNotFound(err) {
			log.V(1).Error(err, "Unknown error")
			return err
		}
		// Relation owner run -> pipeline(owner)
		if err := ctrl.SetControllerReference(pipeline, run, r.Scheme); err != nil {
			log.V(1).Error(err, "Unable to reference between pipeline and new run")
			return err
		}
		if err := r.Create(ctx, run); err != nil {
			log.V(1).Error(err, "Unable to create run")
			return err
		}
	}
	return nil
}

func (r *PipelineReconciler) updatePipelineStatus(ctx context.Context, pipeline *pipelinev1.Pipeline) error {
	log := log.FromContext(ctx)
	log.V(1).Info(fmt.Sprintf("update pipeline status."))
	runList := &pipelinev1.RunList{}

	listQueryOpts := []client.ListOption{
		client.InNamespace(pipeline.ObjectMeta.Namespace),
		client.MatchingLabels(labels.Set{pipelinev1.PipelineNameLabel: pipeline.ObjectMeta.Name}),
	}

	objKey := client.ObjectKey{
		Name:      pipeline.ObjectMeta.Name,
		Namespace: pipeline.ObjectMeta.Namespace,
	}

	if err := r.List(ctx, runList, listQueryOpts...); err != nil {
		return err
	}

	// Retry backoff
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pipeline = &pipelinev1.Pipeline{}
		err := r.Get(ctx, objKey, pipeline)
		if err != nil {
			return err
		}
		pipeline.Status.Runs = len(runList.Items)
		pipeline.Status.CreatedDate = &pipeline.ObjectMeta.CreationTimestamp
		pipeline.Status.LastUpdatedDate = &metav1.Time{Time: time.Now()}

		if pipeline.Spec.Schedule == nil && pipeline.Status.Schedule == nil {
			return r.Status().Update(ctx, pipeline)
		}

		if pipeline.Spec.Schedule == nil && pipeline.Status.Schedule != nil {
			log.V(1).Info(fmt.Sprintf("Removing schedule status due to schedule spec removal."))
			pipeline.Status.Schedule = nil
			pipeline.Status.ScheduleStartDate = nil
			pipeline.Status.ScheduleLastExecutedDate = nil
			pipeline.Status.ScheduleRepeated = 0
			pipeline.Status.SchedulePendingExecuctionDate = nil
			return r.Status().Update(ctx, pipeline)
		}

		hasDiff := true

		if pipeline.Spec.Schedule != nil && pipeline.Status.Schedule != nil {
			changelog, err := diff.Diff(*pipeline.Spec.Schedule, *pipeline.Status.Schedule)
			if err != nil {
				return err
			}
			hasDiff = len(changelog) > 0
		}

		if !hasDiff {
			return r.Status().Update(ctx, pipeline)
		}

		if pipeline.Status.SchedulePendingExecuctionDate != nil && pipeline.Spec.Schedule.ScheduleDate == pipeline.Status.Schedule.ScheduleDate {
			if pipeline.Status.SchedulePendingExecuctionDate.After(time.Now()) {
				log.V(1).Info(fmt.Sprintf("Do not modify schedule on status due to schedule is already running (SchedulePendingExecutionDate: %v, ScheduleDate: %v)", pipeline.Status.SchedulePendingExecuctionDate, pipeline.Spec.Schedule.ScheduleDate))
				return nil
			} else {
				log.V(1).Info(fmt.Sprintf("Removing previous pending execution due to has unexpectedly terminated schedule handler (SchedulePendingExecutionDate: %v, ScheduleDate: %v)", pipeline.Status.SchedulePendingExecuctionDate, pipeline.Spec.Schedule.ScheduleDate))
				pipeline.Status.SchedulePendingExecuctionDate = nil
			}
		}

		if pipeline.Spec.Schedule.ScheduleDate == "" {
			pipeline.Status.ScheduleStartDate = nil
			return r.Status().Update(ctx, pipeline)
		}

		duration, err := pipeline.Spec.Schedule.ScheduleDate.Duration()
		if err != nil {
			return err
		}

		if pipeline.Spec.Schedule.EndDate != nil && pipeline.Status.SchedulePendingExecuctionDate != nil && pipeline.Status.SchedulePendingExecuctionDate.Before(pipeline.Spec.Schedule.EndDate) {
			log.V(1).Info(fmt.Sprintf("ScheduleExecutionDate is before than EndDate; Do not schedule for run. (ScheduleExecutionDate: %v, EndDate: %v)", pipeline.Status.SchedulePendingExecuctionDate, pipeline.Spec.Schedule.EndDate))
			pipeline.Status.SchedulePendingExecuctionDate = nil
			return r.Status().Update(ctx, pipeline)
		}

		if !pipeline.Spec.Schedule.Repeat && pipeline.Status.ScheduleRepeated > 0 {
			log.V(1).Info(fmt.Sprintf("Do not create schedule for run due to repeat is disabled and schedule is already executed. (ScheduleExecutionDate: %v, EndDate: %v)", pipeline.Status.SchedulePendingExecuctionDate, pipeline.Spec.Schedule.EndDate))
			return nil
		}

		if pipeline.Status.Schedule == nil || pipeline.Status.Schedule.ScheduleDate != pipeline.Spec.Schedule.ScheduleDate {
			pipeline.Status.ScheduleLastExecutedDate = nil
			pipeline.Status.ScheduleRepeated = 0
		}

		pipeline.Status.Schedule = pipeline.Spec.Schedule.DeepCopy()
		pipeline.Status.ScheduleStartDate = &metav1.Time{Time: time.Now()}
		pipeline.Status.SchedulePendingExecuctionDate = &metav1.Time{Time: pipeline.Status.ScheduleStartDate.Add(duration)}

		if err := r.Status().Update(ctx, pipeline); err != nil {
			return err
		}

		log.V(1).Info(fmt.Sprintf("Scheduling pipeline %v at %v", pipeline.ObjectMeta.Name, pipeline.Status.SchedulePendingExecuctionDate))
		go r.ScheduleExecution(ctx, pipeline.DeepCopy())

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (r *PipelineReconciler) ScheduleExecution(ctx context.Context, pipeline *pipelinev1.Pipeline) error {
	log := log.FromContext(ctx)
	scheduledTime := time.Now()

	if pipeline.Status.Schedule == nil {
		return nil
	}
	if pipeline.Status.SchedulePendingExecuctionDate == nil {
		return nil
	}

	pendingDuration := pipeline.Status.SchedulePendingExecuctionDate.Sub(scheduledTime)

	time.Sleep(pendingDuration)

	objKey := client.ObjectKey{
		Name:      pipeline.ObjectMeta.Name,
		Namespace: pipeline.ObjectMeta.Namespace,
	}

	now := time.Now()

	log.V(1).Info(fmt.Sprintf("Executing schedule at expected time %v, actual time %v", pipeline.Status.SchedulePendingExecuctionDate, now))

	currentPipeline := &pipelinev1.Pipeline{}

	if err := r.Get(ctx, objKey, currentPipeline); err != nil {
		log.V(1).Info(fmt.Sprintf("Unable to get currentPipeline: %v", err))
		return err
	}

	if !pipeline.Status.ScheduleStartDate.Time.Equal(currentPipeline.Status.ScheduleStartDate.Time) {
		log.V(1).Info(fmt.Sprintf("SchduleStartDate of currentPipeline changed. Aborting Creating Run."))
		return nil
	}

	if !pipeline.Status.SchedulePendingExecuctionDate.Time.Equal(currentPipeline.Status.SchedulePendingExecuctionDate.Time) {
		log.V(1).Info(fmt.Sprintf("SchedulePendingExecutionDate of currentPipeline changed. Aborting Creating Run."))
		return nil
	}

	if (pipeline.Status.Schedule.EndDate == nil && currentPipeline.Status.Schedule.EndDate != nil) || (pipeline.Status.Schedule.EndDate != nil && currentPipeline.Status.Schedule.EndDate == nil) {
		log.V(1).Info(fmt.Sprintf("EndDate of currentPipeline changed. Aborting Creating Run."))
		return nil
	}

	if pipeline.Status.Schedule.EndDate != nil && currentPipeline.Status.Schedule.EndDate != nil && !pipeline.Status.Schedule.EndDate.Time.Equal(currentPipeline.Status.Schedule.EndDate.Time) {
		log.V(1).Info(fmt.Sprintf("EndDate of currentPipeline changed. Aborting Creating Run."))
		return nil
	}

	if pipeline.Status.Schedule.Repeat != currentPipeline.Status.Schedule.Repeat {
		log.V(1).Info(fmt.Sprintf("Repeat of currentPipeline changed. Aborting Creating Run."))
		return nil
	}

	if pipeline.Status.ScheduleRepeated != currentPipeline.Status.ScheduleRepeated {
		log.V(1).Info(fmt.Sprintf("ScheduleRepeated of currentPipeline changed. Aborting Creating Run."))
		return nil
	}

	now = time.Now()

	if pipeline.Status.Schedule.EndDate != nil && now.After(pipeline.Status.Schedule.EndDate.Time) {
		log.V(1).Info(fmt.Sprintf("Executed time is after EndDate. Aborting Creating Run."))
		return nil
	}

	run := &pipelinev1.Run{}

	if err := pipelinev1.ConstructRunFromPipeline(ctx, currentPipeline, run); err != nil {
		log.V(1).Error(err, "Unable to parse run from currentPipeline during Schedule Execution")
		return err
	}

	run.Status.ScheduleExecutedDate = &metav1.Time{Time: now}

	if err := r.Create(ctx, run); err != nil {
		log.V(1).Error(err, "Unable to create run")
		return err
	}

	currentPipeline.Status.ScheduleRepeated = currentPipeline.Status.ScheduleRepeated + 1
	currentPipeline.Status.ScheduleLastExecutedDate = &metav1.Time{Time: now}
	currentPipeline.Status.SchedulePendingExecuctionDate = nil

	if err := r.Status().Update(ctx, currentPipeline); err != nil {
		log.V(1).Error(err, "Unable to update pipeline status")
		return err
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
// TODO: testing pipieline watcher queue
func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &pipelinev1.Run{}, runOwnerKey, func(rawObj client.Object) []string {
		run := rawObj.(*pipelinev1.Run)
		return []string{run.ObjectMeta.Labels[pipelinev1.PipelineNameLabel]}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.Pipeline{}).
		Watches(
			&pipelinev1.Pipeline{},
			handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &pipelinev1.Run{}),
		).
		Owns(&pipelinev1.Run{}).
		Complete(r)
}
