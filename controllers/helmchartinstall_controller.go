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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/suskin/stack-template-engine/api/v1alpha1"
	helmv1alpha1 "github.com/suskin/stack-template-engine/api/v1alpha1"

	"github.com/crossplaneio/crossplane-runtime/pkg/meta"
	"sigs.k8s.io/yaml"

	//generate "k8s.io/kubectl/pkg/generate/versioned"
	"k8s.io/kubectl/pkg/util/hash"
)

// HelmChartInstallReconciler reconciles a HelmChartInstall object
type HelmChartInstallReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=helmchartinstalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=helmchartinstalls/status,verbs=get;update;patch

func (r *HelmChartInstallReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("helmchartinstall", req.NamespacedName)

	// your logic here
	i := &helmv1alpha1.HelmChartInstall{}
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
	// - Inject values from a claim
	//   * Potential implementation: turn the values into a config map, then inject it into
	//     the configuration engine job using a config map volume mapping
	// - Surface the job result somehow (for usability)
	// - Support for deletion, unless we assume that garbage collection will do it for us
	r.render(ctx, i)

	return ctrl.Result{}, nil
}

func (r *HelmChartInstallReconciler) setup(ctx context.Context, stack *helmv1alpha1.HelmChartInstall) error {
	return nil
}

func (r *HelmChartInstallReconciler) render(ctx context.Context, claim *helmv1alpha1.HelmChartInstall) error {
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

	// TODO the status will have more than *just* the result in it,
	// but we can start with just the result
	result, err := r.processConfiguration(ctx, claim, configuration)

	err = r.setClaimStatus(claim, result)

	return err
}

func (r *HelmChartInstallReconciler) getClaim() (unstructured.Unstructured, error) {
	// The claim is the CR that triggered this whole thing.
	return unstructured.Unstructured{}, nil
}

func (r *HelmChartInstallReconciler) getStackConfiguration(
	ctx context.Context,
	claim *v1alpha1.HelmChartInstall,
) (v1alpha1.StackConfiguration, error) {
	// See the template stacks internal design doc for details, but
	// the most likely source of the stack configuration is the stack object itself.
	// Other potential sources include a configmap

	// TODO
	// - Stack configuration will be coming from a Kubernetes object, probably the
	//   Stack itself
	namespacedName, err := client.ObjectKeyFromObject(claim)

	if err != nil {
		r.Log.V(0).Info("getStackConfiguration returning early because of error creating object key", "err", err, "claim", claim)
		return v1alpha1.StackConfiguration{}, err
	}

	configuration := v1alpha1.StackConfiguration{}
	if err := r.Client.Get(ctx, namespacedName, &configuration); err != nil {
		r.Log.V(0).Info("getStackConfiguration returning early because of error fetching configuration", "err", err, "claim", claim)
		return v1alpha1.StackConfiguration{}, err
	}

	r.Log.V(0).Info("getStackConfiguration returning configuration", "configuration", configuration)
	return configuration, nil
}

func (r *HelmChartInstallReconciler) processConfiguration(
	ctx context.Context,
	claim *helmv1alpha1.HelmChartInstall,
	configuration v1alpha1.StackConfiguration,
) (unstructured.Unstructured, error) {
	// Given a claim and a stack configuration, transform the claim into something
	// to inject into a stack's configuration renderer, and then render the Stack's
	// resources from the configuration and the transformed claim

	// TODO we may want some of these steps to be in this controller instead of
	//      being outsourced

	// TODO for each CRD, set up some watches, configured to trigger the configured hooks
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
		return unstructured.Unstructured{}, nil

	}

	if len(targetResourceBehavior.Resources) == 0 {
		// TODO error condition with a real error returned
		// TODO it'd be nice to enforce this on acceptance or creation if possible
		r.Log.V(0).Info("Couldn't find resources for configured behavior!", "claim", claim, "configuration", configuration)
		return unstructured.Unstructured{}, nil

	}

	targetStackImage := configuration.Spec.Source.Image

	engineConfig, _ := r.createBehaviorEngineConfiguration(ctx, claim, &configuration)
	// TODO error handling

	r.executeBehavior(ctx, claim, engineConfig, targetStackImage, &targetResourceBehavior)

	return unstructured.Unstructured{}, nil
}

/**
 * When a behavior executes, the resource engine is configured by the
 * object which triggered the behavior.
 */
func (r *HelmChartInstallReconciler) createBehaviorEngineConfiguration(
	ctx context.Context,
	claim *v1alpha1.HelmChartInstall,
	stackConfig *v1alpha1.StackConfiguration,
) (*corev1.ConfigMap, error) {
	// yamlyamlyamlyamlyaml
	configContents, _ := yaml.Marshal(claim.Spec)
	// TODO handle errors

	// Underneath, the yamler uses https://godoc.org/encoding/json#Marshal,
	// which means that the bytes are UTF-8 encoded
	stringConfigContents := fmt.Sprint(configContents)

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

	/*
		configGenerator := generate.ConfigMapGeneratorV1{
			Name:           claim.GetUID(),
			LiteralSources: fmt.Sprintf("%s=%s", configKeyName, stringConfigContents),
			AppendHash:     true,
		}
	*/

	configName := string(claim.GetUID())
	generatedMap, _ := generateConfigMap(configName, configKeyName, stringConfigContents)
	// TODO handle errors

	err := r.Client.Create(ctx, generatedMap)
	// TODO handle errors

	return generatedMap, err
}

// The main reason this exists as its own method is to encapsulate the hashing logic
func generateConfigMap(name string, fileName string, fileContents string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	configMap.Name = name
	configMap.Data = map[string]string{}

	configMap.Data[fileName] = fileContents
	h, err := hash.ConfigMapHash(configMap)
	if err != nil {
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
	claim *v1alpha1.HelmChartInstall,
	stackConfig *v1alpha1.StackConfiguration,
) string {
	return "helm2"
}

func (r *HelmChartInstallReconciler) executeBehavior(
	ctx context.Context,
	claim *v1alpha1.HelmChartInstall,
	engineConfiguration *corev1.ConfigMap,
	targetStackImage string,
	behavior *v1alpha1.StackConfigurationBehavior,
) (unstructured.Unstructured, error) {
	for _, resource := range behavior.Resources {
		// TODO error handling
		r.executeHook(ctx, claim, engineConfiguration, targetStackImage, resource)
	}
	return unstructured.Unstructured{}, nil
}

// TODO we could have a method create the job, and a higher-level one execute it.
func (r *HelmChartInstallReconciler) executeHook(
	ctx context.Context,
	claim *v1alpha1.HelmChartInstall,
	engineConfig *corev1.ConfigMap,
	targetStackImage string,
	targetResourceDir string,
) (unstructured.Unstructured, error) {
	ownerRef := meta.AsOwner(meta.ReferenceTo(claim, claim.GroupVersionKind()))
	var jobBackoff int32
	jobBackoff = 0

	// TODO target stack image will come from the stack object, or maybe the stack install object.
	// Then for each resource behavior hook, we want to run the hook
	// TODO update this to use the most recent format, where a hook is a structured object

	resourceDirName := fmt.Sprintf("/.registry/resources/%s", targetResourceDir)

	engineConfigurationVolumeName := "engine-configuration"
	engineConfigurationDir := "/usr/share/engine-configuration/"

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
		return unstructured.Unstructured{}, err
	}

	return unstructured.Unstructured{}, nil
}

func (r *HelmChartInstallReconciler) setClaimStatus(
	claim *helmv1alpha1.HelmChartInstall, result unstructured.Unstructured,
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
