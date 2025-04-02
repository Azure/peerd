# Peerd Usage

The following sections describe how to use Peerd in your Kubernetes cluster.

## Prerequisites

- An existing Kubernetes cluster.

| Environment                    | Compatibility Verified |
| ------------------------------ | ---------------------- |
| Azure Kubernetes Service (AKS) | <p>&#9989;</p>         |
| Kind                           | <p>&#9989;</p>         |

- `helm` installed and configured.
- `kubectl` installed and configured.

### Overlaybd Peer-to-Peer Configuration for Artifact Streaming

This step is `OPTIONAL` and should be performed only if you are using Artifact Streaming. Please skip directly to the next
section if you are pulling images instead.

Artifact Streaming leverages the `overlaybd-snapshotter` which is well integrated with `containerd`. More information on
`overlaybd-snapshotter` can be found [here](overlaybd-snapshotter). On AKS, `overlaybd-snapshotter` is already installed
and ready to use.

In order to configure `overlaybd-snapshotter` to work with Peerd, its configuration file must be updated. The default
location of the configuration file is `/etc/overlaybd/overlaybd.json` and the relevant configuration section is as follows:

```json
"p2pConfig": {
    "enable": true,
    "address": "http://localhost:30000/blobs"
},
```

After the configuration file is updated, the `overlaybd-snapshotter` and `overlaybd-tcmu` services must be restarted for
the changes to take effect. *Note that this will impact any ongoing streaming and must be done with caution*. The restart
commands are illustrated in the example below.

#### Configure Overlaybd P2P Helm Chart on Azure Kubernetes Service (AKS)

You may use the included [configure-overlaybd-p2p-helm] tool to do these steps easily if using AKS
To run the tool:

```bash
CLUSTER_CONTEXT=<your-cluster-context> && \
  helm --kube-context=$CLUSTER_CONTEXT install --wait overlaybd ./tools/configure-overlaybd-p2p-helm
```

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

[azure.sh]: ../build/ci/scripts/azure.sh
[ci-script-readiness]: ../build/ci/scripts/azure.sh
[configure-overlaybd-p2p-helm]: ../tools/configure-overlaybd-p2p-helm/
[Grafana dashboard]: ../build/package/peerd-grafana/dashboard.json
[overlaybd-snapshotter]: https://github.com/containerd/accelerated-container-image?tab=readme-ov-file#components
[values.yml]: ../build/package/peerd-helm/values.yaml