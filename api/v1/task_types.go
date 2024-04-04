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

// ToolBox의 개념으로 바라본다.
// TaskSpec defines the desired state of Task
// metadata에도 name이 있고, spec에도 name이 있다. 이유는 inline task 때문에
type TaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name    string   `json:"name,omitempty"`
	Image   string   `json:"image,omitempty"`
	Mode    ModeType `json:"mode,omitempty"`    // Schedule와 Manual mode가 상충할 수 없다.
	Command string   `json:"command,omitempty"` // image의 entrypoint/command를 덮어 쓴다.
	Args    []string `json:"args,omitempty"`    // image의 command는 두고 arg만 추가한다.
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
