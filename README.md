# peerd

[![Build Status]][build-status]
[![Kind CI Status]][kind-ci-status]
[![Docker Release CI]][release-ci]
[![CodeQL]][code-ql]
[![Go Report Card]][go-report-card]

This project implements peer to peer distribution of content (such as files or OCI container images) in a Kubernetes
cluster. The source of the content could be another node in the same cluster, an OCI container registry (like Azure
Container Registry) or a remote blob store (such as Azure Blob Storage).

#### Important Disclaimer

This project is work in progress and can be used for experimental and development purposes. 
It is not yet production ready, but under active development.

## Quickstart

This section shows how to get started with `peerd`. To see all available commands, run `make help`.

```bash
$ make help

         _____                  _
        |  __ \                | |
        | |__) |__  ___ _ __ __| |
        |  ___/ _ \/ _ \ '__/ _` |
        | |  |  __/  __/ | | (_| |
        |_|   \___|\___|_|  \__,_|

all                            Runs the peerd build targets in the correct order.
build-image                    Build the peerd docker image.
build                          Build the peerd packages.
coverage                       Generates test results for code coverage.
help                           Generates help for all targets with a description.
...
```

### Deploy `peerd` to Your Cluster Using `helm`

`peerd` is a self-contained binary that can be run directly on each node of a cluster, as a systemd service ([peerd.service]). 
Alternatively, it can also be deployed as DaemonSet pods using the [helm chart].

#### Prerequisites

* An existing Kubernetes cluster with
* containerd as the container runtime.

With containerd, `peerd` leverages the [hosts configuration][containerd hosts] to act as a mirror for container images.
The [helm chart] deploys a DameonSet to the `peerd-ns` namespace, and mounts the containerd socket to the `peerd` containers.

The `peerd` container image is available at `ghcr.io/azure/acr/peerd`. To deploy, run the following.

```bash
CLUSTER_CONTEXT=<your-cluster-context> && \
  TAG=v0.0.2-alpha && \
  HELM_RELEASE_NAME=peerd && \
  HELM_CHART_DIR=./build/ci/k8s/peerd-helm && \
  helm --kube-context=$CLUSTER_CONTEXT install --wait $HELM_RELEASE_NAME $HELM_CHART_DIR \
    --set peerd.image.ref=ghcr.io/azure/acr/dev/peerd:$TAG
```

By default, `mcr.microsoft.com` and `ghcr.io` are mirrored, but this is configurable. For example, to mirror `docker.io`
as well, run the following.

```bash
CLUSTER_CONTEXT=<your-cluster-context> && \
  TAG=v0.0.2-alpha && \
  HELM_RELEASE_NAME=peerd && \
  HELM_CHART_DIR=./build/ci/k8s/peerd-helm && \
  helm --kube-context=$CLUSTER_CONTEXT install --wait $HELM_RELEASE_NAME $HELM_CHART_DIR \
    --set peerd.image.ref=ghcr.io/azure/acr/dev/peerd:$TAG
    --set peerd.hosts="mcr.microsoft.com ghcr.io docker.io"
```

On deployment, each `peerd` instance will try to connect to its peers in the cluster. 

* When connected successfully, each pod will generate an event `P2PConnected`. This event is used to signal that the 
  `peerd` instance is ready to serve requests to its peers.

* When a request is served by downloading data from a peer, `peerd` will emit an event called `P2PActive`, 
  signalling that it's actively communicating with a peer and serving data from it.

To see logs from the `peerd` pods, run the following.

```bash
kubectl --context=$CLUSTER_CONTEXT -n peerd-ns logs -l app=peerd -f
```

### Build and Deploy to a Local Kind Cluster

For local development or experimentation, you can build the `peerd` docker image, create a kind cluster, and deploy the
`peerd` application to each node in it. To build and deploy to a 3 node kind cluster, run the following.

```bash
$ make build-image && \
    make kind-create kind-deploy
  ...
  ...
  daemonset.apps/peerd created
  service/peerd created
  waiting for pods to connect
  pods: peerd-5trwv peerd-q2c45 peerd-tkj5k
  checking pod 'peerd-5trwv' for event 'P2PConnected'
  checking pod 'peerd-q2c45' for event 'P2PConnected'
  checking pod 'peerd-tkj5k' for event 'P2PConnected'
  Success: All pods have event 'P2PConnected'.
```

Clean up your deployment.

```bash
$ make kind-delete
```

### Run a Test Workload

There are two kinds of test workloads available in this repository:

1. Simple peer to peer file sharing by specifying the range of bytes to read.
   * This enables block level file drivers, such as [Overlaybd], to use `peerd` as the p2p proxy.
   * This test is run by deploying the `random` test workload to the kind cluster.
   * The workload is deployed to each node, and outputs performance metrics that are observed by it, such as the speed
      of downloads and error rates.

    ```bash
    $ make build-image tests-random-image && \
        make kind-create kind-deploy kind-test-random
      ...
      {"level":"info","node":"random-zb9vm","version":"bb7ee6a","mode":"upstream","size":22980743,"readsPerBlob":5,"time":"2024-03-07T21:50:29Z","message":"downloading blob"}
      {"level":"info","node":"random-9gcvw","version":"bb7ee6a","upstream.p50":21.25170790666404,"upstream.p75":5.834663359546446,"upstream.p90":0.7871542327673121,"upstream.p95":0.2965091294200036,"upstream.p100":0.2645602612715345,"time":"2024-03-07T21:50:34Z","message":"speeds (MB/s)"}
      {"level":"info","node":"random-9gcvw","version":"bb7ee6a","p2p.p50":5.802082290454193,"p2p.p75":1.986398855488793,"p2p.p90":0.6210418172329215,"p2p.p95":0.0523776186045032,"p2p.p100":0.023341096448268952,"time":"2024-03-07T21:50:34Z","message":"speeds (MB/s)"}
      {"level":"info","node":"random-9gcvw","version":"bb7ee6a","p2p.error_rate":0,"upstream.error_rate":0,"time":"2024-03-07T21:50:34Z","message":"error rates"}
      ...

      # Clean up
    $ make kind-delete
    ```

2. Peer to peer sharing of container images that are available in the containerd content store of a node.
   * This enables pulling container images from peers in the cluster, instead of from the registry.
   * This test is run by deploying the `ctr` test workload to the kind cluster.
   * The workload is deployed to each node, and outputs performance metrics that are observed by it, such as the speed
      of downloads and error rates.
 
    ```bash
    $ make build-image tests-scanner-image && \
        make kind-create kind-deploy kind-test-ctr
        ...
        ...        
        ...

      # Clean up
    $ make kind-delete
    ```

### Build `peerd` Binary

To build the `peerd` binary, run the following.

```bash
$ make
  ...
```

The build produces a binary and a systemd service unit file. Additionally, it bin-places the API swagger file.

```bash
|-- peerd          # The binary
|-- peerd.service  # The service unit file for systemd
|-- swagger.yml    # The swagger file for the REST API
```

## Features

`peerd` allows a Kubenetes node to share its container images (when using containerd) with other nodes in the cluster.
It also allows a node to act as a mirror for files obtained from any HTTP upstream source (such as an [Azure Blob] using
a [SAS URL]), and can discover and serve a specified byte range of the file to/from other nodes in the cluster.

### Peer-to-Peer File Sharing

When a range of an HTTP file is requested, `peerd` first attempts to discover if any of its peers already have that exact
range. The file is identified by its SHA256 digest, and only upstream URLs that specify this digest are supported.

For example, to download the first 100 bytes of a layer of the `mcr.microsoft.com/hello-world:latest` container image, 
whose SAS URL can be obtained by querying `GET https://mcr.microsoft.com/v2/hello-world/blobs/sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4`, the following request can be made to `peerd`,
which is assumed to run at `http://localhost:30000`.

```bash
GET http://localhost:30000/blobs/https://westus2.data.mcr.microsoft.com/01031d61e1024861afee5d512651eb9f-h36fskt2ei//docker/registry/v2/blobs/sha256/a3/a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4/data?se=2024-03-13T21%3A35%3A45Z&amp;sig=mSdsz%2FXkQjze%2Bzhy7pEAlr0WPrUnlhbcgnPfAoxVzuE%3D&amp;sp=r&amp;spr=https&amp;sr=b&amp;sv=2018-03-28&amp;regid=01031d61e1024861afee5d512651eb9f

Range: bytes=0-100
```

`peerd` will first attempt to discover this range from its peers, and if not found, will download the range from one of 
them. If not found, it will download the range from the upstream HTTP source and serve it to the client. Additionally,
the instance will cache the bytes (and optionally, prefetch the entire file) and advertise the cached bytes to its peers,
so that they can serve it in the future without having to download it from the upstream source.

This approach requires an exact knowledge of parsing the digest from the HTTP URL, and is currently supported for the 
following:

1. `mcr.microsoft.com`
2. Azure Container Registry

With this facility, `peerd` can be used as the [p2p proxy] for [Overlaybd].

```json
"p2pConfig": {
  "enable": true,
  "address": "localhost:30000/blobs"
}
```

### Peer-to-Peer Container Image Sharing

Pulling a container image to a node in Kubernetes is often a time consuming process, especially in scenarios where the 
registry becomes a bottleneck, such as deploying a large cluster or scaling out in response to bursty traffic. To increase
throughput, nodes in the cluster which already have the image can be used as an alternate image source. `peerd` subscribes
to events in the containerd content store, and advertises local images to peers. When a node needs an image, it can query
its peers for the image, and download it from them instead of the registry. Containerd has a [mirror][containerd hosts]
facility that can be used to configure `peerd` as the mirror for container images.

The APIs are described in the [swagger.yaml].

## Design and Architecture

![cluster-arch]

The design is inspired from the [Spegel] project, which is a peer to peer proxy for container images that uses libp2p.
In this section, we describe the design and architecture of `peerd`.

### Background

An OCI image is composed of multiple layers, where each layer is stored as a blob in the registry. When a container
application is deployed to a cluster such as an AKS or ACI, the container image must first be downloaded to each node
where it’s scheduled to run. If the image is too large, downloading it often becomes the most time-consuming step of
starting the application. This download step most impacts two scenarios:

a) Where the application needs to scale immediately to handle a burst of requests, such as an e-commerce application
   dealing with a sudden increase in shoppers; or

