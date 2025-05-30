/*
Copyright The Kubernetes Authors.

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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1beta1

import (
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// ClusterQueueSpecApplyConfiguration represents a declarative configuration of the ClusterQueueSpec type for use
// with apply.
type ClusterQueueSpecApplyConfiguration struct {
	ResourceGroups          []ResourceGroupApplyConfiguration          `json:"resourceGroups,omitempty"`
	Cohort                  *kueuev1beta1.CohortReference              `json:"cohort,omitempty"`
	QueueingStrategy        *kueuev1beta1.QueueingStrategy             `json:"queueingStrategy,omitempty"`
	NamespaceSelector       *v1.LabelSelectorApplyConfiguration        `json:"namespaceSelector,omitempty"`
	FlavorFungibility       *FlavorFungibilityApplyConfiguration       `json:"flavorFungibility,omitempty"`
	Preemption              *ClusterQueuePreemptionApplyConfiguration  `json:"preemption,omitempty"`
	AdmissionChecks         []kueuev1beta1.AdmissionCheckReference     `json:"admissionChecks,omitempty"`
	AdmissionChecksStrategy *AdmissionChecksStrategyApplyConfiguration `json:"admissionChecksStrategy,omitempty"`
	StopPolicy              *kueuev1beta1.StopPolicy                   `json:"stopPolicy,omitempty"`
	FairSharing             *FairSharingApplyConfiguration             `json:"fairSharing,omitempty"`
	AdmissionScope          *AdmissionScopeApplyConfiguration          `json:"admissionScope,omitempty"`
}

// ClusterQueueSpecApplyConfiguration constructs a declarative configuration of the ClusterQueueSpec type for use with
// apply.
func ClusterQueueSpec() *ClusterQueueSpecApplyConfiguration {
	return &ClusterQueueSpecApplyConfiguration{}
}

// WithResourceGroups adds the given value to the ResourceGroups field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ResourceGroups field.
func (b *ClusterQueueSpecApplyConfiguration) WithResourceGroups(values ...*ResourceGroupApplyConfiguration) *ClusterQueueSpecApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithResourceGroups")
		}
		b.ResourceGroups = append(b.ResourceGroups, *values[i])
	}
	return b
}

// WithCohort sets the Cohort field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Cohort field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithCohort(value kueuev1beta1.CohortReference) *ClusterQueueSpecApplyConfiguration {
	b.Cohort = &value
	return b
}

// WithQueueingStrategy sets the QueueingStrategy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the QueueingStrategy field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithQueueingStrategy(value kueuev1beta1.QueueingStrategy) *ClusterQueueSpecApplyConfiguration {
	b.QueueingStrategy = &value
	return b
}

// WithNamespaceSelector sets the NamespaceSelector field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the NamespaceSelector field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithNamespaceSelector(value *v1.LabelSelectorApplyConfiguration) *ClusterQueueSpecApplyConfiguration {
	b.NamespaceSelector = value
	return b
}

// WithFlavorFungibility sets the FlavorFungibility field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FlavorFungibility field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithFlavorFungibility(value *FlavorFungibilityApplyConfiguration) *ClusterQueueSpecApplyConfiguration {
	b.FlavorFungibility = value
	return b
}

// WithPreemption sets the Preemption field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Preemption field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithPreemption(value *ClusterQueuePreemptionApplyConfiguration) *ClusterQueueSpecApplyConfiguration {
	b.Preemption = value
	return b
}

// WithAdmissionChecks adds the given value to the AdmissionChecks field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the AdmissionChecks field.
func (b *ClusterQueueSpecApplyConfiguration) WithAdmissionChecks(values ...kueuev1beta1.AdmissionCheckReference) *ClusterQueueSpecApplyConfiguration {
	for i := range values {
		b.AdmissionChecks = append(b.AdmissionChecks, values[i])
	}
	return b
}

// WithAdmissionChecksStrategy sets the AdmissionChecksStrategy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AdmissionChecksStrategy field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithAdmissionChecksStrategy(value *AdmissionChecksStrategyApplyConfiguration) *ClusterQueueSpecApplyConfiguration {
	b.AdmissionChecksStrategy = value
	return b
}

// WithStopPolicy sets the StopPolicy field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StopPolicy field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithStopPolicy(value kueuev1beta1.StopPolicy) *ClusterQueueSpecApplyConfiguration {
	b.StopPolicy = &value
	return b
}

// WithFairSharing sets the FairSharing field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the FairSharing field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithFairSharing(value *FairSharingApplyConfiguration) *ClusterQueueSpecApplyConfiguration {
	b.FairSharing = value
	return b
}

// WithAdmissionScope sets the AdmissionScope field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AdmissionScope field is set to the value of the last call.
func (b *ClusterQueueSpecApplyConfiguration) WithAdmissionScope(value *AdmissionScopeApplyConfiguration) *ClusterQueueSpecApplyConfiguration {
	b.AdmissionScope = value
	return b
}
