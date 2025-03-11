# Peerd Usage

The following sections describe how to use Peerd in your Kubernetes cluster.

## Prerequisites

- An existing Kubernetes cluster.

| Environment | Compatibility Verified |
| ----------- | ---------------------- |
| AKS         | :white_check_mark:     |
| Kind        | :white_check_mark:     |

- `helm` installed and configured.
- `kubectl` installed and configured.

## Deployment

> See [values.yml] for all available options.

```bash
CLUSTER_CONTEXT=<your-cluster-context> && \
  helm --kube-context=$CLUSTER_CONTEXT install --wait peerd ./build/package/peerd-helm \
    --set peerd.image.ref=ghcr.io/azure/acr/dev/peerd:stable
```

## Wait for Readiness

Wait for Peerd to establish connections with its peers. Each pod will emit an event `P2PConnected` when it's connected.

> See the function `wait_for_peerd_pods` in the [CI script][ci-script-readiness] that programmatically waits for readiness.

## Stream or Pull Images

When the application image is pulled or streamed from a peer, the peerd pod will emit a `P2PActive` event, signalling that
a peer-to-peer transfer is in progress.

> For best results, ensure that at least one peer has fully downloaded the image or begun streaming before scaling out.
## Observe Peerd

### Events

| Pod Event         | Description                                                                                   |
| ----------------- | --------------------------------------------------------------------------------------------- |
| `P2PConnected`    | Peerd pod has connected to p2p network and is ready to serve requests.                        |
| `P2PActive`       | Peerd pod is actively streaming or pulling an image from a peer.                              |
| `P2PDisconnected` | Peerd pod encountered a transient error and is temporarily disconnected from the p2p network. |
| `P2PFailed`       | Peerd pod encountered an error and failed to serve a request.                                 |

### Logs

To see logs from the Peerd pods, run the following.

```bash
kubectl --context=$CLUSTER_CONTEXT -n peerd-ns logs -l app=peerd -f
```

### Metrics

Peerd exposes Prometheus metrics on the `/metrics/prometheus` endpoint. Metrics are prefixed with `peerd_`. `libp2p` metrics
are prefixed with `libp2p_`.

### Grafana Dashboard

The accompanying [Grafana dashboard] can be used to visualize the metrics emitted by Peerd.

> On AKS, automatic metrics scraping is enabled by setting `--set peerd.metrics.prometheus.aksAutoDiscovery=true` in the
> helm chart.

##### Example

On a 100 nodes AKS cluster of VM size `Standard_D2s_v3`, sample throughput observed by a single pod is shown below.

<img src="../assets/images/peer-metrics.png" alt="peer metrics" width="1000">

---

[ci-script-readiness]: ../build/ci/scripts/azure.sh
[Grafana dashboard]: ../build/package/peerd-grafana/dashboard.json
[values.yml]: ../build/package/peerd-helm/values.yaml