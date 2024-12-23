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
	"fmt"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/patch"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deployerTraitID    = "deployer"
	deployerTraitOrder = 900
)

type deployerTrait struct {
	BasePlatformTrait
	traitv1.DeployerTrait `property:",squash"`
}

var _ ControllerStrategySelector = &deployerTrait{}

func newDeployerTrait() Trait {
	return &deployerTrait{
		BasePlatformTrait: NewBasePlatformTrait(deployerTraitID, deployerTraitOrder),
	}
}

func (t *deployerTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	var condition *TraitCondition
	//nolint:staticcheck
	if !ptr.Deref(t.UseSSA, true) {
		condition = NewIntegrationCondition(
			"Deployer",
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			TraitConfigurationReason,
			"The use-ssa parameter is deprecated and may be removed in future releases.",
		)
	}
	return e.Integration != nil, condition, nil
}

func (t *deployerTrait) Apply(e *Environment) error {
	// Register a post action that patches the resources generated by the traits
	e.PostActions = append(e.PostActions, func(env *Environment) error {
		applier := e.Client.ServerOrClientSideApplier()
		for _, resource := range env.Resources.Items() {
			//nolint:staticcheck
			if ptr.Deref(t.UseSSA, true) {
				if err := applier.Apply(e.Ctx, resource); err != nil {
					return err
				}
			} else {
				if err := t.clientSideApply(env, resource); err != nil {
					return err
				}
			}
		}
		return nil
	})

	return nil
}

// Deprecated: use ServerOrClientSideApplier() instead.
func (t *deployerTrait) clientSideApply(env *Environment, resource ctrl.Object) error {
	err := env.Client.Create(env.Ctx, resource)
	if err == nil {
		return nil
	} else if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("error during create resource: %s/%s: %w", resource.GetNamespace(), resource.GetName(), err)
	}
	object := &unstructured.Unstructured{}
	object.SetNamespace(resource.GetNamespace())
	object.SetName(resource.GetName())
	object.SetGroupVersionKind(resource.GetObjectKind().GroupVersionKind())
	err = env.Client.Get(env.Ctx, ctrl.ObjectKeyFromObject(object), object)
	if err != nil {
		return err
	}
	p, err := patch.MergePatch(object, resource)
	if err != nil {
		return err
	} else if len(p) == 0 {
		// Update the resource with the object returned from the API server
		return t.unstructuredToRuntimeObject(object, resource)
	}
	err = env.Client.Patch(env.Ctx, resource, ctrl.RawPatch(types.MergePatchType, p))
	if err != nil {
		return fmt.Errorf("error during patch %s/%s: %w", resource.GetNamespace(), resource.GetName(), err)
	}
	return nil
}

// Deprecated: use ServerOrClientSideApplier() instead.
func (t *deployerTrait) unstructuredToRuntimeObject(u *unstructured.Unstructured, obj ctrl.Object) error {
	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

func (t *deployerTrait) SelectControllerStrategy(e *Environment) (*ControllerStrategy, error) {
	if t.Kind != "" {
		strategy := ControllerStrategy(t.Kind)
		return &strategy, nil
	}
	return nil, nil
}

func (t *deployerTrait) ControllerStrategySelectorOrder() int {
	return 0
}

// RequiresIntegrationPlatform overrides base class method.
func (t *deployerTrait) RequiresIntegrationPlatform() bool {
	return false
}
