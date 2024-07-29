graph TD;
     subgraph Cluster[Normal Image Streaming in a Kubernetes Cluster]
        subgraph fs-1[Filesystem]
            subgraph store-1["Files"]
                sf-2[sha256:l2, bytes=10-4500]
            end
        end

        subgraph fs-2[Filesystem]
            subgraph store-2["Files"]
                sf-6[sha256:l6, bytes=100-1000]
                sf-3[sha256:l3, bytes=0-10000]
            end
        end

        subgraph fs-3[Filesystem]
            subgraph store-3["Files"]
                sf-4[sha256:l4, bytes=90-1000]
                sf-5[sha256:l5, bytes=0-700]
            end
        end

        subgraph Node1[Node A]
            direction TB
            kubelet["kubectl run mcr.microsoft.com/nginx:streamable"]
            fs-1

            kubelet ~~~ fs-1
        end

        subgraph Node2[Node B]
            fs-2
        end

        subgraph Node3[Node C]
            fs-3
        end
    end
    
    subgraph Upstream[Upstream Container Registry]
        acr(mcr.microsoft.com)
    end

    Node1 --> |<b style="color:blue">GET sha256:l6<br>bytes=101-500</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l3<br>bytes10-790</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l4<br>bytes=91-500</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l5<br>bytes=0-700</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:l1<br>bytes=800-9000</b>| acr
    Node1 --> |<b style="color:blue">GET sha256:c1<br>bytes=0-10000</b>| acr    

    classDef cluster fill:#fafafa,stroke:#bbb,stroke-width:2px,color:#326ce5;
    class Node1,NodeN cluster

    classDef registry fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#326ce5;
    class acr registry

    classDef outer fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#a9a9a9;
    class Cluster outer
