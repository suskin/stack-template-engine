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

// RenderPhaseReconciler reconciles an object which we're watching for a template stack
type RenderPhaseReconciler struct {
	Client    client.Client
	Log       logr.Logger
	GVK       *schema.GroupVersionKind
	EventName v1alpha1.EventName
}

const (
	renderTimeout = 60 * time.Second
	spec          = "spec"
)

// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=helmchartinstalls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=helmchartinstalls/status,verbs=get;update;patch

func (r *RenderPhaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), renderTimeout)
	defer cancel()

	// We grab the claim as an unstructured so that we can have the same code handle
	// arbitrary claim types. The types will be erased by this point, so if a stack
	// author wants to validate the schema of a claim, they can do it by putting a
	// schema in the CRD of the claim type so that the claim's schema is validated
	// at the time that the object is accepted by the api server.
	i := &unstructured.Unstructured{}
	i.SetGroupVersionKind(*r.GVK)
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
	return ctrl.Result{}, r.render(ctx, i)
}

func (r *RenderPhaseReconciler) render(ctx context.Context, claim *unstructured.Unstructured) error {
	cfg, err := r.getStackConfiguration(ctx, claim)
	// TODO check for errors

	trb, err := r.getBehavior(ctx, claim, cfg)

	if trb == nil {
		// TODO error condition with a real error returned
		r.Log.V(0).Info("Couldn't find a configured behavior!",
			"claim", claim,
			"configuration", cfg,
		)
		return err
	}

	for _, hookCfg := range trb {
		engineCfg, err := r.createBehaviorEngineConfiguration(ctx, claim, &hookCfg)

		if err != nil {
			r.Log.Error(err, "Error creating engine configuration!", "claim", claim, "hook config", hookCfg)
			return err
		}

		// TODO support specifying the image on the hook
		stackImage := cfg.Spec.Behaviors.Source.Image

		result, err := r.executeHook(ctx, claim, engineCfg, stackImage, &hookCfg)
		// TODO check for errors

		err = r.setClaimStatus(claim, result)
	}

	return err
}

func (r *RenderPhaseReconciler) getStackConfiguration(
	ctx context.Context,
	claim *unstructured.Unstructured,
) (*v1alpha1.StackConfiguration, error) {
	// See the template stacks internal design doc for details, but
	// the most likely source of the stack configuration is the stack object itself.
	// Other potential sources include a configmap

	// TODO
	// - Stack configuration will be coming from a Kubernetes object, probably the
	//   Stack itself
	name, err := client.ObjectKeyFromObject(claim)

	if err != nil {
		r.Log.V(0).Info("getStackConfiguration returning early because of error creating object key", "err", err, "claim", claim)
		return nil, err
	}

	sc := &v1alpha1.StackConfiguration{}
	if err := r.Client.Get(ctx, name, sc); err != nil {
		r.Log.V(0).Info("getStackConfiguration returning early because of error fetching configuration", "err", err, "claim", claim)
		return nil, err
	}

	r.Log.V(0).Info("getStackConfiguration returning configuration", "configuration", sc)
	return sc, nil
}

// When a behavior is triggered, we want to know which behavior exactly we are executing.
//
// In most cases, this will probably be configured ahead of time by the setup controller, rather
// than being fetched at runtime by the render controller.
func (r *RenderPhaseReconciler) getBehavior(
	ctx context.Context,
	claim *unstructured.Unstructured,
	sc *v1alpha1.StackConfiguration,
) ([]v1alpha1.HookConfiguration, error) {
	gv, k := claim.GetObjectKind().GroupVersionKind().ToAPIVersionAndKind()
	gvk := v1alpha1.GVK(fmt.Sprintf("%s.%s", k, gv))

	// TODO handle missing keys gracefully
	scb, ok := sc.Spec.Behaviors.CRDs[gvk]

	if !ok {
		// TODO error condition with a real error returned
		r.Log.V(0).Info("Couldn't find a configured behavior!",
			"claim", claim,
			"configuration", sc,
			"targetGroupKindVersion", gvk,
		)
		return nil, nil
	}

	hooks := scb.Hooks

	if len(hooks) == 0 {
		// TODO error condition with a real error returned
		// TODO it'd be nice to enforce this on acceptance or creation if possible
		// TODO theoretically we should be able to enforce this at the schema level
		r.Log.V(0).Info("Couldn't find hooks for configured behavior!", "claim", claim, "configuration", sc)
		return nil, nil

	}

	hookCfgs := hooks[r.EventName]
	if len(hookCfgs) == 0 {
		// TODO error condition with a real error returned
		// TODO it'd be nice to enforce this on acceptance or creation if possible
		r.Log.V(0).Info("Couldn't find resources for configured behavior!", "claim", claim, "configuration", sc)
		return nil, nil

	}

	// If a directory is not provided, we will use the root of the stack artifact. However, it is recommended to
	// specify a directory for clarity.
	resolvedCfgs := make([]v1alpha1.HookConfiguration, 0)
	for _, cfg := range hookCfgs {
		// If no engine is specified at the hook level, we want to use the engine specified at the CRD level.
		// If no engine is specified at the hook *or* CRD level, we want to use the engine specified at the configuration level.
		if cfg.Engine.Type == "" {
			if scb.Engine.Type != "" {
				r.Log.V(0).Info("Inheriting engine for hook from CRD-level behavior configuration", "engineType", scb.Engine.Type)
				cfg.Engine.Type = scb.Engine.Type
			} else {
				r.Log.V(0).Info("Inheriting engine for hook from top-level behavior configuration", "engineType", sc.Spec.Behaviors.Engine.Type)
				cfg.Engine.Type = sc.Spec.Behaviors.Engine.Type
			}
		}

		resolvedCfgs = append(resolvedCfgs, cfg)
	}

	r.Log.V(0).Info("Returning hook configurations", "hook configurations", resolvedCfgs)

	return resolvedCfgs, nil
}

