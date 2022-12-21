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

package integration

import (
	"context"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/log"
	"github.com/apache/camel-k/pkg/util/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/stretchr/testify/assert"
)

func TestCanHandle_CanHandle(t *testing.T) {
	t.Helper()

	integration := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Builder: &traitv1.BuilderTrait{
					Trait: traitv1.Trait{
						Enabled: pointer.Bool(true),
					},
				},
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseError,
		},
	}

	a := monitorAction{}
	a.InjectLogger(log.Log)

	canHandleResult := a.CanHandle(integration)

	assert.Equal(t, true, canHandleResult)
}

func TestHandle_Handle(t *testing.T) {
	t.Helper()
	kit := &v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration-kit",
			Labels: map[string]string{
				v1.IntegrationKitTypeLabel: v1.IntegrationKitTypePlatform,
			},
		},
		Spec: v1.IntegrationKitSpec{
			Dependencies: []string{
				"camel-core",
				"camel-irc",
			},
		},
		Status: v1.IntegrationKitStatus{
			Phase: v1.IntegrationKitPhaseReady,
		},
	}

	integration := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-integration",
		},
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Builder: &traitv1.BuilderTrait{
					Trait: traitv1.Trait{
						Enabled: pointer.Bool(true),
					},
				},
			},
		},
		Status: v1.IntegrationStatus{
			Phase:  v1.IntegrationPhaseDeploying,
			Digest: "fakefake",
			IntegrationKit: &corev1.ObjectReference{
				Namespace: "ns",
				Name:      "my-integration-kit",
			},
			Conditions: []v1.IntegrationCondition{
				{Type: v1.IntegrationConditionPlatformAvailable, Status: corev1.ConditionTrue},
				{Type: v1.IntegrationConditionKitAvailable, Status: corev1.ConditionTrue},
				{Type: v1.IntegrationConditionDeploymentAvailable, Status: corev1.ConditionTrue},
				{Type: v1.IntegrationConditionReady},
			},
		},
	}

	mydigest, _ := digest.ComputeForIntegration(integration)
	integration.Status.Digest = mydigest

	c, err := test.NewFakeClient(kit)

	assert.Nil(t, err)

	a := monitorAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)

	handleIntegration, handleError := a.Handle(context.TODO(), integration)

	assert.Nil(t, handleError)
	assert.NotNil(t, handleIntegration)
	assert.Equal(t, v1.IntegrationPhaseInitialization, handleIntegration.Status.Phase)

}
