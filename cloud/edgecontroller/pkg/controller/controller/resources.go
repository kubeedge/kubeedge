package controller

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"sync"
)

type resources struct {
	Pods       []corev1.Pod        `json:"pods,omitempty"`
	ConfigMaps []*corev1.ConfigMap `json:"configMaps,omitempty"`
	Secrets    []*corev1.Secret    `json:"secrets,omitempty"`
	Error      string              `json:"error,omitempty"`
}

func listResourcesOnNode(client kubernetes.Interface, nodeID, namespace string) (*resources, error) {
	pods, configMaps, secrets, err := listResources(client, nodeID, namespace)
	if err != nil {
		return nil, err
	}
	filteredConfigMaps, filteredSecrets := filterConfigMapsAndSecrets(pods, configMaps, secrets)
	r := new(resources)
	r.Pods = pods
	r.ConfigMaps = make([]*corev1.ConfigMap, 0, len(filteredConfigMaps))
	for _, configMap := range filteredConfigMaps {
		r.ConfigMaps = append(r.ConfigMaps, configMap)
	}
	r.Secrets = make([]*corev1.Secret, 0, len(filteredSecrets))
	for _, secret := range filteredSecrets {
		r.Secrets = append(r.Secrets, secret)
	}
	return r, nil
}

func filterConfigMapsAndSecrets(pods []corev1.Pod, configMaps map[string]*corev1.ConfigMap, secrets map[string]*corev1.Secret) (
	filteredConfigMaps map[string]*corev1.ConfigMap, filteredSecrets map[string]*corev1.Secret) {

	filteredConfigMaps = make(map[string]*corev1.ConfigMap)
	filteredSecrets = make(map[string]*corev1.Secret)
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			envs := container.Env
			if envs == nil {
				continue
			}
			for _, env := range envs {
				valueFrom := env.ValueFrom
				if valueFrom == nil {
					continue
				}
				if configMap := valueFrom.ConfigMapKeyRef; configMap != nil {
					filteredConfigMaps[configMap.Name] = configMaps[configMap.Name]
				}
				if secret := valueFrom.SecretKeyRef; secret != nil {
					filteredSecrets[secret.Name] = secrets[secret.Name]
				}
			}
		}
		volumes := pod.Spec.Volumes
		if volumes == nil {
			continue
		}
		for _, volume := range volumes {
			if configMap := volume.ConfigMap; configMap != nil {
				filteredConfigMaps[configMap.Name] = configMaps[configMap.Name]
			}
			if secret := volume.Secret; secret != nil {
				filteredSecrets[secret.SecretName] = secrets[secret.SecretName]
			}
		}
	}
	return
}

func listResources(client kubernetes.Interface, nodeID, namespace string) (
	pods []corev1.Pod, configMaps map[string]*corev1.ConfigMap, secrets map[string]*corev1.Secret, err error) {

	join := new(sync.WaitGroup)
	join.Add(3)
	errs := new(sync.Map)
	go func() {
		defer join.Done()
		p, e := listPods(client, namespace, fmt.Sprintf("spec.nodeName=%s", nodeID))
		if e != nil {
			errs.Store("pods", e)
		}
		pods = p
	}()
	go func() {
		defer join.Done()
		c, e := listConfigMaps(client, namespace)
		if e != nil {
			errs.Store("configMap", e)
		}
		configMaps = c
	}()
	go func() {
		defer join.Done()
		s, e := listSecrets(client, namespace)
		if e != nil {
			errs.Store("secret", e)
		}
		secrets = s
	}()
	join.Wait()
	err = checkErrors(errs)
	return
}

func checkErrors(errs *sync.Map) (err error) {
	es := make([]error, 0)
	errs.Range(func(_, value interface{}) bool {
		es = append(es, value.(error))
		return true
	})
	if count := len(es); count == 1 {
		err = es[0]
	} else if count > 1 {
		err = errors.NewAggregate(es)
	}
	return
}

func listPods(client kubernetes.Interface, namespace, selector string) (pods []corev1.Pod, err error) {
	var list *corev1.PodList
	if list, err = client.CoreV1().Pods(namespace).List(metav1.ListOptions{FieldSelector: selector}); err != nil {
		return
	}
	pods = list.Items
	return
}

func listConfigMaps(client kubernetes.Interface, namespace string) (configMaps map[string]*corev1.ConfigMap, err error) {
	var list *corev1.ConfigMapList
	if list, err = client.CoreV1().ConfigMaps(namespace).List(metav1.ListOptions{}); err != nil {
		return
	}
	if list.Items == nil {
		return
	}
	count := len(list.Items)
	configMaps = make(map[string]*corev1.ConfigMap, count)
	for i := 0; i < count; i++ {
		configMap := &(list.Items[i])
		configMaps[configMap.Name] = configMap
	}
	return
}

func listSecrets(client kubernetes.Interface, namespace string) (secrets map[string]*corev1.Secret, err error) {
	var list *corev1.SecretList
	if list, err = client.CoreV1().Secrets(namespace).List(metav1.ListOptions{}); err != nil {
		return
	}
	if list.Items == nil {
		return
	}
	count := len(list.Items)
	secrets = make(map[string]*corev1.Secret, count)
	for i := 0; i < count; i++ {
		secret := &(list.Items[i])
		secrets[secret.Name] = secret
	}
	return
}