// When a behavior executes, the resource engine is configured by the
// object which triggered the behavior. This method encapsulates the logic to
// create the resource engine configuration from the object's fields.
func (r *RenderPhaseReconciler) createBehaviorEngineConfiguration(
	ctx context.Context,
	claim *unstructured.Unstructured,
	hc *v1alpha1.HookConfiguration,
) (*corev1.ConfigMap, error) {
	// yamlyamlyamlyamlyaml
	// TODO if spec is missing, this won't work very well
	s, ok := claim.Object[spec]

	if !ok {
		r.Log.V(0).Info("Spec not found on claim; not creating engine configuration", "claim", claim)
	}

	r.Log.V(0).Info("Converting configuration", "spec", s)
	configContents, err := yaml.Marshal(s)

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
	generatedMap, err := r.generateConfigMap(configName, configKeyName, stringConfigContents)

	if err != nil {
		r.Log.V(0).Info("Error generating config map!", "claim", claim, "error", err)
		return nil, err
	}

	generatedMap.SetNamespace(claim.GetNamespace())

	r.Log.V(0).Info("Generated config map to pass engine configuration", "configMap", generatedMap)

	if err := r.Client.Create(ctx, generatedMap); err != nil {
		if kerrors.IsAlreadyExists(err) {
			r.Log.V(1).Info("Config map already exists! Ignoring error", "claim", claim, "configMap", generatedMap)
		} else {
			r.Log.V(0).Info("Error creating config map!", "claim", claim, "error", err, "configMap", generatedMap)
			return nil, err
		}
	}

	return generatedMap, err
}

// The main reason this exists as its own method is to encapsulate the hashing logic
func (r *RenderPhaseReconciler) generateConfigMap(name string, fileName string, fileContents string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	cm.Name = name
	cm.Data = map[string]string{}

	cm.Data[fileName] = fileContents
	h, err := hash.ConfigMapHash(cm)
	if err != nil {
		r.Log.V(0).Info("Error hashing config map!", "error", err)
		return cm, err
	}
	cm.Name = fmt.Sprintf("%s-%s", cm.Name, h)

	return cm, nil
}

// TODO we could have a method create the job, and a higher-level one execute it.
func (r *RenderPhaseReconciler) executeHook(
	ctx context.Context,
	claim *unstructured.Unstructured,
	engineCfg *corev1.ConfigMap,
	targetStackImage string,
	hookCfg *v1alpha1.HookConfiguration,
) (*unstructured.Unstructured, error) {
	// TODO if there is no config specified, either use an empty config or don't specify
	// one at all.

	// TODO if we change this to meta.AsController, and we have the controller-runtime controller configured
	// to Own Jobs, then we'll get a reconcile call when Jobs finish. However, we'd need to change the logic
	// for the reconcile a bit to support that effectively. For example:
	// - We wouldn't want to create jobs every time reconcile is run
	//   * This means keeping track of created jobs somewhere and could also mean using deterministic job names
	ownerRef := meta.AsOwner(meta.ReferenceTo(claim, claim.GroupVersionKind()))
	var jobBackoff int32

	// TODO target stack image will come from the stack object, or maybe the stack install object.
	// Then for each resource behavior hook, we want to run the hook
	// TODO update this to use the most recent format, where a hook is a structured object

	resourceDir := fmt.Sprintf("/.registry/resources/%s", hookCfg.Directory)

	engineCfgVolumeName := "engine-configuration"
	engineCfgDir := "/usr/share/engine-configuration/"

	// TODO this file name should not be hard-coded
	engineCfgFile := fmt.Sprintf("%svalues.yaml", engineCfgDir)

	stackVolumeName := "stack-configuration"
	stackDestDir := "/usr/share/input/"

	resourceCfgVolumeName := "resource-configuration"
	resourceCfgDestDir := "/usr/share/resource-configuration/"
	namespace := claim.GetNamespace()

	// TODO we should generate a name and save a reference on the claim
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "helm-template-apply-",
			Namespace:    namespace,
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
								"cp", "-R", fmt.Sprintf("%s/.", resourceDir), stackDestDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      stackVolumeName,
									MountPath: stackDestDir,
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
								"--output-dir", resourceCfgDestDir,
								"--namespace", namespace,
								"--values", engineCfgFile,
								stackDestDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      stackVolumeName,
									MountPath: stackDestDir,
								},
								{
									Name:      resourceCfgVolumeName,
									MountPath: resourceCfgDestDir,
								},
								{
									Name:      engineCfgVolumeName,
									MountPath: engineCfgDir,
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
								"--namespace", namespace,
								"-R",
								"-f",
								resourceCfgDestDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      resourceCfgVolumeName,
									MountPath: resourceCfgDestDir,
								},
							},
							ImagePullPolicy: corev1.PullNever,
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: stackVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: resourceCfgVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: engineCfgVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: engineCfg.GetName(),
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

func (r *RenderPhaseReconciler) setClaimStatus(
	claim *unstructured.Unstructured, result *unstructured.Unstructured,
) error {
	// The claim is the CR that triggered this whole thing.
	// The result is the result of trying to apply or delete the templates rendered from processing the claim.
	// If the processing happens in a job, the status should be updated after the job completes, which may mean
	// waiting for it to complete by using a watcher.
	return nil
}

func (r *RenderPhaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.HelmChartInstall{}).
		Complete(r)
}
