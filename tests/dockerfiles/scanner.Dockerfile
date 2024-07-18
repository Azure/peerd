FROM mcr.microsoft.com/cbl-mariner/base/core:2.0 AS scannerbase

ARG FILE_PATH=/usr/local/bin/scannerbase

RUN dd if=/dev/urandom of=$FILE_PATH bs=1 count=$((600 * 1024 * 1024))

FROM mcr.microsoft.com/oss/go/microsoft/golang:1.22-fips-cbl-mariner2.0 as builder

COPY ./ /src/

RUN tdnf install make -y && \
    tdnf install git -y

WORKDIR /src

RUN make tests-build

FROM mcr.microsoft.com/cbl-mariner/base/core:2.0 as scanner

ARG USER_ID=6190

RUN tdnf update -y && \
    tdnf install ca-certificates-microsoft -y && \
    tdnf install shadow-utils -y

RUN groupadd -g $USER_ID scanner && \
    useradd -g scanner -u $USER_ID scanner

COPY --from=scannerbase --chown=scanner:root /usr/local/bin/scannerbase /usr/local/bin/scannerbase
COPY --from=builder --chown=scanner:root /src/bin/tests/tests /src/bin/tests/tests

EXPOSE 5004

ENTRYPOINT ["/src/bin/tests/tests", "scanner"]
