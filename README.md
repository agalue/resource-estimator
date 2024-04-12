# Kubernetes Resource Estimator Tool

This tool aims to analyze all the pods from all the namespaces (except system ones) and suggest CPU and Memory for each resource based on the maximums calculated from the Prometheus Statistics.

For that purpose, Prometheus and the Prometheus Kube State Metrics must be running in your Kubernetes cluster.

## Calculate Maximum CPU Usage

```
max_over_time(
    node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{
        namespace="xxxx",
        container!=""
    }
    [24h:1h]
)
```

The source is the Recording Rule from the Pod Usage dashboard. The query will give you the [max_over_time](https://prometheus.io/docs/prometheus/latest/querying/functions/#aggregation_over_time) for all the containers on a given namespace for the last 24 hours (a range is configurable through the script).

## Calculate Maximum Memory Usage

```
max_over_time(
    container_memory_working_set_bytes{
        job="kubelet",
        metrics_path="/metrics/cadvisor",
        namespace="xxxx",
        container!="",
        image!=""
    }
    [24h:1h]
)
```

The source is the from the Pod Usage dashboard. The query will give you the [max_over_time](https://prometheus.io/docs/prometheus/latest/querying/functions/#aggregation_over_time) for all the containers on a given namespace for the last 24 hours (a range is configurable through the script).

## Compile

Assuming you run the tool from a Linux machine, run the following from the main folder after cloning the repository.

```bash
GOOS=linux GOARCH=amd64 go build -o resource-estimator .
```

## Execute

The tool will extract the `kubeconfig` from your local directory to communicate with your Kubernetes cluster and assume Prometheus is exposed as `http://localhost:9090`. You can tune the URL with the `-url` parameter, although the current code doesn't support authentication. However, when using Cortex, Mimir, or Thanos, you can pass the TenantID via the `-tenant` parameter.

The easiest way to have access is via `port-forward`, for instance:

For Prometheus:
```bash
kubectl port-forward -n observability svc/monitor-prometheus 9090
```

With the above, the tool runs without alterations (assuming your `kubeconfig` gives you read access to all the namespaces).

For Mimir via Gateway:
```bash
kubectl port-forward -n mimir svc/mimir-gateway 9090:80
```

With the above, the tools needs an adjustment to run. For instance, with multi-tenancy disabled:
```bash
resource-estimator -url http://localhost:9090/prometheus
```

With multi-tenancy and targeting the Query Frontend:
```bash
kubectl port-forward -n mimir svc/mimir-query-frontend 9090:8080
```

Then,

```bash
resource-estimator -url http://localhost:9090/prometheus -tenant _local
```

## Understand results

The script will produce statistics per node, including pod and container details. For instance:

```
Node(name: ip-10-10-1-1.ec2.internal, allocatable-cpu: 4, used-cpu: 1.3199 (33.67%), allocatable-mem: 14.5Gi, used-mem: 5.2Gi (35.84%))
  Label: eks.amazonaws.com/capacityType = ON_DEMAND
  Label: kubernetes.io/arch = amd64
  Label: topology.kubernetes.io/region = us-east-1
  Label: topology.kubernetes.io/zone = us-east-1c
  Label: beta.kubernetes.io/instance-type = t3.xlarge
  Label: kubernetes.io/hostname = ip-10-10-1-1.ec2.internal
  Label: topology.ebs.csi.aws.com/zone = us-east-1c
  Label: beta.kubernetes.io/arch = amd64
  Label: node.kubernetes.io/instance-type = t3.xlarge
  Label: beta.kubernetes.io/os = linux
  Label: failure-domain.beta.kubernetes.io/region = us-east-1
  Label: kubernetes.io/os = linux
  Pod(name: monitoring-prometheus-node-exporter-99n4b, host-ip: 10.10.1.1, pod-ip: 10.10.1.1, used-cpu: 0.0013 (0.03%), used-mem: 11.2Mi (0.08%))
    Container(name: node-exporter, used-cpu: 0.0013 (0.03%), used-mem: 11.2Mi (0.08%))
  Pod(name: tempo-compactor-5fff987c95-w9ntv, host-ip: 10.10.1.1, pod-ip: 10.10.1.11, used-cpu: 0.4829 (12.32%), used-mem: 1.7Gi (11.79%))
    Container(name: compactor, used-cpu: 0.4829 (12.32%), used-mem: 1.7Gi (11.79%))
  Pod(name: tempo-distributor-868bdfd984-7drtb, host-ip: 10.10.1.1, pod-ip: 10.10.1.12, used-cpu: 0.0040 (0.10%), used-mem: 70.4Mi (0.48%))
    Container(name: distributor, used-cpu: 0.0040 (0.10%), used-mem: 70.4Mi (0.48%))
  Pod(name: tempo-gateway-769f9bc5b4-q5gb4, host-ip: 10.10.1.1, pod-ip: 10.10.1.13, used-cpu: 0.0000 (0.00%), used-mem: 11.9Mi (0.08%))
    Container(name: nginx, used-cpu: 0.0000 (0.00%), used-mem: 11.9Mi (0.08%))
  Pod(name: tempo-ingester-3, host-ip: 10.10.1.1, pod-ip: 10.10.1.14, used-cpu: 0.8234 (21.00%), used-mem: 3.3Gi (22.77%))
    Container(name: ingester, used-cpu: 0.8234 (21.00%), used-mem: 3.3Gi (22.77%))
  Pod(name: tempo-meta-monitoring-logs-fc9vz, host-ip: 10.10.1.1, pod-ip: 10.10.1.15, used-cpu: 0.0075 (0.19%), used-mem: 42.8Mi (0.29%))
    Container(name: config-reloader, used-cpu: 0.0001 (0.00%), used-mem: 3.8Mi (0.03%))
    Container(name: grafana-agent, used-cpu: 0.0074 (0.19%), used-mem: 39.1Mi (0.26%))
  Pod(name: tempo-query-frontend-6f4587b55-qjxv9, host-ip: 10.10.1.1, pod-ip: 10.10.1.16, used-cpu: 0.0008 (0.02%), used-mem: 52.9Mi (0.36%))
    Container(name: query-frontend, used-cpu: 0.0008 (0.02%), used-mem: 52.9Mi (0.36%))
```

> Note: Some labels were removed, but I left a few for reference. Also, the IPs were changed to avoid reflecting those actively used. The CPU and Memory values are from a real EKS cluster running Tempo, among other applications.

As you can see, several Tempo instances are running on this node. As explained before, the values are the maximums for the selected range (not the current ones). Note that there might be other nodes running other replicas of the same components, and the following results will take the maximum of all of them:

```
Group monitoring-prometheus-node-exporter, suggested-cpu: 1m, suggested-memory: 10.6Mi
Group tempo-compactor, suggested-cpu: 483m, suggested-memory: 1.8Gi
Group tempo-distributor, suggested-cpu: 114m, suggested-memory: 1.3Gi
Group tempo-gateway, suggested-cpu: 0m, suggested-memory: 11.9Mi
Group tempo-ingester, suggested-cpu: 823m, suggested-memory: 4.0Gi
Group tempo-memcached, suggested-cpu: 0m, suggested-memory: 68.8Mi
Group tempo-meta-monitoring, suggested-cpu: 21m, suggested-memory: 118.8Mi
Group tempo-meta-monitoring-logs, suggested-cpu: 9m, suggested-memory: 54.5Mi
Group tempo-metrics-generator, suggested-cpu: 7m, suggested-memory: 184.4Mi
Group tempo-querier, suggested-cpu: 5m, suggested-memory: 167.2Mi
Group tempo-query-frontend, suggested-cpu: 1m, suggested-memory: 86.6Mi
```

The idea is to use the suggestions from the above list to determine a good starting point for the `resources.requests` (for Memory and CPU).
