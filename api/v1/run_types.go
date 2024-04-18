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

package v1

import (
	"context"
	"fmt"
	"hash/fnv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	pipelineNameLabel = "pipeline.1eedaegon.github.io/pipeline-name"
)

// RunStatus defines the observed state of Run
// pre-run: initializing, stopping, waiting
// run: running, deleting
// post-run: completed, deleted, failed
// deleted는 뭐냐, run시킬 수 없지만 보존기간동안 volume이 보관되는 상태
// # - Create/Update: Initializing|Waiting|Running|Completed 새로운 pipeline을 통해 run을 만드는 개념
// # - Delete: Stopping|Deleting|Deleted
// # - Failed: Completed상태 대신 failed로 빠지며 가장 우선순위가 높음
type JobState string

const (
	// Pre run
	JobStateInit JobState = "initializing"
	JobStateWait JobState = "waiting"
	JobStateStop JobState = "stopping"
	// run
	JobStateRun      JobState = "running"
	JobStateDeleting JobState = "deleting"
	// post run
	JobStateCompleted JobState = "completed"
	JobStateDeleted   JobState = "deleted"
	JobStateFailed    JobState = "failed"
)

/*
RunState order: run > pre-run > post-run

Running으로 한번 진입하면 completed 아니면 Failed이다.

Deleting으로 한번 진입하면 deleted 아니면 failed이다.

JobState를 참고해서 작성되었다.

pre-run: waiting, initializing, stoppin

run: running, deleting

post-run: completed, failed, deleted

deleted	: run시킬 수 없지만 보존기간동안 volume이 보관되는 상태
*/
type RunState string

const (
	// Pre run
	RunStateInit JobState = "initializing"
	RunStateWait JobState = "waiting"
	RunStateStop JobState = "stopping"
	// Run
	RunStateRun      JobState = "running"
	RunStateDeleting JobState = "deleting"
	// Post run
	RunStateCompleted JobState = "completed"
	RunStateDeleted   JobState = "deleted"
	RunStateFailed    JobState = "failed"
)

//
/*
RunSpec defines the desired state of Run

metadata:
  namespace: pipeline
  name: pipeline-chain-test
  annotations:
    pipeline.1eedaegon.github.io/schedule-at: "2024-04-04T01:50:31Z"
  labels:
    pipeline.1eedaegon.github.io/pipeline-name: pipeline-chain-test

*/

