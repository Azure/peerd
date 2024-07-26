graph RL;
     subgraph Cluster[Peer-to-Peer Artifact Streaming in a Kubernetes Cluster]
        direction LR
        subgraph kernel-1[Kernel]
            fs-1[Filesystem]
        end
        
        subgraph app-1[Nginx]
            app-pod-1((Pod))
        end

        subgraph overlaybd-1["(User Space)"]
            driver-1["Overlaybd
            TCMU"]
        end
        
        subgraph peerd-1[Peerd]
            proxy-1(Proxy)
            files-1(("Files
            Cache"))
        end

        subgraph Node1[Node A]
            kernel-1
            app-1
            overlaybd-1
            peerd-1
        end
        
        subgraph NodeN[Node N]
            peerd-n(Peerd)
        end

        files-1 o-.-o |<b style="color:darkgray"><br>Advertise</b>| peerd-n

        app-pod-1 --> |<b style="color:orange"><br>1</b>| fs-1
        fs-1 -.-> |<b style="color:orange"><br>2</b>| driver-1
        driver-1 --> |<b style="color:orange"><br>3</b>| proxy-1
        proxy-1 <-.-> |<b style="color:orange"><br>4</b>| peerd-n
    end

    subgraph Upstream[Upstream Container Registry]
        acr(mcr.microsoft.com)
    end

    proxy-1 -.-> |<b style="color:orange"><br>5</b>| acr

    classDef userspace fill:#e0ffff,stroke:#000,stroke-width:4px,color:#000;
    class proxy-1,files-1,driver-1,app-pod-1,peerd-n userspace

    classDef cluster fill:#fafafa,stroke:#bbb,stroke-width:2px,color:#326ce5;
    class Node1,NodeN cluster

    classDef registry fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#326ce5;
    class acr registry

    classDef outer fill:#e0f7fa,stroke:#00008b,stroke-width:2px,color:#a9a9a9;
    class Cluster outer
