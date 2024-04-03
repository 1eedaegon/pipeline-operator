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
	kbatchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

type RunState string

const (
	RunStateInit      RunState = "Initializing"
	RunStateRun       RunState = "Running"
	RunStateCompleted RunState = "Completed"
	RunStateStop      RunState = "Stopping"
	RunStateDeleting  RunState = "Deleting"
	RunStateDeleted   RunState = "Deleted"
)

// RunSpec defines the desired state of Run
type RunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name       string             `json:"name,omitempty"`
	VolumeName string             `json:"volumeName,omitempty"` // Volume이 run으로 진입했을 때 겹칠 수 있으니 새로 생성해야한다. +prefix
	Schedule   Schedule           `json:"schedule,omitempty"`
	RunAfter   []string           `json:"runAfter,omitempty"`
	RunBefore  []string           `json:"runBefore,omitempty"`
	Inputs     []string           `json:"inputs,omitempty"`
	Outputs    []string           `json:"outputs,omitempty"`
	Jobs       []kbatchv1.JobSpec `json:"tasks,omitempty"`
}

// RunStatus defines the observed state of Run
type RunStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	RunState RunState
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
