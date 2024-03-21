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

/*
# Pipeline/Run:
# - Create/Update: Initializing|Running|Completed
# - Delete: Stopping|Deleting|Deleted
# Task/Job: pipeline의 하위에 존재한다.
# - Create: Initializing|Waiting|Running|Completed
# - Update/Delete: Initializing|Waiting|Running|Completed 새로운 pipeline을 통해 run을 만드는 개념

apiVersion: pipeline.1eedaegon.github.io/v1
kind: Pipeline
metadata:
  projectName: test-project
  name: pipeline-chain-test
spec:
  schedule: # Schedule의 cron과 runAfter/runBefore가 동시에 걸리면 경고를 띄운다. => after는 하위 시간 무시, before는 상위가 나를 무시
    type: cron
    interval: "5 * * * * *"
  # schedule:
  #   type: date
  #   startAt: "2024-06-30:16:00:00"
  runAfter: []
  runBefore: [] # runBefore를 걸었을 때 tasks/pipeline에 해당 이름이 없으면 경고를 띄운다.
  tasks:
  - name: load-data
  	image: s3
	command:
	- echo
	args:
	- "hello 1eedaegon.github tasks"
    # Schedule의 cron과 runAfter/runBefore가 동시에 걸리면 경고를 띄운다. => after는 하위 시간 무시, before는 상위가 나를 무시
    # cron일 때 pipeline에 schedule이 걸려있으면 경고를 띄운다.
    schedule:
      type: date
      startAt: "2024-06-30:16:00:00"
    runBefore: []
    runAfter:
    - transform-data
    inputs: [] # runBefore와 inputs가 동시에 걸리고 runBefore의 task에 output이 없으면 경고를 띄운다.
    outputs: []

    resource:
      cpu: 1
      memory: 2Mi
      gpu:
        gpuType: nvidia # gpu를 걸고 타입을 걸었을 때 node label에 없으면 경고를 띄운다.
        amount: 2
  - name: transform-data
    runBefore:
    - load-data
    runAfter: []
    inputs: [] # runBefore와 inputs가 동시에 걸리고 runBefore의 task에 output이 없으면 경고를 띄운다.
    outputs: []
    steps:
    - name: from-s3
      image: nginx
      command:
      - echo
      args:
      - "hello 1eedaegon.github tasks"
    resource:
      cpu: 1
      memory: 2Mi

status:
  # name 뒤에 postfix가 붙어야한다.
  state: running
  currentTask:
  - load-data-1
  - load-data-2
  startDate: 2024-05-13:16:00:00
  finishDate: 2024-05-13:17:00:00
  initializing: 1
  running: 2
  completed: 1
  jobs:
  - name: initialize-data
    state: completed
    runAfter:
	  - load-data-1
	  - load-data-2
    startDate: 2024-05-13:16:00:00
    finishDate: 2024-05-13:17:00:00S
*/

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

type ScheduleType string

const (
	Cron ScheduleType = "cron"
	Date ScheduleType = "date"
)

type Schedule struct {
	ScheduleType ScheduleType `json:"type,omitempty"`
	StartAt      string       `json:"startAt,omitempty"`
}

// - name: load-data
// image: nginx
// command:
// - aws s3
// args:
// - "cp s3://something"
// # Schedule의 cron과 runAfter/runBefore가 동시에 걸리면 경고를 띄운다. => after는 하위 시간 무시, before는 상위가 나를 무시
// # cron일 때 pipeline에 schedule이 걸려있으면 경고를 띄운다.
// schedule:
//
//	type: date
//	startAt: "2024-06-30:16:00:00"
//
// runBefore: []
// runAfter:
// - transform-data
// inputs: [] # runBefore와 inputs가 동시에 걸리고 runBefore의 task에 output이 없으면 경고를 띄운다.
// outputs: []
// resource:
//
//	  cpu: 1
//	  memory: 2Mi
//	  gpu:
//		type: nvidia # gpu를 걸고 타입을 걸었을 때 node label에 없으면 경고를 띄운다.
//		amount: 2
//
// TaskSpec defines the desired state of Task
type Task struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name      string   `json:"name,omitempty"`
	Image     string   `json:"image,omitempty"`
	Command   []string `json:"command,omitempty"` // image의 entrypoint/command를 덮어 쓴다.
	Args      []string `json:"args,omitempty"`    // image의 command는 두고 arg만 추가한다.
	Resource  Resource `json:"resource,omitempty"`
	Schedule  Schedule `json:"schedule,omitempty"`
	RunBefore []string `json:"runBefore,omitempty"`
	RunAfter  []string `json:"runAfter,omitempty"`
	Inputs    []string `json:"inputs,omitempty"`
	Outputs   []string `json:"outputs,omitempty"`
}

// status:
//
//	  # name 뒤에 postfix가 붙어야한다.
//	  state: running
//	  currentJobs:
//	  - load-data-1
//	  - load-data-2
//	  startDate: 2024-05-13:16:00:00
//	  finishDate: 2024-05-13:17:00:00
//	  initializing: 1
//	  running: 2
//	  completed: 1
//	  jobs:
//	  - name: initialize-data
//	    state: completed
//	    runAfter:
//		  - load-data-1
//		  - load-data-2
//	    startDate: 2024-05-13:16:00:00
//	    finishDate: 2024-05-13:17:00:00S
//
// TaskStatus defines the observed state of Task
// Task가 status를 갖는다는 의미는 job으로 변환되었다는 의미다.
type TaskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name       string   `json:"name,omitempty"`
	State      JobState `json:"state,omitempty"`
	RunAfter   []string `json:"runAfter,omitempty"`
	RunBefore  []string `json:"runBefore,omitempty"`
	StartDate  string   `json:"startDate,omitempty"`
	FinishDate string   `json:"finishDate,omitempty"`
}

// schedule: # Schedule의 cron과 runAfter/runBefore가 동시에 걸리면 경고를 띄운다. => after는 하위 시간 무시, before는 상위가 나를 무시
//
//	type: cron
//	interval: "5 * * * * *"
//
// # schedule:
// #   type: date
// #   startAt: "2024-06-30:16:00:00"
// runAfter: []
// runBefore: [] # runBefore를 걸었을 때 tasks/pipeline에 해당 이름이 없으면 경고를 띄운다.
// tasks:
// PipelineSpec defines the desired state of Pipeline
type PipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name      string   `json:"name,omitempty"`
	Schedule  Schedule `json:"schedule,omitempty"`
	RunAfter  []string `json:"runAfter,omitempty"`
	RunBefore []string `json:"runBefore,omitempty"`
	Inputs    []string `json:"inputs,omitempty"`
	Outputs   []string `json:"outputs,omitempty"`
	Tasks     []Task   `json:"tasks,omitempty"`
}

//	state: running
//	 startDate: 2024-05-13:16:00:00
//	 finishDate: 2024-05-13:17:00:00
//	 initializing: 1
//	 running: 1
//	 completed: 0
//	 currentJobs:
//	 - load-data
//
// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	State        RunState     `json:"state,omitempty"`
	StartDate    *metav1.Time `json:"startDate,omitempty"`
	FinishDate   *metav1.Time `json:"finishDate,omitempty"`
	Initailizing int          `json:"initializing,omitempty"`
	Running      int          `json:"running,omitempty"`
	Completed    int          `json:"completed,omitempty"`
	CurrentJobs  []string     `json:"currentJobs,omitempty"`
	Jobs         []TaskStatus `json:"jobs,omitempty"`
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
