package controllers

import (
	"fmt"
	"time"

	devorgv1 "github.com/kalynv/coinbase-pinger/operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CRD_UID       string = "webapp-pinger"
	CRD_NAME      string = "notify-name"
	CRD_NAMESPACE string = "notify-namespace"
)

func constructCronJob(pinger devorgv1.CoinbasePinger) *batchv1.CronJob {
	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(pinger.UID),
			Namespace: pinger.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: pinger.APIVersion,
					Kind:       pinger.Kind,
					Name:       pinger.Name,
					UID:        pinger.UID,
				},
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          intervalToCrontabSchedule(pinger.Spec.Interval),
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								CRD_UID:       string(pinger.UID),
								CRD_NAME:      pinger.Name,
								CRD_NAMESPACE: pinger.Namespace,
							},
						},
						Spec: *constructPodSpec(pinger),
					},
				},
			},
		},
	}
	return cronjob
}

// TODO crd CoinbasePinger should contain desired PodSpec, so hardcoded values
// should be replaced with values from CoinbasePinger spec or default one.
func constructPodSpec(_pinger devorgv1.CoinbasePinger) *v1.PodSpec {
	return &v1.PodSpec{
		ServiceAccountName: "web-pinger-sa",
		RestartPolicy:      v1.RestartPolicyNever,
		Containers: []v1.Container{
			v1.Container{
				Name:    "pinger",
				Image:   "kalynv/webapp-pinger",
				Command: []string{"/webping"},
				Args:    []string{"/prices/BTC-USD/buy"},
				Env: []v1.EnvVar{
					v1.EnvVar{
						Name:  "BASE_URL",
						Value: "https://api.coinbase.com/v2",
					},
				},
				VolumeMounts: []v1.VolumeMount{
					v1.VolumeMount{
						Name:      "podinfo",
						ReadOnly:  true,
						MountPath: "/etc/podinfo",
					},
				},
			},
		},
		Volumes: []v1.Volume{
			v1.Volume{
				Name: "podinfo",
				VolumeSource: v1.VolumeSource{
					DownwardAPI: &v1.DownwardAPIVolumeSource{
						Items: []v1.DownwardAPIVolumeFile{
							v1.DownwardAPIVolumeFile{
								Path: "namespace",
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
							v1.DownwardAPIVolumeFile{
								Path: "name",
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
					},
				},
			},
		},
	}
}

func intervalToCrontabSchedule(interval string) (schedule string) {
	duration, err := time.ParseDuration(interval)
	if err != nil {
		// duration must be parsable
		panic(err.Error())
	}
	minutes := int(duration.Minutes())
	if minutes < 1 {
		panic(fmt.Sprintf("Bad duration, must be at least a minute, but got %d minute", minutes))
	}
	if minutes < 60 {
		return fmt.Sprintf("*/%d * * * *", minutes)
	}
	hours := int(duration.Hours())
	if hours < 24 {
		return fmt.Sprintf("* */%d * * *", hours)
	}
	// duration must be less than 24 hours
	panic(fmt.Sprintf("Bad duration, must be less than 24 hours, but got %d hours", hours))
}
