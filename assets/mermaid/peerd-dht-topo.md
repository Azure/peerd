graph TD;
     subgraph Cluster[DHT Topology in a Kubernetes Cluster]
        direction LR
        subgraph peerd-1[Peerd]
            dht-1(DHT)
        end

        subgraph peerd-2[Peerd]
            dht-2(DHT)
        end

        subgraph peerd-3[Peerd]
            dht-3(DHT)
        end

        subgraph Node1[Node A]
            peerd-1
        end
        
        subgraph Node2[Node B]
            peerd-2(Peerd)
        end

        subgraph Node3[Node C]
            peerd-3(Peerd)
        end

        subgraph k8s-api[K8s API Server]
            lease-1((("Peerd Leader
            Lease Resource")))
        end
    end

    dht-1 o-.-o |<b style="color:orange">Initialize<br><br></b>| lease-1
    dht-2 o-.-o |<b style="color:orange">Initialize<br><br></b>| lease-1
    dht-3 o-.-o |<b style="color:orange">Initialize<br><br></b>| lease-1

    dht-1 <==> |<b style="color:blue">State<br><br></b>| dht-2
    dht-1 <==> |<b style="color:blue">State<br><br></b>| dht-3
    dht-2 <==> |<b style="color:blue">State<br><br></b>| dht-3

    classDef cluster fill:#fafafa,stroke:#bbb,stroke-width:2px,color:#326ce5;
    class Node1,NodeN cluster

    classDef outer fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#a9a9a9;
    class Cluster outer

    subgraph Legend[Legend]
        direction TB
        tls[<b style="color:orange">Initialize</b> - TLS connections]
        mtls[<b style="color:blue">State</b> - mTLS connections]
    end

    Cluster ~~~ Legend
