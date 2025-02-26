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

	"dario.cat/mergo"

	hashset "github.com/1eedaegon/go-hashset"
	kbatchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: error codes should be in one place for each concepts
// Error codes
const (
	ErrVolumeName   = "volume name must be defined"
	ErrCapacityType = "must define the volume capacity correctly like: 1Gi|2Mi|3G|1M"
	ErrStorageName  = "storage name must be defined"
)

const (
	RunNameLabel         = "pipeline.1eedaegon.github.io/run-name"
	RunJobNameLabel      = "pipeline.1eedaegon.github.io/run-job-name"
	JobNameLabel         = "batch.kubernetes.io/job-name"
	StatusAnnotation     = "pipeline.1eedaegon.github.io/status"
	ReasonAnnotation     = "pipeline.1eedaegon.github.io/reason"
	RunDeletionFinalizer = "pipeline.1eedaegon.github.io/finalizer"
	GpuTypeLabel         = "nvidia.com/gpu.product"
	GpuAmountLabel       = "nvidia.com/gpu.count"
)

type TriggerString string

var (
	IsTriggeredString    TriggerString = "true"
	IsNotTriggeredString TriggerString = "false"
)

func (t TriggerString) Bool() bool {
	trigger, err := strconv.ParseBool(string(t))
	if err != nil {
		return false
	}
	return trigger
}

func (t TriggerString) Trigger() Trigger {
	return Trigger(t.Bool())
}

func (t TriggerString) String() string {
	return string(t)
}

// RunStatus defines the observed state of Run
// pre-run: initializing, stopping, waiting
// run: running, deleting
// post-run: completed, deleted, failed
// deleted는 뭐냐, run시킬 수 없지만 보존기간동안 volume이 보관되는 상태
// # - Create/Update: Initializing|Waiting|Running|Completed 새로운 pipeline을 통해 run을 만드는 개념
// # - Delete: Stopping|Deleting|Deleted
// # - Failed: Completed상태 대신 failed로 빠지며 가장 우선순위가 높음

type JobCategory string

const (
	PreRunCategory  JobCategory = "preRun"
	RunCategory     JobCategory = "run"
	PostRunCategory JobCategory = "postRun"
)

var JobCategoryMap = map[JobState]JobCategory{
	JobStateInit:      PreRunCategory,
	JobStateWait:      PreRunCategory,
	JobStateStop:      PreRunCategory,
	JobStateRun:       RunCategory,
	JobStateDeleting:  RunCategory,
	JobStateCompleted: PostRunCategory,
	JobStateFailed:    PostRunCategory,
	JobStateDeleted:   PostRunCategory,
}

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
	JobStateUnknown   JobState = "unknown"
)

var StateOrder = map[JobState]int{
	JobStateFailed:    8,
	JobStateRun:       7,
	JobStateInit:      6,
	JobStateWait:      5,
	JobStateCompleted: 4,
	JobStateDeleting:  3,
	JobStateStop:      2,
	JobStateDeleted:   1,
	JobStateUnknown:   0,
}

