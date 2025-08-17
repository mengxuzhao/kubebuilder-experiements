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
	"crypto/sha256"
	"encoding/hex"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cfg2deployv1 "meng.xu/config-to-deploy/api/v1"
)

// ConfigDeploymentReconciler reconciles a ConfigDeployment object
type ConfigDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Compute a hash of the ConfigMap data
func hashConfigMap(data map[string]string) string {
	h := sha256.New()
	for k, v := range data {
		h.Write([]byte(k + v))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// +kubebuilder:rbac:groups=cfg2deploy.meng.xu,resources=configdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cfg2deploy.meng.xu,resources=configdeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cfg2deploy.meng.xu,resources=configdeployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ConfigDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// 1. Fetch the ConfigDeployment instance
	configDeployment := &cfg2deployv1.ConfigDeployment{}
	if err := r.Get(ctx, req.NamespacedName, configDeployment); err != nil {
		logf.FromContext(ctx).Error(err, "unable to fetch ConfigDeployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Initialize the ConfigMap data
	data := map[string]string{
		"greetingMsg": configDeployment.Spec.ConfigGreetingMsg,
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configDeployment.Spec.DeployName + "-config",
			Namespace: configDeployment.Spec.DeployNamespace,
		},
		Data: data,
	}

	// 3. Set the owner reference to the ConfigDeployment
	if err := ctrl.SetControllerReference(configDeployment, configMap, r.Scheme); err != nil {
		logf.FromContext(ctx).Error(err, "unable to set controller reference for ConfigMap")
		return ctrl.Result{}, err
	}

	// 4. Check if the ConfigMap already exists
	existingConfigMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(configMap), existingConfigMap); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logf.FromContext(ctx).Error(err, "unable to fetch ConfigMap")
			return ctrl.Result{}, err
		}

		// 5. Create the ConfigMap if it does not exist
		if err := r.Create(ctx, configMap); err != nil {
			logf.FromContext(ctx).Error(err, "unable to create ConfigMap")
			return ctrl.Result{}, err
		}
	} else {
		// 6. Update the existing ConfigMap with new data
		if configMap.Data["greetingMsg"] != existingConfigMap.Data["greetingMsg"] {
			logf.FromContext(ctx).Info("Updating ConfigMap with new greeting message")
			existingConfigMap.Data = data
			if err := r.Update(ctx, existingConfigMap); err != nil {
				logf.FromContext(ctx).Error(err, "unable to update ConfigMap")
				return ctrl.Result{}, err
			}
			logf.FromContext(ctx).Info("ConfigMap updated successfully")
		}
	}

	// 7. Create or update the Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configDeployment.Spec.DeployName + "-deploy",
			Namespace: configDeployment.Spec.DeployNamespace,
			Labels: map[string]string{
				"app": configDeployment.Spec.DeployName,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &configDeployment.Spec.DeploySize,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": configDeployment.Spec.DeployName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": configDeployment.Spec.DeployName,
					},
					Annotations: map[string]string{
						"configmap-hash": hashConfigMap(configMap.Data),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  configDeployment.Spec.DeployName + "-container",
							Image: configDeployment.Spec.DeployImage,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "greeting-volume",
									MountPath: "/usr/local/apache2/htdocs/index.html",
									SubPath:   "index.html",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "greeting-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMap.Name,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "greetingMsg",
											Path: "index.html",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// 8. Set the owner reference to the ConfigDeployment
	if err := ctrl.SetControllerReference(configDeployment, deployment, r.Scheme); err != nil {
		logf.FromContext(ctx).Error(err, "unable to set controller reference for Deployment")
		return ctrl.Result{}, err
	}

	// 9. Check if the Deployment already exists
	existingDeployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(deployment), existingDeployment); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logf.FromContext(ctx).Error(err, "unable to fetch Deployment")
			return ctrl.Result{}, err
		}

		// 10. Create the Deployment if it does not exist
		if err := r.Create(ctx, deployment); err != nil {
			logf.FromContext(ctx).Error(err, "unable to create Deployment")
			return ctrl.Result{}, err
		}
	} else {
		// 11. Update the existing Deployment with new configuration
		if deployment.Spec.Replicas != existingDeployment.Spec.Replicas ||
			deployment.Spec.Template.Spec.Containers[0].Image != existingDeployment.Spec.Template.Spec.Containers[0].Image {
			logf.FromContext(ctx).Info("Updating Deployment with new configuration")
			existingDeployment.Spec = deployment.Spec
			if err := r.Update(ctx, existingDeployment); err != nil {
				logf.FromContext(ctx).Error(err, "unable to update Deployment")
				return ctrl.Result{}, err
			}
			logf.FromContext(ctx).Info("Deployment updated successfully")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cfg2deployv1.ConfigDeployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