b) Where the application must be deployed on each node of a large cluster (say 1000+ nodes) and the container image
   itself is very large (multiple Gbs), such as training a large language model.

ACR Teleport addresses scenario a by allowing a container to quickly start up using the registry as a remote filesystem
and downloading only specific parts of files needed for it to serve requests. However, scenario b will continue to be
impacted by increased latencies due to the requirement of downloading entire layers from the registry to all nodes before
the application can run. Here, the registry can become a bottleneck for the downloads.

To minimize network I/O to the remote registry and improve speed, once an image (or parts of it) has been downloaded by
a node, other nodes in the cluster can leverage this peer and download from it rather than from the remote ACR. This can
reduce network traffic to the registry and improve the average download speed per node. Peers must be able to discover
content already downloaded to the network and share it with others. Such p2p distribution would benefit both scenarios
above, a (Teleport) and b (regular non-Teleport).

### Design

There are four main components to the design that together make up the `peerd` binary:

1.	Peer to Peer Router
2.	File Cache
3.	Containerd Content Store Subscriber
4.	P2P Proxy Server

#### Peer to Peer Router

The p2p router is the core component responsible for discovering peers in the local network and maintaining a distributed
hash table (DHT) for content lookup. It provides the ability to advertise local content to the network, as well as
discover peers that have specific content. The DHT protocol is called Kademlia, which provides provable consistency and
performance. Please reference the [white paper] for details.

