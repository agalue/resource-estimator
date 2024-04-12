package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/util/homedir"
)

var (
	prometheusURL string
	tenant        string
	kubeconfig    string
	queryRange    time.Duration
)

func main() {
	// Define CLI Flags

	flag.StringVar(&prometheusURL, "url", "http://localhost:9090", "The base URL for a Prometheus compatible PromQL API")
	flag.StringVar(&tenant, "tenant", "anonymous", "The Tenant ID (X-Scope-OrgID) for Cortex/Mimir if applicable")
	flag.DurationVar(&queryRange, "r", 24*time.Hour, "The duration of the query aggregation")

	if home := homedir.HomeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kc", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kc", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	if queryRange.Hours() < 1 {
		fmt.Println("Invalid range; must be greater than 1 hour")
		os.Exit(1)
	}
	fmt.Printf("Executing Prometheus Query aggregations for max_over_time with [%s:1h]\n\n", queryRange)

	// Get Clients

	clientset := getKubernetesClient()
	promcli := getPrometheusClient(tenant)

	// Analyze Nodes

	nodeStats := NewNodeStats(clientset, promcli)
	analyzeNodes(clientset, nodeStats)
}
