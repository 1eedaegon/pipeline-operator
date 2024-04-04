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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

// JobState를 참고해서 작성되었다.
type RunState string

const (
	// Pre run
	RunStateInit JobState = "initializing"
	RunStateWait JobState = "waiting"
	RunStateStop JobState = "stopping"
	// run
	RunStateRun      JobState = "running"
	RunStateDeleting JobState = "deleting"
	// post run
	RunStateCompleted JobState = "completed"
	RunStateDeleted   JobState = "deleted"
	RunStateFailed    JobState = "failed"
)

// RunSpec defines the desired state of Run
type RunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name      string   `json:"name,omitempty"` - Name은 Spec이 아니라 metadata이다.
	Schedule     Schedule     `json:"schedule,omitempty"`
	VolumeName   string       `json:"volumeName,omitempty"`   // Volume이 run으로 진입했을 때 겹칠 수 있으니 새로 생성해야한다. +prefix
	HistoryLimit HistoryLimit `json:"historyLimit,omitempty"` // post-run 상태의 pipeline들의 최대 보존 기간: Default - 1D
	RunAfter     []string     `json:"runAfter,omitempty"`
	RunBefore    []string     `json:"runBefore,omitempty"`
	Inputs       []string     `json:"inputs,omitempty"`   // RX
	Outputs      []string     `json:"outputs,omitempty"`  // RWX
	Resource     Resource     `json:"resource,omitempty"` // task에 리소스가 없을 때, pipeline에 리소스가 지정되어있다면 이것을 적용
	Jobs         []string     `json:"jobs,omitempty"`
}

// RunStatus defines the observed state of Run
// pre-run: waiting, initializing, stopping
// run: running, deleting
// post-run: completed, failed, deleted
// deleted는 뭐냐, run시킬 수 없지만 보존기간동안 volume이 보관되는 상태

// RunState order: run > pre-run > post-run
// Running으로 한번 진입하면 completed 아니면 Failed이다.
// Deleting으로 한번 진입하면 deleted 아니면 failed이다.
type RunStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	RunState          RunState     `json:"runState,omitempty"`          // run > pre-run > post-run
	CurrentWorkingJob string       `json:"currentWorkingJob,omitempty"` // current-working-job-name(string)
	Initializing      uint         `json:"initializing,omitempty"`      // initializing/total
	Waiting           uint         `json:"waiting,omitempty"`           // waiting/total
	Stopping          uint         `json:"stopping,omitempty"`          // stopping/total
	Running           uint         `json:"running,omitempty"`           // running/total
	Deleting          uint         `json:"deleting,omitempty"`          // deleting/total
	Completed         uint         `json:"completed,omitempty"`         // completed/total
	Deleted           uint         `json:"deleted,omitempty"`           // deleted/total
	Failed            uint         `json:"failed,omitempty"`            // failed/total
	CreateDate        *metav1.Time `json:"createDate,omitempty"`
	LastUpdateDate    *metav1.Time `json:"lastUpdateDate,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Run is the Schema for the runs API
type Run struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunSpec   `json:"spec,omitempty"`
	Status RunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RunList contains a list of Run
type RunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Run `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Run{}, &RunList{})
}
