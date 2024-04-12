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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
	kbatchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	apiGVStr = pipelinev1.GroupVersion.String()
)

const (
	scheduledTimeAnnotation = "pipeline.1eedaegon.github.io/scheduled-at"
	jobOwnerKey             = ".metadata.controller"
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

	// 1. pipeline yaml이 하나라도 있는지 확인
	log.Info("Reconciling pipeline.")
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		log.Error(err, "unable to fetch pipeline")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. enduring volume name
	volume := pipeline.Spec.VolumeName
	log.Info("Ensuring volume", "volume", volume)
	r.ensureVolume(pipeline.Spec.VolumeName)

	// 3. enduring history limit
	runList := &pipelinev1.RunList{}
	err := r.Get(ctx, client.ObjectKey{Name: })
	return ctrl.Result{}, nil
}

func (r *PipelineReconciler) ValidatePipelineRequest(ctx context.Context, req ctrl.Request, pipeline *pipelinev1.Pipeline) error {
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		return err
	}
	return nil
}

func (r *PipelineReconciler) SyncPipelineResourceStatus(ctx context.Context, req ctrl.Request, tasks kbatchv1.JobList) error {
	var childJobs kbatchv1.JobList

	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		return err
	}
	return nil
}

func (r *PipelineReconciler) ConstructPipeline(ctx context.Context, req ctrl.Request, pipeline *pipelinev1.Pipeline) (pipelinev1.Pipeline, error) {
	ConstructJobsFromPipelineTasks(ctx, pipeline)
	return pipelinev1.Pipeline{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.Pipeline{}).
		Owns(&kbatchv1.Job{}).
		Complete(r)
}
