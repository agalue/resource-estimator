package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	MEM_PROMQL = `max_over_time(container_memory_working_set_bytes{job="kubelet", metrics_path="/metrics/cadvisor", namespace="%s", container!="", image!=""}[%s:1h])`
	CPU_PROMQL = `max_over_time(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{namespace="%s", container!=""}[%s:1h])`
)

type TenantTransport struct {
	TenantID string
}

func (t *TenantTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-Scope-OrgID", t.TenantID)
	return http.DefaultTransport.RoundTrip(req)
}

func getKubernetesClient() *kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return clientset
}

func getPrometheusClient(tenant string) promv1.API {
	promclient, err := promapi.NewClient(promapi.Config{
		Address:      prometheusURL,
		RoundTripper: &TenantTransport{TenantID: tenant},
	})
	if err != nil {
		panic(err.Error())
	}
	return promv1.NewAPI(promclient)
}

func executePromQL(cli promv1.API, queryTemplate, namespace string) model.Vector {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	promQL := fmt.Sprintf(queryTemplate, namespace, queryRange.String())

	result, warnings, err := cli.Query(ctx, promQL, time.Now())
	if err != nil {
		panic(err.Error())
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}
	return result.(model.Vector)
}

func byteCountIEC(b float64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%f B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ci",
		float64(b)/float64(div), "KMGTPE"[exp])
}
