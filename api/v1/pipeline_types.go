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

type HistoryLimit *metav1.Time

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
	ScheduleType ScheduleType `json:"scheduleType,omitempty"` // ScheduleType이 cron이면 cron의 최초 도달시점, date면 시스템 시간에 시작
	ScheduleDate string       `json:"scheduleDate,omitempty"` // ScheduleDate를 기점으로 scheduling 시작
	EndDate      string       `json:"endDate,omitempty"`      // 현재 *time.Time이 EndDate보다 높으면 complete and no queuing
}

type ModeType string

const (
	Auto   ModeType = "auto"
	Manual ModeType = "manual"
)

type PipelineTaskType string

const (
	Inline PipelineTaskType = "inline"
	Import PipelineTaskType = "import"
)

type PipelineTask struct {
	// Inline 타입이면 Task를 수동으로 기입해줘야한다. inline에서 정의한 task가 task 템플릿으로 들어가진 않는다.
	// Import 타입이면 이미있는 Task를 기준으로 Task가 채워진다.
	Type      PipelineTaskType `json:"pipelineTaskType,omitempty"`
	Name      string           `json:"name,omitempty"`
	Task      TaskSpec         `json:"task,omitempty"`
	Resource  Resource         `json:"resource,omitempty"`
	Schedule  Schedule         `json:"schedule,omitempty"`
	RunAfter  []string         `json:"runAfter,omitempty"`
	RunBefore []string         `json:"runBefore,omitempty"`
	Inputs    []string         `json:"inputs,omitempty"`
	Outputs   []string         `json:"outputs,omitempty"`
}

// PipelineSpec defines the desired state of Pipeline
type PipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name       string   `json:"name,omitempty"` - Spec이 아니라 Metadata에 들어가야할 내용임.
	Tasks        []PipelineTaskType `json:"tasks,omitempty"`
	Schedule     Schedule           `json:"schedule,omitempty"`
	VolumeName   string             `json:"volumeName,omitempty"`   // Volume이 run으로 진입했을 때 겹칠 수 있으니 새로 생성해야한다. +prefix
	HistoryLimit HistoryLimit       `json:"historyLimit,omitempty"` // post-run 상태의 pipeline들의 최대 보존 기간
	RunAfter     []string           `json:"runAfter,omitempty"`
	RunBefore    []string           `json:"runBefore,omitempty"`
	Inputs       []string           `json:"inputs,omitempty"`   // RX
	Outputs      []string           `json:"outputs,omitempty"`  // RWX
	Resource     Resource           `json:"resource,omitempty"` // task에 리소스가 없을 때, pipeline에 리소스가 지정되어있다면 이것을 적용
}

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Runs           int          `json:"runs,omitempty"`           // Number of run
	CreateDate     *metav1.Time `json:"createDate,omitempty"`     // Date of created pipeline
	LastUpdateDate *metav1.Time `json:"lastUpdateDate,omitempty"` // Last modified date pipeline
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
