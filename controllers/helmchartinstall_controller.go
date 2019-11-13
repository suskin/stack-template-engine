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

	helmv1alpha1 "github.com/suskin/stack-helm/api/v1alpha1"

	"github.com/crossplaneio/crossplane-runtime/pkg/meta"
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

	r.render(ctx, i)

	return ctrl.Result{}, nil
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
	*/

	// configuration is a typed object
	configuration, err := r.getStackConfiguration()

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

func (r *HelmChartInstallReconciler) getStackConfiguration() (unstructured.Unstructured, error) {
	// See the template stacks internal design doc for details, but
	// the most likely source of the stack configuration is the stack object itself.
	// Other potential sources include a configmap

	// TODO
	// - Stack configuration should be a structured object
	// - Stack configuration will be coming from a Kubernetes object, probably the
	//   Stack itself
	//stackConfiguration := ''
	return unstructured.Unstructured{}, nil
}

func (r *HelmChartInstallReconciler) processConfiguration(
	ctx context.Context,
	claim *helmv1alpha1.HelmChartInstall,
	configuration unstructured.Unstructured,
) (unstructured.Unstructured, error) {
	// Given a claim and a stack configuration, transform the claim into something
	// to inject into a stack's configuration renderer, and then render the Stack's
	// resources from the configuration and the transformed claim
	ownerRef := meta.AsOwner(meta.ReferenceTo(claim, claim.GroupVersionKind()))
	var jobBackoff int32
	jobBackoff = 0
	// TODO Handle multiple resource dirs. Perhaps try rolling everything into a single job,
	//      so have multiple engine containers, one for each path.
	//      And multiple apply containers; one for each path. Or, make multiple jobs.
	// TODO we may want some of these steps to be in this controller instead of
	//      being outsourced

	// TODO resource dir will come from the stack configuration
	targetResourceDir := "myResourceDir"
	// TODO target stack image will come from the claim
	targetStackImage := "crossplane/template-stack:latest"
	resourceDirName := fmt.Sprintf("/.registry/resources/%s", targetResourceDir)

	stackConfigurationVolumeName := "stack-configuration"
	stackContentsDestinationDir := "/usr/share/input/"

	resourceConfigurationVolumeName := "resource-configuration"
	resourceConfigurationDestinationDir := "/usr/share/resource-configuration/"
	targetNamespace := claim.GetNamespace()

	// TODO we should generate a name and save a reference on the claim
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "helm-template-apply",
			Namespace: targetNamespace,
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
					},
				},
			},
		},
	}

	if err := r.Create(ctx, job); err != nil {
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
