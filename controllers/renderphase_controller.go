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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/suskin/stack-template-engine/api/v1alpha1"
	"github.com/suskin/stack-template-engine/engines"
)

// RenderPhaseReconciler reconciles an object which we're watching for a template stack
type RenderPhaseReconciler struct {
	Client     client.Client
	Log        logr.Logger
	GVK        *schema.GroupVersionKind
	EventName  v1alpha1.EventName
	ConfigName types.NamespacedName
}

const (
	renderTimeout = 60 * time.Second
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
		engineType := hookCfg.Engine.Type

		var engineRunner engines.ResourceEngineRunner

		// TODO this should probably not be a hard-coded raw string
		if engineType == "helm2" {
			engineRunner = engines.NewHelm2EngineRunner(r.Log)
		} else {
			r.Log.V(0).Info("Unrecognized engine type! Skipping hook.", "claim", claim, "hookConfig", hookCfg)
			continue
		}

		cm, err := engineRunner.CreateConfig(claim, &hookCfg)

		// engineCfg, err := r.createBehaviorEngineConfiguration(ctx, claim, &hookCfg)

		if err != nil {
			r.Log.Error(err, "Error creating engine configuration!", "claim", claim, "hookConfig", hookCfg)
			return err
		}

		err = r.createConfigMap(ctx, cm)
		if err != nil {
			r.Log.Error(err, "Error creating config map!", "claim", claim, "hookConfig", hookCfg)
			return err
		}

		// TODO support specifying the image on the hook. We could start by just injecting the source image on the
		// hook configuration, in the same way we do for engine type.
		stackImage := cfg.Spec.Behaviors.Source.Image

		result, err := engineRunner.RunEngine(ctx, r.Client, claim, cm, stackImage, &hookCfg)
		// TODO check for errors

		err = r.setClaimStatus(claim, result)
	}

	return err
}

// This mostly exists to encapsulate the logging and the ignoring of already exists errors
func (r *RenderPhaseReconciler) createConfigMap(ctx context.Context, cm *corev1.ConfigMap) error {
	if err := r.Client.Create(ctx, cm); err != nil {
		if kerrors.IsAlreadyExists(err) {
			r.Log.V(1).Info("Config map already exists! Ignoring error", "configMap", cm)
		} else {
			// One might consider logging an error here, but the logging is handled at a higher level
			// where more context can be logged.
			return err
		}
	}

	return nil
}

func (r *RenderPhaseReconciler) getStackConfiguration(
	ctx context.Context,
	claim *unstructured.Unstructured,
) (*v1alpha1.StackConfiguration, error) {
	// See the template stacks internal design doc for details, but
	// the most likely source of the stack configuration is the stack object itself.
	// Other potential sources include a configmap

	sc := &v1alpha1.StackConfiguration{}
	if err := r.Client.Get(ctx, r.ConfigName, sc); err != nil {
		// TODO if the error is that the config no longer exists, we may want to kill this controller. But, maybe that'll be handled at a different level.
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

func (r *RenderPhaseReconciler) setClaimStatus(
	claim *unstructured.Unstructured, result *unstructured.Unstructured,
) error {
	// The claim is the CR that triggered this whole thing.
	// The result is the result of trying to apply or delete the templates rendered from processing the claim.
	// If the processing happens in a job, the status should be updated after the job completes, which may mean
	// waiting for it to complete by using a watcher.
	return nil
}
