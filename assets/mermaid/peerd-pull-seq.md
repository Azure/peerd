sequenceDiagram
    Title: Peer-to-Peer Image Pulling in a Kubernetes Cluster

    box white Node A
        participant Nginx Pod
        participant Containerd Client
        participant Peerd-A
    end

    box white Node N
        participant Peerd-N
    end

    box white Upstream Registry
        participant Upstream
    end

    loop Every layer
        Containerd Client->>Peerd-A: GET sha256:l1
        Note over Containerd Client,Peerd-A: 1

        alt peer found
            Peerd-A->>Peerd-N: GET sha256:l1
            Note over Peerd-A,Peerd-N: 2
            activate Peerd-N
            Peerd-N->>Peerd-A: result
            Peerd-A->>Containerd Client: result
        else upstream request
            Containerd Client->>Upstream: GET sha256:l1
            Note over Peerd-A,Upstream: 3
            Upstream->>Containerd Client: result
        end

        opt Advertise state (async)
            activate Peerd-A
            Note right of Peerd-A: Advertise state from containerd content store
        end
    end

    Containerd Client-->Nginx Pod: start
