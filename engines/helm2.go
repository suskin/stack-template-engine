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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/suskin/stack-template-engine/api/v1alpha1"
)

type Helm2EngineRunner struct {
	Log logr.Logger
}

const (
	spec = "spec"
)

// When a behavior executes, the resource engine is configured by the
// object which triggered the behavior. This method encapsulates the logic to
// create the resource engine configuration from the object's fields.
// TODO it seems as though a lot of the transformation logic is probably reusable
func (her *Helm2EngineRunner) CreateConfig(claim *unstructured.Unstructured, hc *v1alpha1.HookConfiguration) (*corev1.ConfigMap, error) {
	// yamlyamlyamlyamlyaml
	// TODO if spec is missing, this won't work very well
	s, ok := claim.Object[spec]

	if !ok {
		her.Log.V(0).Info("Spec not found on claim; not creating engine configuration", "claim", claim)
	}

	her.Log.V(0).Info("Converting configuration", "spec", s)
	configContents, err := yaml.Marshal(s)

	her.Log.V(0).Info("Configuration contents as yaml", "configContents", configContents)

	if err != nil {
		her.Log.Error(err, "Error marshaling claim spec as yaml!", "claim", claim)
		return nil, err
	}

	// Underneath, the yamler uses https://godoc.org/encoding/json#Marshal,
	// which means that the bytes are UTF-8 encoded
	// Theoretically we could get better performance by using a binary config
	// map, but having a string makes it better for humans who may want to observe
	// or troubleshoot behavior.
	stringConfigContents := string(configContents)

	// TODO get the engine type from the configuration
	engineType := hc.Engine.Type

	// TODO engine type should have a bit more structure;
	// probably better to use an enum type pattern, with an
	// engine name and its corresponding configuration file
	// name in the same object
	configKeyName := ""

	if engineType == "helm2" {
		configKeyName = "values.yaml"
	}

	configName := string(claim.GetUID())
	generatedMap, err := generateConfigMap(configName, configKeyName, stringConfigContents, her.Log)

	if err != nil {
		her.Log.V(0).Info("Error generating config map!", "claim", claim, "error", err)
		return nil, err
	}

	generatedMap.SetNamespace(claim.GetNamespace())

	her.Log.V(0).Info("Generated config map to pass engine configuration", "configMap", generatedMap)

	return generatedMap, err
}

func (her *Helm2EngineRunner) RunEngine(claim *unstructured.Unstructured, config *corev1.ConfigMap) error {
	return nil
}

func NewHelm2EngineRunner(log logr.Logger) *Helm2EngineRunner {
	return &Helm2EngineRunner{
		Log: log,
	}
}
