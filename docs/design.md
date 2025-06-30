# Peerd Design

**Peer D**aemon is designed to be deployed as a daemonset on every node in a Kubernetes cluster and acts as a registry
mirror.

* It discovers other nodes in the cluster and establishes a peer-to-peer overlay network in the cluster using the
  [Kademlia DHT][kademlia-white-paper] protocol.

* It discovers streamable container files such as used in [Azure Artifact Streaming], and advertises them to its peers.

* It can serve discovered/cached content to other nodes in the cluster, acting as a mirror for the content.

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

  Peerd is compatible with [Azure Artifact Streaming], and can be used to improve performance further.

  | **Without Peerd**           | **With Peerd**             |
  | --------------------------- | -------------------------- |
  | ![normal-streaming-summary] | ![peerd-streaming-summary] |

The APIs are described in the [swagger.yaml].

The design is inspired from the [Spegel] project, which is a peer to peer proxy for container images that uses libp2p.
In this section, we describe the design and architecture of `peerd`.

#### **DHT Topology**

|                   |
| ----------------- |
| ![peerd-dht-topo] |

#### **Image Streaming Description**

|                    |                        |
| ------------------ | ---------------------- |
| ![peerd-streaming] | ![peerd-streaming-seq] |

### Background

An OCI image is composed of multiple layers, where each layer is stored as a blob in the registry. When a container
application is deployed to a cluster such as an AKS or ACI, the container image must first be downloaded to each node
where it’s scheduled to run. If the image is too large, downloading it often becomes the most time-consuming step of
starting the application. This download step most impacts two scenarios:

a) Where the application needs to scale immediately to handle a burst of requests, such as an e-commerce application
   dealing with a sudden increase in shoppers; or

b) Where the application must be deployed on each node of a large cluster (say 1000+ nodes) and the container image
   itself is very large (multiple Gbs), such as training a large language model.

ACR Teleport addresses scenario `a` by allowing a container to quickly start up using the registry as a remote filesystem
and downloading only specific parts of files needed for it to serve requests. However, scenario `b` will continue to be
impacted by increased latencies due to the requirement of downloading entire layers from the registry to all nodes before
the application can run. Here, the registry can become a bottleneck for the downloads.

To minimize network I/O to the remote registry and improve speed, once an image (or parts of it) have been downloaded by
a node, other nodes in the cluster can leverage this peer and download from it rather than from the remote ACR. This can
reduce network traffic to the registry and improve the average download speed per node. Peers must be able to discover
content already downloaded to the network and share it with others.

### Design

There are three main components to the design that together make up the `peerd` binary:

1.	Peer to Peer Router
2.	File Cache
3.	P2P Proxy Server

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

Once the node has completed bootstrapping, it is ready to advertise its content to the network. The source for this content
is the file cache; this is where files pulled to the node are available.

Advertising means adding the content's key to the node's DHT, and optionally, announcing the available content on the
network. The key used is the sha256 digest of the content, together with the byte range. 

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

#### P2P Proxy Server

The p2p proxy server (a.k.a. p2p mirror) serves the node’s content from the file cache.
The Overlaybd TCMU driver accesses this proxy server at container runtime.

The driver makes requests like the following to the p2p proxy.

```bash
GET http://localhost:5000/blobs/https://westus2.data.mcr.microsoft.com/01031d61e1024861afee5d512651eb9f36fskt2ei//docker/registry/v2/blobs/sha256/1b/1b930d010525941c1d56ec53b97bd057a67ae1865eebf042686d2a2d18271ced/data?se=20230920T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f

Range: bytes=456-990
```

Here, the p2p proxy is listening at `localhost:5000`, and it is passed in the full SAS URL of the layer. The SAS URL was
previously obtained by the driver from the ACR. The proxy will first attempt to locate this content in the p2p network
using the router. If found, the peer will be used to reverse proxy the request. Otherwise, after the configured resolution
timeout, the request will be proxied to the upstream storage account.

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

[Azure Artifact Streaming]: https://learn.microsoft.com/en-us/azure/container-registry/container-registry-artifact-streaming
[Azure Blob]: https://learn.microsoft.com/en-us/azure/storage/blobs/storage-blobs-introduction
[containerd hosts]: https://github.com/containerd/containerd/blob/main/docs/hosts.md
[file-system-layout]: ../assets/images/file-system-layout.png
[kademlia-white-paper]: https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf
[normal-pull-summary]: ../assets/mermaid/rendered/normal-pull-summary.png
[normal-streaming-summary]: ../assets/mermaid/rendered/normal-streaming-summary.png
[Overlaybd]: https://github.com/containerd/overlaybd
[p2p proxy]: https://github.com/containerd/overlaybd/blob/main/src/example_config/overlaybd.json#L27C5-L30C7
[peerd-pull]: ../assets/mermaid/rendered/peerd-pull.png
[peerd-pull-seq]: ../assets/mermaid/rendered/peerd-pull-seq.png
[peerd-pull-summary]: ../assets/mermaid/rendered/peerd-pull-summary.png
[peerd-streaming]: ../assets/mermaid/rendered/peerd-streaming.png
[peerd-streaming-seq]: ../assets/mermaid/rendered/peerd-streaming-seq.png
[peerd-streaming-summary]: ../assets/mermaid/rendered/peerd-streaming-summary.png
[peerd-dht-topo]: ../assets/mermaid/rendered/peerd-dht-topo.png
[SAS URL]: https://learn.microsoft.com/en-us/azure/storage/common/storage-sas-overview
[Spegel]: https://github.com/XenitAB/spegel
[swagger.yaml]: ./api/swagger.yaml