##### Bootstrap

When a node is created, it must obtain some basic information to join the p2p network, such as the addresses and public
keys of nodes already in the network to initialize its DHT. One way to do this is to connect to an existing node in the
network and ask it for this information. So, which node should it connect to? To make this process completely automatic,
we leverage leader election in k8s, and connect to the leader to bootstrap.

Although this introduces a dependency on the k8s runtime APIs and kubelet credentials for leader election and is the
current approach, an alternative would be to use a statically assigned node as a bootstrapper.

##### Configuration

The router uses the following configuration to connect to peers:

| Name           | Value | Description                                   |
| -------------- | ----- | --------------------------------------------- |
| ResolveTimeout | 20ms  | The time to wait for a peer to resolve        |
| ResolveRetries | 3     | The number of times to retry resolving a peer |

##### Advertisements

Once the node has completed bootstrapping, it is ready to advertise its content to the network. There are two sources
for this content:

1. Containerd Content Store: this is where images pulled to the node are available, see section
   [Containerd Content Store Subscriber]. 

2. File cache: this is where files pulled to the node are available, see section [File Cache].

Advertising means adding the content's key to the node's DHT, and optionally, announcing the available content on the
network. The key used is the sha256 digest of the content. 

##### Resolution

A key is resolved to a node based on the closeness metric discussed in the Kademlia paper. With advertisements,
resolution is very fast (overhead of ~1ms in AKS).

