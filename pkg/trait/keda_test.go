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

import (
	"context"
	"fmt"
	"testing"

	"github.com/apache/camel-k/v2/addons/keda/duck/v1alpha1"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
)

func TestManualConfig(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = ptr.To(true)
	keda.Auto = ptr.To(false)
	meta := map[string]string{
		"prop":      "val",
		"camelCase": "VAL",
	}
	keda.Triggers = append(keda.Triggers, traitv1.KedaTrigger{
		Type:     "mytype",
		Metadata: meta,
	})
	env := createBasicTestEnvironment()

	res, condition, err := keda.Configure(env)
	require.NoError(t, err)
	assert.True(t, res)
	assert.Nil(t, condition)
	require.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "mytype", so.Spec.Triggers[0].Type)
	assert.Equal(t, meta, so.Spec.Triggers[0].Metadata)
	assert.Nil(t, so.Spec.Triggers[0].AuthenticationRef)
	assert.Nil(t, getTriggerAuthentication(env))
	assert.Nil(t, getSecret(env))
}

func TestConfigFromSecret(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = ptr.To(true)
	keda.Auto = ptr.To(false)
	meta := map[string]string{
		"prop":      "val",
		"camelCase": "VAL",
	}
	keda.Triggers = append(keda.Triggers, traitv1.KedaTrigger{
		Type: "mytype",
		//require.NoError(t, err)
		Metadata:             meta,
		AuthenticationSecret: "my-secret",
	})
	env := createBasicTestEnvironment(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-secret",
		},
		Data: map[string][]byte{
			"bbb": []byte("val1"),
			"aaa": []byte("val2"),
		},
	})

	res, condition, err := keda.Configure(env)
	require.NoError(t, err)
	assert.True(t, res)
	assert.Nil(t, condition)
	require.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "mytype", so.Spec.Triggers[0].Type)
	assert.Equal(t, meta, so.Spec.Triggers[0].Metadata)
	triggerAuth := getTriggerAuthentication(env)
	assert.NotNil(t, triggerAuth)
	assert.Equal(t, so.Spec.Triggers[0].AuthenticationRef.Name, triggerAuth.Name)
	assert.NotEqual(t, "my-secret", triggerAuth.Name)
	assert.Len(t, triggerAuth.Spec.SecretTargetRef, 2)
	assert.Equal(t, "aaa", triggerAuth.Spec.SecretTargetRef[0].Key)
	assert.Equal(t, "aaa", triggerAuth.Spec.SecretTargetRef[0].Parameter)
	assert.Equal(t, "my-secret", triggerAuth.Spec.SecretTargetRef[0].Name)
	assert.Equal(t, "bbb", triggerAuth.Spec.SecretTargetRef[1].Key)
	assert.Equal(t, "bbb", triggerAuth.Spec.SecretTargetRef[1].Parameter)
	assert.Equal(t, "my-secret", triggerAuth.Spec.SecretTargetRef[1].Name)
	assert.Nil(t, getSecret(env)) // Secret is already present, not generated
}

func TestKameletAutoDetection(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = ptr.To(true)
	env := createBasicTestEnvironment(
		&v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-kamelet",
				Annotations: map[string]string{
					"camel.apache.org/keda.type": "my-scaler",
				},
			},
			Spec: v1.KameletSpec{
				KameletSpecBase: v1.KameletSpecBase{
					Definition: &v1.JSONSchemaProps{
						Properties: map[string]v1.JSONSchemaProp{
							"a": {
								XDescriptors: []string{
									"urn:keda:metadata:a",
								},
							},
							"b": {
								XDescriptors: []string{
									"urn:keda:metadata:bb",
								},
							},
							"c": {
								XDescriptors: []string{
									"urn:keda:authentication:cc",
								},
							},
						},
					},
				},
			},
		},
		&v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-it",
			},
			Spec: v1.IntegrationSpec{
				Sources: []v1.SourceSpec{
					{
						DataSpec: v1.DataSpec{
							Name: "my-it.yaml",
							Content: "" +
								"- route:\n" +
								"    from:\n" +
								"      uri: kamelet:my-kamelet\n" +
								"      parameters:\n" +
								"        a: v1\n" +
								"        b: v2\n" +
								"        c: v3\n" +
								"    steps:\n" +
								"    - to: log:sink\n",
						},
						Language: v1.LanguageYaml,
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		})

	res, condition, err := keda.Configure(env)
	require.NoError(t, err)
	assert.True(t, res)
	assert.Nil(t, condition)
	require.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "my-scaler", so.Spec.Triggers[0].Type)
	assert.Equal(t, map[string]string{
		"a":  "v1",
		"bb": "v2",
	}, so.Spec.Triggers[0].Metadata)
	triggerAuth := getTriggerAuthentication(env)
	assert.NotNil(t, triggerAuth)
	assert.Equal(t, so.Spec.Triggers[0].AuthenticationRef.Name, triggerAuth.Name)
	assert.Len(t, triggerAuth.Spec.SecretTargetRef, 1)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Key)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Parameter)
	secretName := triggerAuth.Spec.SecretTargetRef[0].Name
	secret := getSecret(env)
	assert.NotNil(t, secret)
	assert.Equal(t, secretName, secret.Name)
	assert.Len(t, secret.StringData, 1)
	assert.Contains(t, secret.StringData, "cc")
}

