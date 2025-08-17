/*
Copyright 2025.

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

package controller

import (
	"context"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	trackerv1alpha1 "meng.xu/checkin/api/v1alpha1"
)

// LongLivingPodReconciler reconciles a LongLivingPod object
type LongLivingPodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tracker.meng.xu,resources=longlivingpods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tracker.meng.xu,resources=longlivingpods/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tracker.meng.xu,resources=longlivingpods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LongLivingPod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *LongLivingPodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// 1. Fetch the LongLivingPod instance
	longLivingPod := &trackerv1alpha1.LongLivingPod{}
	if err := r.Get(ctx, req.NamespacedName, longLivingPod); err != nil {
		logf.FromContext(ctx).Error(err, "unable to fetch LongLivingPod")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Fetch the CheckIn resource
	var checkin trackerv1alpha1.CheckIn
	if err := r.Get(ctx, types.NamespacedName{
		Name:      "checkin-rsc",
		Namespace: "default",
	}, &checkin); err != nil {
		logf.FromContext(ctx).Error(err, "unable to fetch CheckIn")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 3. Fetch the active Pod from the CheckIn status
	if checkin.Status.ActivePod != "" {
		activePod := &corev1.Pod{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      "checkin-rsc-pod",
			Namespace: "default",
		}, activePod); err != nil {
			logf.FromContext(ctx).Error(err, "unable to fetch active Pod from CheckIn status")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		// 4. Check if the Pod has been runing for more than 2 minutes
		if activePod.Status.StartTime != nil {
			runningDuration := time.Since(activePod.Status.StartTime.Time)
			if runningDuration > 2*time.Minute {
				// 5. If the Pod has been running for more than 2 minutes and not already present, add it to the LongLivingPods list
				exists := slices.Contains(longLivingPod.Status.LongLivingPods, string(activePod.UID))
				if !exists {
					longLivingPod.Status.LongLivingPods = append(longLivingPod.Status.LongLivingPods, string(activePod.UID))
					// 6. Update the status of LongLivingPod
					if err := r.Status().Update(ctx, longLivingPod); err != nil {
						logf.FromContext(ctx).Error(err, "unable to update LongLivingPod status")
						return ctrl.Result{}, err
					}
				}
			}
		}
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LongLivingPodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trackerv1alpha1.LongLivingPod{}).
		Named("longlivingpod").
		Complete(r)
}
