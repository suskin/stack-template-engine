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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	helmv1alpha1 "github.com/suskin/stack-helm/api/v1alpha1"
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

	return ctrl.Result{}, nil
}

func (r *HelmChartInstallReconciler) render() error {
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
	*/

	// claim is an unstructured.Unstructured
	claim, err := r.getClaim()

	// configuration is a typed object
	configuration, err := r.getStackConfiguration()

	// TODO the status will have more than *just* the result in it,
	// but we can start with just the result
	result, err := r.processConfiguration(claim, configuration)

	err = r.setClaimStatus(claim, result)

	return err
}

func (r *HelmChartInstallReconciler) getClaim() (unstructured.Unstructured, error) {
	// The claim is the CR that triggered this whole thing.
	return unstructured.Unstructured{}, nil
}

func (r *HelmChartInstallReconciler) getStackConfiguration() (unstructured.Unstructured, error) {
	// See the template stacks internal design doc for details, but
	// the most likely source of the stack configuration is the stack object itself.
	// Other potential sources include a configmap
	return unstructured.Unstructured{}, nil
}

func (r *HelmChartInstallReconciler) processConfiguration(
	claim unstructured.Unstructured, configuration unstructured.Unstructured,
) (unstructured.Unstructured, error) {
	//job := Job{}
	return unstructured.Unstructured{}, nil
}

func (r *HelmChartInstallReconciler) setClaimStatus(
	claim unstructured.Unstructured, result unstructured.Unstructured,
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
