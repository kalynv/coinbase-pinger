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

package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	devorgv1 "github.com/kalynv/coinbase-pinger/operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// CoinbasePingerReconciler reconciles a CoinbasePinger object
type CoinbasePingerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=batch.dev.org,resources=coinbasepingers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.dev.org,resources=coinbasepingers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.dev.org,resources=coinbasepingers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Create, Update and Delete corresponding child CronJob resource to reflect
// the CoinbasePinger spec. Updates CoinbasePinger resource with ping results.
func (r *CoinbasePingerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	finalizer := "codepinger.dev.org/finalizer"
	l := log.FromContext(ctx)

	coinbasePinger := devorgv1.CoinbasePinger{}
	getCoinbasePingerErr := r.Get(ctx, req.NamespacedName, &coinbasePinger)
	if getCoinbasePingerErr != nil {
		l.Error(getCoinbasePingerErr, "unable to fetch CoinbasePinger")
		return reconcile.Result{}, client.IgnoreNotFound(getCoinbasePingerErr)
	}
	resourceUnderDeletion := !coinbasePinger.ObjectMeta.DeletionTimestamp.IsZero()

	cronJob, getCronJobErr := r.getCronJob(ctx, &coinbasePinger)
	cronJobNotFound := apierrors.IsNotFound(getCronJobErr)

	if cronJobNotFound && resourceUnderDeletion {
		controllerutil.RemoveFinalizer(&coinbasePinger, finalizer)
		err := r.Update(ctx, &coinbasePinger)
		return reconcile.Result{}, err
	}
	if cronJobNotFound && !resourceUnderDeletion {
		l.Info("CronJob for CoinbasePinger not found. Creating")
		cronJob := constructCronJob(coinbasePinger)
		createErr := r.Create(ctx, cronJob)
		requeue := false
		if createErr != nil {
			l.Error(createErr, "unable to create CronJob for CoinbasePinger")
			requeue = true
		}
		if apierrors.IsAlreadyExists(createErr) {
			requeue = false
		}
		return ctrl.Result{Requeue: requeue, RequeueAfter: time.Second * 10}, createErr
	}
	if getCronJobErr != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, getCronJobErr
	}
	if resourceUnderDeletion {
		if err := r.Delete(ctx, cronJob); err != nil {
			l.Error(err, "Could not delete CronJob")
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
		}
		controllerutil.RemoveFinalizer(&coinbasePinger, finalizer)
		err := r.Update(ctx, &coinbasePinger)
		return ctrl.Result{}, err
	}

	updatedCronJob := constructCronJob(coinbasePinger)
	if cronjobChanged(cronJob, updatedCronJob) {
		r.recreateCronJob(ctx, cronJob, updatedCronJob)
	}

	updateCoinbasePingerErr := r.updateCoinbasePingerStatus(ctx, coinbasePinger)
	if updateCoinbasePingerErr != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, updateCoinbasePingerErr
	}
	return ctrl.Result{}, nil
}

func (r *CoinbasePingerReconciler) getCronJob(
	ctx context.Context,
	pinger *devorgv1.CoinbasePinger,
) (*batchv1.CronJob, error) {
	cronJob := &batchv1.CronJob{}
	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      string(pinger.UID),
			Namespace: pinger.Namespace,
		},
		cronJob,
	)
	return cronJob, err
}

func (r *CoinbasePingerReconciler) recreateCronJob(
	ctx context.Context,
	oldCronJob *batchv1.CronJob,
	updatedCronJob *batchv1.CronJob,
) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info(
		"CoinbasePinger spec changed, recreating CronJob",
		"Old Cronjob name",
		oldCronJob.Name,
		"Old Cronjob UID",
		oldCronJob.UID,
		"Old Schedule",
		oldCronJob.Spec.Schedule,
		"New Schedule",
		updatedCronJob.Spec.Schedule,
	)
	if err := r.Delete(ctx, oldCronJob); err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}
	if err := r.Create(ctx, updatedCronJob); err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 10}, err
	}
	l.Info("recreated cronjob", "Schedule", updatedCronJob.Spec.Schedule)
	return ctrl.Result{}, nil
}

func (r *CoinbasePingerReconciler) updateCoinbasePingerStatus(
	ctx context.Context,
	pinger devorgv1.CoinbasePinger,
) error {
	l := log.FromContext(ctx)
	pods, getPodsErr := r.getOwnPods(ctx, pinger)
	if getPodsErr != nil {
		return getPodsErr
	}

	conditions := podsToConditions(pods, log.FromContext(ctx))
	if len(conditions) == 0 {
		l.Info("status is empty")
		return nil
	}

	l.Info("updating status", "Conditions", conditions)

	updatedCodebasePinger := pinger.DeepCopy()
	updatedCodebasePinger.Status.Conditions = conditions
	updateErr := r.Status().Update(ctx, updatedCodebasePinger)

	return updateErr
}

func (r *CoinbasePingerReconciler) getOwnPods(
	ctx context.Context,
	pinger devorgv1.CoinbasePinger,
) ([]corev1.Pod, error) {
	requirement, _ := labels.NewRequirement(
		CRD_UID,
		selection.Equals,
		[]string{string(pinger.UID)},
	)
	selector := labels.NewSelector().Add(*requirement)

	list := corev1.PodList{}
	err := r.List(
		ctx,
		&list,
		&client.ListOptions{
			LabelSelector: selector,
			Namespace:     pinger.Namespace,
		},
	)

	return list.Items, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *CoinbasePingerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devorgv1.CoinbasePinger{}).
		Watches(
			&source.Kind{Type: &corev1.Pod{}},
			handler.EnqueueRequestsFromMapFunc(
				func(obj client.Object) []reconcile.Request {
					labels := obj.GetLabels()
					name, namePresent := labels[CRD_NAME]
					if !namePresent {
						return nil
					}
					namespace, namespacePresent := labels[CRD_NAMESPACE]
					if !namespacePresent {
						return nil
					}
					return []reconcile.Request{
						reconcile.Request{
							NamespacedName: types.NamespacedName{
								Name:      name,
								Namespace: namespace,
							},
						},
					}
				},
			),
			builder.WithPredicates(
				&predicate.Funcs{
					UpdateFunc: func(event.UpdateEvent) bool {
						return true
					},
				},
			),
		).
		Complete(r)
}
