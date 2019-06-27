/*
Copyright 2019 The Skaffold Authors

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

package deploy

import (
	"fmt"
	"sync"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

)

func TestGetPods(t *testing.T) {
	labeller := NewLabeller("")
	var tests = []struct {
		description string
		pods []*v1.Pod
		shouldErr   bool
	}{
		{
			description: "multiple deployments in same namespace",
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
							"random":            "foo",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
				},
			},

		},
		{
			description: "multiple deployments with no progress deadline set",
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
				},
			},

		},
		{
			description: "no deployments",

		},
		{
			description: "multiple deployments in different namespaces",
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod2",
						Namespace: "test1",
						Labels: map[string]string{
							K8ManagedByLabelKey: labeller.skaffoldVersion(),
						},
					},
				},
			},

		},
		{
			description: "deployment in correct namespace but not deployed by skaffold",
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							"some-other-tool": "helm",
						},
					},
				},
			},

		},
		{
			description: "deployment in correct namespace  deployed by skaffold but previous version",
			pods: []*v1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1",
						Namespace: "test",
						Labels: map[string]string{
							K8ManagedByLabelKey: "skaffold-0.26.0",
						},
					},
				},
			},

		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			objs := make([]runtime.Object, len(test.pods))
			for i, dep := range test.pods {
				objs[i] = dep
			}
			client := fakekubeclientset.NewSimpleClientset(objs...)
			actual, err := getPods(client.CoreV1().Pods("test"), labeller)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, nil, actual)
		})
	}
}

func TestGetPodStatus(t *testing.T) {
	var tests = []struct {
		description    string
		deps           map[string]interface{}
		expectedErrMsg []string
		shouldErr      bool
	}{
		{
			description: "one error",
			deps: map[string]interface{}{
				"pod1": "SUCCESS",
				"pod2": fmt.Errorf("could not return within default timeout"),
			},
			expectedErrMsg: []string{"deployment pod2 failed due to could not return within default timeout"},
			shouldErr:      true,
		},
		{
			description: "no error",
			deps: map[string]interface{}{
				"pod1": "SUCCESS",
				"pod2": "RUNNING",
			},
		},
		{
			description: "multiple errors",
			deps: map[string]interface{}{
				"pod1": "SUCCESS",
				"pod2": fmt.Errorf("could not return within default timeout"),
				"pod3": fmt.Errorf("ERROR"),
			},
			expectedErrMsg: []string{"deployment pod2 failed due to could not return within default timeout",
				"deployment pod3 failed due to ERROR"},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			syncMap := &sync.Map{}
			for k, v := range test.deps {
				syncMap.Store(k, v)
			}
			//err := getPodStatus(syncMap)
			//t.CheckError(test.shouldErr, err)
			//for _, msg := range test.expectedErrMsg {
			//	t.CheckErrorContains(msg, err)
			//}
		})
	}
}
