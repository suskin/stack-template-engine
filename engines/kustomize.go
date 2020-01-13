package engines

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/api/types"

	"k8s.io/apimachinery/pkg/runtime"

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
	kustomizationFieldName = "kustomization.yaml"
	overlaysFieldName      = "overlays.yaml"
)

func NewKustomizeEngine(logger logr.Logger) *KustomizeEngine {
	return &KustomizeEngine{}
}

type KustomizeEngine struct {
	logger logr.Logger
}

func (k *KustomizeEngine) CreateConfig(claim *unstructured.Unstructured, hc *v1alpha1.HookConfiguration) (*corev1.ConfigMap, error) {
	if hc.Engine.KustomizeConfiguration == nil {
		return nil, fmt.Errorf("kustomize configuration is empty")
	}
	// Kustomize only works with relative paths.
	stackRelativeDestDir, err := filepath.Rel(engineCfgDir, stackDestinationDir)
	if err != nil {
		return nil, err
	}

	// TODO(muvaf): investigate a better way to convert *Unstructured to Kustomization.
	kustomizationJSON, err := yaml.Marshal(hc.Engine.Kustomization)
	if err != nil {
		return nil, err
	}
	kustomization := &types.Kustomization{}
	if err := yaml.Unmarshal(kustomizationJSON, kustomization); err != nil {
		return nil, err
	}
	// Final overlay has to refer to the base Kustomization directory.
	kustomization.Resources = append(kustomization.Resources, stackRelativeDestDir)
	kustomization.PatchesStrategicMerge = []types.PatchStrategicMerge{
		overlaysFieldName,
	}
	kustomization.NamePrefix = fmt.Sprintf("%v-%v", claim.GetName(), kustomization.NamePrefix)
	if kustomization.CommonLabels == nil {
		kustomization.CommonLabels = map[string]string{}
	}
	//TODO(muvaf): use the user's domain.
	kustomization.CommonLabels["crossplane.io/name"] = claim.GetName()
	kustomization.CommonLabels["crossplane.io/namespace"] = claim.GetNamespace()
	kustomization.CommonLabels["crossplane.io/uid"] = string(claim.GetUID())
	kustomizationValue, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, err
	}
	var overlayObjects []runtime.Object
	for _, overlay := range hc.Engine.Overlays {
		// First make sure there is a value in the referred path.
		val, exists, err := unstructured.NestedFieldCopy(claim.Object, strings.Split(overlay.From, ".")...)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		// Create the patch with the given value.
		// TODO(muvaf): support more than one binding pair for the same resource entry.
		obj := &unstructured.Unstructured{}
		obj.SetAPIVersion(overlay.To.APIVersion)
		obj.SetKind(overlay.To.Kind)
		obj.SetName(overlay.To.Name)
		obj.SetNamespace(overlay.To.Namespace)
		if err := unstructured.SetNestedField(obj.Object, val, strings.Split(overlay.To.FieldPath, ".")...); err != nil {
			return nil, err
		}
		overlayObjects = append(overlayObjects, obj)
	}
	overlayYAMLs, err := yaml.Marshal(overlayObjects)
	if err != nil {
		return nil, err
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(claim.GetUID()),
			Namespace: claim.GetNamespace(),
		},
		Data: map[string]string{
			kustomizationFieldName: string(kustomizationValue),
			overlaysFieldName:      string(overlayYAMLs),
		},
	}
	return cm, nil
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
