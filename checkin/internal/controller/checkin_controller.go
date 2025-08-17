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

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	trackerv1alpha1 "meng.xu/checkin/api/v1alpha1"
)

// CheckInReconciler reconciles a CheckIn object
type CheckInReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tracker.meng.xu,resources=checkins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tracker.meng.xu,resources=checkins/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tracker.meng.xu,resources=checkins/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CheckIn object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *CheckInReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// 1. Fetch the CheckIn instance
	checkin := &trackerv1alpha1.CheckIn{}
	if err := r.Get(ctx, req.NamespacedName, checkin); err != nil {
		logf.FromContext(ctx).Error(err, "unable to fetch CheckIn")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Ensure the Pod exists
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      checkin.Name + "-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "checkin",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "checkin-container",
					Image: checkin.Spec.PodImage,
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	// 3. Set CheckIn instance as the owner of the Pod
	if err := ctrl.SetControllerReference(checkin, pod, r.Scheme); err != nil {
		logf.FromContext(ctx).Error(err, "unable to set controller reference for Pod")
		return ctrl.Result{}, err
	}

	// 4. Check if the Pod already exists
	existingPod := &corev1.Pod{}
	err := r.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: pod.Namespace}, existingPod)
	isNewPod := false
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			logf.FromContext(ctx).Error(err, "unable to fetch Pod")
			return ctrl.Result{}, err
		}
		// Pod does not exist, create it
		isNewPod = true
		logf.FromContext(ctx).Info("Creating Pod for CheckIn", "name", pod.Name)
		if err := r.Create(ctx, pod); err != nil {
			logf.FromContext(ctx).Error(err, "unable to create Pod")
			return ctrl.Result{}, err
		}
		existingPod = pod
	}

	// 5. Track the active Pod ID
	checkin.Status.ActivePod = string(existingPod.UID)
	if checkin.Status.PodHistory == nil {
		checkin.Status.PodHistory = []string{}
	}
	if isNewPod {
		checkin.Status.PodHistory = append(checkin.Status.PodHistory, string(existingPod.UID))
		// 6. Update CheckIn status to indicate completion
		apimeta.SetStatusCondition(&checkin.Status.Conditions, metav1.Condition{
			Type:    trackerv1alpha1.CheckInCompleted,
			Status:  metav1.ConditionTrue,
			Reason:  "PodRunning",
			Message: "Checked in successfully with Pod " + string(existingPod.UID),
		})
	}

	if err := r.Status().Update(ctx, checkin); err != nil {
		logf.FromContext(ctx).Error(err, "unable to update CheckIn status")
		return ctrl.Result{}, err
	}

	logf.FromContext(ctx).Info("CheckIn reconciled successfully", "name", checkin.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CheckInReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&trackerv1alpha1.CheckIn{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
