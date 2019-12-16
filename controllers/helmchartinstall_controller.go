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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/crossplaneio/crossplane-runtime/pkg/meta"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/util/hash"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/suskin/stack-template-engine/api/v1alpha1"
	helmv1alpha1 "github.com/suskin/stack-template-engine/api/v1alpha1"
)

// HelmChartInstallReconciler reconciles a HelmChartInstall object
type HelmChartInstallReconciler struct {
	client.Client
	Log logr.Logger
}

const (
	timeout = 60 * time.Second
)

// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=helmchartinstalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=helmchartinstalls/status,verbs=get;update;patch

func (r *HelmChartInstallReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// TODO NOTE the group, version, and kind would normally come from the
	// stack configuration, and would be part of the configuration for the render
	// callback
	//
	// We grab the claim as an unstructured so that we can have the same code handle
	// arbitrary claim types. The types will be erased by this point, so if a stack
	// author wants to validate the schema of a claim, they can do it by putting a
	// schema in the CRD of the claim type so that the claim's schema is validated
	// at the time that the object is accepted by the api server.
	i := &unstructured.Unstructured{}
	i.SetGroupVersionKind(
		schema.GroupVersionKind{
			Group:   "helm.samples.stacks.crossplane.io",
			Version: "v1alpha1",
			Kind:    "HelmChartInstall",
		},
	)
	if err := r.Client.Get(ctx, req.NamespacedName, i); err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.V(0).Info("Hello World!", "instanceName", req.NamespacedName, "instance", i)

	// FIXME TODO we are conflating the setup and render phases here, because originally
	// I was writing this controller as an experiment for a controller which only supported
	// rendering.
	//
	// Here's what we want in the long run, for a controller which supports both:
	// - Setup phase: reads configuration, and sets up event handlers. An event handler has:
	//   * A target CRD GVK (specified by the stack configuration for the behavior)
	//   * A watcher (the controller would create this, to watch CRDs of the desired type)
	//   * A set of hooks (specified by the stack configuration for the behavior). Each hook
	//     takes in a set of configuration and the triggering CRD, and outputs some K8s resources
	// - Render phase: this is what happens when a hook is triggered, and the hook runs
	//   * Inputs:
	//     ^ A claim, which is the object which triggered the render
	//     ^ A behavior, which has all of the configuration for what should happen now
	//       that the render has been triggered
	//   * Outputs:
	//     ^ Changes to new or existing K8s resources
	//     ^ The result of the render (probably as the status of the claim)
	//       + The job which was run
	//       + The result of the job
	//       + The logs of the job (maybe; it might be too much)

	// TODO remaining functionality for the render phase:
	// - Surface the job result somehow (for usability)
	// - Support for deletion, unless we assume that garbage collection will do it for us
	r.render(ctx, i)

	return ctrl.Result{}, nil
}

func (r *HelmChartInstallReconciler) setup(ctx context.Context, stack *helmv1alpha1.HelmChartInstall) error {
	return nil
}

func (r *HelmChartInstallReconciler) render(ctx context.Context, claim *unstructured.Unstructured) error {
	/**
	* Steps to rendering:
	- Grab the original claim
	- Load the stack's stack.yaml configuration
	- From the stack.yaml, load the stack's configuration to be processed
	- Process the configuration as specified
	- Grab the output / result, and Apply the output or report the failure
	- Report the result to the status of the original claim

	Other steps:
	- Watch for new types being provided by stacks
	- When there is a new type provided, watch for instances being created
	- When an instance is created, feed it into the renderer

	TODO This could use a bit of refactoring so it flows more nicely
	*/

	// configuration is a typed object
	configuration, err := r.getStackConfiguration(ctx, claim)
	// TODO check for errors

	targetResourceBehavior, err := r.getBehavior(ctx, claim, configuration)

	if targetResourceBehavior == nil {
		// TODO error condition with a real error returned
		r.Log.V(0).Info("Couldn't find a configured behavior!",
			"claim", claim,
			"configuration", configuration,
		)
		return err
	}

	engineConfig, err := r.createBehaviorEngineConfiguration(ctx, claim, configuration)

	if err != nil {
		r.Log.Error(err, "Error creating engine configuration!", "claim", claim)
		return err
	}

	targetStackImage := configuration.Spec.Source.Image

	result, err := r.executeBehavior(ctx, claim, engineConfig, targetStackImage, targetResourceBehavior)
	// TODO check for errors

	err = r.setClaimStatus(claim, result)

	return err
}

func (r *HelmChartInstallReconciler) getStackConfiguration(
	ctx context.Context,
	claim *unstructured.Unstructured,
) (*v1alpha1.StackConfiguration, error) {
	// See the template stacks internal design doc for details, but
	// the most likely source of the stack configuration is the stack object itself.
	// Other potential sources include a configmap

	// TODO
	// - Stack configuration will be coming from a Kubernetes object, probably the
	//   Stack itself
	namespacedName, err := client.ObjectKeyFromObject(claim)

	if err != nil {
		r.Log.V(0).Info("getStackConfiguration returning early because of error creating object key", "err", err, "claim", claim)
		return nil, err
	}

	configuration := &v1alpha1.StackConfiguration{}
	if err := r.Client.Get(ctx, namespacedName, configuration); err != nil {
		r.Log.V(0).Info("getStackConfiguration returning early because of error fetching configuration", "err", err, "claim", claim)
		return nil, err
	}

	r.Log.V(0).Info("getStackConfiguration returning configuration", "configuration", configuration)
	return configuration, nil
}

