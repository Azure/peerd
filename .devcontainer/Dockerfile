# This is a dockerfile specifically for running as a devcontainer
FROM mcr.microsoft.com/oss/go/microsoft/golang:1.22-fips-cbl-mariner2.0
RUN tdnf update -y && tdnf install make -y && tdnf install git -y && tdnf install gawk -y && tdnf install jq -y

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
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.54.2

RUN mkdir -p /go/src/github.com/azure/peerd

WORKDIR /go/src/github.com/azure/peerd
