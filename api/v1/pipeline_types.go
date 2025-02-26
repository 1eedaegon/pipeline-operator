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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ScheduleDateAnnotation = "pipeline.1eedaegon.github.io/schedule-date"
	EndDateAnnotation      = "pipeline.1eedaegon.github.io/end-date"
	ScheduledAtAnnotation  = "pipeline.1eedaegon.github.io/scheduled-at"
	TriggerAnnotation      = "pipeline.1eedaegon.github.io/trigger"
	PipelineNameLabel      = "pipeline.1eedaegon.github.io/pipeline-name"
)

type Trigger bool

const (
	IsTriggered    Trigger = true
	IsNotTriggered Trigger = false
)

func (t Trigger) String() string {
	return strconv.FormatBool(bool(t))
}
func (t Trigger) TriggerString() TriggerString {
	return TriggerString(t.String())
}

type HistoryLimit struct {
	Amount uint   `json:"amount,omitempty"`
	Date   string `json:"date,omitempty"`
}

type Resource struct {
	Cpu    string      `json:"cpu,omitempty"`
	Memory string      `json:"memory,omitempty"`
	Gpu    GpuResource `json:"gpu,omitempty"`
}

type GpuResource struct {
	GpuType string `json:"type,omitempty"`
	Amount  int    `json:"amount,omitempty"`
}

type VolumeResource struct {
	Name     string `json:"name,omitempty"`
	Capacity string `json:"capacity,omitempty"`
	Storage  string `json:"storage,omitempty"`
}

// Pipeline / Task 내 Input / Output 정의에 필요한 Spec
type IOVolumeSpec struct {
	// Volume의 name과 동일해야 한다. (기존 or 신규 생성 Volume 공통)
	Name string `json:"name,omitempty"`
	// pvc root 하위의 run별로 격리된 hashed path를 subPath로 사용할 것인지 정의한다. (PipelineSpec 참고)
	UseIntermediateDirectory bool `json:"useIntermediateDirectory,omitempty"`
	// parseVolumeMountList() parameter까지 run 정의를 넘겨줄 수 없기에 사용하는 private field.
	intermediateDirectoryName string
}

type ScheduleDate string

func (sd ScheduleDate) durationFromDateString() (time.Duration, error) {
	var duration time.Duration
	// date string parser
	scheduleDatePattern := `(\d+)([smhdMy])`
	re := regexp.MustCompile(scheduleDatePattern)
	matcheSlice := re.FindAllStringSubmatch(string(sd), -1)
	for _, match := range matcheSlice {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			return 0, err
		}
		dateRune := match[2]
		switch dateRune {
		case "s":
			duration += time.Duration(value) * time.Second
		case "m":
			duration += time.Duration(value) * time.Minute
		case "h":
			duration += time.Duration(value) * time.Hour
		case "d":
			duration += time.Duration(value) * 24 * time.Hour
		case "w":
			duration += time.Duration(value) * 7 * 24 * time.Hour
		case "M":
			duration += time.Duration(value) * 30 * 24 * time.Hour // 간단화를 위해 모든 달을 30일로 가정
		case "y":
			duration += time.Duration(value) * 365 * 24 * time.Hour // 윤년은 고려하지 않음
		default:
			return 0, fmt.Errorf("unknown date string: %s", strings.Join(match, ""))
		}
	}
	return duration, nil
}

func (sd ScheduleDate) durationFromCron() (time.Duration, error) {
	// cron parser
	cronExpr, err := cron.ParseStandard(string(sd))
	duration := cronExpr.Next(time.Now()).Sub(time.Now())
	if err != nil {
		return 0, err
	}
	return duration, nil
}
func (sd ScheduleDate) Duration() (time.Duration, error) {
	cronDuration, err := sd.durationFromCron()
	dateDuration, err2 := sd.durationFromDateString()
	if err != nil && err2 != nil {
		return 0, fmt.Errorf("unknown format schedule")
	}
	if err != nil {
		return dateDuration, nil
	}
	return cronDuration, nil
}

