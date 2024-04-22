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
	"strconv"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"

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
	runOwnerKey = ".metadata." + pipelinev1.PipelineNameLabel
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

	// Checking pipeline CRD
	log.Info("Reconciling pipeline.")
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		log.V(1).Error(err, "unable to fetch pipeline")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Update pipeline with status(with sum)
	if err := r.ensurePipelineMetadata(ctx, pipeline); err != nil {
		log.V(1).Error(err, "unable to ensure pipeline")
		return ctrl.Result{}, err
	}

	// Check the list of runs and create one if there are changes or none
	if err := r.ensureRunExists(ctx, pipeline); err != nil {
		log.V(1).Error(err, "Unable to ensure run exists for pipeline")
		return ctrl.Result{}, err
	}

	if err := r.updatePipelineStatus(ctx, pipeline); err != nil {
		log.V(1).Error(err, "Unable to update pipeline status")
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
	if err := r.Get(ctx, objKey, pipeline); err != nil {
		return err
	}
	objectMeta := pipeline.ObjectMeta
	if objectMeta.Annotations == nil {
		objectMeta.Annotations = map[string]string{
			pipelinev1.ScheduleDateAnnotation: string(pipeline.Spec.Schedule.ScheduleDate),
			pipelinev1.TriggerAnnotation:      strconv.FormatBool(pipeline.Spec.Trigger),
		}
	}
	if objectMeta.Labels == nil {
		objectMeta.Labels = map[string]string{
			pipelinev1.PipelineNameLabel: pipeline.ObjectMeta.Name,
		}
	}
	objectMeta.Annotations[pipelinev1.ScheduleDateAnnotation] = string(pipeline.Spec.Schedule.ScheduleDate)
	objectMeta.Annotations[pipelinev1.TriggerAnnotation] = strconv.FormatBool(pipeline.Spec.Trigger)
	objectMeta.Labels[pipelinev1.PipelineNameLabel] = pipeline.ObjectMeta.Name

	if !reflect.DeepEqual(pipeline.ObjectMeta, objectMeta) {
		pipeline.ObjectMeta = objectMeta
	}
	if err := r.Update(ctx, pipeline); err != nil {
		return err
	}
	return nil
}

func (r *PipelineReconciler) ensureRunExists(ctx context.Context, pipeline *pipelinev1.Pipeline) error {
	log := log.FromContext(ctx)
	run := &pipelinev1.Run{}

	if err := pipelinev1.NewRunFromPipeline(ctx, pipeline, run); err != nil {
		log.V(1).Error(err, "Unable to parse from pipeline")
		return err
	}
	objKey := client.ObjectKey{
		Name:      run.ObjectMeta.Name,
		Namespace: run.ObjectMeta.Namespace,
	}
	log.V(1).Info(fmt.Sprintf("pipeline run obj key %v", objKey))
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
	runList := pipelinev1.RunList{}

	listQueryOpts := []client.ListOption{
		client.InNamespace(pipeline.ObjectMeta.Namespace),
		client.MatchingLabels(map[string]string{pipelinev1.PipelineNameLabel: pipeline.ObjectMeta.Name}),
	}

	objKey := client.ObjectKey{
		Name:      pipeline.ObjectMeta.Name,
		Namespace: pipeline.ObjectMeta.Namespace,
	}
	if err := r.List(ctx, &runList, listQueryOpts...); err != nil {
		return err
	}
	// Retry backoff
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		err := r.Get(ctx, objKey, pipeline)
		if err != nil {
			return err
		}
		pipeline.Status.Runs = len(runList.Items)
		pipeline.Status.CreatedDate = &pipeline.ObjectMeta.CreationTimestamp
		pipeline.Status.LastUpdatedDate = &metav1.Time{Time: time.Now()}

		return r.Status().Update(ctx, pipeline)
	}); err != nil {
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
