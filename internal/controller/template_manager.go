package controller

import (
	"context"

	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/api/resource"
)

/*
 1. pipeline template
 2. job template
 3. resource template
*/

// TODO: Validate binary expression
// Binary expression: 2^10 -> 2Ki
const (
	pvcPrefix       string = "pvc-"
	mountPathPrefix string = "/data/pipeline"
)

func PvcByStorageClassTemplate(ctx context.Context, namespace, volumeName, capacity string, storageClassName *string) (*corev1.PersistentVolumeClaim, error) {
	storageCapacity, err := resource.ParseQuantity(capacity)
	if err != nil {
		return nil, err
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      volumeName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageCapacity,
				},
			},
			StorageClassName: storageClassName,
		},
	}, nil
}

func JobTemplate(ctx context.Context, namespace, volumeName string, task pipelinev1.Task) (*kbatchv1.Job, error) {
	var inputsOutputsMount []corev1.VolumeMount
	var inputsOutputsVolume []corev1.Volume

	for input := range task.Spec.Inputs {
		inputsOutputsMount = append(inputsOutputsMount, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPathPrefix + volumeName,
			SubPath:   string(input),
		})
		inputsOutputsVolume = append(inputsOutputsVolume, corev1.Volume{
			Name: volumeName,
		})
	}
	for output := range task.Spec.Outputs {
		inputsOutputsMount = append(inputsOutputsMount, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPathPrefix + volumeName,
			SubPath:   string(output),
		})
		inputsOutputsVolume = append(inputsOutputsVolume, corev1.Volume{
			Name: volumeName,
		})
	}
	job := &kbatchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.Name,
			Namespace: namespace,
		},
		Spec: kbatchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         task.Spec.Name,
							Image:        task.Spec.Image,
							Command:      task.Spec.Command,
							VolumeMounts: inputsOutputsMount,
						},
					},
					RestartPolicy: "Never",
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: volumeName,
								},
							},
						},
					},
				},
			},
		},
	}
	return job, nil
}

func ConstructJobsFromPipelineTasks(ctx context.Context, pipeline *pipelinev1.Pipeline) ([]*kbatchv1.Job, error) {
	log := log.FromContext(ctx)
	var jobs []*kbatchv1.Job
	for _, task := range pipeline.Spec.Tasks {
		job, err := JobTemplate(ctx, pipeline.Namespace, pipeline.Spec.VolumeName, task)
		if err != nil {
			log.V(1).Error(err, "Can't generate Job from request")
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}
