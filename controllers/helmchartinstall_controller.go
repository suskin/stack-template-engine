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

func (r *HelmChartInstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmv1alpha1.HelmChartInstall{}).
		Complete(r)
}
