FROM mcr.microsoft.com/oss/go/microsoft/golang:1.24-fips-azurelinux3.0 AS builder

COPY ./ /

RUN tdnf install make -y && \
    tdnf install git -y

WORKDIR /

RUN make install

FROM mcr.microsoft.com/azurelinux/base/core:3.0 AS runtime

ARG USER_ID=6191

ENV GIN_MODE=release

# Install useful packages
RUN tdnf update -y && \
    tdnf install ca-certificates-microsoft -y && \
    tdnf install shadow-utils -y && \
    tdnf install net-tools -y && \
    tdnf install tcpdump -y

RUN groupadd -g $USER_ID peerd && \
    useradd -g peerd -u $USER_ID peerd

COPY --from=builder --chown=peerd:root /bin/ /bin/

ENTRYPOINT ["/bin/peerd"]
