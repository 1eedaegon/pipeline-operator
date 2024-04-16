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
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	apiGVStr = pipelinev1.GroupVersion.String()
)

const (
	jobOwnerKey             = ".metadata.controller"
	updatedByAnnotation     = "pipeline.1eedaegon.github.io/updated-at"
	createdByAnnotation     = "pipeline.1eedaegon.github.io/created-by"
	createdTimeAnnotation   = "pipeline.1eedaegon.github.io/created-at"
	scheduledTimeAnnotation = "pipeline.1eedaegon.github.io/schedule-at"
	pipelineNameLabel       = "pipeline.1eedaegon.github.io/pipeline-name"
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
	pipeline := &pipelinev1.Pipeline{}
	// 하나의 파이프라인에게 동작한다고 생각하자.

	// Checking pipeline CRD
	log.Info("Reconciling pipeline.")
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		log.V(1).Error(err, "unable to fetch pipeline")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update pipeline status(with sum)
	if err := r.ensurePipeline(ctx, pipeline); err != nil {
		log.V(1).Error(err, "unable to ensure pipeline")
		return ctrl.Result{}, err
	}
	// Construct run(CRD) from pipeline template
	run := &pipelinev1.Run{}
	if err := pipelinev1.NewRunFromPipeline(ctx, pipeline, run); err != nil {
		log.V(1).Error(err, "Unable to parse from pipeline")
		return ctrl.Result{}, err
	}
	// Relation owner run -> pipeline(owner)
	if err := ctrl.SetControllerReference(pipeline, run, r.Scheme); err != nil {
		log.V(1).Error(err, "Unable to reference between pipeline and new run")
		return ctrl.Result{}, err
	}
	// Create run
	if err := r.Create(ctx, run); err != nil {
		log.V(1).Error(err, "Unable to create run")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PipelineReconciler) ensurePipeline(ctx context.Context, pipeline *pipelinev1.Pipeline) error {
	if _, find := pipeline.ObjectMeta.Labels["pipeline.1eedaegon.github.io/pipeline-name"]; !find {
		pipeline.ObjectMeta.Labels["pipeline.1eedaegon.github.io/pipeline-name"] = pipeline.Name
	}
	if pipeline.Spec.Schedule.ScheduleDate != "" {
		pipeline.ObjectMeta.Annotations["pipeline.1eedaegon.github.io/schedule-date"] = string(pipeline.Spec.Schedule.ScheduleDate)
		// TODO: 아래 타입은 runs로
		// pipeline.ObjectMeta.Annotations["pipeline.1eedaegon.github.io/scheduled-at"] = duration
	}
	if pipeline.Spec.Trigger {
		pipeline.ObjectMeta.Annotations["pipeline.1eedaegon.github.io/trigger"] = strconv.FormatBool(pipeline.Spec.Trigger)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.Pipeline{}).
		Named("Pipeline").
		Watches(
			&pipelinev1.Pipeline{},
			handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &pipelinev1.Run{}),
		).
		Owns(&pipelinev1.Run{}).
		Complete(r)
}