#### File Cache

The file cache is a cache of files on the local file system. These files correspond to layers of a teleported image.

##### Prefetching

The first time a request for a file is made, the range of requested bytes is served from the remote source (either peer
or upstream). At the same time, multiple prefetch tasks are kicked off, which download fixed size chunks of the file
parallelly (from peer or upstream) and store them in the cache. The default configuration is as follows:

| Name            | Value | Description                                                                             |
| --------------- | ----- | --------------------------------------------------------------------------------------- |
| ChunkSize       | 1 Mib | The size of a single chunk of a file that is downloaded from remote and cached locally. |  |
| PrefetchWorkers | 50    | The total number of workers available for downloading file chunks.                      |

##### File System Layout

Below is an example of what the file cache looks like. Here, five files are cached (the folder name of each is its digest,
shortened in the example below), and for each file, some chunks have been downloaded. For example, for the file
095e6bc048, four chunks are available in the cache. The name of each chunk corresponds to an offset in the file. So,
chunk 0 is the portion of 095e6bc048 starting at offset 0 of size ChunkSize. Chunk 1048576 is the portion of 095e6bc048
starting at offset 1048576 of size ChunkSize. And so on.

![file-system-layout]

#### Containerd Content Store Subscriber

This component is responsible for discovering layers in the local containerd content store and advertising them to the
p2p network using the p2p router component, enabling p2p distribution for regular image pulls.

#### P2P Proxy Server

The p2p proxy server (a.k.a. p2p mirror) serves the node’s content from the file cache or containerd content store.
There are two scenarios for accessing the proxy: 

1. Overlaybd TCMU driver: this is the Teleport scenario.

The driver makes requests like the following to the p2p proxy. 

```bash
GET http://localhost:5000/blobs/https://westus2.data.mcr.microsoft.com/01031d61e1024861afee5d512651eb9f36fskt2ei//docker/registry/v2/blobs/sha256/1b/1b930d010525941c1d56ec53b97bd057a67ae1865eebf042686d2a2d18271ced/data?se=20230920T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f

Range: bytes=456-990
```

Here, the p2p proxy is listening at `localhost:5000`, and it is passed in the full SAS URL of the layer. The SAS URL was
previously obtained by the driver from the ACR. The proxy will first attempt to locate this content in the p2p network
using the router. If found, the peer will be used to reverse proxy the request. Otherwise, after the configured resolution
timeout, the request will be proxied to the upstream storage account.

2. Containerd Hosts: this is the non-Teleport scenario.

Here, containerd is configured to use the p2p mirror using its hosts configuration. The p2p mirror will receive registry
requests to the /v2 API, following the OCI distribution API spec. The mirror will support GET and HEAD requests to `/v2/`
routes. When a request is received, the digest is first looked up in the p2p network, and if a peer has the layer, it is
used to serve the request. Otherwise, the mirror returns a 404, and containerd client falls back to the ACR directly (or
any next configured mirror.)

