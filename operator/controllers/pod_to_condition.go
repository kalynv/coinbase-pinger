package controllers

import (
	"github.com/go-logr/logr"
	devorgv1 "github.com/kalynv/coinbase-pinger/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TypeLabel          string = "type"
	StatusLabel        string = "status"
	ReasonLabel        string = "reason"
	MessageAnnotation  string = "message"
	PingTimeAnnotation string = "ping-time"
)

func podToCondition(pod corev1.Pod, l logr.Logger) devorgv1.Condition {
	condition := devorgv1.Condition{}
	labels := pod.GetLabels()
	annotations := pod.GetAnnotations()
	if labels != nil {
		condition.Type = labels[TypeLabel]
		if labels[StatusLabel] == "true" {
			condition.Status = true
		}
		condition.Reason = labels[ReasonLabel]
	}
	if annotations != nil {
		condition.Message = annotations[MessageAnnotation]
		pingTime := annotations[PingTimeAnnotation]
		t := metav1.Time{}
		e := t.UnmarshalJSON([]byte(pingTime))
		if e != nil {
			l.Error(
				e,
				"Could not UnmarshalJSON ping-time annotation",
				"pod",
				pod.Name,
				"ping-time",
				pingTime,
			)
			t = metav1.Time{}
		}
		condition.PingTime = t
	}
	return condition
}

func podsToConditions(pods []corev1.Pod, l logr.Logger) []devorgv1.Condition {
	podsNumber := len(pods)
	if podsNumber == 0 {
		return nil
	}
	conditions := make([]devorgv1.Condition, 0, podsNumber)
	for _, pod := range pods {
		condition := podToCondition(pod, l)
		conditions = append(conditions, condition)
	}
	return conditions
}
