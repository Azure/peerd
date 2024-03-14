# Peerd Design

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

---

[cluster-arch]: ../assets/images/cluster.png
[file-system-layout]: ../assets/images/file-system-layout.png
