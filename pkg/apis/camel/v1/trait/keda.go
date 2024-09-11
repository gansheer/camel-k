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

package trait

// The KEDA trait can be used for automatic integration with KEDA autoscalers.
// The trait can be either manually configured using the `triggers` option or automatically configured
// via markers in the Kamelets.
//
// For information on how to use KEDA enabled Kamelets with the KEDA trait, refer to
// xref:ROOT:kamelets/kamelets-user.adoc#kamelet-keda-user[the KEDA section in the Kamelets user guide].
// If you want to create Kamelets that contain KEDA metadata, refer to
// xref:ROOT:kamelets/kamelets-dev.adoc#kamelet-keda-dev[the KEDA section in the Kamelets development guide].
//
// The KEDA trait is disabled by default.
//
// +camel-k:trait=keda.
type KedaTrait struct {
	Trait `property:",squash" json:",inline"`
	// Enables automatic configuration of the trait. Allows the trait to infer KEDA triggers from the Kamelets.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Set the spec->replicas field on the top level controller to an explicit value if missing, to allow KEDA to recognize it as a scalable resource.
	HackControllerReplicas *bool `property:"hack-controller-replicas" json:"hackControllerReplicas,omitempty"`
	// Interval (seconds) to check each trigger on.
	PollingInterval *int32 `property:"polling-interval" json:"pollingInterval,omitempty"`
	// The wait period between the last active trigger reported and scaling the resource back to 0.
	CooldownPeriod *int32 `property:"cooldown-period" json:"cooldownPeriod,omitempty"`
	// Enabling this property allows KEDA to scale the resource down to the specified number of replicas.
	IdleReplicaCount *int32 `property:"idle-replica-count" json:"idleReplicaCount,omitempty"`
	// Minimum number of replicas.
	MinReplicaCount *int32 `property:"min-replica-count" json:"minReplicaCount,omitempty"`
	// Maximum number of replicas.
	MaxReplicaCount *int32 `property:"max-replica-count" json:"maxReplicaCount,omitempty"`
	// Definition of triggers according to the KEDA format. Each trigger must contain `type` field corresponding
	// to the name of a KEDA autoscaler and a key/value map named `metadata` containing specific trigger options.
	// An optional `authentication-secret` can be declared per trigger and the operator will link each entry of
	// the secret to a KEDA authentication parameter.
	Triggers []KedaTrigger `property:"triggers" json:"triggers,omitempty"`
}

type KedaTrigger struct {
	Type                 string            `property:"type" json:"type,omitempty"`
	Metadata             map[string]string `property:"metadata" json:"metadata,omitempty"`
	AuthenticationSecret string            `property:"authentication-secret" json:"authenticationSecret,omitempty"`
	Authentication       map[string]string
}
