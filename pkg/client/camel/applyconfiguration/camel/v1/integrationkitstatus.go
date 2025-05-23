/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// IntegrationKitStatusApplyConfiguration represents a declarative configuration of the IntegrationKitStatus type for use
// with apply.
type IntegrationKitStatusApplyConfiguration struct {
	ObservedGeneration *int64                                      `json:"observedGeneration,omitempty"`
	Phase              *camelv1.IntegrationKitPhase                `json:"phase,omitempty"`
	RootImage          *string                                     `json:"rootImage,omitempty"`
	BaseImage          *string                                     `json:"baseImage,omitempty"`
	Image              *string                                     `json:"image,omitempty"`
	Digest             *string                                     `json:"digest,omitempty"`
	Artifacts          []ArtifactApplyConfiguration                `json:"artifacts,omitempty"`
	Failure            *FailureApplyConfiguration                  `json:"failure,omitempty"`
	RuntimeVersion     *string                                     `json:"runtimeVersion,omitempty"`
	RuntimeProvider    *camelv1.RuntimeProvider                    `json:"runtimeProvider,omitempty"`
	Catalog            *CatalogApplyConfiguration                  `json:"catalog,omitempty"`
	Platform           *string                                     `json:"platform,omitempty"`
	Version            *string                                     `json:"version,omitempty"`
	Conditions         []IntegrationKitConditionApplyConfiguration `json:"conditions,omitempty"`
}

// IntegrationKitStatusApplyConfiguration constructs a declarative configuration of the IntegrationKitStatus type for use with
// apply.
func IntegrationKitStatus() *IntegrationKitStatusApplyConfiguration {
	return &IntegrationKitStatusApplyConfiguration{}
}

// WithObservedGeneration sets the ObservedGeneration field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ObservedGeneration field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithObservedGeneration(value int64) *IntegrationKitStatusApplyConfiguration {
	b.ObservedGeneration = &value
	return b
}

// WithPhase sets the Phase field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Phase field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithPhase(value camelv1.IntegrationKitPhase) *IntegrationKitStatusApplyConfiguration {
	b.Phase = &value
	return b
}

// WithRootImage sets the RootImage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RootImage field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithRootImage(value string) *IntegrationKitStatusApplyConfiguration {
	b.RootImage = &value
	return b
}

// WithBaseImage sets the BaseImage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the BaseImage field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithBaseImage(value string) *IntegrationKitStatusApplyConfiguration {
	b.BaseImage = &value
	return b
}

// WithImage sets the Image field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Image field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithImage(value string) *IntegrationKitStatusApplyConfiguration {
	b.Image = &value
	return b
}

// WithDigest sets the Digest field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Digest field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithDigest(value string) *IntegrationKitStatusApplyConfiguration {
	b.Digest = &value
	return b
}

// WithArtifacts adds the given value to the Artifacts field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Artifacts field.
func (b *IntegrationKitStatusApplyConfiguration) WithArtifacts(values ...*ArtifactApplyConfiguration) *IntegrationKitStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithArtifacts")
		}
		b.Artifacts = append(b.Artifacts, *values[i])
	}
	return b
}

// WithFailure sets the Failure field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Failure field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithFailure(value *FailureApplyConfiguration) *IntegrationKitStatusApplyConfiguration {
	b.Failure = value
	return b
}

// WithRuntimeVersion sets the RuntimeVersion field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RuntimeVersion field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithRuntimeVersion(value string) *IntegrationKitStatusApplyConfiguration {
	b.RuntimeVersion = &value
	return b
}

// WithRuntimeProvider sets the RuntimeProvider field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the RuntimeProvider field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithRuntimeProvider(value camelv1.RuntimeProvider) *IntegrationKitStatusApplyConfiguration {
	b.RuntimeProvider = &value
	return b
}

// WithCatalog sets the Catalog field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Catalog field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithCatalog(value *CatalogApplyConfiguration) *IntegrationKitStatusApplyConfiguration {
	b.Catalog = value
	return b
}

// WithPlatform sets the Platform field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Platform field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithPlatform(value string) *IntegrationKitStatusApplyConfiguration {
	b.Platform = &value
	return b
}

// WithVersion sets the Version field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Version field is set to the value of the last call.
func (b *IntegrationKitStatusApplyConfiguration) WithVersion(value string) *IntegrationKitStatusApplyConfiguration {
	b.Version = &value
	return b
}

// WithConditions adds the given value to the Conditions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Conditions field.
func (b *IntegrationKitStatusApplyConfiguration) WithConditions(values ...*IntegrationKitConditionApplyConfiguration) *IntegrationKitStatusApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithConditions")
		}
		b.Conditions = append(b.Conditions, *values[i])
	}
	return b
}
