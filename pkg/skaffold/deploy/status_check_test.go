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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestGetPods(t *testing.T) {
	labeller := NewLabeller("")
	var tests = []struct {
		description      string
		pods             []*v1.Pod
		expectedPodNames map[string]bool
		shouldErr        bool
	}{
		{
			description: "multiple pods in same namespace",
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
			expectedPodNames: map[string]bool{"pod1": true, "pod2": true},
		},
		{
			description: "no pods",
		},
		{
			description: "multiple pods in different namespaces",
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
			expectedPodNames: map[string]bool{"pod1": true},
		},
		{
			description: "pod in correct namespace but not deployed by skaffold",
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
			description: "pod in correct namespace  deployed by skaffold but previous version",
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
			var expectedPods []v1.Pod
			if test.expectedPodNames != nil {
				expectedPods = []v1.Pod{}
				for _, po := range test.pods {
					if _, ok := test.expectedPodNames[po.Name]; ok {
						expectedPods = append(expectedPods, *po)
					}
				}
			}
			actual, err := getPods(client.CoreV1().Pods("test"), labeller)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, expectedPods, actual)
		})
	}
}
