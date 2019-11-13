/*

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

// StackConfigurationSpec defines the desired state of StackConfiguration
type StackConfigurationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Behaviors StackConfigurationBehaviors `json:"behaviors,omitempty"`
}

// StackConfigurationBehaviors specifies behaviors for the stack
// Strings should be in GroupVersion format, so Kind.group
type StackConfigurationBehaviors struct {
	CRDs map[string]StackConfigurationBehavior `json:"crds,omitempty"`
}

// StackConfigurationBehavior specifies an individual behavior, by listing resources
// which should be processed.
type StackConfigurationBehavior struct {
	Resources []string `json:"resources"`
}

// StackConfigurationStatus defines the observed state of StackConfiguration
type StackConfigurationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// StackConfiguration is the Schema for the stackconfigurations API
type StackConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StackConfigurationSpec   `json:"spec,omitempty"`
	Status StackConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StackConfigurationList contains a list of StackConfiguration
type StackConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StackConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StackConfiguration{}, &StackConfigurationList{})
}
