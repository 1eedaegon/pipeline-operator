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
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	hashset "github.com/1eedaegon/go-hashset"
	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

pre-run: waiting, initializing, stopping

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
	Name      string            `json:"name,omitempty"`
	Namespace string            `json:"namespace,omitempty"`
	Image     string            `json:"image,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Schedule  Schedule          `json:"schedule,omitempty"`
	Resource  Resource          `json:"resource,omitempty"`
	Trigger   bool              `json:"trigger,omitempty"`
	RunBefore []string          `json:"runBefore,omitempty"`
	Inputs    []string          `json:"inputs,omitempty"`
	Outputs   []string          `json:"outputs,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

type RunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name      string   `json:"name,omitempty"` - Name은 Spec이 아니라 metadata이다.
	Schedule     Schedule          `json:"schedule,omitempty"`
	Volumes      []VolumeResource  `json:"volumes,omitempty"` // Volume이 run으로 진입했을 때 겹칠 수 있으니 새로 생성해야한다. +prefix
	Trigger      bool              `json:"trigger,omitempty"`
	HistoryLimit HistoryLimit      `json:"historyLimit,omitempty"` // post-run 상태의 pipeline들의 최대 보존 기간: Default - 1D
	Jobs         []Job             `json:"jobs,omitempty"`
	RunBefore    []string          `json:"runBefore,omitempty"`
	Inputs       []string          `json:"inputs,omitempty"`   // RX
	Outputs      []string          `json:"outputs,omitempty"`  // RWX
	Resource     Resource          `json:"resource,omitempty"` // task에 리소스가 없을 때, pipeline에 리소스가 지정되어있다면 이것을 적용
	Env          map[string]string `json:"env,omitempty"`
}

// RunStatus defines the observed state of Run
type RunStatus struct {
	RunState          RunState     `json:"runState,omitempty"` // run > pre-run > post-run
	CreatedDate       *metav1.Time `json:"createDate,omitempty"`
	LastUpdatedDate   *metav1.Time `json:"lastUpdateDate,omitempty"`
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
// TODO: 이곳에서 PVC가 없으면 에러를 뱉는 로직을 추가해야한다.
// TODO: Pipeline과의 차이는, pipeline은 iuputs의 목록이 volume에 있는지 확인이고
// 이곳은 실제 get() 함수를 통해 pvc가 존재하는지 확인 후 생성해야한다.
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
			PipelineNameLabel: pipeline.ObjectMeta.Name,
		},
		Name:      runName,
		Namespace: pipeline.ObjectMeta.Namespace,
	}
	run.Spec = RunSpec{
		Schedule:     pipeline.Spec.Schedule,
		Volumes:      pipeline.Spec.Volumes,
		HistoryLimit: pipeline.Spec.HistoryLimit,
		RunBefore:    pipeline.Spec.RunBefore,
		Inputs:       pipeline.Spec.Inputs,
		Outputs:      pipeline.Spec.Outputs,
		Resource:     pipeline.Spec.Resource,
		Env:          pipeline.Spec.Env,
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
		Env:       ptask.Env,
	}, nil
}

func NewKjobListFromRun(ctx context.Context, run *Run) ([]kbatchv1.Job, error) {
	kjobList := []kbatchv1.Job{}
	for _, runjob := range run.Spec.Jobs {
		kjob, err := convertRunJobToKjob(ctx, run.ObjectMeta, runjob)
		if err != nil {
			return nil, err
		}
		kjobList = append(kjobList, *kjob)
	}
	return kjobList, nil
}

// TODO: 임의로 리소스를 추가하려고 하는 경우를 방지하기 위해 validationWebhook에서 제한을 걸어야한다.
// TODO: 리콘실러가 아닌 다른 조작에 의해 리소스가 삭제되지않도록  finalizer 제약을 걸어야한다.
func convertRunJobToKjob(ctx context.Context, meta metav1.ObjectMeta, job Job) (*kbatchv1.Job, error) {
	kjobMeta := constructKjobMetaFromJob(ctx, meta.Labels[PipelineNameLabel], job)
	podSpec, err := ParsePodSpecFromJob(ctx, job)
	if err != nil {
		return nil, err
	}
	kJob := &kbatchv1.Job{
		ObjectMeta: kjobMeta,
		Spec: kbatchv1.JobSpec{
			Template: *podSpec,
		},
	}
	return kJob, nil
}

func constructKjobMetaFromJob(ctx context.Context, pipelineNameLabel string, job Job) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{
		Annotations: make(map[string]string),
		Labels:      make(map[string]string),
	}
	hsByString := job.Name + fmt.Sprintf("%v", job)
	meta.Name = getShortHashPostFix(job.Name, hsByString)
	meta.Namespace = job.Namespace
	meta.Annotations[ScheduleDateAnnotation] = string(job.Schedule.ScheduleDate)
	meta.Annotations[TriggerAnnotation] = strconv.FormatBool(job.Trigger)
	meta.Labels[PipelineNameLabel] = pipelineNameLabel

	return meta
}

func ParsePodSpecFromJob(ctx context.Context, job Job) (*corev1.PodTemplateSpec, error) {
	container, err := ParseContainerFromJob(ctx, job)
	if err != nil {
		return nil, err
	}

	volumes, err := parseVolumeWithPVCFromJob(ctx, job)
	if err != nil {
		return nil, err
	}

	podTempSpec := corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Volumes:       volumes,
			Containers:    container,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	return &podTempSpec, nil
}

