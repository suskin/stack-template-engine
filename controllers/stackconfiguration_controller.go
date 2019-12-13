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
	"strings"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	batchv1 "k8s.io/api/batch/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1alpha1 "github.com/suskin/stack-template-engine/api/v1alpha1"
)

// StackConfigurationReconciler reconciles a StackConfiguration object
type StackConfigurationReconciler struct {
	Client  client.Client
	Log     logr.Logger
	Manager manager.Manager
}

type Behavior struct {
	cfg *v1alpha1.StackConfigurationBehavior
	gvk *schema.GroupVersionKind
}

// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=stackconfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.samples.stacks.crossplane.io,resources=stackconfigurations/status,verbs=get;update;patch

func (r *StackConfigurationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("stackconfiguration", req.NamespacedName)

	// your logic here
	// r.setup()
	i := &v1alpha1.StackConfiguration{}
	if err := r.Client.Get(ctx, req.NamespacedName, i); err != nil {
		if kerrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.V(0).Info("Hello World!", "instanceName", req.NamespacedName, "instance", i)
	_ = r.setup(i)

	return ctrl.Result{}, nil
}

func (r *StackConfigurationReconciler) setup(config *v1alpha1.StackConfiguration) error {
	// For each behavior:
	// - Grab the configuration values:
	//   * Source stack; image or url
	//   * Source GVK
	//   * Event type
	//   * The coordinates of the behavior in the configuration, so it can be found again. Or,
	//     the behavior itself
	// - Create a render controller, passing it the configuration values

	/* Questions
	Should we grab all of the configuration at setup time, or at render time?
	- At render time, so that we're always using the latest version of the object
	- Though, the ideal would be if we cached the configuration and changed it if it changed
	*/

	// behaviors := config.Spec.Behaviors
	// Even though this is using a type which is part of the manager's scheme and which is a type
	// known at compile time, this has also been tested with types which are neither (such as Crossplane's
	// KubernetesApplication), and it works for that too.

	behaviors := r.getBehaviors(config)

	for _, behavior := range behaviors {
		gvk := behavior.gvk

		if err := r.NewRenderController(gvk); err != nil {
			// TODO what do we want to do if some of the registrations succeed and some of them fail?
			r.Log.Error(err, "Error creating new render controller!", "gvk", gvk)
		}
	}

	return nil
}

// This exists because getting the individual behaviors may be a bit tricker in the future.
// For example, the engine may be configured at multiple levels. Another example is that
// behaviors may be configured at multiple levels, if there are stack-level behaviors in
// addition to object-level behaviors.
func (r *StackConfigurationReconciler) getBehaviors(config *v1alpha1.StackConfiguration) []Behavior {
	scbs := config.Spec.Behaviors.CRDs

	behaviors := make([]Behavior, 0)

	for rawGvk, scb := range scbs {
		// We are assuming strings look like "Kind.group.com/version"
		gvkSplit := strings.SplitN(rawGvk, ".", 2)
		gvk := schema.FromAPIVersionAndKind(gvkSplit[1], gvkSplit[0])

		behaviors = append(behaviors, Behavior{
			gvk: &gvk,
			cfg: &scb,
		})
	}

	return behaviors
}

func (r *StackConfigurationReconciler) NewRenderController(gvk *schema.GroupVersionKind) error {
	/**
	TODO
	- In the future, we may want to be able to stop listening when a stack is uninstalled.
	- What if we have multiple controller workers watching the stack configuration? Do we need to worry about trying to not
	  create multiple render controllers for a single gvk?
	*/
	// scheme

	apiType := &unstructured.Unstructured{}
	apiType.SetGroupVersionKind(*gvk)

	reconciler := (&HelmChartInstallReconciler{
		Client: r.Manager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName(fmt.Sprintf("%s.%s/%s", gvk.Kind, gvk.Group, gvk.Version)),
		GVK:    gvk,
	})

	r.Log.V(0).Info("Adding new controller to manager")

	err := ctrl.NewControllerManagedBy(r.Manager).
		For(apiType).
		Owns(&batchv1.Job{}).
		Complete(reconciler)

	if err != nil {
		r.Log.V(0).Info("unable to create controller", "gvk", gvk, "err", err)
		return err
	}

	return nil
}

func (r *StackConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.StackConfiguration{}).
		Complete(r)
}
