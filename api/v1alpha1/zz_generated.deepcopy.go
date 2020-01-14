// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FieldBinding) DeepCopyInto(out *FieldBinding) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FieldBinding.
func (in *FieldBinding) DeepCopy() *FieldBinding {
	if in == nil {
		return nil
	}
	out := new(FieldBinding)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmChartInstall) DeepCopyInto(out *HelmChartInstall) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmChartInstall.
func (in *HelmChartInstall) DeepCopy() *HelmChartInstall {
	if in == nil {
		return nil
	}
	out := new(HelmChartInstall)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HelmChartInstall) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmChartInstallList) DeepCopyInto(out *HelmChartInstallList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HelmChartInstall, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmChartInstallList.
func (in *HelmChartInstallList) DeepCopy() *HelmChartInstallList {
	if in == nil {
		return nil
	}
	out := new(HelmChartInstallList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HelmChartInstallList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmChartInstallSpec) DeepCopyInto(out *HelmChartInstallSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmChartInstallSpec.
func (in *HelmChartInstallSpec) DeepCopy() *HelmChartInstallSpec {
	if in == nil {
		return nil
	}
	out := new(HelmChartInstallSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmChartInstallStatus) DeepCopyInto(out *HelmChartInstallStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmChartInstallStatus.
func (in *HelmChartInstallStatus) DeepCopy() *HelmChartInstallStatus {
	if in == nil {
		return nil
	}
	out := new(HelmChartInstallStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HookConfiguration) DeepCopyInto(out *HookConfiguration) {
	*out = *in
	in.Engine.DeepCopyInto(&out.Engine)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HookConfiguration.
func (in *HookConfiguration) DeepCopy() *HookConfiguration {
	if in == nil {
		return nil
	}
	out := new(HookConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in HookConfigurations) DeepCopyInto(out *HookConfigurations) {
	{
		in := &in
		*out = make(HookConfigurations, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HookConfigurations.
func (in HookConfigurations) DeepCopy() HookConfigurations {
	if in == nil {
		return nil
	}
	out := new(HookConfigurations)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KustomizeConfiguration) DeepCopyInto(out *KustomizeConfiguration) {
	*out = *in
	if in.Overlays != nil {
		in, out := &in.Overlays, &out.Overlays
		*out = make([]Overlay, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Kustomization != nil {
		in, out := &in.Kustomization, &out.Kustomization
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KustomizeConfiguration.
func (in *KustomizeConfiguration) DeepCopy() *KustomizeConfiguration {
	if in == nil {
		return nil
	}
	out := new(KustomizeConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Overlay) DeepCopyInto(out *Overlay) {
	*out = *in
	out.ObjectReference = in.ObjectReference
	if in.Bindings != nil {
		in, out := &in.Bindings, &out.Bindings
		*out = make([]FieldBinding, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Overlay.
func (in *Overlay) DeepCopy() *Overlay {
	if in == nil {
		return nil
	}
	out := new(Overlay)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceEngineConfiguration) DeepCopyInto(out *ResourceEngineConfiguration) {
	*out = *in
	if in.KustomizeConfiguration != nil {
		in, out := &in.KustomizeConfiguration, &out.KustomizeConfiguration
		*out = new(KustomizeConfiguration)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceEngineConfiguration.
func (in *ResourceEngineConfiguration) DeepCopy() *ResourceEngineConfiguration {
	if in == nil {
		return nil
	}
	out := new(ResourceEngineConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StackConfiguration) DeepCopyInto(out *StackConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StackConfiguration.
func (in *StackConfiguration) DeepCopy() *StackConfiguration {
	if in == nil {
		return nil
	}
	out := new(StackConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StackConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StackConfigurationBehavior) DeepCopyInto(out *StackConfigurationBehavior) {
	*out = *in
	if in.Hooks != nil {
		in, out := &in.Hooks, &out.Hooks
		*out = make(map[EventName]HookConfigurations, len(*in))
		for key, val := range *in {
			var outVal []HookConfiguration
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(HookConfigurations, len(*in))
				for i := range *in {
					(*in)[i].DeepCopyInto(&(*out)[i])
				}
			}
			(*out)[key] = outVal
		}
	}
	in.Engine.DeepCopyInto(&out.Engine)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StackConfigurationBehavior.
func (in *StackConfigurationBehavior) DeepCopy() *StackConfigurationBehavior {
	if in == nil {
		return nil
	}
	out := new(StackConfigurationBehavior)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StackConfigurationBehaviors) DeepCopyInto(out *StackConfigurationBehaviors) {
	*out = *in
	if in.CRDs != nil {
		in, out := &in.CRDs, &out.CRDs
		*out = make(map[GVK]StackConfigurationBehavior, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	in.Engine.DeepCopyInto(&out.Engine)
	out.Source = in.Source
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StackConfigurationBehaviors.
func (in *StackConfigurationBehaviors) DeepCopy() *StackConfigurationBehaviors {
	if in == nil {
		return nil
	}
	out := new(StackConfigurationBehaviors)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StackConfigurationList) DeepCopyInto(out *StackConfigurationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]StackConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StackConfigurationList.
func (in *StackConfigurationList) DeepCopy() *StackConfigurationList {
	if in == nil {
		return nil
	}
	out := new(StackConfigurationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *StackConfigurationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StackConfigurationSource) DeepCopyInto(out *StackConfigurationSource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StackConfigurationSource.
func (in *StackConfigurationSource) DeepCopy() *StackConfigurationSource {
	if in == nil {
		return nil
	}
	out := new(StackConfigurationSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StackConfigurationSpec) DeepCopyInto(out *StackConfigurationSpec) {
	*out = *in
	in.Behaviors.DeepCopyInto(&out.Behaviors)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StackConfigurationSpec.
func (in *StackConfigurationSpec) DeepCopy() *StackConfigurationSpec {
	if in == nil {
		return nil
	}
	out := new(StackConfigurationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StackConfigurationStatus) DeepCopyInto(out *StackConfigurationStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StackConfigurationStatus.
func (in *StackConfigurationStatus) DeepCopy() *StackConfigurationStatus {
	if in == nil {
		return nil
	}
	out := new(StackConfigurationStatus)
	in.DeepCopyInto(out)
	return out
}
