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

package controllers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	// "github.com/ubie-oss/cluster-lending-manager/api/v1alpha1"
	clusterlendingmanagerv1alpha1 "github.com/ubie-oss/cluster-lending-manager/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

type LendingConfig clusterlendingmanagerv1alpha1.LendingConfig

var hoursPattern = regexp.MustCompile(`(\d{2}):(\d{2}) *(am|pm)?`)

type LendingConfigEvent = string
type LendingScheduleMode = string

const annotationNameSkip = "clusterlendingmanager.ubie-oss.github.com/skip"
const annotationNameDefaultReplicas = "clusterlendingmanager.ubie-oss.github.com/default-replicas"

const (
	SchedulesUpdated LendingConfigEvent = "SchedulesUpdated"
	SchedulesCleared LendingConfigEvent = "SchedulesCleared"
	LendingStarted   LendingConfigEvent = "LendingStarted"
	LendingEnded     LendingConfigEvent = "endingEnded"

	ScheduleModeAlways LendingScheduleMode = "Always"
	ScheduleModeNever  LendingScheduleMode = "Never"
	ScheduleModeCron   LendingScheduleMode = "Cron"
)

func (config *LendingConfig) ClearSchedules(ctx context.Context, reconciler *LendingConfigReconciler) error {
	logger := log.FromContext(ctx)

	logger.Info("Clear schedules")

	reconciler.Cron.Clear(config.ToNamespacedName())

	reconciler.Recorder.Event(config.ToCompatible(), corev1.EventTypeNormal, SchedulesCleared, "Schedules cleared.")

	return nil
}

func (config *LendingConfig) UpdateSchedules(ctx context.Context, reconciler *LendingConfigReconciler) error {
	logger := log.FromContext(ctx)

	logger.Info("Update schedules")

	reconciler.Cron.Clear(config.ToNamespacedName())

	// ScheduleMode
	// - Always: Always lend the cluster.
	// - Never: Never lend the cluster.
	// - Cron: Lend the cluster according to the schedule. (Default)
	if config.Spec.ScheduleMode == ScheduleModeAlways {
		logger.Info(fmt.Sprintf("Allow always activate resources (SchedulerMode=%s)", ScheduleModeAlways))
		if err := config.ActivateTargetResources(ctx, reconciler); err != nil {
			return err
		}
		return nil
	}
	if config.Spec.ScheduleMode == ScheduleModeNever {
		logger.Info(fmt.Sprintf("Allow always deactivate resources (SchedulerMode=%s)", ScheduleModeNever))
		if _, err := config.DeactivateTargetResources(ctx, reconciler); err != nil {
			return err
		}
		return nil
	}

	// TODO: Remove in future
	if config.Spec.Schedule.Always {
		logger.Info("Allow always lend the cluster")
		if err := config.ActivateTargetResources(ctx, reconciler); err != nil {
			return err
		}
		return nil
	}

	logger.Info(fmt.Sprintf("Set individual day schedules (SchedulerMode=%s)", ScheduleModeCron))

	items := []CronItem{}

	var monHours []clusterlendingmanagerv1alpha1.Schedule
	if config.Spec.Schedule.Monday != nil && config.Spec.Schedule.Monday.Hours != nil {
		monHours = config.Spec.Schedule.Monday.Hours
	} else if config.Spec.Schedule.Default != nil && config.Spec.Schedule.Default.Hours != nil {
		monHours = config.Spec.Schedule.Default.Hours
	}
	if monHours != nil {
		crons, err := config.getCrons(reconciler, "mon", monHours)
		if err != nil {
			return err
		}
		items = append(items, crons...)
	}

	var tueHours []clusterlendingmanagerv1alpha1.Schedule
	if config.Spec.Schedule.Tuesday != nil && config.Spec.Schedule.Tuesday.Hours != nil {
		tueHours = config.Spec.Schedule.Tuesday.Hours
	} else if config.Spec.Schedule.Default != nil && config.Spec.Schedule.Default.Hours != nil {
		tueHours = config.Spec.Schedule.Default.Hours
	}
	if tueHours != nil {
		crons, err := config.getCrons(reconciler, "tue", tueHours)
		if err != nil {
			return err
		}
		items = append(items, crons...)
	}

	var wedHours []clusterlendingmanagerv1alpha1.Schedule
	if config.Spec.Schedule.Wednesday != nil && config.Spec.Schedule.Wednesday.Hours != nil {
		wedHours = config.Spec.Schedule.Wednesday.Hours
	} else if config.Spec.Schedule.Default != nil && config.Spec.Schedule.Default.Hours != nil {
		wedHours = config.Spec.Schedule.Default.Hours
	}
	if wedHours != nil {
		crons, err := config.getCrons(reconciler, "wed", wedHours)
		if err != nil {
			return err
		}
		items = append(items, crons...)
	}

	var thuHours []clusterlendingmanagerv1alpha1.Schedule
	if config.Spec.Schedule.Thursday != nil && config.Spec.Schedule.Thursday.Hours != nil {
		thuHours = config.Spec.Schedule.Thursday.Hours
	} else if config.Spec.Schedule.Default != nil && config.Spec.Schedule.Default.Hours != nil {
		thuHours = config.Spec.Schedule.Default.Hours
	}
	if thuHours != nil {
		crons, err := config.getCrons(reconciler, "thu", thuHours)
		if err != nil {
			return err
		}
		items = append(items, crons...)
	}

	var friHours []clusterlendingmanagerv1alpha1.Schedule
	if config.Spec.Schedule.Friday != nil && config.Spec.Schedule.Friday.Hours != nil {
		friHours = config.Spec.Schedule.Friday.Hours
	} else if config.Spec.Schedule.Default != nil && config.Spec.Schedule.Default.Hours != nil {
		friHours = config.Spec.Schedule.Default.Hours
	}
	if friHours != nil {
		crons, err := config.getCrons(reconciler, "fri", friHours)
		if err != nil {
			return err
		}
		items = append(items, crons...)
	}

	var satHours []clusterlendingmanagerv1alpha1.Schedule
	if config.Spec.Schedule.Saturday != nil && config.Spec.Schedule.Saturday.Hours != nil {
		satHours = config.Spec.Schedule.Saturday.Hours
	} else if config.Spec.Schedule.Default != nil && config.Spec.Schedule.Default.Hours != nil {
		satHours = config.Spec.Schedule.Default.Hours
	}
	if satHours != nil {
		crons, err := config.getCrons(reconciler, "sat", satHours)
		if err != nil {
			return err
		}
		items = append(items, crons...)
	}

	var sunHours []clusterlendingmanagerv1alpha1.Schedule
	if config.Spec.Schedule.Sunday != nil && config.Spec.Schedule.Sunday.Hours != nil {
		sunHours = config.Spec.Schedule.Sunday.Hours
	} else if config.Spec.Schedule.Default != nil && config.Spec.Schedule.Default.Hours != nil {
		sunHours = config.Spec.Schedule.Default.Hours
	}
	if sunHours != nil {
		crons, err := config.getCrons(reconciler, "sun", sunHours)
		if err != nil {
			return err
		}
		items = append(items, crons...)
	}

	err := reconciler.Cron.Add(config.ToNamespacedName(), items)
	if err != nil {
		return errors.WithStack(err)
	}

	reconciler.Recorder.Event(config.ToCompatible(), corev1.EventTypeNormal, SchedulesUpdated, "Schedules updated.")

	return nil
}

