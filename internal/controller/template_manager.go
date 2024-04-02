package controller

import (
	"context"

	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	mountPathPrefix string = "/data/"
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

func jobTemplate(ctx context.Context, namespace, volumeName string, task pipelinev1.Task) (*kbatchv1.Job, error) {
	var inputsOutputsMount []corev1.VolumeMount
	var inputsOutputsVolume []corev1.Volume

	for input := range task.Inputs.ToSlice() {
		inputsOutputsMount = append(inputsOutputsMount, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPathPrefix + volumeName,
			SubPath:   string(input),
		})
		inputsOutputsVolume = append(inputsOutputsVolume, corev1.Volume{
			Name: volumeName,
		})
	}
	for output, _ := range task.Outputs.ToSlice() {
		inputsOutputsMount = append(inputsOutputsMount, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPathPrefix + volumeName,
			SubPath:   string(output),
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
							Name:         task.Name,
							Image:        task.Image,
							Command:      task.Command,
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

/*
1. job과 pod은 1:1
*/
func constructJobForPipelineTasks(pipeline *pipelinev1.Pipeline) ([]*kbatchv1.Job, error) {
	var jobs []*kbatchv1.Job
	for _, task := range pipeline.Spec.Tasks {
		job := &kbatchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        task.Name,
				Namespace:   pipeline.Namespace,
			},
			Spec: &kbatchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Image:   task.Image,
								Command: task.Command,
								Args:    task.Args,
							},
						},
						Volumes: []corev1.Volume{
							corev1.Volume{
								VolumeSource: corev1.VolumeSource{
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimSpec{
										ClaimName: "",
									},
								},
							},
						},
					},
				},
			},
		}
	}
}