### Performance

The following numbers were gathered from a 3-node AKS cluster.

#### Peer Discovery

In broadcast mode, any locally available content is broadcasted to the k closest peers f the content ID. As seen below,
the performance improves significantly, with the tradeoff that network traffic also increases.

**Broadcast off**

| Operation | Samples | Min (s) | Mean (s) | Max (s) | Std. Deviation |
| --------- | ------- | ------- | -------- | ------- | -------------- |
| Discovery | 30      | 0.006   | 0.021    | 0.039   | 0.009          |

**Broadcast on**

| Operation | Samples | Min (s) | Mean (s) | Max (s) | Std. Deviation |
| --------- | ------- | ------- | -------- | ------- | -------------- |
| Discovery | 416     | 0       | 0.001    | 0.023   | 0.003          |

#### File Scanner Application Container

An Overlaybd image was created for a simple application that reads an entire file. The performance is compared between 
running this container in p2p vs non-p2p mode on a 3 node AKS cluster with Artifact Streaming.

| Mode                              | File Size Read (Mb) | Speed (3 nodes) (Mbps) |
| --------------------------------- | ------------------- | ---------------------- |
| Teleport without p2p              | 200                 | 3.5, 3.8, 3.9          |
| Teleport with p2p, no prefetching | 600                 | 3.8, 3.9, 4.9          |
| Teleport with p2p and prefetching | 200                 | 6.5, **11, 13**        |
| Teleport with p2p and prefetching | 600                 | 5.5, 6.1, 6.8          |


## Contributing

Please read our [CONTRIBUTING.md] which outlines all of our policies, procedures, and requirements for contributing to
this project.

## Acknowledgments

A hat tip to:

* [Spegel]
* [DADI P2P Proxy]

## Glossary

| Term | Definition                   |
| ---- | ---------------------------- |
| ACR  | Azure Container Registry     |
| AKS  | Azure Kubernetes Service     |
| ACI  | Azure Container Instances    |
| DHT  | Distributed Hash Table       |
| OCI  | Open Container Initiative    |
| P2P  | Peer to Peer                 |
| POC  | Proof of Concept             |
| TCMU | Target Core Module Userspace |

---

[CONTRIBUTING.md]: CONTRIBUTING.md
[kubectl-node-shell]: https://github.com/kvaps/kubectl-node-shell
[Go Report Card]: https://goreportcard.com/badge/github.com/azure/peerd
[go-report-card]: https://goreportcard.com/report/github.com/azure/peerd
[Build Status]: https://github.com/azure/peerd/actions/workflows/build.yml/badge.svg
[build-status]: https://github.com/azure/peerd/actions/workflows/build.yml
[Kind CI Status]: https://github.com/azure/peerd/actions/workflows/kind.yml/badge.svg
[kind-ci-status]: https://github.com/azure/peerd/actions/workflows/kind.yml
[Docker Release CI]: https://github.com/azure/peerd/actions/workflows/release.yml/badge.svg
[release-ci]: https://github.com/azure/peerd/actions/workflows/release.yml
[Code Coverage]: https://img.shields.io/badge/coverage-54.9%25-orange
[cluster-arch]: ./assets/images/cluster.png
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
[helm chart]: ./build/ci/k8s/peerd-helm
[CodeQL]: https://github.com/Azure/peerd/actions/workflows/github-code-scanning/codeql/badge.svg?branch=main
[code-ql]: https://github.com/Azure/peerd/actions/workflows/github-code-scanning/codeql
[Azure Blob]: https://learn.microsoft.com/en-us/azure/storage/blobs/storage-blobs-introduction
[SAS URL]: https://learn.microsoft.com/en-us/azure/storage/common/storage-sas-overview
[p2p proxy]: https://github.com/containerd/overlaybd/blob/main/src/example_config/overlaybd.json#L27C5-L30C7
[peerd.service]: ./init/systemd/peerd.service
[white paper]: https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf
[file-system-layout]: ./assets/images/file-system-layout.png
