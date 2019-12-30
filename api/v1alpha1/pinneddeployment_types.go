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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReplicasRoundingStrategyType defines strategies for rounding the number of old and new replicas.
type ReplicasRoundingStrategyType string

const (
	// Rounds the number of new replicas down to the nearest integer.
	DownReplicasRoundingStrategyType = "Down"
	// Rounds the number of new replicas up to the nearest integer.
	UpReplicasRoundingStrategyType = "Up"
	// Rounds the number of new replicas to the nearest integer.
	// This is the default ReplicasRoundingStrategy.
	NearestReplicasRoundingStrategyType = "Nearest"
)

// PinnedDeploymentSpec defines the desired state of PinnedDeployment
type PinnedDeploymentSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas dictates total number number of replicas to have, between old and new versions.
	Replicas int32 `json:"replicas,omitempty"`

	// ReplicasPercentNext dictates the percent of replicas that would use the new pod template.
	// Accepted values are 0 to 100.
	ReplicasPercentNext int32 `json:"replicasPercentNext,omitempty"`

	// ReplicasRoundingStrategy dictates the rounding behavior when ReplicasPercentNext does not translate to a whole number of instances.
	// Available options are Up, Down, and Nearest.
	ReplicasRoundingStrategy ReplicasRoundingStrategyType `json:"replicasRoundingStrategy,omitempty"`

	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// Templates contains the old and new pod templates.
	Templates PreviousNextPodTemplateSpecPair `json:"templates,omitempty"`

	/*
		// NewMustBecomeOld dictates how the Templates.Previous field may be updated.
		// If false, there are no restrictions to how the field can be updated.
		// If true, Templates.Previous can only be updated to the previous value of Templates.Next, and if the previous value of ReplicasPercentNext was 100.
		// In other words, if NewMustBecomeOld is true, a version must be fully rolled out to be considered old.
		// Default false.
		NewMustBecomeOld bool
	*/
}

type PreviousNextPodTemplateSpecPair struct {
	// Previous is the pod template for the previous version.
	Previous FakePodTemplateSpec `json:"previous,omitempty"`

	// Next is the pod template for the next version.
	Next FakePodTemplateSpec `json:"next,omitempty"`
}

// FakePodTemplateSpec is called "fake" because it's a workaround for CRD issues with metadata fields in PodTemplateSpec.
// It provides a very similar interface to PodTempleteSpec.
// https://github.com/zeitgeistlabs/kubernetes-rollout/issues/3
type FakePodTemplateSpec struct {
	Metadata BasicMetadata  `json:"metadata,omitempty"`
	Spec     corev1.PodSpec `json:"spec,omitempty"`
}

type BasicMetadata struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// PinnedDeploymentStatus defines the observed state of PinnedDeployment
type PinnedDeploymentStatus struct {
	Replicas int32  `json:"replicas,omitempty"`
	Selector string `json:"selector,omitempty"` // TODO should this go in the spec? It's fixed based on Spec.Selector
	/*
		Need the following:
			- Total replicas
			- Total ready replicas
			- Total & desired & ready per-version
	*/
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.replicas,selectorpath=.status.selector
// +kubebuilder:subresource:status

// PinnedDeployment is the Schema for the pinneddeployments API
type PinnedDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PinnedDeploymentSpec   `json:"spec,omitempty"`
	Status PinnedDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PinnedDeploymentList contains a list of PinnedDeployment
type PinnedDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PinnedDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PinnedDeployment{}, &PinnedDeploymentList{})
}
