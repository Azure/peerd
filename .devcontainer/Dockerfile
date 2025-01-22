# This is a dockerfile specifically for running as a devcontainer
FROM mcr.microsoft.com/oss/go/microsoft/golang:1.23-bullseye
RUN apt-get update -y && \
    apt-get install make -y && \
    apt-get install git -y && \
    apt-get install gawk -y && \
    apt-get install jq -y && \
    apt-get install gettext-base -y

RUN go install github.com/cweill/gotests/gotests@latest && \
    go install github.com/fatih/gomodifytags@latest && \
    go install github.com/josharian/impl@latest && \
    go install github.com/haya14busa/goplay/cmd/goplay@latest && \
    go install github.com/go-delve/delve/cmd/dlv@latest && \
    go install honnef.co/go/tools/cmd/staticcheck@latest && \
    go install golang.org/x/tools/gopls@latest && \
    go install github.com/axw/gocov/gocov@latest && \
    go install gotest.tools/gotestsum@latest && \
    go install github.com/jandelgado/gcov2lcov@latest && \
    go install github.com/AlekSi/gocov-xml@latest && \
    go install sigs.k8s.io/kind@v0.22.0 && \
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.63.4 && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && chmod +x kubectl && mkdir -p ~/.local/bin && mv ./kubectl /usr/local/bin/kubectl && \
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 && chmod 700 get_helm.sh && ./get_helm.sh

RUN mkdir -p /go/src/github.com/azure/peerd

WORKDIR /go/src/github.com/azure/peerd
