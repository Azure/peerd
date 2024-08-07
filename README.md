# Peerd

[![Build Status]][build-status]
[![Kind CI Status]][kind-ci-status]
[![Release CI]][release-ci]
[![CodeQL]][code-ql]
[![Go Report Card]][go-report-card]
[![codecov]][code-cov]
[![release-tag]][peerd-pkgs]

This project implements peer to peer distribution of content (such as content-addressable files or OCI container images)
in a Kubernetes cluster. The source of the content could be another node in the same cluster, an OCI container registry
(like Azure Container Registry) or a remote blob store (such as Azure Blob Storage).

![cluster-ops]

#### Important Disclaimer

**Work in Progress**: We are actively working on this project and would love to hear your feedback.
Please feel free to open an issue or a pull request.

## Usage

**Peer D**aemon is designed to be deployed as a daemonset on every node in a Kubernetes cluster and acts as a registry
mirror.

* It discovers other nodes in the cluster and establishes a peer-to-peer overlay network in the cluster using the
  [Kademlia DHT][white paper] protocol.

* It discovers content such as OCI images in the node's containerd content store as well as streamable container files,
  such as used in [Azure Artifact Streaming][ACR Artifact Streaming], and advertises them to its peers.

* It can serve discovered/cached content to other nodes in the cluster, acting as a mirror for the content.

This is useful in the following scenarios:

1. **Increased Throughput**: For downloading large images or deploying large clusters, the container/artifact registry
   can become a bottleneck. Peerd can be used to download images from other nodes in the cluster that have already downloaded it,
   increasing throughput.

2. **Improved Fault Tolerance**: If the upstream registry is unavailable, Peerd can still serve images from other nodes
   in the cluster.

3. **Firewall configuration**: Peerd can be used to download images from other nodes in the cluster. This can be useful
    in scenarios where outbound internet access is restricted on some nodes.

## Features

* **Peer to Peer Streaming**: Peerd allows a node to act as a mirror for files obtained from any HTTP upstream source
  (such as an [Azure Blob] using a [SAS URL]), and can discover and serve a specified byte range of the file to/from
  other nodes in the cluster. Peerd will first attempt to discover and serve this range from its peers. If not found, it
  will  fallback to download the range from the upstream URL. Peerd caches downloaded ranges as well as optionally, can
  prefetch the entire file.

  With this facility, `peerd` can be used as the [p2p proxy] for [Overlaybd].

  ```json
  "p2pConfig": {
    "enable": true,
    "address": "localhost:30000/blobs"
  }
  ```

  Peerd is compatible with Azure Container Registry's [Artifact Streaming][ACR Artifact Streaming] feature, and can be
  used to improve performance further.

  | **Without Peerd**           | **With Peerd**             |
  | --------------------------- | -------------------------- |
  | ![normal-streaming-summary] | ![peerd-streaming-summary] |

* **Peer to Peer Container Image Pulls**: Pulling a container image to a node in Kubernetes is often a time consuming
  process, especially in scenarios where the registry becomes a bottleneck, such as deploying a large cluster or scaling
  out in response to bursty traffic. To increase throughput, nodes in the cluster which already have the image can be
  used as an alternate image source. Peerd subscribes to events in the containerd content store, and advertises local
  images to peers. When a node needs an image, it can query its peers for the image, and download it from them instead
  of the registry. Containerd has a [mirror][containerd hosts] facility that can be used to configure Peerd as the 
  mirror for container images.

  | **Without Peerd**      | **With Peerd**        |
  | ---------------------- | --------------------- |
  | ![normal-pull-summary] | ![peerd-pull-summary] |

The APIs are described in the [swagger.yaml].

## Quickstart

To see all available commands, run `make help`.

### Deploy Peerd to Your Cluster Using Helm

If you have a k8s cluster that uses containerd as the runtime, you can use the provided [helm chart] to deploy Peerd
pods on every node. With containerd, Peerd leverages the [hosts configuration][containerd hosts] to act as a mirror for
container images.

```bash
CLUSTER_CONTEXT=<your-cluster-context> && \
  helm --kube-context=$CLUSTER_CONTEXT install --wait peerd ./build/package/peerd-helm \
    --set peerd.image.ref=ghcr.io/azure/acr/dev/peerd:stable
```

By default, some well known registries are mirrored, but this is configurable using the [values.yml] file.
Update the `peerd.hosts` field to include the registries you want to mirror.

On deployment, each Peerd instance will try to connect to its peers in the cluster. 

* When connected successfully, each pod will generate an event `P2PConnected`. This event is used to signal that the 
  Peerd instance is ready to serve requests to its peers.