type Job struct {
	Name      string   `json:"name,omitempty"`
	Namespace string   `json:"namespace,omitempty"`
	Image     string   `json:"image,omitempty"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	Schedule  Schedule `json:"schedule,omitempty"`
	Resource  Resource `json:"resource,omitempty"`
	Trigger   bool     `json:"trigger,omitempty"`
	RunBefore []string `json:"runBefore,omitempty"`
	Inputs    []string `json:"inputs,omitempty"`
	Outputs   []string `json:"outputs,omitempty"`
}

type RunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name      string   `json:"name,omitempty"` - Name은 Spec이 아니라 metadata이다.
	Schedule     Schedule       `json:"schedule,omitempty"`
	Volume       VolumeResource `json:"volume,omitempty"` // Volume이 run으로 진입했을 때 겹칠 수 있으니 새로 생성해야한다. +prefix
	Trigger      bool           `json:"trigger,omitempty"`
	HistoryLimit HistoryLimit   `json:"historyLimit,omitempty"` // post-run 상태의 pipeline들의 최대 보존 기간: Default - 1D
	RunBefore    []string       `json:"runBefore,omitempty"`
	Inputs       []string       `json:"inputs,omitempty"`   // RX
	Outputs      []string       `json:"outputs,omitempty"`  // RWX
	Resource     Resource       `json:"resource,omitempty"` // task에 리소스가 없을 때, pipeline에 리소스가 지정되어있다면 이것을 적용
	Jobs         []Job          `json:"jobs,omitempty"`
}

// RunStatus defines the observed state of Run
type RunStatus struct {
	RunState          RunState     `json:"runState,omitempty"` // run > pre-run > post-run
	CreateDate        *metav1.Time `json:"createDate,omitempty"`
	LastUpdateDate    *metav1.Time `json:"lastUpdateDate,omitempty"`
	CurrentWorkingJob string       `json:"currentWorkingJob,omitempty"` // current-working-job-name(string)
	Initializing      int          `json:"initializing,omitempty"`      // initializing/total
	Waiting           int          `json:"waiting,omitempty"`           // waiting/total
	Stopping          int          `json:"stopping,omitempty"`          // stopping/total
	Running           int          `json:"running,omitempty"`           // running/total
	Deleting          int          `json:"deleting,omitempty"`          // deleting/total
	Completed         int          `json:"completed,omitempty"`         // completed/total
	Deleted           int          `json:"deleted,omitempty"`           // deleted/total
	Failed            int          `json:"failed,omitempty"`            // failed/total
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Runs",type="integer",JSONPath=".status.runs",description="Number of executed runs"
// +kubebuilder:printcolumn:name="CreatedDate",type="string",JSONPath=".status.createdDate",description="Creation time of the pipeline"
// +kubebuilder:printcolumn:name="LastUpdateDate",type="string",JSONPath=".status.lastUpdateDate",description="Last update time of the pipeline"
// Run is the Schema for the runs API
type Run struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunSpec   `json:"spec,omitempty"`
	Status RunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// RunList contains a list of Run
type RunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Run `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Run{}, &RunList{})
}

// Construct Run template from pipeline
func NewRunFromPipeline(ctx context.Context, pipeline *Pipeline, run *Run) error {
	// jobs := []kbatchv1.Job{}
	jobs := []Job{}
	for _, task := range pipeline.Spec.Tasks {
		job, err := newRunJobFromPipelineTask(ctx, pipeline.ObjectMeta.Namespace, task)
		if err != nil {
			return err
		}
		jobs = append(jobs, *job)
	}
	hsByString := pipeline.ObjectMeta.Name + fmt.Sprintf("%v", pipeline.Spec)
	runName := getShortHashPostFix(pipeline.ObjectMeta.Name, hsByString)
	run.ObjectMeta = metav1.ObjectMeta{
		Annotations: make(map[string]string),
		Labels: map[string]string{
			pipelineNameLabel: pipeline.ObjectMeta.Name,
		},
		Name:      runName,
		Namespace: pipeline.ObjectMeta.Namespace,
	}
	run.Spec = RunSpec{
		Schedule:     pipeline.Spec.Schedule,
		Volume:       pipeline.Spec.Volume,
		HistoryLimit: pipeline.Spec.HistoryLimit,
		RunBefore:    pipeline.Spec.RunBefore,
		Inputs:       pipeline.Spec.Inputs,
		Outputs:      pipeline.Spec.Outputs,
		Resource:     pipeline.Spec.Resource,
		Jobs:         jobs,
	}
	return nil
}

// Construct job template using pipieline task and pipeline volume resource
func newRunJobFromPipelineTask(ctx context.Context, namespace string, ptask PipelineTask) (*Job, error) {
	return &Job{
		Name:      ptask.Name,
		Namespace: namespace,
		Image:     ptask.Image,
		Command:   ptask.Command,
		Args:      ptask.Args,
		Schedule:  ptask.Schedule,
		Resource:  ptask.Resource,
		Trigger:   ptask.Trigger,
		RunBefore: ptask.RunBefore,
		Inputs:    ptask.Inputs,
		Outputs:   ptask.Outputs,
	}, nil
}

