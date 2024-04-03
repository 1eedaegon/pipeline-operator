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

// TaskSpec defines the desired state of Task
type TaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name      string   `json:"name,omitempty"` - Name은 Spec이 아니라 metadata이다.
	Image   string   `json:"image,omitempty"`
	Mode    ModeType `json:"mode,omitempty"`    // Schedule와 Manual mode가 상충할 수 없다.
	Command []string `json:"command,omitempty"` // image의 entrypoint/command를 덮어 쓴다.
	Args    []string `json:"args,omitempty"`    // image의 command는 두고 arg만 추가한다.
	// 밑의 6가지 내용은 pipeline에서 유동적으로 넣어줄 수 있는 타입
	// Resource  Resource `json:"resource,omitempty"`
	// Schedule  Schedule `json:"schedule,omitempty"`
	// RunAfter  []string `json:"runAfter,omitempty"`
	// RunBefore []string `json:"runBefore,omitempty"`
	// Inputs    []string `json:"inputs,omitempty"`
	// Outputs   []string `json:"outputs,omitempty"`
}

// TaskStatus defines the observed state of Task
type TaskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CreateDate      *metav1.Time `json:"createDate,omitempty"`
	LastUpdateDate  *metav1.Time `json:"lastUpdateDate,omitempty"`
	UsedByPipelines []string     `json:"usedByPipelines,omitempty"`
	UsedByJobs      int          `json:"usedByJobs,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Task is the Schema for the tasks API
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TaskList contains a list of Task
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
