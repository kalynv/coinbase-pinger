package controllers

import (
	batchv1 "k8s.io/api/batch/v1"
)

func cronjobChanged(current, constructed *batchv1.CronJob) bool {
	return current.Spec.Schedule != constructed.Spec.Schedule
}