func TestPipeAutoDetection(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = ptr.To(true)
	logEndpoint := "log:info"
	klb := v1.Pipe{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "my-binding",
		},
		Spec: v1.PipeSpec{
			Source: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "my-kamelet",
				},
				Properties: asEndpointProperties(map[string]string{
					"a": "v1",
					"b": "v2",
					"c": "v3",
				}),
			},
			Sink: v1.Endpoint{
				URI: &logEndpoint,
			},
		},
	}

	env := createBasicTestEnvironment(
		&v1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-kamelet",
				Annotations: map[string]string{
					"camel.apache.org/keda.type": "my-scaler",
				},
			},
			Spec: v1.KameletSpec{
				KameletSpecBase: v1.KameletSpecBase{
					Definition: &v1.JSONSchemaProps{
						Properties: map[string]v1.JSONSchemaProp{
							"a": {
								XDescriptors: []string{
									"urn:keda:metadata:a",
								},
							},
							"b": {
								XDescriptors: []string{
									"urn:keda:metadata:bb",
								},
							},
							"c": {
								XDescriptors: []string{
									"urn:keda:authentication:cc",
								},
							},
						},
					},
				},
			},
		},
		&klb,
		&v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "camel-k",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
				Profile: v1.TraitProfileKubernetes,
			},
			Status: v1.IntegrationPlatformStatus{
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		})

	// lifecycle error here on pipe
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-binding",
			Namespace: "test",
			Labels: map[string]string{
				"camel.apache.org/created.by.kind": "",
				"camel.apache.org/created.by.name": "my-binding",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         klb.APIVersion,
					Kind:               klb.Kind,
					Name:               klb.Name,
					UID:                klb.UID,
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(true),
				},
			},
		},
		Spec: v1.IntegrationSpec{
			Profile: v1.DefaultTraitProfile,
			Configuration: []v1.ConfigurationSpec{
				{
					Type:  "property",
					Value: "camel.kamelet.my-kamelet.source.a = v1",
				},
				{
					Type:  "property",
					Value: "camel.kamelet.my-kamelet.source.b = v2",
				},
				{
					Type:  "property",
					Value: "camel.kamelet.my-kamelet.source.c = v3",
				},
			},
		},
	}

	flowRoute := map[string]interface{}{
		"route": map[string]interface{}{
			"id": "binding",
			"from": map[string]interface{}{
				"uri": "kamelet:my-kamelet/source",
				"steps": []interface{}{map[string]string{
					"to": "log:info",
				}},
			},
		},
	}
	encodedRoute, err := json.Marshal(flowRoute)
	require.NoError(t, err)
	it.Spec.Flows = append(it.Spec.Flows, v1.Flow{RawMessage: encodedRoute})
	//it, err := pipe.CreateIntegrationFor(env.Ctx, env.Client, &klb)
	//assert.NotNil(t, it)
	env.Integration = it

	it.Status.Phase = v1.IntegrationPhaseInitialization
	init := NewInitTrait()
	ok, condition, err := init.Configure(env)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Nil(t, condition)
	require.NoError(t, init.Apply(env))

	it.Status.Phase = v1.IntegrationPhaseDeploying
	res, condition, err := keda.Configure(env)
	require.NoError(t, err)
	assert.True(t, res)
	assert.Nil(t, condition)
	require.NoError(t, keda.Apply(env))
	so := getScaledObject(env)
	assert.NotNil(t, so)
	assert.Len(t, so.Spec.Triggers, 1)
	assert.Equal(t, "my-scaler", so.Spec.Triggers[0].Type)
	assert.Equal(t, map[string]string{
		"a":  "v1",
		"bb": "v2",
	}, so.Spec.Triggers[0].Metadata)
	triggerAuth := getTriggerAuthentication(env)
	assert.NotNil(t, triggerAuth)
	assert.Equal(t, so.Spec.Triggers[0].AuthenticationRef.Name, triggerAuth.Name)
	assert.Len(t, triggerAuth.Spec.SecretTargetRef, 1)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Key)
	assert.Equal(t, "cc", triggerAuth.Spec.SecretTargetRef[0].Parameter)
	secretName := triggerAuth.Spec.SecretTargetRef[0].Name
	secret := getSecret(env)
	assert.NotNil(t, secret)
	assert.Equal(t, secretName, secret.Name)
	assert.Len(t, secret.StringData, 1)
	assert.Contains(t, secret.StringData, "cc")
}

