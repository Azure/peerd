graph TB;
     subgraph Cluster[Peer-to-Peer Image Pulling in a Kubernetes Cluster]
        direction LR
        subgraph app-1[mcr.microsoft.com/nginx:latest]
            app-pod-1((Pod))
        end

        subgraph ctr-1[Containerd]
            client-1{Client}
            store-1[[Content Store]]

            subgraph hosts-1["Containerd Hosts Configuration"]
                h-1[(mcr.microsoft.com pull mirror: peerd)]
            end
        end

        subgraph peerd-1[Peerd]
            proxy-1(Proxy)
            sub-1(((Subscription)))
        end

        subgraph Node1[Node A]
            hosts-1
            app-1
            peerd-1
            ctr-1
        end
        
        subgraph NodeN[Node N]
            peerd-n(Peerd)
        end

    end

    subgraph manifest-1[mcr.microsft.com/nginx@sha256:m1]
        direction TB
        c-1[config sha256:c1]
        l-1[layer sha256:l1]
        l-2[layer sha256:l2]
    end

    subgraph Upstream[Upstream Container Registry]
        acr(mcr.microsoft.com)
    end

    hosts-1 ~~~ client-1
    c-1 ~~~ l-1
    l-1 ~~~ l-2

    client-1 --> |<b style="color:orange">1</b>| proxy-1
    proxy-1 -.-> |<b style="color:orange">&nbsp&nbsp&nbsp&nbsp2</b>| peerd-n
    client-1 -.-> |<b style="color:orange">&nbsp&nbsp&nbsp&nbsp3</b>| acr
    client-1 --o |<b style="color:orange">&nbsp&nbsp&nbsp&nbsp4</b>| app-1

    sub-1 o-.-o store-1
    sub-1 o-.-o |<b style="color:darkgray">Advertise</b>| peerd-n
    
    classDef containerd fill:#e0ffff,stroke:#000,stroke-width:4px,color:#000;

    classDef cluster fill:#fafafa,stroke:#bbb,stroke-width:2px,color:#326ce5;
    class Node1,NodeN cluster

    classDef registry fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#326ce5;
    class acr registry

    classDef outer fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#a9a9a9;
    class Cluster outer