type Schedule struct {
	// ScheduleType ScheduleType `json:"type,omitempty"`         // ScheduleType이 cron이면 cron의 최초 도달시점, date면 시스템 시간에 시작
	ScheduleDate ScheduleDate `json:"scheduleDate,omitempty"` // ScheduleDate를 기점으로 scheduling 시작
	EndDate      string       `json:"endDate,omitempty"`      // 현재 *time.Time이 EndDate보다 높으면 complete and no queuing
}

type ModeType string

const (
	Auto   ModeType = "auto" // Default
	Manual ModeType = "manual"
)

type PipelineTask struct {
	// Inline 타입이면 Task를 수동으로 기입해줘야한다. inline에서 정의한 task가 task 템플릿으로 들어가진 않는다.
	// Import 타입이면 이미있는 Task를 기준으로 Task가 채워진다.
	TaskSpec  `json:",inline,omitempty"` // Task의 image 키워드가 없으면 name을 불러온다. 존재하지 않으면 에러가 발생한다.
	Schedule  Schedule                   `json:"schedule,omitempty"`
	Resource  Resource                   `json:"resource,omitempty"`
	Trigger   Trigger                    `json:"trigger,omitempty"`
	RunBefore []string                   `json:"runBefore,omitempty"`
	Inputs    []IOVolumeSpec             `json:"inputs,omitempty"`
	Outputs   []IOVolumeSpec             `json:"outputs,omitempty"`
	Env       map[string]string          `json:"env,omitempty"`
}

/*
	Need Annotations:
	- pipeline.1eedaegon.github.io/scheduled-at
	- pipeline.1eedaegon.github.io/created-at
	- pipeline.1eedaegon.github.io/created-by
	- pipeline.1eedaegon.github.io/updated-at
*/

// PipelineSpec defines the desired state of Pipeline
type PipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Name       string   `json:"name,omitempty"` - Spec이 아니라 Metadata에 들어가야할 내용임.
	Schedule Schedule `json:"schedule,omitempty"`
	/*
		하나의 pipeline의 run들은 동일한 volume을 사용한다.
		hashed intermediate directory (IOVolumeSpec.UseIntermediateDirectory) 구성에 따른 영향과 목적은 다음과 같다.
		  true: 개별 run의 input / output에서 source hashed intermediate directory를 사용함으로써 개별 run이 다룰 수 있는 volume 내 데이터가 격리된다.
				Share intermediate directory on same run based jobs
				It is attended to output of a job becomes input of other job.
				To avoid this sharing, append subDirectories on task input / output name.
		  false: pvc에 초기 주입된 기존 데이터를 사용하거나 기존 run에서 write한 결과를 재사용할 수 있다.
	*/
	Volumes      []VolumeResource  `json:"volumes,omitempty"`
	Trigger      Trigger           `json:"trigger,omitempty"`
	HistoryLimit HistoryLimit      `json:"historyLimit,omitempty"` // post-run 상태의 pipeline들의 최대 보존 기간
	Tasks        []PipelineTask    `json:"tasks,omitempty"`
	RunBefore    []string          `json:"runBefore,omitempty"`
	Inputs       []IOVolumeSpec    `json:"inputs,omitempty"`   // RX
	Outputs      []IOVolumeSpec    `json:"outputs,omitempty"`  // RWX
	Resource     Resource          `json:"resource,omitempty"` // task에 리소스가 없을 때, pipeline에 리소스가 지정되어있다면 이것을 적용
	Env          map[string]string `json:"env,omitempty"`
}

// PipelineStatus defines the observed state of Pipeline
type PipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Runs            int          `json:"runs,omitempty"`            // Number of run
	CreatedDate     *metav1.Time `json:"createdDate,omitempty"`     // Date of created pipeline
	LastUpdatedDate *metav1.Time `json:"lastUpdatedDate,omitempty"` // Last modified date pipeline
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Runs",type="integer",JSONPath=".status.runs",description="Number of executed run"
// +kubebuilder:printcolumn:name="CreatedDate",type="string",JSONPath=".status.createdDate",description="Time of when created pipeline"
// +kubebuilder:printcolumn:name="LastUpdatedDate",type="string",JSONPath=".status.lastUpdatedDate",description="Lastest tiem when pipeline updated it."
// Pipeline is the Schema for the pipelines API
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PipelineSpec   `json:"spec,omitempty"`
	Status            PipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// PipelineList contains a list of Pipeline
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
