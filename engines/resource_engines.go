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
package engines

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/suskin/stack-template-engine/api/v1alpha1"
)

type ResourceEngineRunner interface {
	CreateConfig(
		claim *unstructured.Unstructured,
		hc *v1alpha1.HookConfiguration,
		// engine type

	) (*corev1.ConfigMap, error)
	// ) (string fileName, string fileContents)
	RunEngine(
		claim *unstructured.Unstructured,
		config *corev1.ConfigMap,
	) error // (result, error)
}