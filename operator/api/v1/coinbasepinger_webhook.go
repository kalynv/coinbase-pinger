/*
Copyright 2021.

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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type crdConstError string

func (s crdConstError) Error() string {
	return string(s)
}

const (
	NonParsableInterval crdConstError = crdConstError("Non parsable interval. Must consist of decimal number and unit suffix s, m or h")
	OutOfRangeInterval  crdConstError = crdConstError("Must be greater than 1 minute and less then 24 hours")
	PingPath            crdConstError = crdConstError("Wrong web ping path")
)

// log is for logging in this package.
var coinbasepingerlog = logf.Log.WithName("coinbasepinger-resource")

func (r *CoinbasePinger) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-batch-dev-org-v1-coinbasepinger,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.dev.org,resources=coinbasepingers,verbs=create;update,versions=v1,name=vcoinbasepinger.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &CoinbasePinger{}

func (r *CoinbasePinger) validateInterval() error {
	duration, err := time.ParseDuration(r.Spec.Interval)
	if err != nil {
		return NonParsableInterval
	}
	if duration < time.Minute {
		return OutOfRangeInterval
	}
	if duration >= (time.Hour * 24) {
		return OutOfRangeInterval
	}
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *CoinbasePinger) ValidateCreate() error {
	coinbasepingerlog.Info("validate create", "name", r.Name)
	return r.validateInterval()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *CoinbasePinger) ValidateUpdate(old runtime.Object) error {
	coinbasepingerlog.Info("validate update", "name", r.Name)
	return r.validateInterval()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *CoinbasePinger) ValidateDelete() error {
	// coinbasepingerlog.Info("validate delete", "name", r.Name)
	return nil
}
