# peerd

[![Go Report Card]][go-report-card]
[![Build Status]][build-status]
[![Kind CI Status]][kind-ci-status]
[![Docker Release CI]][release-ci]
![Code Coverage]

This project implements peer to peer distribution of content (such as files or OCI container images) in a Kubernetes
cluster. The source of the content could be another node in the same cluster, an OCI container registry (like Azure
Container Registry) or a remote blob store (such as Azure Blob Storage).

#### Important Disclaimer

This project is work in progress and can be used for experimental and development purposes. 
It is not yet production ready, but we're getting there.

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

#### Prerequisites

* An existing Kubernetes cluster with
* containerd as the container runtime.

You can deploy `peerd` to your existing cluster using the included [helm chart]. With containerd, `peerd` leverages the 
[hosts configuration][containerd hosts] to act as a mirror for container images. The helm chart deploys a DameonSet to
the `peerd-ns` namespace, and mounts the containerd socket to the `peerd` containers.

The `peerd` container image is available at `ghcr.io/azure/acr/peerd`. To deploy, run the following.

```bash
CLUSTER_CONTEXT=<your-cluster-context> && \
  TAG=v0.0.2-alpha && \
  HELM_RELEASE_NAME=peerd && \
  HELM_CHART_DIR=./build/ci/k8s/peerd-helm && \
  helm --kube-context=$CLUSTER_CONTEXT install --wait $HELM_RELEASE_NAME $HELM_CHART_DIR \
    --set peerd.image.ref=ghcr.io/azure/acr/dev/peerd:$TAG
```

By default, `mcr.microsoft.com` and `ghcr.io` are mirrored, but this is configurable. For example, to `docker.io`, run
the following.

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

There are two kinds of test workloads avaialbe in this repository:

1. Simple peer to peer sharing of a file, specified by the range of bytes to read.
   * This scenario is useful for block level file drivers, such as [Overlaybd].
   * This test is run by deploying the `random` test workload to the kind cluster.
   * The test deploys a workload to each node, and outputs performance metrics that are observed by the test app,
      such as the speed of download aggregated at the 50th, 75th and 90th percentiles, and error rates.

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

2. Peer to peer sharing of container images that are available in the containerd store of a node.
   * This scenario is useful for downloading container images to a cluster.
   * This test is run by deploying the `ctr` test workload to the kind cluster.
   * The test deploys a workload to each node, and outputs performance metrics that are observed by the test app,
     such as the speed of download aggregated at the 50th, 75th and 90th percentiles, and error rates.
 
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

### Throughput Improvements

An Overlaybd image was created for a simple application that reads an entire file (see [scanner]). The performance is
compared when running this container in p2p vs non-p2p mode on a 3 node AKS cluster with [ACR Artifact Streaming].

| Mode                 | File Size (Mb) | Throughput (MB/s) |
| -------------------- | -------------- | ----------------- |
| Non P2P              | 200            | 3.5, 3.8, 3.9     |
| P2P (no prefetching) | 600            | 3.8, 3.9, 4.9     |
| P2P with prefetching | 200            | 6.5, 11, 13       |

## Features

`peerd` allows a node to share content with other nodes in a cluster. Specifically:

* A `peerd` node can share (parts of) a file with another node. The file itself may have been acquired from an upstream
  source by `peerd`, if no other node in the cluster had it to begin with.

* A `peerd` node can share a container image from the local `containerd` content store with another node.

The APIs are described in the [swagger.yaml].

## Design and Architecture

`peerd` is a self-contained binary that is designed to run as on each node of a cluster. It can be deployed as a 
systemd service (`peerd.service`), or as a container, such as by using a Kubernetes DaemonSet. It relies on accessing
the Kubernetes API to run a leader election, and to discover other `peerd` instances in the cluster. 

> The commands `make kind-create kind-deploy` can be used as a reference for deployment.

### Cluster Operations

![cluster-arch] \

[Work in Progress]

## Contributing

Please read our [CONTRIBUTING.md] which outlines all of our policies, procedures, and requirements for contributing to
this project.

## Acknowledgments

A hat tip to:

* [Spegel]
* [DADI P2P Proxy]

## Glossary

| Term | Definition                |
| ---- | ------------------------- |
| ACR  | Azure Container Registry  |
| AKS  | Azure Kubernetes Service  |
| ACI  | Azure Container Instances |
| DHT  | Distributed Hash Table    |
| OCI  | Open Container Initiative |
| P2P  | Peer to Peer              |
| POC  | Proof of Concept          |

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
[DADI P2P Proxy]: https://github.com/data-accelerator/dadi-p2proxy
[containerd hosts]: https://github.com/containerd/containerd/blob/main/docs/hosts.md
[containerd-mirror]: ./internal/containerd/mirror.go
[helm chart]: ./build/ci/k8s/peerd-helm
```
