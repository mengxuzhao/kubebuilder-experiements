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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LongLivingPodSpec defines the desired state of LongLivingPod
type LongLivingPodSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
}

// LongLivingPodStatus defines the observed state of LongLivingPod.
type LongLivingPodStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// LongLivingPods is the list of currently observed long-living pods
	LongLivingPods []string `json:"longLivingPods,omitempty"`
	// Conditions represent the latest observations of the controller's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// LongLivingPod is the Schema for the longlivingpods API
type LongLivingPod struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of LongLivingPod
	// +required
	Spec LongLivingPodSpec `json:"spec"`

	// status defines the observed state of LongLivingPod
	// +optional
	Status LongLivingPodStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// LongLivingPodList contains a list of LongLivingPod
type LongLivingPodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LongLivingPod `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LongLivingPod{}, &LongLivingPodList{})
}
