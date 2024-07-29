graph TD;
     subgraph Cluster[Peer to Peer Image Pull in a Kubernetes Cluster]
        subgraph ctr-1[Containerd]
            subgraph store-1["Content Store"]
                sl-2[sha256:l2]
            end
        end

        subgraph ctr-2[Containerd]
            subgraph store-2["Content Store"]
                sl-6[sha256:l6]
                sl-3[sha256:l3]
            end
        end

        subgraph ctr-3[Containerd]
            subgraph store-3["Content Store"]
                sl-4[sha256:l4]
                sl-5[sha256:l5]
            end
        end

        subgraph Node1[Node A]
            direction TB
            kubelet["kubectl run mcr.microsoft.com/nginx:latest"]
            ctr-1

            kubelet ~~~ ctr-1
        end

        subgraph Node2[Node B]
            ctr-2
        end

        subgraph Node3[Node C]
            ctr-3
        end
    end

    subgraph manifest-1[mcr.microsft.com/nginx@sha256:m1]
        direction TB
        c-1[config sha256:c1]
        l-1[layer sha256:l1]
        l-2[layer sha256:l2]
        l-3[layer sha256:l3]
        l-4[layer sha256:l4]
        l-5[layer sha256:l5]
        l-6[layer sha256:l6]

        c-1 ~~~ l-1
        l-1 ~~~ l-2
        l-2 ~~~ l-3
        l-3 ~~~ l-4
        l-4 ~~~ l-5
        l-5 ~~~ l-6
    end

    subgraph Upstream[Upstream Container Registry]
        acr(mcr.microsoft.com)
    end

    subgraph Legend[Legend]
        direction TB
        mtls[<b style="color:orange">Pull from Peer</b> - mTLS connections]
        tls[<b style="color:blue">Pull from Upstream</b> - TLS connections]

        mtls ~~~ tls
    end

    Legend ~~~ Upstream

    Node1 -.-> |<b style="color:orange"><br>GET sha256:l6</b>| sl-6
    Node1 -.-> |<b style="color:orange"><br>GET sha256:l3</b>| sl-3
    Node1 -.-> |<b style="color:orange"><br><br>GET sha256:l4</b>| sl-4
    Node1 -.-> |<b style="color:orange"><br>GET sha256:l5</b>| sl-5

    Node1 --> |<b style="color:blue"><br>GET sha256:l1</b>| acr
    Node1 --> |<b style="color:blue"><br>GET sha256:c1</b>| acr    

    classDef cluster fill:#fafafa,stroke:#bbb,stroke-width:2px,color:#326ce5;
    class Node1,NodeN cluster

    classDef registry fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#326ce5;
    class acr registry

    classDef outer fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#a9a9a9;
    class Cluster outer
