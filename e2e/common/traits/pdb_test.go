//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package traits

import (
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestPodDisruptionBudgetTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-traits-pdb"
		g.Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		name := RandomizedSuffixName("java")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "pdb.enabled=true",
			"-t", "pdb.min-available=2",
		).Execute()).To(Succeed())

		g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// check integration schema does not contains unwanted default trait value.
		g.Eventually(UnstructuredIntegration(t, ns, name)).ShouldNot(BeNil())
		unstructuredIntegration := UnstructuredIntegration(t, ns, name)()
		pdbTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "pdb")
		g.Expect(pdbTrait).ToNot(BeNil())
		g.Expect(len(pdbTrait)).To(Equal(2))
		g.Expect(pdbTrait["enabled"]).To(Equal(true))
		g.Expect(pdbTrait["minAvailable"]).To(Equal("2"))

		// Check PodDisruptionBudget
		g.Eventually(podDisruptionBudget(t, ns, name), TestTimeoutShort).ShouldNot(BeNil())
		pdb := podDisruptionBudget(t, ns, name)()
		// Assert PDB Spec
		g.Expect(pdb.Spec.MinAvailable).To(PointTo(Equal(intstr.FromInt(2))))
		// Assert PDB Status
		g.Eventually(podDisruptionBudget(t, ns, name), TestTimeoutShort).
			Should(MatchFieldsP(IgnoreExtras, Fields{
				"Status": MatchFields(IgnoreExtras, Fields{
					"ObservedGeneration": BeNumerically("==", 1),
					"DisruptionsAllowed": BeNumerically("==", 0),
					"CurrentHealthy":     BeNumerically("==", 1),
					"DesiredHealthy":     BeNumerically("==", 2),
					"ExpectedPods":       BeNumerically("==", 1),
				}),
			}))

		// Scale Integration
		g.Expect(ScaleIntegration(t, ns, name, 2)).To(Succeed())
		g.Eventually(IntegrationPods(t, ns, name), TestTimeoutMedium).Should(HaveLen(2))
		g.Eventually(IntegrationStatusReplicas(t, ns, name), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 2)))
		g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// Check PodDisruptionBudget
		pdb = podDisruptionBudget(t, ns, name)()
		g.Expect(pdb).NotTo(BeNil())
		// Assert PDB Status according to the scale change
		g.Eventually(podDisruptionBudget(t, ns, name), TestTimeoutShort).
			Should(MatchFieldsP(IgnoreExtras, Fields{
				"Status": MatchFields(IgnoreExtras, Fields{
					"ObservedGeneration": BeNumerically("==", 1),
					"DisruptionsAllowed": BeNumerically("==", 0),
					"CurrentHealthy":     BeNumerically("==", 2),
					"DesiredHealthy":     BeNumerically("==", 2),
					"ExpectedPods":       BeNumerically("==", 2),
				}),
			}))

		// Eviction attempt
		pods := IntegrationPods(t, ns, name)()
		g.Expect(pods).To(HaveLen(2))
		err := TestClient(t).CoreV1().Pods(ns).EvictV1(TestContext, &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name: pods[0].Name,
			},
		})
		g.Expect(err).To(MatchError(&k8serrors.StatusError{
			ErrStatus: metav1.Status{
				Status:  "Failure",
				Message: "Cannot evict pod as it would violate the pod's disruption budget.",
				Reason:  "TooManyRequests",
				Code:    http.StatusTooManyRequests,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    "DisruptionBudget",
							Message: "The disruption budget " + name + " needs 2 healthy pods and has 2 currently",
						},
					},
				},
			},
		}))

		// Scale Integration to Scale > PodDisruptionBudgetSpec.MinAvailable
		// for the eviction request to succeed once replicas are ready
		g.Expect(ScaleIntegration(t, ns, name, 3)).To(Succeed())
		g.Eventually(IntegrationPods(t, ns, name), TestTimeoutMedium).Should(HaveLen(3))
		g.Eventually(IntegrationStatusReplicas(t, ns, name), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 3)))
		g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		pods = IntegrationPods(t, ns, name)()
		g.Expect(pods).To(HaveLen(3))
		g.Expect(TestClient(t).CoreV1().Pods(ns).EvictV1(TestContext, &policyv1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name: pods[0].Name,
			},
		})).To(Succeed())

		// Clean up
		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func podDisruptionBudget(t *testing.T, ns string, name string) func() *policyv1.PodDisruptionBudget {
	return func() *policyv1.PodDisruptionBudget {
		pdb := policyv1.PodDisruptionBudget{
			TypeMeta: metav1.TypeMeta{
				APIVersion: policyv1.SchemeGroupVersion.String(),
				Kind:       "PodDisruptionBudget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}
		err := TestClient(t).Get(TestContext, ctrl.ObjectKeyFromObject(&pdb), &pdb)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &pdb
	}
}
