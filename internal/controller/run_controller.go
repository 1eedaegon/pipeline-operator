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

	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	if err := r.ensureVolumeList(ctx, run); err != nil {
		log.V(1).Error(err, "unable to ensure volume list")
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
	log.Info("ensure run metadata.")

	objKey := client.ObjectKey{
		Name:      run.ObjectMeta.Name,
		Namespace: run.ObjectMeta.Namespace,
	}
	runQuery := pipelinev1.Run{}
	if err := r.Get(ctx, objKey, &runQuery); err != nil {
		return err
	}
	objectMeta := run.ObjectMeta
	if objectMeta.Annotations == nil {
		objectMeta.Annotations = make(map[string]string)
	}
	if objectMeta.Labels == nil {
		objectMeta.Labels = make(map[string]string)
	}
	objectMeta.Annotations[pipelinev1.ScheduleDateAnnotation] = string(run.Spec.Schedule.ScheduleDate)
	objectMeta.Annotations[pipelinev1.TriggerAnnotation] = strconv.FormatBool(run.Spec.Trigger)
	objectMeta.Labels[pipelinev1.PipelineNameLabel] = run.ObjectMeta.Name

	if !reflect.DeepEqual(run.ObjectMeta, objectMeta) {
		run.ObjectMeta = objectMeta
	}
	if err := r.Update(ctx, run); err != nil {
		return err
	}
	return nil
}

func (r *RunReconciler) ensureVolumeList(ctx context.Context, run *pipelinev1.Run) error {
	log := log.FromContext(ctx)
	for _, volume := range run.Spec.Volumes {

		objKey := client.ObjectKey{
			Name:      volume.Name,
			Namespace: run.ObjectMeta.Namespace,
		}
		pvcQuery := &corev1.PersistentVolumeClaim{}
		if err := r.Get(ctx, objKey, pvcQuery); err != nil {
			// If network error, return unknown
			if !apierrors.IsNotFound(err) {
				log.V(1).Error(err, "Unknown error")
				return err
			}

			// Construct pvc template
			log.V(1).Info(fmt.Sprintf("Creating pvc %v", objKey))
			meta := metav1.ObjectMeta{
				Name:      volume.Name,
				Namespace: run.ObjectMeta.Namespace,
			}
			pvc, err := pipelinev1.ParsePvcFromVolumeResourceWithMeta(ctx, meta, volume)
			if err != nil {
				log.V(1).Error(err, "Unable to parse pvc from run")
			}

			// Relation owner run -> pvc(owner)
			log.V(1).Info(fmt.Sprintf("Referencing pvc %v", objKey))
			if err := ctrl.SetControllerReference(run, pvc, r.Scheme); err != nil {
				log.V(1).Error(err, "Unable to reference between run and new pvc")
				return err
			}
			if err := r.Create(ctx, pvc); err != nil {
				log.V(1).Error(err, "Unable to create pvc")
				return err
			}
		}
	}

	return nil
}

func (r *RunReconciler) ensureJobList(ctx context.Context, run *pipelinev1.Run) error {
	log := log.FromContext(ctx)
	log.Info("ensure job list.")

	kjobList, err := pipelinev1.NewKjobListFromRun(ctx, run)
	if err != nil {
		log.V(1).Error(err, "Unable to parse from job list")
		return err
	}

	for _, kjob := range kjobList {
		objKey := client.ObjectKey{
			Name:      kjob.ObjectMeta.Name,
			Namespace: kjob.ObjectMeta.Namespace,
		}
		log.V(1).Info(fmt.Sprintf("Job obj key %v", objKey))
		if err := r.Get(ctx, objKey, &kjob); err != nil {
			log.V(1).Info(fmt.Sprintf("Getting jobs %v", objKey))
			// If network error, return unknown
			if !apierrors.IsNotFound(err) {
				log.V(1).Error(err, "Unknown error: unstable network connection")
				return err
			}

			// Relation owner run -> pipeline(owner)
			log.V(1).Info(fmt.Sprintf("Referencing jobs %v", objKey))
			if err := ctrl.SetControllerReference(run, &kjob, r.Scheme); err != nil {
				log.V(1).Error(err, "Unable to reference between run and new job")
				return err
			}

			// Create Job
			log.V(1).Info(fmt.Sprintf("Creating jobs %v", objKey))
			if err := r.Create(ctx, &kjob); err != nil {
				log.V(1).Error(err, "Unable to create job")
				return err
			}
		}
	}

	return nil
}

// TODO: Job내 pod의 상태를 보고 run job의 상태를 결정지어야한다.
func (r *RunReconciler) updateRunStatus(ctx context.Context, run *pipelinev1.Run) error {
	log := log.FromContext(ctx)
	jobList := kbatchv1.JobList{}

	listQueryOpts := []client.ListOption{
		client.InNamespace(run.ObjectMeta.Namespace),
		client.MatchingLabels(labels.Set{pipelinev1.PipelineNameLabel: run.ObjectMeta.Name}),
	}

	objKey := client.ObjectKey{
		Name:      run.ObjectMeta.Name,
		Namespace: run.ObjectMeta.Namespace,
	}
	if err := r.List(ctx, &jobList, listQueryOpts...); err != nil {
		return err
	}
	// Retry backoff
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		log.V(1).Info("update run status")
		run = &pipelinev1.Run{}
		err := r.Get(ctx, objKey, run)
		if err != nil {
			return err
		}
		run.Status.Initializing = len(jobList.Items)
		run.Status.CreatedDate = &run.ObjectMeta.CreationTimestamp
		run.Status.LastUpdatedDate = &metav1.Time{Time: time.Now()}

		return r.Status().Update(ctx, run)
	}); err != nil {
		return err
	}
	return nil
}
