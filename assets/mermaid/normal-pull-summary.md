graph TD;
     subgraph Cluster[Normal Image Pull in a Kubernetes Cluster]
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
    
    subgraph Upstream[Upstream Container Registry]
        acr(mcr.microsoft.com)
    end

    Node1 --> |<b style="color:blue">GET sha256:l6</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l3</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l4</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l5</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l1</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:c1</b>| acr    

    classDef cluster fill:#fafafa,stroke:#bbb,stroke-width:2px,color:#326ce5;
    class Node1,NodeN cluster

    classDef registry fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#326ce5;
    class acr registry

    classDef outer fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#a9a9a9;
    class Cluster outer
