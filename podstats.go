package main

import (
	"strings"

	v1 "k8s.io/api/core/v1"
)

type ContainerStats struct {
	Name   string
	CPU    float64
	Memory float64
}

type PodStats struct {
	Name       string
	Namespace  string
	NodeName   string
	HostIP     string
	PodIP      string
	Prefix     string
	Containers []*ContainerStats
}

func (p *PodStats) GetTotals() (float64, float64) {
	var cpu, mem float64 = 0, 0
	for _, c := range p.Containers {
		cpu += c.CPU
		mem += c.Memory
	}
	return cpu, mem
}

func NewPodStats(pod v1.Pod) *PodStats {
	prefix := ""
	if pod.GenerateName != "" {
		prefix = pod.GenerateName[:len(pod.GenerateName)-1]
	}
	if hash, ok := pod.Labels["pod-template-hash"]; ok {
		prefix = strings.ReplaceAll(pod.GenerateName, "-"+hash+"-", "")
	}

	podStat := &PodStats{
		Namespace:  pod.Namespace,
		Name:       pod.Name,
		NodeName:   pod.Spec.NodeName,
		HostIP:     pod.Status.HostIP,
		PodIP:      pod.Status.PodIP,
		Prefix:     prefix,
		Containers: make([]*ContainerStats, 0),
	}

	for _, c := range pod.Spec.Containers {
		podStat.Containers = append(podStat.Containers, &ContainerStats{
			Name: c.Name,
		})
	}

	return podStat
}