* When an instance serves a request by downloading data from a peer, it will emit an event called `P2PActive`, 
  signalling that it's actively communicating with a peer and serving data from it.

To see logs from the Peerd pods, run the following.

```bash
kubectl --context=$CLUSTER_CONTEXT -n peerd-ns logs -l app=peerd -f
```

### Observe Metrics

Peerd exposes metrics on the `/metrics/prometheus` endpoint. Metrics are prefixed with `peerd_`. `libp2p` metrics are
prefixed with `libp2p_`.

#### Example

On a 100 nodes AKS cluster of VM size `Standard_D2s_v3`, sample throughput observed by a single pod is shown below.


<img src="./assets/images/peer-metrics.png" alt="peer metrics" width="1000">

## Build

See [build.md].

## Design and Architecture

See [design.md].

## Contributing

Please read our [CONTRIBUTING.md] which outlines all of our policies, procedures, and requirements for contributing to
this project.

## Acknowledgments

The [Spegel] project has inspired this work; thanks to Philip Laine and Simon Gottschlag at Xenit
for generously sharing their insights with us. A hat tip also to the [DADI P2P Proxy] project for demonstrating the
integration with [Overlaybd].

---

[CONTRIBUTING.md]: CONTRIBUTING.md
[kubectl-node-shell]: https://github.com/kvaps/kubectl-node-shell
[Go Report Card]: https://goreportcard.com/badge/github.com/azure/peerd
[go-report-card]: https://goreportcard.com/report/github.com/azure/peerd
[Build Status]: https://github.com/azure/peerd/actions/workflows/build.yml/badge.svg
[build-status]: https://github.com/azure/peerd/actions/workflows/build.yml
[Kind CI Status]: https://github.com/azure/peerd/actions/workflows/kind.yml/badge.svg
[kind-ci-status]: https://github.com/azure/peerd/actions/workflows/kind.yml
[Release CI]: https://github.com/azure/peerd/actions/workflows/release.yml/badge.svg
[release-ci]: https://github.com/azure/peerd/actions/workflows/release.yml
[Code Coverage]: https://img.shields.io/badge/coverage-54.9%25-orange
[node-arch]: ./assets/images/http-flow.png
[Overlaybd]: https://github.com/containerd/overlaybd
[scanner]: ./tests/scanner/scanner.go
[ACR Artifact Streaming]: https://learn.microsoft.com/en-us/azure/container-registry/container-registry-artifact-streaming
[swagger.yaml]: ./api/swagger.yaml
[Spegel]: https://github.com/XenitAB/spegel
[Kraken]: https://github.com/uber/kraken
[Dragonfly]: https://github.com/dragonflyoss/Dragonfly2
[DADI P2P Proxy]: https://github.com/data-accelerator/dadi-p2proxy
[containerd hosts]: https://github.com/containerd/containerd/blob/main/docs/hosts.md
[containerd-mirror]: ./internal/containerd/mirror.go
[helm chart]: ./build/package/peerd-helm
[CodeQL]: https://github.com/Azure/peerd/actions/workflows/github-code-scanning/codeql/badge.svg?branch=main
[code-ql]: https://github.com/Azure/peerd/actions/workflows/github-code-scanning/codeql
[Azure Blob]: https://learn.microsoft.com/en-us/azure/storage/blobs/storage-blobs-introduction
[SAS URL]: https://learn.microsoft.com/en-us/azure/storage/common/storage-sas-overview
[p2p proxy]: https://github.com/containerd/overlaybd/blob/main/src/example_config/overlaybd.json#L27C5-L30C7
[peerd.service]: ./init/systemd/peerd.service
[white paper]: https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf
[design.md]: ./docs/design.md
[cluster-ops]: ./assets/images//cluster-ops.gif
[codecov]: https://codecov.io/gh/Azure/peerd/branch/main/graph/badge.svg
[code-cov]: https://codecov.io/gh/Azure/peerd
[release-tag]: https://img.shields.io/github/v/tag/Azure/peerd?label=Docker%20Image%20Tag
[peerd-pkgs]: https://github.com/Azure/peerd/pkgs/container/acr%2Fdev%2Fpeerd
[build.md]: ./docs/build.md
[values.yml]: ./build/package/peerd-helm/values.yaml
[normal-pull-summary]: ./assets/mermaid/rendered/normal-pull-summary.png
[peerd-pull-summary]: ./assets/mermaid/rendered/peerd-pull-summary.png
[normal-streaming-summary]: ./assets/mermaid/rendered/normal-streaming-summary.png
[peerd-streaming-summary]: ./assets/mermaid/rendered/peerd-streaming-summary.png