/**
 * When a behavior is triggered, we want to know which behavior exactly we are executing.
 *
 * In most cases, this will probably be configured ahead of time by the setup controller, rather
 * than being fetched at runtime by the render controller.
 */
func (r *HelmChartInstallReconciler) getBehavior(
	ctx context.Context,
	claim *unstructured.Unstructured,
	configuration *v1alpha1.StackConfiguration,
) (*v1alpha1.StackConfigurationBehavior, error) {
	targetResourceGroupVersion, targetResourceKind := claim.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()
	targetGroupKindVersion := fmt.Sprintf("%s.%s", targetResourceKind, targetResourceGroupVersion)

	var targetResourceBehavior v1alpha1.StackConfigurationBehavior
	// TODO handle missing keys gracefully
	targetResourceBehavior, ok := configuration.Spec.Behaviors.CRDs[targetGroupKindVersion]

	if !ok {
		// TODO error condition with a real error returned
		r.Log.V(0).Info("Couldn't find a configured behavior!",
			"claim", claim,
			"configuration", configuration,
			"targetGroupKindVersion", targetGroupKindVersion,
		)
		return nil, nil

	}

	if len(targetResourceBehavior.Resources) == 0 {
		// TODO error condition with a real error returned
		// TODO it'd be nice to enforce this on acceptance or creation if possible
		r.Log.V(0).Info("Couldn't find resources for configured behavior!", "claim", claim, "configuration", configuration)
		return nil, nil

	}

	return &targetResourceBehavior, nil
}

/**
 * When a behavior executes, the resource engine is configured by the
 * object which triggered the behavior. This method encapsulates the logic to
 * create the resource engine configuration from the object's fields.
 *
 * TODO Currently creation fails if the config map already exists, but it should
 * succeed instead.
 */
func (r *HelmChartInstallReconciler) createBehaviorEngineConfiguration(
	ctx context.Context,
	claim *unstructured.Unstructured,
	stackConfig *v1alpha1.StackConfiguration,
) (*corev1.ConfigMap, error) {
	// yamlyamlyamlyamlyaml
	// TODO if spec is missing, that won't work very well
	spec, ok := claim.Object["spec"]

	if !ok {
		r.Log.V(0).Info("Spec not found on claim; not creating engine configuration", "claim", claim)
	}

	r.Log.V(0).Info("Converting configuration", "spec", spec)
	configContents, err := yaml.Marshal(spec)

	r.Log.V(0).Info("Configuration contents as yaml", "configContents", configContents)

	if err != nil {
		r.Log.Error(err, "Error marshaling claim spec as yaml!", "claim", claim)
		return nil, err
	}

	// Underneath, the yamler uses https://godoc.org/encoding/json#Marshal,
	// which means that the bytes are UTF-8 encoded
	// Theoretically we could get better performance by using a binary config
	// map, but having a string makes it better for humans who may want to observe
	// or troubleshoot behavior.
	stringConfigContents := string(configContents)

	// TODO get the engine type from the configuration
	engineType := r.getEngineType(claim, stackConfig)

	// TODO engine type should have a bit more structure;
	// probably better to use an enum type pattern, with an
	// engine name and its corresponding configuration file
	// name in the same object
	configKeyName := ""

	if engineType == "helm2" {
		configKeyName = "values.yaml"
	}

	configName := string(claim.GetUID())
	generatedMap, err := r.generateConfigMap(configName, configKeyName, stringConfigContents)

	if err != nil {
		r.Log.V(0).Info("Error generating config map!", "claim", claim, "error", err)
		return nil, err
	}

	generatedMap.SetNamespace(claim.GetNamespace())

	r.Log.V(0).Info("Generated config map to pass engine configuration", "configMap", generatedMap)

	err = r.Client.Create(ctx, generatedMap)

	if err != nil {
		r.Log.V(0).Info("Error creating config map!", "claim", claim, "error", err, "configMap", generatedMap)
		return nil, err
	}

	return generatedMap, err
}

// The main reason this exists as its own method is to encapsulate the hashing logic
func (r *HelmChartInstallReconciler) generateConfigMap(name string, fileName string, fileContents string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	configMap.Name = name
	configMap.Data = map[string]string{}

	configMap.Data[fileName] = fileContents
	h, err := hash.ConfigMapHash(configMap)
	if err != nil {
		r.Log.V(0).Info("Error hashing config map!", "error", err)
		return configMap, err
	}
	configMap.Name = fmt.Sprintf("%s-%s", configMap.Name, h)

	return configMap, nil
}

