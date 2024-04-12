package main

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func analyzeNodes(clientset *kubernetes.Clientset, nodeStats *NodeStats) {
	cpuUsagePerNode := make(map[string]float64)
	memUsagePerNode := make(map[string]float64)
	for node, pods := range nodeStats.Data {
		for _, pod := range pods {
			for _, c := range pod.Containers {
				cpuUsagePerNode[node] += c.CPU
				memUsagePerNode[node] += c.Memory
			}
		}
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	maxPerGroup := make(map[string]*ContainerStats)
	for _, node := range nodes.Items {
		allocatableCpu := node.Status.Allocatable.Cpu().AsApproximateFloat64()
		allocatableMem := node.Status.Allocatable.Memory().AsApproximateFloat64()

		fmt.Printf("Node(name: %s, allocatable-cpu: %.0f, used-cpu: %.4f (%.2f%%), allocatable-mem: %s, used-mem: %s (%.2f%%))\n",
			node.Name,
			allocatableCpu,
			cpuUsagePerNode[node.Name],
			100*(cpuUsagePerNode[node.Name]/allocatableCpu),
			byteCountIEC(allocatableMem),
			byteCountIEC(memUsagePerNode[node.Name]),
			100*(memUsagePerNode[node.Name]/allocatableMem),
		)
		for k, v := range node.Labels {
			fmt.Printf("  Label: %s = %s\n", k, v)
		}
		for _, pod := range nodeStats.Data[node.Name] {
			cpu, mem := pod.GetTotals()
			if sts, ok := maxPerGroup[pod.Prefix]; ok {
				if cpu > sts.CPU {
					sts.CPU = cpu
				}
				if mem > sts.Memory {
					sts.Memory = mem
				}
			} else {
				maxPerGroup[pod.Prefix] = &ContainerStats{CPU: cpu, Memory: mem}
			}
			fmt.Printf("  Pod(name: %s, host-ip: %s, pod-ip: %s, used-cpu: %.4f (%.2f%%), used-mem: %s (%.2f%%))\n",
				pod.Name, pod.HostIP, pod.PodIP, cpu, 100*(cpu/allocatableCpu), byteCountIEC(mem), 100*(mem/allocatableMem))
			for _, c := range pod.Containers {
				fmt.Printf("    Container(name: %s, used-cpu: %.4f (%.2f%%), used-mem: %s (%.2f%%))\n", c.Name, c.CPU, 100*(c.CPU/allocatableCpu), byteCountIEC(c.Memory), 100*(c.Memory/allocatableMem))
			}
		}
		fmt.Println()
	}

	keys := make([]string, 0, len(maxPerGroup))
	for k := range maxPerGroup {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, group := range keys {
		if group == "" {
			continue
		}
		cpu := maxPerGroup[group].CPU * 1000
		mem := maxPerGroup[group].Memory
		fmt.Printf("Group %s, suggested-cpu: %.0fm, suggested-memory: %s\n", group, cpu, byteCountIEC(mem))
	}
}