// Parsing Container specs
func ParseContainerFromJob(ctx context.Context, job Job) ([]corev1.Container, error) {

	requests, err := ParseComputingResource(ctx, &job.Resource)
	if err != nil {
		return nil, err
	}
	limits, _ := ParseComputingResource(ctx, &job.Resource)
	mountVolumeList, err := parseVolumeMountList(ctx, job)
	if err != nil {
		return nil, err
	}

	envList := parseContainerEnv(ctx, job.Env)
	image := defaultImageRegistry(job.Image)
	command := parseCommand(job.Command)

	containers := []corev1.Container{
		{
			Name:    job.Name,
			Image:   image,
			Command: command,
			Args:    job.Args,
			Resources: corev1.ResourceRequirements{
				Requests: *requests,
				Limits:   *limits,
			},
			VolumeMounts: mountVolumeList,
			Env:          envList,
		},
	}

	return containers, nil
}

// Parsing computing resouce: cpu: 500m / memory: 5GiB
func ParseComputingResource(ctx context.Context, computingResource *Resource) (*corev1.ResourceList, error) {
	resourceList := corev1.ResourceList{}

	if computingResource == nil {
		return &resourceList, nil
	}

	if computingResource.Cpu != "" {
		cpu, err := resource.ParseQuantity(string(computingResource.Cpu))
		if err != nil {
			return nil, err
		}
		resourceList[corev1.ResourceCPU] = cpu
	}

	if computingResource.Memory != "" {
		mem, err := resource.ParseQuantity(string(computingResource.Memory))
		if err != nil {
			return nil, err
		}
		resourceList[corev1.ResourceMemory] = mem
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

// Parsing Volume with PVC
func parseVolumeWithPVCFromJob(ctx context.Context, job Job) ([]corev1.Volume, error) {
	hashSet := hashset.New()

	volumeList := []corev1.Volume{}
	// 들어온 volume이름 목록으로 PVC template을 만든다.
	volumeStringList := []string{}
	volumeStringList = append(volumeStringList, job.Inputs...)
	volumeStringList = append(volumeStringList, job.Outputs...)

	for _, volumeString := range volumeStringList {
		volumeCopus, err := splitVolumeCopus(volumeString)
		if err != nil {
			return nil, err
		}
		// hsBy := volumeCopus[0] + fmt.Sprintf("%v", job)
		// volumeName := getShortHashPostFix(volumeCopus[0], hsBy)
		volumeName := volumeCopus[0]
		if !hashSet.Contains(volumeName) {
			hashSet.Add(volumeName)
			volume := corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: volumeName,
					},
				},
			}
			volumeList = append(volumeList, volume)
		}
	}
	return volumeList, nil
}

const (
	mountPathPrefix string = "/data/pipeline"
)

// Parsing Volume mount for using containers
func parseVolumeMountList(ctx context.Context, job Job) ([]corev1.VolumeMount, error) {
	volumeMounts := []corev1.VolumeMount{}

	volumeMountStringList := []string{}
	volumeMountStringList = append(volumeMountStringList, job.Inputs...)
	volumeMountStringList = append(volumeMountStringList, job.Outputs...)

	for _, mountString := range volumeMountStringList {
		mountCopus, err := splitVolumeCopus(mountString)
		if err != nil {
			return nil, err
		}
		// hsBy := mountCopus[0] + fmt.Sprintf("%v", job)
		// volumeName := getShortHashPostFix(mountCopus[0], hsBy)
		volumeName, subPath := mountCopus[0], mountCopus[1]
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: mountPathPrefix + "/" + volumeName + "/" + subPath,
			SubPath:   subPath,
		})
	}
	return volumeMounts, nil
}

func parseContainerEnv(ctx context.Context, env map[string]string) []corev1.EnvVar {
	envList := []corev1.EnvVar{}
	for key, value := range env {
		envList = append(envList, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	return envList
}

func ParsePvcListFromRun(ctx context.Context, run *Run) ([]*corev1.PersistentVolumeClaim, error) {
	pvcList := []*corev1.PersistentVolumeClaim{}

	for _, volume := range run.Spec.Volumes {
		meta := metav1.ObjectMeta{
			Name:      volume.Name,
			Namespace: run.ObjectMeta.Namespace,
		}
		pvc, err := ParsePvcFromVolumeResourceWithMeta(ctx, meta, volume)
		if err != nil {
			return nil, err
		}
		pvcList = append(pvcList, pvc)
	}

	return pvcList, nil
}

func ParsePvcFromVolumeResourceWithMeta(ctx context.Context, meta metav1.ObjectMeta, volumeResource VolumeResource) (*corev1.PersistentVolumeClaim, error) {
	quota, err := resource.ParseQuantity(volumeResource.Capacity)
	if err != nil {
		return nil, err
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: meta,
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
				corev1.ReadWriteMany,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceName(corev1.ResourceStorage): quota,
				},
			},
			StorageClassName: &volumeResource.Storage,
		},
	}
	return pvc, nil
}

func parseCommand(command string) []string {
	commandString := []string{}
	if command != "" {
		commandString = append(commandString, command)
	}
	return commandString
}

func defaultImageRegistry(imagePath string) string {
	imagePathCopus := strings.SplitN(imagePath, "/", 2)
	url := strings.Split(imagePathCopus[0], ".")
	if len(url) == 1 {
		return "docker.io/" + imagePath
	}
	return imagePath
}

// volume 이름의 "/"를 기준으로 자른다.(copus)
// 자른 이름의 좌측을 pvc의 이름으로 사용, 우측을 subpath로 사용한다.
func splitVolumeCopus(volumeString string) ([]string, error) {
	volumeCopus := strings.SplitN(volumeString, "/", 2)
	if len(volumeCopus) == 0 || volumeCopus[1] == "" {
		return nil, errors.New("volume has no prefix or postfix like: 'volumeName/filePath'")
	}
	return volumeCopus, nil
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