func (config *LendingConfig) ToCompatible() *clusterlendingmanagerv1alpha1.LendingConfig {
	return (*clusterlendingmanagerv1alpha1.LendingConfig)(config)
}

func (config *LendingConfig) ToNamespacedName() types.NamespacedName {
	return types.NamespacedName{Namespace: config.Namespace, Name: config.Name}
}

func (config *LendingConfig) getCrons(reconciler *LendingConfigReconciler, dayOfWeek string, schedules []clusterlendingmanagerv1alpha1.Schedule) ([]CronItem, error) {
	res := []CronItem{}
	for _, hours := range schedules {
		var startTimeMinutes *int32
		if hours.Start != nil {
			hour, minute, err := parseHours(*hours.Start)
			if err != nil {
				return nil, err
			}
			startTimeMinutes = pointer.Int32Ptr(60*hour + minute)
			tsz := fmt.Sprintf("CRON_TZ=%s %d %d * * %s", config.Spec.Timezone, minute, hour, dayOfWeek)
			res = append(res, CronItem{Cron: tsz, Job: NewCronContext(
				reconciler, config, LendingStart,
			)})
		}
		if hours.End != nil {
			hour, minute, err := parseHours(*hours.End)
			if err != nil {
				return nil, err
			}
			if startTimeMinutes != nil && 60*hour+minute <= *startTimeMinutes {
				return nil, errors.New("The end time must be later than the start time.")
			}
			tsz := fmt.Sprintf("CRON_TZ=%s %d %d * * %s", config.Spec.Timezone, minute, hour, dayOfWeek)
			res = append(res, CronItem{Cron: tsz, Job: NewCronContext(
				reconciler, config, LendingEnd,
			)})
		}
	}
	return res, nil
}

