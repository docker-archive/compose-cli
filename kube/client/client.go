// +build kube

/*
   Copyright 2021 Docker Compose CLI authors

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

package client

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/utils"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// KubeClient API to access kube objects
type KubeClient struct {
	client    *kubernetes.Clientset
	namespace string
}

// NewKubeClient new kubernetes client
func NewKubeClient(config genericclioptions.RESTClientGetter) (*KubeClient, error) {
	restConfig, err := config.ToRESTConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	namespace, _, err := config.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		client:    clientset,
		namespace: namespace,
	}, nil
}

// GetContainers get containers for a given compose project
func (kc KubeClient) GetContainers(ctx context.Context, projectName string, all bool) ([]compose.ContainerSummary, error) {
	fieldSelector := ""
	if !all {
		fieldSelector = "status.phase=Running"
	}

	pods, err := kc.client.CoreV1().Pods(kc.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", compose.ProjectTag, projectName),
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}
	result := []compose.ContainerSummary{}
	for _, pod := range pods.Items {
		result = append(result, podToContainerSummary(pod))
	}

	return result, nil
}

func podToContainerSummary(pod corev1.Pod) compose.ContainerSummary {
	return compose.ContainerSummary{
		ID:      pod.GetObjectMeta().GetName(),
		Name:    pod.GetObjectMeta().GetName(),
		Service: pod.GetObjectMeta().GetLabels()[compose.ServiceTag],
		State:   string(pod.Status.Phase),
		Project: pod.GetObjectMeta().GetLabels()[compose.ProjectTag],
	}
}

// GetLogs retrieves pod logs
func (kc *KubeClient) GetLogs(ctx context.Context, projectName string, consumer compose.LogConsumer, follow bool) error {
	pods, err := kc.client.CoreV1().Pods(kc.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", compose.ProjectTag, projectName),
	})
	if err != nil {
		return err
	}
	eg, ctx := errgroup.WithContext(ctx)
	for _, pod := range pods.Items {
		request := kc.client.CoreV1().Pods(kc.namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Follow: follow})
		service := pod.Labels[compose.ServiceTag]
		w := utils.GetWriter(pod.Name, service, string(pod.UID), func(event compose.ContainerEvent) {
			consumer.Log(event.Name, event.Service, event.Source, event.Line)
		})

		eg.Go(func() error {
			r, err := request.Stream(ctx)
			defer r.Close() // nolint errcheck
			if err != nil {
				return err
			}
			_, err = io.Copy(w, r)
			return err
		})
	}
	return eg.Wait()
}
