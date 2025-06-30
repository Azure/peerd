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

Peer to peer file sharing by specifying the range of bytes to read.

* This enables block level file drivers, such as [Overlaybd], to use `peerd` as the p2p proxy.
* This test is run by deploying the `random` test workload to the kind cluster.
* The workload is deployed to each node, and outputs performance metrics that are observed by it, such as the speed
  of downloads and error rates.

```bash
$ make ci-kind-random
  ...
  {"level":"info","node":"random-zb9vm","version":"bb7ee6a","mode":"upstream","size":22980743,"readsPerBlob":5,"time":"2024-03-07T21:50:29Z","message":"downloading blob"}
  {"level":"info","node":"random-9gcvw","version":"bb7ee6a","upstream.p50":21.25170790666404,"upstream.p75":5.834663359546446,"upstream.p90":0.7871542327673121,"upstream.p95":0.2965091294200036,"upstream.p100":0.2645602612715345,"time":"2024-03-07T21:50:34Z","message":"speeds (MB/s)"}
  {"level":"info","node":"random-9gcvw","version":"bb7ee6a","p2p.p50":5.802082290454193,"p2p.p75":1.986398855488793,"p2p.p90":0.6210418172329215,"p2p.p95":0.0523776186045032,"p2p.p100":0.023341096448268952,"time":"2024-03-07T21:50:34Z","message":"speeds (MB/s)"}
  {"level":"info","node":"random-9gcvw","version":"bb7ee6a","p2p.error_rate":0,"upstream.error_rate":0,"time":"2024-03-07T21:50:34Z","message":"error rates"}
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