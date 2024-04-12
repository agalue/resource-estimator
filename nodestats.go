package main

import (
	"context"
	"fmt"
	"strings"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	CPU    = "cpu"
	MEMORY = "memory"
)

type NodeStats struct {
	Data map[string][]*PodStats
}

func (n *NodeStats) appendPod(pod *PodStats) {
	node := pod.NodeName
	if n.Data[node] == nil {
		n.Data[node] = make([]*PodStats, 0)
	}
	n.Data[node] = append(n.Data[node], pod)
}

func (n *NodeStats) updateContainerStats(kind string, sample *model.Sample) {
	pod := string(sample.Metric["pod"])
	container := string(sample.Metric["container"])
	value := float64(sample.Value)

	for _, pods := range n.Data {
		for _, p := range pods {
			if p.Name == pod {
				for _, c := range p.Containers {
					if c.Name == container {
						switch kind {
						case MEMORY:
							c.Memory = value
						case CPU:
							c.CPU = value
						}
						break
					}
				}
			}
		}
	}
}

func NewNodeStats(clientset *kubernetes.Clientset, promcli promv1.API) *NodeStats {
	nodeStats := &NodeStats{Data: make(map[string][]*PodStats)}

	// Get Namespaces

	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Process Containers

	for _, ns := range namespaces.Items {
		namespace := ns.Name
		if strings.HasPrefix(namespace, "kube-") {
			continue
		}

		// Retrieve Memory Statistics per Namespace

		memData := executePromQL(promcli, MEM_PROMQL, namespace)
		cpuData := executePromQL(promcli, CPU_PROMQL, namespace)

		// Retrieve Pods

		pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != "Running" {
				fmt.Printf("Warning: pod %s is not running (%s), skipping...\n", pod.Name, pod.Status.Phase)
				continue
			}
			nodeStats.appendPod(NewPodStats(pod))
		}

		// Update Pods with Statistics

		for _, val := range memData {
			nodeStats.updateContainerStats(MEMORY, val)
		}
		for _, val := range cpuData {
			nodeStats.updateContainerStats(CPU, val)
		}
	}

	return nodeStats
}
