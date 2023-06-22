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

package openshift

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// IsOpenShift returns true if we are connected to a OpenShift cluster.
func IsOpenShift(client kubernetes.Interface) (bool, error) {
	_, err := client.Discovery().ServerResourcesForGroupVersion("image.openshift.io/v1")
	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// GetOpenShiftPodSecurityContext return the uid as the minimum value in the "openshift.io/sa.scc.uid-range" annotation from the namespace if present.
// TODO : manage gid and fsgroup from "openshift.io/sa.scc.supplemental-groups".
func GetOpenshiftPodUID(ctx context.Context, client kubernetes.Interface, namespace string) (int64, error) {

	ns, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return -1, fmt.Errorf("failed to get namespace %q: %w", namespace, err)
	}

	uidRange, ok := ns.ObjectMeta.Annotations["openshift.io/sa.scc.uid-range"]
	if !ok {
		return -1, errors.New("annotation 'openshift.io/sa.scc.uid-range' not found")
	}

	uidStr := strings.Split(uidRange, "/")[0]
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("failed to convert uid to integer %q: %w", uidStr, err)
	}

	return uid, nil

}