type RunJobState struct {
	Name       string   `json:"name,omitempty"`
	RunJobName string   `json:"runJobName,omitempty"`
	JobState   JobState `json:"jobState,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

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
	RunStateInit RunState = "initializing"
	RunStateWait RunState = "waiting"
	RunStateStop RunState = "stopping"
	// Run
	RunStateRun      RunState = "running"
	RunStateDeleting RunState = "deleting"
	// Post run
	RunStateCompleted RunState = "completed"
	RunStateDeleted   RunState = "deleted"
	RunStateFailed    RunState = "failed"
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
	Name                     string            `json:"name,omitempty"`
	Namespace                string            `json:"namespace,omitempty"`
	Image                    string            `json:"image,omitempty"`
	Command                  string            `json:"command,omitempty"`
	Args                     []string          `json:"args,omitempty"`
	Schedule                 Schedule          `json:"schedule,omitempty"`
	Resource                 Resource          `json:"resource,omitempty"`
	Trigger                  TriggerString     `json:"trigger,omitempty"`
	RunBefore                []string          `json:"runBefore,omitempty"`
	Inputs                   []IOVolumeSpec    `json:"inputs,omitempty"`
	Outputs                  []IOVolumeSpec    `json:"outputs,omitempty"`
	Env                      map[string]string `json:"env,omitempty"`
	AdditionalContainerSpecs corev1.Container  `json:"additionalContainerSpecs,omitempty"`
	AdditionalPodSpecs       corev1.PodSpec    `json:"additionalPodSpecs,omitempty"`
}

type RunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name      string   `json:"name,omitempty"` - Name은 Spec이 아니라 metadata이다.
	Schedule Schedule `json:"schedule,omitempty"`
	// See comments on api/v1/run_types.go
	Volumes                  []VolumeResource  `json:"volumes,omitempty"`
	Trigger                  TriggerString     `json:"trigger,omitempty"`
	HistoryLimit             HistoryLimit      `json:"historyLimit,omitempty"` // post-run 상태의 pipeline들의 최대 보존 기간: Default - 1D
	Jobs                     []Job             `json:"jobs,omitempty"`
	RunBefore                []string          `json:"runBefore,omitempty"`
	Inputs                   []IOVolumeSpec    `json:"inputs,omitempty"`   // RX
	Outputs                  []IOVolumeSpec    `json:"outputs,omitempty"`  // RWX
	Resource                 Resource          `json:"resource,omitempty"` // task에 리소스가 없을 때, pipeline에 리소스가 지정되어있다면 이것을 적용
	Env                      map[string]string `json:"env,omitempty"`
	AdditionalContainerSpecs corev1.Container  `json:"additionalContainerSpecs,omitempty"`
	AdditionalPodSpecs       corev1.PodSpec    `json:"additionalPodSpecs,omitempty"`
}

// RunStatus defines the observed state of Run
type RunStatus struct {
	RunState        RunState      `json:"runState,omitempty"` // run > pre-run > post-run
	CreatedDate     *metav1.Time  `json:"createdDate,omitempty"`
	LastUpdatedDate *metav1.Time  `json:"lastUpdateDate,omitempty"`
	JobStates       []RunJobState `json:"JobStates,omitempty"`    // current-working-job-name(string)
	Initializing    *int          `json:"initializing,omitempty"` // initializing/total
	Waiting         *int          `json:"waiting,omitempty"`      // waiting/total
	Stopping        *int          `json:"stopping,omitempty"`     // stopping/total
	Running         *int          `json:"running,omitempty"`      // running/total
	Deleting        *int          `json:"deleting,omitempty"`     // deleting/total
	Completed       *int          `json:"completed,omitempty"`    // completed/total
	Deleted         *int          `json:"deleted,omitempty"`      // deleted/total
	Failed          *int          `json:"failed,omitempty"`       // failed/total
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="RunState",type="string",JSONPath=".status.runState",description=""
// +kubebuilder:printcolumn:name="Initializing",type="integer",JSONPath=".status.initializing",description=""
// +kubebuilder:printcolumn:name="Waiting",type="integer",JSONPath=".status.waiting",description=""
// +kubebuilder:printcolumn:name="Stopping",type="integer",JSONPath=".status.stopping",description=""
// +kubebuilder:printcolumn:name="Running",type="integer",JSONPath=".status.running",description=""
// +kubebuilder:printcolumn:name="Deleting",type="integer",JSONPath=".status.deleting",description=""
// +kubebuilder:printcolumn:name="Completed",type="integer",JSONPath=".status.completed",description=""
// +kubebuilder:printcolumn:name="Deleted",type="integer",JSONPath=".status.deleted",description=""
// +kubebuilder:printcolumn:name="Failed",type="integer",JSONPath=".status.failed",description=""
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
func ConstructRunFromPipeline(ctx context.Context, pipeline *Pipeline, run *Run) error {
	// Construct run metadata
	if err := newRunMetaFromPipeline(ctx, run, pipeline); err != nil {
		return err
	}
	// Construct run volumes
	if err := newRunVolumes(ctx, run, pipeline); err != nil {
		return err
	}
	// Construct job list from pipeline
	if err := newRunJobFromPipeline(ctx, run, pipeline); err != nil {
		return err
	}

	// Construct run input/output from pipeline
	if err := newRunScopeInputOutputsFromPipeline(ctx, run, pipeline); err != nil {
		return err
	}

	// Construct run spec from pipeline
	run.Spec.Schedule = pipeline.Spec.Schedule
	run.Spec.HistoryLimit = pipeline.Spec.HistoryLimit
	run.Spec.RunBefore = pipeline.Spec.RunBefore
	run.Spec.Resource = pipeline.Spec.Resource
	run.Spec.Env = pipeline.Spec.Env
	run.Spec.AdditionalContainerSpecs = pipeline.Spec.AdditionalContainerSpecs
	run.Spec.AdditionalPodSpecs = pipeline.Spec.AdditionalPodSpecs

	return nil
}
func newRunMetaFromPipeline(ctx context.Context, run *Run, pipeline *Pipeline) error {
	run.ObjectMeta = metav1.ObjectMeta{
		Annotations: make(map[string]string),
		Labels:      make(map[string]string),
	}

	hsByString := pipeline.ObjectMeta.Namespace + pipeline.ObjectMeta.Name + fmt.Sprintf("%v", pipeline.Spec)
	runName := getShortHashPostFix(pipeline.ObjectMeta.Name, hsByString)
	run.ObjectMeta.Name = runName
	run.ObjectMeta.Namespace = pipeline.ObjectMeta.Namespace
	run.ObjectMeta.Labels[PipelineNameLabel] = pipeline.ObjectMeta.Name
	run.ObjectMeta.Annotations[ScheduleDateAnnotation] = string(pipeline.Spec.Schedule.ScheduleDate)
	run.ObjectMeta.Annotations[EndDateAnnotation] = string(pipeline.Spec.Schedule.EndDate)
	run.ObjectMeta.Annotations[TriggerAnnotation] = pipeline.Spec.Trigger.String()
	return nil
}

func newRunVolumes(ctx context.Context, run *Run, pipeline *Pipeline) error {
	run.Spec.Volumes = pipeline.Spec.Volumes
	return nil
}

// Convert pipeline input/output and Convert job input/ouput to uniq string with short hash
func newRunScopeInputOutputsFromPipeline(ctx context.Context, run *Run, pipeline *Pipeline) error {
	initialRunIOs := [][]IOVolumeSpec{}
	for _, es := range [][]IOVolumeSpec{pipeline.Spec.Inputs, pipeline.Spec.Outputs} {
		res := []IOVolumeSpec{}

		for _, e := range es {
			res = append(res, IOVolumeSpec{Name: e.Name, UseIntermediateDirectory: true, IntermediateDirectoryName: run.ObjectMeta.Name})
		}
		initialRunIOs = append(initialRunIOs, res)
	}

	run.Spec.Inputs = initialRunIOs[0]
	run.Spec.Outputs = initialRunIOs[1]
	return nil
}

// Construct job template using pipieline task and pipeline volume resource
func newRunJobFromPipeline(ctx context.Context, run *Run, pipeline *Pipeline) error {
	jobs := []Job{}
	namespace := run.ObjectMeta.Namespace
	for _, task := range pipeline.Spec.Tasks {
		hsByString := run.ObjectMeta.Name + run.ObjectMeta.Namespace
		jobName := getShortHashPostFix(task.Name, hsByString)

		IntermediateDirectoryName := run.ObjectMeta.Name

		uniqInputs, err := toInsertIntermediateDirectoryNameOnIOVolumeSpecs(task.Inputs, IntermediateDirectoryName)
		if err != nil {
			return err
		}

		uniqOutputs, err := toInsertIntermediateDirectoryNameOnIOVolumeSpecs(task.Outputs, IntermediateDirectoryName)
		if err != nil {
			return err
		}

		jobRunBeforeList := []string{}
		for _, taskRunBefore := range task.RunBefore {
			jobRunBefore := getShortHashPostFix(taskRunBefore, hsByString)
			jobRunBeforeList = append(jobRunBeforeList, jobRunBefore)
		}

		additionalContainerSpecs := run.Spec.AdditionalContainerSpecs
		mergo.Merge(&additionalContainerSpecs, task.AdditionalContainerSpecs)

		additionalPodSpecs := run.Spec.AdditionalPodSpecs
		mergo.Merge(&additionalPodSpecs, task.AdditionalPodSpecs)

		job := &Job{
			Name:                     jobName,
			Namespace:                namespace,
			Image:                    task.Image,
			Command:                  task.Command,
			Args:                     task.Args,
			Schedule:                 task.Schedule,
			Resource:                 task.Resource,
			Trigger:                  task.Trigger.TriggerString(),
			RunBefore:                jobRunBeforeList,
			Inputs:                   append(run.Spec.Inputs, uniqInputs...),
			Outputs:                  append(run.Spec.Outputs, uniqOutputs...),
			Env:                      task.Env,
			AdditionalContainerSpecs: additionalContainerSpecs,
			AdditionalPodSpecs:       additionalPodSpecs,
		}
		jobs = append(jobs, *job)
	}
	run.Spec.Jobs = jobs
	// Construct Job
	return nil
}

func toInsertIntermediateDirectoryNameOnIOVolumeSpecs(ioVolumeSpecs []IOVolumeSpec, IntermediateDirectoryName string) ([]IOVolumeSpec, error) {
	res := []IOVolumeSpec{}
	for _, v := range ioVolumeSpecs {
		e := v
		e.IntermediateDirectoryName = IntermediateDirectoryName
		res = append(res, e)
	}
	return res, nil
}

func toInsertIfHasIntermediateDirectoryFromIOVolumeSpecs(pathList []IOVolumeSpec, insertPath string) ([]string, error) {
	res := []string{}
	for _, input := range pathList {
		name := input.Name
		if input.UseIntermediateDirectory {
			nameWithHashedSubdirectory, err := toInsertBetweenFirstPath(name, insertPath)
			if err != nil {
				return nil, err
			}
			name = nameWithHashedSubdirectory
		}
		res = append(res, name)
	}
	return res, nil
}

func toInsertBetweenFirstPathFromList(pathList []string, insertPath string) ([]string, error) {
	hashPathList := []string{}
	for _, path := range pathList {
		hashPath, err := toInsertBetweenFirstPath(path, insertPath)
		if err != nil {
			return nil, err
		}
		hashPathList = append(hashPathList, hashPath)
	}
	return hashPathList, nil
}

func toInsertBetweenFirstPath(pathName string, insertPath string) (string, error) {
	splited := strings.SplitN(pathName, "/", 2)
	if len(splited) <= 0 || splited[1] == "" {
		return "", errors.New("path has not splited by '/' like: 'volumeName/filePath'")
	}
	hashPath := strings.Join([]string{
		splited[0], insertPath, splited[1],
	}, "/")

	return hashPath, nil
}

// Construct kuberentes job from run job
func ConstructKjobListFromRun(ctx context.Context, run *Run) ([]kbatchv1.Job, error) {
	kjobList := []kbatchv1.Job{}
	for _, runjob := range run.Spec.Jobs {
		kjob, err := constructKjobFromRunJob(ctx, run.ObjectMeta, runjob)
		if err != nil {
			return nil, err
		}
		kjobList = append(kjobList, *kjob)
	}
	return kjobList, nil
}

// TODO: 임의로 리소스를 추가하려고 하는 경우를 방지하기 위해 validationWebhook에서 제한을 걸어야한다.
// TODO: 리콘실러가 아닌 다른 조작에 의해 리소스가 삭제되지않도록  finalizer 제약을 걸어야한다.
func constructKjobFromRunJob(ctx context.Context, runMeta metav1.ObjectMeta, job Job) (*kbatchv1.Job, error) {
	// Construct container template
	container, err := parseContainerFromJob(ctx, job)
	if err != nil {
		return nil, err
	}
	// Construct volume template
	volumes, err := parseVolumeWithPVCFromJob(ctx, job)
	if err != nil {
		return nil, err
	}
	// Construct pod Spec
	podSpec := corev1.PodSpec{
		Volumes:       volumes,
		Containers:    container,
		RestartPolicy: corev1.RestartPolicyNever,
	}

	mergo.Merge(&podSpec, job.AdditionalPodSpecs)

	// Construct pod Template
	podTemplateMeta := constructKjobPodMetaFromJob(ctx, runMeta, job)
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: podTemplateMeta,
		Spec:       podSpec,
	}
	kjobMeta := constructKjobMetaFromJob(ctx, runMeta, job)

	trigger := job.Trigger.Bool()
	if err != nil {
		return nil, err
	}
	kJob := &kbatchv1.Job{
		ObjectMeta: kjobMeta,
		Spec: kbatchv1.JobSpec{
			Suspend:  &trigger,
			Template: podTemplate,
		},
	}
	return kJob, nil
}

func constructKjobPodMetaFromJob(ctx context.Context, runMeta metav1.ObjectMeta, job Job) metav1.ObjectMeta {
	podTempMeta := metav1.ObjectMeta{
		Annotations: make(map[string]string),
		Labels:      make(map[string]string),
	}
	podTempMeta.Annotations[ScheduleDateAnnotation] = string(job.Schedule.ScheduleDate)
	podTempMeta.Labels[PipelineNameLabel] = runMeta.Labels[PipelineNameLabel]
	podTempMeta.Labels[RunNameLabel] = runMeta.Name
	return podTempMeta
}

func constructKjobMetaFromJob(ctx context.Context, runMeta metav1.ObjectMeta, job Job) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{
		Annotations: make(map[string]string),
		Labels:      make(map[string]string),
	}
	hsByString := job.Name + runMeta.Name + runMeta.Namespace
	meta.Name = getShortHashPostFix(job.Name, hsByString)
	meta.Namespace = job.Namespace
	meta.Annotations[ScheduleDateAnnotation] = string(job.Schedule.ScheduleDate)
	meta.Annotations[TriggerAnnotation] = job.Trigger.String()
	meta.Labels[PipelineNameLabel] = runMeta.Labels[PipelineNameLabel]
	meta.Labels[RunNameLabel] = runMeta.Name
	meta.Labels[RunJobNameLabel] = job.Name
	meta.Labels[GpuTypeLabel] = job.Resource.Gpu.GpuType
	meta.Labels[GpuAmountLabel] = fmt.Sprintf("%d", job.Resource.Gpu.Amount)
	return meta
}

func parsePodSpecFromJob(ctx context.Context, job Job) (*corev1.PodTemplateSpec, error) {
	container, err := parseContainerFromJob(ctx, job)
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
func parseContainerFromJob(ctx context.Context, job Job) ([]corev1.Container, error) {

	requests, err := parseComputingResource(ctx, &job.Resource)
	if err != nil {
		return nil, err
	}
	limits, _ := parseComputingResource(ctx, &job.Resource)
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

	for _, c := range containers {
		mergo.Merge(&c, job.AdditionalContainerSpecs)
	}

	return containers, nil
}

// Parsing computing resouce: cpu: 500m / memory: 5GiB
func parseComputingResource(ctx context.Context, computingResource *Resource) (*corev1.ResourceList, error) {
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

// Parsing Volume with PVC
func parseVolumeWithPVCFromJob(ctx context.Context, job Job) ([]corev1.Volume, error) {
	hashSet := hashset.New()

	volumeList := []corev1.Volume{}
	// 들어온 volume이름 목록으로 PVC template을 만든다.
	volumeStringList := []IOVolumeSpec{}
	volumeStringList = append(volumeStringList, job.Inputs...)
	volumeStringList = append(volumeStringList, job.Outputs...)

	for _, e := range volumeStringList {
		volumeCopus, err := splitVolumeCopus(e.Name)
		if err != nil {
			return nil, err
		}
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

	for _, es := range [][]IOVolumeSpec{job.Inputs, job.Outputs} {
		for _, e := range es {
			mountCopus, err := splitVolumeCopus(e.Name)
			if err != nil {
				return nil, err
			}

			subPath := strings.Join(mountCopus[1:], "/")

			subPathWithIntermediateDirectory := subPath
			if e.UseIntermediateDirectory && e.IntermediateDirectoryName != "" {
				subPathWithIntermediateDirectory = e.IntermediateDirectoryName + "/" + subPath
			}

			volumeName := mountCopus[0]
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volumeName,
				MountPath: mountPathPrefix + "/" + volumeName + "/" + subPath,
				SubPath:   subPathWithIntermediateDirectory,
				ReadOnly:  true,
			})
		}
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

func parsePvcListFromRun(ctx context.Context, run *Run) ([]*corev1.PersistentVolumeClaim, error) {
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
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: meta,
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
				corev1.ReadWriteMany,
			},
		},
	}

	capacity, err := resource.ParseQuantity(volumeResource.Capacity)
	if err != nil {
		return nil, err
	}
	pvc.Spec.Resources.Requests = corev1.ResourceList{
		corev1.ResourceName(corev1.ResourceStorage): capacity,
	}

	if volumeResource.Storage == "" {
		return nil, errors.New("storage name must be defined")

	}
	pvc.Spec.StorageClassName = &volumeResource.Storage

	return pvc, nil
}

// TODO: Simplify state
func CategorizeJobState(state JobState) JobCategory {
	return JobCategoryMap[state]
}

func DetermineJobStateFrom(kjob *kbatchv1.Job, pod *corev1.Pod) JobState {
	switch {
	case *kjob.Spec.Suspend || kjob.ObjectMeta.Annotations[TriggerAnnotation] == IsTriggered.String():
		return JobStateWait
	case kjob.Status.Active > 0:
		if pod.Status.Phase == corev1.PodPending {
			return JobStateInit // Job 초기화 중
		} else if pod.Status.Phase == corev1.PodRunning {
			return JobStateRun // Job 실행 중
		}
	case kjob.Status.Succeeded > 0:
		return JobStateCompleted // Job 완료

	case kjob.Status.Failed > 0:
		return JobStateFailed // Job 실패

	case !kjob.ObjectMeta.DeletionTimestamp.IsZero():
		if kjob.Status.Active > 0 {
			return JobStateStop // Job 삭제 중
		} else {
			return JobStateDeleting
		}
	default:
		return JobStateInit
	}
	return JobStateDeleted
}

func DetermineJobStateFromOrder(prevState, nextState JobState) JobState {
	if StateOrder[prevState] > StateOrder[nextState] {
		return prevState
	} else {
		return nextState
	}
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
	if len(url) <= 1 {
		return "docker.io/" + imagePath
	}
	return imagePath
}

// volume 이름의 "/"를 기준으로 자른다.(copus)
// 자른 이름의 0 index를 pvc의 이름으로 사용, 1 index를 hash subdirectory로, 2.. index를 subpath로 사용한다.
func splitVolumeCopus(volumeString string) ([]string, error) {
	volumeCopus := strings.Split(volumeString, "/")
	if len(volumeCopus) <= 0 || volumeCopus[1] == "" {
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