func (config *LendingConfig) ActivateTargetResources(ctx context.Context, reconciler *LendingConfigReconciler) error {
	logger := log.FromContext(ctx)

	for _, target := range config.Spec.Targets {
		groupVersionKind, err := getGroupVersionKind(target)
		if err != nil {
			return err
		}
		objs := &unstructured.Unstructured{}
		objs.SetGroupVersionKind(groupVersionKind)

		err = reconciler.List(ctx, objs, &client.ListOptions{Namespace: config.Namespace})
		if err != nil {
			return errors.WithStack(err)
		}
		err = objs.EachListItem(func(obj runtime.Object) error {
			uobj := obj.(*unstructured.Unstructured)
			logger.Info(fmt.Sprintf("Patch %s %s/%s", groupVersionKind, uobj.GetNamespace(), uobj.GetName()))

			replicas, found, err := unstructured.NestedInt64(uobj.UnstructuredContent(), "spec", "replicas")
			if err != nil {
				return errors.WithStack(err)
			}
			if !found {
				return fmt.Errorf("the resource doesn't have replcias field")
			}
			if replicas > 0 {
				logger.Info("Skipped the already running resource.")
				return nil
			}
			annotations := uobj.GetAnnotations()
			if annotations[annotationNameSkip] == "true" {
				logger.Info("Skipped the annotated resource.")
				return nil
			}

			var lastReplicas *int64
			for _, ref := range config.Status.LendingReferences {
				if ref.ObjectReference.APIVersion == target.APIVersion &&
					ref.ObjectReference.Kind == target.Kind &&
					ref.ObjectReference.Name == uobj.GetName() {
					lastReplicas = &ref.Replicas
					logger.Info(fmt.Sprintf("Found last replicas=%d.", ref.Replicas))
					break
				}
			}

			if lastReplicas == nil {
				if defaultReplicasStr, exists := annotations[annotationNameDefaultReplicas]; exists {
					defaultReplicas, err := strconv.ParseInt(defaultReplicasStr, 10, 64)
					if err != nil {
						return errors.WithStack(err)
					}
					logger.Info(fmt.Sprintf("Use per-resource default replicas=%d.", defaultReplicas))
					lastReplicas = &defaultReplicas
				}
			}

			if lastReplicas == nil {
				if target.DefaultReplicas != nil {
					defaultReplicas := *target.DefaultReplicas
					logger.Info(fmt.Sprintf("Use target default replicas=%d.", defaultReplicas))
					lastReplicas = &defaultReplicas
				}
			}

			if lastReplicas == nil {
				logger.Info("Skipped the unlended resource.")
				return nil
			}

			patch := makeReplicasPatch(uobj, groupVersionKind, int64(*lastReplicas))
			err = reconciler.Patch(ctx, patch, client.Apply, &client.PatchOptions{
				FieldManager: "application/apply-patch",
				Force:        pointer.Bool(true),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			logger.Info("Patched the resource.")
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (config *LendingConfig) DeactivateTargetResources(ctx context.Context, reconciler *LendingConfigReconciler) ([]clusterlendingmanagerv1alpha1.LendingReference, error) {
	logger := log.FromContext(ctx)

	lendingReferences := []clusterlendingmanagerv1alpha1.LendingReference{}

	for _, target := range config.Spec.Targets {
		groupVersionKind, err := getGroupVersionKind(target)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		objs := &unstructured.Unstructured{}
		objs.SetGroupVersionKind(groupVersionKind)

		err = reconciler.List(ctx, objs, &client.ListOptions{Namespace: config.Namespace})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		err = objs.EachListItem(func(obj runtime.Object) error {
			uobj := obj.(*unstructured.Unstructured)
			logger.Info(fmt.Sprintf("Patch %s %s/%s", groupVersionKind, uobj.GetNamespace(), uobj.GetName()))

			replicas, found, err := unstructured.NestedInt64(uobj.UnstructuredContent(), "spec", "replicas")
			if err != nil {
				return errors.WithStack(err)
			}
			if !found {
				return fmt.Errorf("the resource doesn't have replcias field")
			}
			if replicas == 0 {
				logger.Info("Skipped the already stopped resource.")
				return nil
			}
			annotations := uobj.GetAnnotations()
			if annotations[annotationNameSkip] == "true" {
				logger.Info("Skipped the annotated resource.")
				return nil
			}

			lendingReferences = append(lendingReferences, clusterlendingmanagerv1alpha1.LendingReference{
				ObjectReference: clusterlendingmanagerv1alpha1.ObjectReference{
					Name:       uobj.GetName(),
					APIVersion: target.APIVersion,
					Kind:       target.Kind,
				},
				Replicas: replicas,
			})
			logger.Info(fmt.Sprintf("Save replicas=%d.", replicas))

			patch := makeReplicasPatch(uobj, groupVersionKind, int64(0))
			err = reconciler.Patch(ctx, patch, client.Apply, &client.PatchOptions{
				FieldManager: "application/apply-patch",
				Force:        pointer.Bool(true),
			})
			if err != nil {
				return errors.WithStack(err)
			}
			logger.Info("Patched the resource.")
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return lendingReferences, nil
}

func parseHours(hours string) (int32, int32, error) {
	res := hoursPattern.FindStringSubmatch(strings.ToLower(hours))
	if len(res) != 4 {
		return 0, 0, errors.New("hours format is invalid")
	}
	hour, err := strconv.Atoi(res[1])
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}
	minute, err := strconv.Atoi(res[2])
	if err != nil {
		return 0, 0, errors.WithStack(err)
	}
	if res[3] == "pm" {
		hour += 12
	}
	return int32(hour), int32(minute), nil
}
