FROM mcr.microsoft.com/oss/go/microsoft/golang:1.22-fips-cbl-mariner2.0 as builder

COPY ./ /src/

RUN tdnf install make -y && \
    tdnf install git -y

WORKDIR /src

RUN make tests-build

FROM mcr.microsoft.com/cbl-mariner/base/core:2.0 as runtime

ARG USER_ID=6192

RUN tdnf update -y && \
    tdnf install ca-certificates-microsoft -y && \
    tdnf install shadow-utils -y

RUN groupadd -g $USER_ID random && \
    useradd -g random -u $USER_ID random

COPY --from=builder --chown=scanner:root /src/bin/tests/tests /src/bin/tests/tests

ENTRYPOINT ["/src/bin/tests/tests", "random"]
