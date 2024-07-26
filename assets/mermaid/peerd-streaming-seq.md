sequenceDiagram
    Title: Peer-to-Peer Artifact Streaming in a Kubernetes Cluster

    box white Node A
        participant Nginx Pod
        participant File System
        participant Overlaybd TCMU
        participant Peerd-A
    end

    box white Node N
        participant Peerd-N
    end

    box white Upstream Registry
        participant Upstream
    end

    Nginx Pod->>File System: Read bytes 30-2000 from data.csv
    Note over Nginx Pod,File System: 1
    
    File System->>Overlaybd TCMU: Read bytes 30-2000 from data.csv
    Note over File System,Overlaybd TCMU: 2

    Overlaybd TCMU->>Peerd-A: Fetch file data.csv 'Range: bytes=30-2000'
    Note over Overlaybd TCMU,Peerd-A: 3
    activate Peerd-A
    
    alt bytes cached
        Peerd-A->>Overlaybd TCMU: result
    else peer found
        Peerd-A->>Peerd-N: Fetch file data.csv 'Range: bytes=30-2000'
        Note over Peerd-A,Peerd-N: 4
        activate Peerd-N
        Peerd-N->>Peerd-A: result
        Peerd-A->>Overlaybd TCMU: result
    else upstream request
        Peerd-A->>Upstream: Fetch file data.csv 'Range: bytes=30-2000'
        Note over Peerd-A,Upstream: 5
        Upstream->>Peerd-A: result
        Peerd-A->>Overlaybd TCMU: result
    end

    opt Optimistic File Prefetch
        activate Peerd-A
        Note right of Peerd-A: Prefetch entire file from peers/upstream
    end

    opt Advertise state (async)
        activate Peerd-A
        Note right of Peerd-A: Advertise state from files cache
    end

    Overlaybd TCMU->>File System: result
    File System->>Nginx Pod: result
