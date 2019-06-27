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
	"context"
	"sync"
	"time"

	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// TODO: Move this to a flag or global config.
	// Default deadline set to 10 minutes. This is default value for progressDeadlineInSeconds
	// See: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/apps/v1/types.go#L305
	defaultStatusCheckDeadlineInSeconds int32 = 600
	// Poll period for checking set to 100 milliseconds
	defaultPollPeriodInMilliseconds = 100
)

func StatusCheckPods(ctx context.Context, defaultLabeller *DefaultLabeller, runCtx *runcontext.RunContext) error {

	client, err := kubernetesutil.GetClientset()
	if err != nil {
		return err
	}
	podInterface := client.CoreV1().Pods(runCtx.Opts.Namespace)
	pods, err := getPods(podInterface, defaultLabeller)
	if err != nil {
		return errors.Wrap(err, "could not fetch deployments")
	}

	wg := sync.WaitGroup{}
	// Its safe to use sync.Map without locks here as each subroutine adds a different key to the map.
	syncMap := &sync.Map{}

	for _, po := range pods {
		deadlineDuration := time.Duration(defaultStatusCheckDeadlineInSeconds) * time.Second
		wg.Add(1)
		go func(po *v1.Pod) {
			defer wg.Done()
			getPodStatus(ctx, podInterface, po, deadlineDuration, syncMap)
		}(&po)
	}

	// Wait for all deployment status to be fetched
	wg.Wait()
	return nil
}

func getPods(pi corev1.PodInterface, l *DefaultLabeller) ([]v1.Pod, error) {
	pods, err := pi.List(metav1.ListOptions{
		LabelSelector: l.K8sManagedByLabelKeyValueString(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch deployments")
	}
	return pods.Items, nil
}

func getPodStatus(ctx context.Context, pi corev1.PodInterface, po *v1.Pod, deadline time.Duration, syncMap *sync.Map) {
	err := kubernetesutil.WaitForPodToStabilize(ctx, pi, po.Name, deadline)
	syncMap.Store(po.Name, err)
}
