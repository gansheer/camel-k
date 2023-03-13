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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1beta2 "github.com/apache/camel-k/v2/addons/strimzi/duck/client/internalclientset/typed/duck/v1beta2"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeKafkaV1beta2 struct {
	*testing.Fake
}

func (c *FakeKafkaV1beta2) Kafkas(namespace string) v1beta2.KafkaInterface {
	return &FakeKafkas{c, namespace}
}

func (c *FakeKafkaV1beta2) KafkaTopics(namespace string) v1beta2.KafkaTopicInterface {
	return &FakeKafkaTopics{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeKafkaV1beta2) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
