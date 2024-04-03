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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Pipeline/Run:
// - Create/Update: Initializing|Running|Completed
// - Delete: Stopping|Deleting|Deleted
// Task/Job: pipeline의 하위에 존재한다.
// - Create: Initializing|Waiting|Running|Completed
// - Update/Delete: Initializing|Waiting|Running|Completed 새로운 pipeline을 통해 run을 만드는 개념
type RunState string

const (
	RunStateInit      RunState = "Initializing"
	RunStateRun       RunState = "Running"
	RunStateCompleted RunState = "Completed"
	RunStateStop      RunState = "Stopping"
	RunStateDeleting  RunState = "Deleting"
	RunStateDeleted   RunState = "Deleted"
)

type JobState string

const (
	JobStateInit      JobState = "Initializing"
	JobStateWait      JobState = "Wating"
	JobStateRun       JobState = "Running"
	JobStateCompleted JobState = "Completed"
)

type Resource struct {
	Cpu    int         `json:"cpu,omitempty"`
	Memory string      `json:"memory,omitempty"`
	Gpu    GpuResource `json:"gpu,omitempty"`
}

type GpuResource struct {
	GpuType string `json:"type,omitempty"`
	Amount  int    `json:"amount,omitempty"`
}

type StorageResource struct {
	Name     string `json: name`
	Capacity string `json: capacity`
}

type ScheduleType string

const (
	Cron ScheduleType = "cron"
	Date ScheduleType = "date"
)

type Schedule struct {
	ScheduleType ScheduleType `json:"scheduleType,omitempty"`
	ScheduleDate string       `json:"scheduleDate,omitempty"`
	EndDate      string       `json:"endDate,omitempty"`
}

type ModeType string

const (
	Auto   ModeType = "auto"
	Manual ModeType = "manual"
)

// PipelineSpec defines the desired state of Pipeline
type PipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name       string   `json:"name,omitempty"` - Spec이 아니라 Metadata에 들어가야할 내용임.
	VolumeName      string   `json:"volumeName,omitempty"`
	Schedule        Schedule `json:"schedule,omitempty"`
	DefaultResource Resource `json:"resource,omitempty"`
	RunAfter        []string `json:"runAfter,omitempty"`
	RunBefore       []string `json:"runBefore,omitempty"`
	Inputs          []string `json:"inputs,omitempty"`
	Outputs         []string `json:"outputs,omitempty"`
	Tasks           []Task   `json:"tasks,omitempty"`
}

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	FinishDate   *metav1.Time `json:"finishDate,omitempty"`
	Initailizing int          `json:"initializing,omitempty"`
	Running      int          `json:"running,omitempty"`
	Completed    int          `json:"completed,omitempty"`
	CurrentJobs  []string     `json:"currentJobs,omitempty"`
	Jobs         []string     `json:"jobs,omitempty"`

	CreateDate     *metav1.Time `json:"createDate,omitempty"`
	LastUpdateDate *metav1.Time `json:"lastUpdateDate,omitempty"`
	Runs           int          `json:"runs,omitempty"`
}

type JobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name           string       `json:"name,omitempty"`
	State          JobState     `json:"state,omitempty"`
	RunAfter       []string     `json:"runAfter,omitempty"`
	RunBefore      []string     `json:"runBefore,omitempty"`
	StartDate      *metav1.Time `json:"startDate,omitempty"`
	LastUpdateDate *metav1.Time `json:"lastUpdateDate,omitempty"`
	FinishDate     *metav1.Time `json:"finishDate,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="State"
// +kubebuilder:printcolumn:name="StartDate",type="string",JSONPath=".status.startDate",description="Start date"
// +kubebuilder:printcolumn:name="FinishDate",type="string",JSONPath=".status.finishDate",description="Finish date"
// +kubebuilder:printcolumn:name="Initializing",type="number",JSONPath=".status.initializing",description="initializing"
// +kubebuilder:printcolumn:name="running",type="number",JSONPath=".status.running",description="running"
// +kubebuilder:printcolumn:name="completed",type="number",JSONPath=".status.completed",description="completed"
// Pipeline is the Schema for the pipelines API
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PipelineSpec   `json:"spec,omitempty"`
	Status            PipelineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