func NewJobFromRun(ctx context.Context, run *Run) error {
	jobMeta := metav1.ObjectMeta{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Namespace:   namespace,
		Name:        getShortHashPostFix(ptask.Name, ptask.name + ),
	}
	volume, err := parseVolume(ctx, volumeResource)
	container, err := ParseContainer(ctx, ptask, volumeResource)
	if err != nil {
		return nil, err
	}
	job := kbatchv1.Job{
		ObjectMeta: jobMeta,
		Spec: kbatchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: jobMeta,
				Spec: corev1.PodSpec{
					Containers:    container,
					RestartPolicy: "Never",
					Volumes:       volume,
				},
			},
		},
	}
	return &job, nil
}

// Parsing computing resouce: cpu: 500m / memory: 5GiB
func ParseComputingResource(ctx context.Context, computingResource Resource) (*corev1.ResourceList, error) {
	cpu, err := resource.ParseQuantity(string(computingResource.Cpu))
	if err != nil {
		return nil, err
	}
	mem, err := resource.ParseQuantity(string(computingResource.Memory))
	if err != nil {
		return nil, err
	}
	resourceList := corev1.ResourceList{
		corev1.ResourceCPU:    cpu,
		corev1.ResourceMemory: mem,
	}

	return &resourceList, nil
}

// Parsing Volume resouce: capacity: 5GiB / storageClass: ceph-fs
func ParseVolumeResource(ctx context.Context, volumeResource VolumeResource) (*corev1.ResourceList, error) {
	volume, err := resource.ParseQuantity(string(volumeResource.Capacity))
	if err != nil {
		return nil, err
	}
	resourceList := corev1.ResourceList{
		corev1.ResourceStorage: volume,
	}
	return &resourceList, nil
}

// Parsing Container specs
func ParseContainer(ctx context.Context, ptask PipelineTask, volumeResource VolumeResource) ([]corev1.Container, error) {
	requests, err := ParseComputingResource(ctx, ptask.Resource)
	limits, err := ParseComputingResource(ctx, ptask.Resource)
	if err != nil {
		return nil, err
	}

	mountVolumeList := []corev1.VolumeMount{}
	inputVolumeMountList, err := parseVolumeMountList(ctx, ptask.Inputs, volumeResource)
	outputVolumeMountList, err := parseVolumeMountList(ctx, ptask.Outputs, volumeResource)
	if err != nil {
		return nil, err
	}
	mountVolumeList = append(mountVolumeList, inputVolumeMountList...)
	mountVolumeList = append(mountVolumeList, outputVolumeMountList...)

	containers := []corev1.Container{
		{
			Name:  ptask.Name,
			Image: ptask.Image,
			Command: []string{
				ptask.Command,
			},
			Args: ptask.Args,
			Resources: corev1.ResourceRequirements{
				Requests: *requests,
				Limits:   *limits,
			},
			VolumeMounts: mountVolumeList,
		},
	}

	return containers, nil
}

// Parsing Volume with PVC
func parseVolume(ctx context.Context, volumeResource VolumeResource) ([]corev1.Volume, error) {
	volumes := []corev1.Volume{
		{
			Name: volumeResource.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volumeResource.Name,
				},
			},
		},
	}
	return volumes, nil
}

const (
	mountPathPrefix string = "/data/pipeline"
)

// Parsing Volume mount for using containers
func parseVolumeMountList(ctx context.Context, mountNameList []string, volumeResource VolumeResource) ([]corev1.VolumeMount, error) {
	volumeMounts := []corev1.VolumeMount{}
	for _, mountName := range mountNameList {
		if mountName == "" {
			continue
		}
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeResource.Name,
			MountPath: mountPathPrefix + volumeResource.Name,
			SubPath:   string(mountName),
		})
	}
	return volumeMounts, nil
}

// Get hash64a
// https://cs.opensource.google/go/go/+/refs/tags/go1.22.2:src/hash/fnv/fnv.go;drc=527829a7cba4ded29f98fae97f8bab9de247d5fe;l=129
func hashString(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// Generate unique resource ID with short hash
func getShortHashPostFix(name, hashBy string) string {
	hs := hashString(hashBy)
	return fmt.Sprintf("%s-%x", name, hs)
}
