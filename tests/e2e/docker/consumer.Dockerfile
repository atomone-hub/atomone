# Dockerfile for ICS1 Consumer Chain (for e2e testing)
# Clones the ICS1 repository and uses its Makefile to build the consumer daemon

ARG GO_VERSION=1.24
ARG IMG_TAG=3.19
ARG ICS_VERSION=v1.0.0-ics1-rc.1

FROM golang:${GO_VERSION}-alpine AS consumer-builder

RUN set -eux; apk add --no-cache ca-certificates build-base git linux-headers

# Set working directory
WORKDIR /src

# Clone the ICS1 repository at the specific tag
ARG ICS_VERSION
RUN git clone --depth 1 --branch ${ICS_VERSION} https://github.com/allinbits/interchain-security.git

WORKDIR /src/interchain-security

# Use the Makefile to install the consumer daemon
# The ICS1 repo should have make targets for building binaries
RUN make install

# Create final image
FROM alpine:${IMG_TAG}

COPY --from=consumer-builder /go/bin/interchain-security-cd /bin/interchain-security-cd

ENV HOME=/consumer
WORKDIR $HOME

EXPOSE 26656 26657 1317 9090

ENTRYPOINT ["interchain-security-cd"]
