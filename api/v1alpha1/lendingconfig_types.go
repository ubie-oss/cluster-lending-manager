/*
Copyright 2022 Daisuke Taniwaki..

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Schedule represents a time schedule with a start and end time.
type Schedule struct {
	// Start is the start time of the schedule.
	Start *string `json:"start,omitempty"`
	// End is the end time of the schedule.
	End *string `json:"end,omitempty"`
}

// DaySchedule represents a schedule for a specific day, consisting of multiple time schedules.
type DaySchedule struct {
	// Hours is a list of time schedules for the day.
	Hours []Schedule `json:"hours,omitempty"`
}

// ScheduleSpec represents the schedule specification for a lending configuration.
type ScheduleSpec struct {
	// Default is the default schedule for all days.
	Default *DaySchedule `json:"default,omitempty"`
	// Monday is the schedule for Monday.
	Monday *DaySchedule `json:"monday,omitempty"`
	// Tuesday is the schedule for Tuesday.
	Tuesday *DaySchedule `json:"tuesday,omitempty"`
	// Wednesday is the schedule for Wednesday.
	Wednesday *DaySchedule `json:"wednesday,omitempty"`
	// Thursday is the schedule for Thursday.
	Thursday *DaySchedule `json:"thursday,omitempty"`
	// Friday is the schedule for Friday.
	Friday *DaySchedule `json:"friday,omitempty"`
	// Saturday is the schedule for Saturday.
	Saturday *DaySchedule `json:"saturday,omitempty"`
	// Sunday is the schedule for Sunday.
	Sunday *DaySchedule `json:"sunday,omitempty"`

	// Always indicates if the schedule is always active.
	Always bool `json:"always,omitempty"`

	// TODO: Support holidays.
}

// Target represents a target object for the lending configuration.
type Target struct {
	// APIVersion is the API version of the target object.
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind is the kind of the target object.
	Kind string `json:"kind,omitempty"`
	// Name is the name of the target object.
	Name *string `json:"name,omitempty"`
	// DefaultReplicas is the default number of replicas for the target object.
	DefaultReplicas *int64 `json:"defaultReplicas,omitempty"`
}

// LendingConfigSpec defines the desired state of a LendingConfig.
type LendingConfigSpec struct {
	// Timezone is the timezone for the lending configuration.
	Timezone string `json:"timezone,omitempty"`
	// Schedule is the schedule specification for the lending configuration.
	Schedule ScheduleSpec `json:"schedule,omitempty"`
	// ScheduleMode is the schedule mode for the lending configuration.
	// +kubebuilder:default=Cron
	// +kubebuilder:validation:Enum=Always;Cron;Schedule
	ScheduleMode string `json:"scheduleMode,omitempty"`
	// Targets is a list of target objects for the lending configuration.
	Targets []Target `json:"targets,omitempty"`
}

// ObjectReference contains enough information to let you locate the typed referenced object inside the same namespace.
type ObjectReference struct {
	// APIVersion is the API version of the referenced object.
	APIVersion string `json:"apiVersion,omitempty"`
	// Kind is the kind of the referenced object.
	Kind string `json:"kind,omitempty"`
	// Name is the name of the referenced object.
	Name string `json:"name,omitempty"`
}

// LendingReference represents a reference to a lending object with the number of replicas.
type LendingReference struct {
	// ObjectReference is the reference to the lending object.
	ObjectReference `json:"objectReference,omitempty"`
	// Replicas is the number of replicas for the lending object.
	Replicas int64 `json:"replicas,omitempty"`
}

// LendingConfigStatus defines the observed state of a LendingConfig.
type LendingConfigStatus struct {
	// LendingReferences is a list of references to lending objects.
	LendingReferences []LendingReference `json:"objectReferences,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LendingConfig is the Schema for the lendingconfigs API.
type LendingConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the desired state of the lending configuration.
	Spec LendingConfigSpec `json:"spec,omitempty"`
	// Status is the observed state of the lending configuration.
	Status LendingConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LendingConfigList contains a list of LendingConfig
type LendingConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is the list of LendingConfig objects.
	Items []LendingConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LendingConfig{}, &LendingConfigList{})
}
