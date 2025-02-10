ARG GO_VERSION="1.22.10"
ARG ALPINE_VERSION=latest

# Compile the atomoned binary
FROM golang:$GO_VERSION-alpine AS builder
WORKDIR /src/app/
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ENV PACKAGES="curl make git libc-dev bash gcc linux-headers eudev-dev python3"
RUN apk add --no-cache $PACKAGES
RUN CGO_ENABLED=0 make install

# Final image
FROM alpine:$ALPINE_VERSION
RUN adduser -D nonroot
ARG ALPINE_VERSION
COPY --from=builder /go/bin/atomoned /usr/local/bin/
EXPOSE 26656 26657 1317 9090
USER nonroot
ENTRYPOINT [ "atomoned" ]
CMD [ "start" ]