func TestHackReplicas(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = ptr.To(true)
	keda.Auto = ptr.To(false)
	keda.Triggers = append(keda.Triggers, traitv1.KedaTrigger{
		Type: "custom",
		Metadata: map[string]string{
			"a": "b",
		},
	})
	keda.HackControllerReplicas = ptr.To(true)
	env := createBasicTestEnvironment(
		&v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-it",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	)

	res, condition, err := keda.Configure(env)
	require.NoError(t, err)
	assert.True(t, res)
	assert.Nil(t, condition)
	require.NoError(t, keda.Apply(env))
	scalesClient, err := env.Client.ScalesClient()
	require.NoError(t, err)
	sc, err := scalesClient.Scales("test").Get(env.Ctx, v1.SchemeGroupVersion.WithResource("integrations").GroupResource(), "my-it", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), sc.Spec.Replicas)
}

func TestHackKLBReplicas(t *testing.T) {
	keda, _ := NewKedaTrait().(*kedaTrait)
	keda.Enabled = ptr.To(true)
	keda.Auto = ptr.To(false)
	keda.Triggers = append(keda.Triggers, traitv1.KedaTrigger{
		Type: "custom",
		Metadata: map[string]string{
			"a": "b",
		},
	})
	keda.HackControllerReplicas = ptr.To(true)
	env := createBasicTestEnvironment(
		&v1.Pipe{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-klb",
			},
		},
		&v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "my-it",
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: v1.SchemeGroupVersion.String(),
						Kind:       "Pipe",
						Name:       "my-klb",
					},
				},
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseInitialization,
			},
		},
	)

	res, condition, err := keda.Configure(env)
	require.NoError(t, err)
	assert.True(t, res)
	assert.Nil(t, condition)
	require.NoError(t, keda.Apply(env))
	scalesClient, err := env.Client.ScalesClient()
	require.NoError(t, err)
	sc, err := scalesClient.Scales("test").Get(env.Ctx, v1.SchemeGroupVersion.WithResource("pipes").GroupResource(), "my-klb", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, int32(1), sc.Spec.Replicas)
}

func getScaledObject(e *Environment) *v1alpha1.ScaledObject {
	var res *v1alpha1.ScaledObject
	for _, o := range e.Resources.Items() {
		if so, ok := o.(*v1alpha1.ScaledObject); ok {
			if res != nil {
				panic("multiple ScaledObjects found in env")
			}
			res = so
		}
	}
	return res
}

func getTriggerAuthentication(e *Environment) *v1alpha1.TriggerAuthentication {
	var res *v1alpha1.TriggerAuthentication
	for _, o := range e.Resources.Items() {
		if so, ok := o.(*v1alpha1.TriggerAuthentication); ok {
			if res != nil {
				panic("multiple TriggerAuthentication found in env")
			}
			res = so
		}
	}
	return res
}

func getSecret(e *Environment) *corev1.Secret {
	var res *corev1.Secret
	for _, o := range e.Resources.Items() {
		if so, ok := o.(*corev1.Secret); ok {
			if res != nil {
				panic("multiple Secret found in env")
			}
			res = so
		}
	}
	return res
}

func createBasicTestEnvironment(resources ...runtime.Object) *Environment {
	fakeClient, err := test.NewFakeClient(resources...)
	if err != nil {
		panic(fmt.Errorf("could not create fake client: %w", err))
	}

	var it *v1.Integration
	for _, res := range resources {
		if integration, ok := res.(*v1.Integration); ok {
			it = integration
		}
	}
	if it == nil {
		it = &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		}
	}

	var pl *v1.IntegrationPlatform
	for _, res := range resources {
		if platform, ok := res.(*v1.IntegrationPlatform); ok {
			pl = platform
		}
	}
	if pl == nil {
		pl = &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "camel-k",
			},
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterKubernetes,
				Profile: v1.TraitProfileKubernetes,
			},
		}
	}

	camelCatalog, _ := camel.DefaultCatalog()

	return &Environment{
		Catalog:               NewCatalog(nil),
		Ctx:                   context.Background(),
		Client:                fakeClient,
		Integration:           it,
		CamelCatalog:          camelCatalog,
		Platform:              pl,
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
}

func asEndpointProperties(props map[string]string) *v1.EndpointProperties {
	serialized, err := json.Marshal(props)
	if err != nil {
		panic(err)
	}
	return &v1.EndpointProperties{
		RawMessage: serialized,
	}
}
