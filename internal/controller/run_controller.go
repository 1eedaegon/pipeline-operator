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

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kbatchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
)

const (
	runNameLabel = "pipeline.1eedaegon.github.io/run-name"
	jobOwnerKey  = ".metadata." + runNameLabel
)

// RunReconciler reconciles a Run object
type RunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=runs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=runs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pipeline.1eedaegon.github.io,resources=runs/finalizers,verbs=update
func (r *RunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	run := &pipelinev1.Run{}

	// Checking pipeline CRD
	log.Info("Reconciling run.")
	if err := r.Get(ctx, req.NamespacedName, run); err != nil {
		log.V(1).Error(err, "unable to fetch run")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.ensureRunMetadata(ctx, run); err != nil {
		log.V(1).Error(err, "unable to ensure run")
		return ctrl.Result{}, err
	}

	if err := r.ensureJobList(ctx, run); err != nil {
		log.V(1).Error(err, "Unable to ensure job list for pipeline")
		return ctrl.Result{}, err
	}

	if err := r.updateRunStatus(ctx, run); err != nil {
		log.V(1).Error(err, "Unable to ensure run exists for pipeline")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kbatchv1.Job{}, jobOwnerKey, func(rawObj client.Object) []string {
		run := rawObj.(*kbatchv1.Job)
		return []string{run.ObjectMeta.Labels[runNameLabel]}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.Run{}).
		Watches(
			&pipelinev1.Run{},
			handler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), &kbatchv1.Job{}),
		).
		Owns(&kbatchv1.Job{}).
		Complete(r)
}

func (r *RunReconciler) ensureRunMetadata(ctx context.Context, run *pipelinev1.Run) error {
	log := log.FromContext(ctx)
	objKey := client.ObjectKey{
		Name:      run.ObjectMeta.Name,
		Namespace: run.ObjectMeta.Namespace,
	}
	if err := r.Get(ctx, objKey, run); err != nil {
		return err
	}
	objectMeta := run.ObjectMeta
	if objectMeta.Annotations == nil {
		log.V(1).Info(fmt.Sprintf("annotations is empty"))
		objectMeta.Annotations = map[string]string{
			scheduleDateAnnotation: string(run.Spec.Schedule.ScheduleDate),
			triggerAnnotation:      strconv.FormatBool(run.Spec.Trigger),
		}
	}
	if objectMeta.Labels == nil {
		log.V(1).Info(fmt.Sprintf("labels is empty"))
		objectMeta.Labels = map[string]string{
			pipelineNameLabel: run.ObjectMeta.Name,
		}
	}
	objectMeta.Annotations[scheduleDateAnnotation] = string(run.Spec.Schedule.ScheduleDate)
	objectMeta.Annotations[triggerAnnotation] = strconv.FormatBool(run.Spec.Trigger)
	objectMeta.Labels[pipelineNameLabel] = run.ObjectMeta.Name

	if !reflect.DeepEqual(run.ObjectMeta, objectMeta) {
		run.ObjectMeta = objectMeta
	}
	if err := r.Update(ctx, run); err != nil {
		return err
	}
	return nil
}

func (r *RunReconciler) ensureJobList(ctx context.Context, run *pipelinev1.Run) error {
	if err := pipelinev1.NewJobFromRun(ctx, run); err != nil {
		log.V(1).Error(err, "Unable to parse from pipeline")
		return err
	}
	objKey := client.ObjectKey{
		Name:      run.ObjectMeta.Name,
		Namespace: run.ObjectMeta.Namespace,
	}

	log.V(1).Info(fmt.Sprintf("Obj key %v", objKey))

	if err := r.Get(ctx, objKey, run); err != nil {
		if !apierrors.IsNotFound(err) {
			log.V(1).Error(err, "Unknown error")
			return err
		}

		// Relation owner run -> pipeline(owner)
		if err := ctrl.SetControllerReference(run, job, r.Scheme); err != nil {
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
func (r *RunReconciler) updateRunStatus(ctx context.Context, run *pipelinev1.Run) error {
	return nil
}
