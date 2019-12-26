package engines

import (
	"context"
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kustomize/api/types"

	"github.com/crossplaneio/crossplane-runtime/pkg/meta"
	"github.com/go-logr/logr"
	"github.com/suskin/stack-template-engine/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	kustomizationFileName = "kustomization.yaml"
)

func NewKustomizeEngine(logger logr.Logger) *KustomizeEngine {
	return &KustomizeEngine{}
}

type KustomizeEngine struct {
	logger logr.Logger
}

func (k *KustomizeEngine) CreateConfig(claim *unstructured.Unstructured, hc *v1alpha1.HookConfiguration) (*corev1.ConfigMap, error) {
	kustomization, ok := claim.Object[spec].(*types.Kustomization)
	if !ok {
		return nil, fmt.Errorf("could not marshall claim spec into Kustomization type")
	}
	// Kustomize only works with relative paths.
	stackRelativeDestDir, err := filepath.Rel(engineCfgDir, stackDestinationDir)
	if err != nil {
		return nil, err
	}
	// Final overlay has to refer to the base Kustomization directory.
	kustomization.Resources = []string{
		stackRelativeDestDir,
	}

	k.logger.V(0).Info("Converting configuration", "spec", kustomization)
	configContents, err := yaml.Marshal(kustomization)

	k.logger.V(0).Info("Configuration contents as yaml", "configContents", configContents)

	if err != nil {
		k.logger.Error(err, "Error marshaling claim spec as yaml!", "claim", claim)
		return nil, err
	}

	// Underneath, the yamler uses https://godoc.org/encoding/json#Marshal,
	// which means that the bytes are UTF-8 encoded
	// Theoretically we could get better performance by using a binary config
	// map, but having a string makes it better for humans who may want to observe
	// or troubleshoot behavior.
	stringConfigContents := string(configContents)

	configName := string(claim.GetUID())
	generatedMap, err := generateConfigMap(configName, kustomizationFileName, stringConfigContents, k.logger)

	if err != nil {
		k.logger.V(0).Info("Error generating config map!", "claim", claim, "error", err)
		return nil, err
	}

	generatedMap.SetNamespace(claim.GetNamespace())

	k.logger.V(0).Info("Generated config map to pass engine configuration", "configMap", generatedMap)

	return generatedMap, err
}

func (k *KustomizeEngine) RunEngine(ctx context.Context, client client.Client, claim *unstructured.Unstructured, config *corev1.ConfigMap, stackSource string, hc *v1alpha1.HookConfiguration) (*unstructured.Unstructured, error) {
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

	resourceDir := filepath.Join("/.registry/resources", hc.Directory)

	engineCfgVolumeName := "engine-configuration"

	stackVolumeName := "stack-configuration"

	resourceCfgVolumeName := "resource-configuration"
	resourceCfgDestDir := "/usr/share/resource-configuration/"

	namespace := claim.GetNamespace()

	// TODO we should generate a name and save a reference on the claim
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kustomize-template-apply-",
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
							Image: stackSource,
							Command: []string{
								// The "." suffix causes the cp -R to copy the contents of the directory instead of
								// the directory itself
								"cp", "-R", fmt.Sprintf("%s/.", resourceDir), stackDestinationDir,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      stackVolumeName,
									MountPath: stackDestinationDir,
								},
							},
							ImagePullPolicy: corev1.PullNever,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "engine",
							Image: "crossplane/kubectl:latest",
							Command: []string{
								// To use streaming via `>` we need to invoke bash -c
								"bash", "-c",
							},
							Args: []string{
								fmt.Sprintf("kubectl apply --kustomize %s", engineCfgDir),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      stackVolumeName,
									MountPath: stackDestinationDir,
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
										Name: config.GetName(),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// TODO theoretically this won't be creating a job every time, and eventually we'll want to return a status or result of some sort
	// so that our shared reconciler logic can expose it, probably by updating the claim status.
	return nil, client.Create(ctx, job)
}