/**
 * This is in its own method because it will involve
 * reading through multiple levels of the configuration to see
 * what the engine type is. It may even involve inferring an engine
 * type based on the stack contents (though that may happen earlier
 * in the lifecycle than this).
 */
func (r *HelmChartInstallReconciler) getEngineType(
	claim *unstructured.Unstructured,
	stackConfig *v1alpha1.StackConfiguration,
) string {
	return "helm2"
}

func (r *HelmChartInstallReconciler) executeBehavior(
	ctx context.Context,
	claim *unstructured.Unstructured,
	engineConfiguration *corev1.ConfigMap,
	targetStackImage string,
	behavior *v1alpha1.StackConfigurationBehavior,
) (*unstructured.Unstructured, error) {
	for _, resource := range behavior.Resources {
		// TODO error handling
		// TODO use result
		r.executeHook(ctx, claim, engineConfiguration, targetStackImage, resource)
	}
	// TODO return a real result
	return &unstructured.Unstructured{}, nil
}

// TODO we could have a method create the job, and a higher-level one execute it.
func (r *HelmChartInstallReconciler) executeHook(
	ctx context.Context,
	claim *unstructured.Unstructured,
	engineConfig *corev1.ConfigMap,
	targetStackImage string,
	targetResourceDir string,
) (*unstructured.Unstructured, error) {
	// TODO if there is no config specified, either use an empty config or don't specify
	// one at all.

	ownerRef := meta.AsOwner(meta.ReferenceTo(claim, claim.GroupVersionKind()))
	var jobBackoff int32
	jobBackoff = 0

	// TODO target stack image will come from the stack object, or maybe the stack install object.
	// Then for each resource behavior hook, we want to run the hook
	// TODO update this to use the most recent format, where a hook is a structured object

	resourceDirName := fmt.Sprintf("/.registry/resources/%s", targetResourceDir)

	engineConfigurationVolumeName := "engine-configuration"
	engineConfigurationDir := "/usr/share/engine-configuration/"
	engineConfigurationFile := fmt.Sprintf("%svalues.yaml", engineConfigurationDir)

	stackConfigurationVolumeName := "stack-configuration"
	stackContentsDestinationDir := "/usr/share/input/"

	resourceConfigurationVolumeName := "resource-configuration"
	resourceConfigurationDestinationDir := "/usr/share/resource-configuration/"
	targetNamespace := claim.GetNamespace()

	// TODO we should generate a name and save a reference on the claim
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "helm-template-apply-",
			Namespace:    targetNamespace,
			OwnerReferences: []metav1.OwnerReference{
				ownerRef,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &jobBackoff,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:  "load-stack",
							Image: targetStackImage,
							Command: []string{
								// The "." suffix causes the cp -R to copy the contents of the directory instead of
								// the directory itself
								"cp", "-R", fmt.Sprintf("%s/.", resourceDirName), stackContentsDestinationDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      stackConfigurationVolumeName,
									MountPath: stackContentsDestinationDir,
								},
							},
							ImagePullPolicy: corev1.PullNever,
						},
						{
							Name:  "engine",
							Image: "crossplane/helm-engine:latest",
							Command: []string{
								"helm",
							},
							Args: []string{
								"template",
								"--output-dir", resourceConfigurationDestinationDir,
								"--namespace", targetNamespace,
								"--values", engineConfigurationFile,
								stackContentsDestinationDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      stackConfigurationVolumeName,
									MountPath: stackContentsDestinationDir,
								},
								{
									Name:      resourceConfigurationVolumeName,
									MountPath: resourceConfigurationDestinationDir,
								},
								{
									Name:      engineConfigurationVolumeName,
									MountPath: engineConfigurationDir,
								},
							},
							ImagePullPolicy: corev1.PullNever,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "kubectl",
							Image: "crossplane/kubectl:latest",
							// "--debug" can be added to this list of Args to get debug output from the job,
							// but note that will be included in the stdout from the pod, which makes it
							// impossible to create the resources that the job unpacks.
							Command: []string{
								"kubectl",
							},
							Args: []string{
								"apply",
								"--namespace", targetNamespace,
								"-R",
								"-f",
								resourceConfigurationDestinationDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      resourceConfigurationVolumeName,
									MountPath: resourceConfigurationDestinationDir,
								},
							},
							ImagePullPolicy: corev1.PullNever,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: stackConfigurationVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: resourceConfigurationVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: engineConfigurationVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: engineConfig.GetName(),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := r.Client.Create(ctx, job); err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{}, nil
}

func (r *HelmChartInstallReconciler) setClaimStatus(
	claim *unstructured.Unstructured, result *unstructured.Unstructured,
) error {
	// The claim is the CR that triggered this whole thing.
	// The result is the result of trying to apply or delete the templates rendered from processing the claim.
	// If the processing happens in a job, the status should be updated after the job completes, which may mean
	// waiting for it to complete by using a watcher.
	return nil
}

func (r *HelmChartInstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.HelmChartInstall{}).
		Complete(r)
}
