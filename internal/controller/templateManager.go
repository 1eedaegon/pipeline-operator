package controller

import (
	pipelinev1 "github.com/1eedaegon/pipeline-operator/api/v1"
	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
									PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
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
