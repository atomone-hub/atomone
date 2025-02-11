ARG GO_VERSION
ARG IMG_TAG=latest

# Compile the atomoned binary
FROM golang:$GO_VERSION-alpine AS builder
WORKDIR /src/app/
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ENV PACKAGES="curl make git libc-dev bash gcc linux-headers eudev-dev python3"
RUN apk add --no-cache $PACKAGES

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 make install

# Add to a distroless container
FROM alpine:$IMG_TAG
RUN adduser -D nonroot
ARG IMG_TAG
COPY --from=builder /go/bin/atomoned /usr/local/bin/
EXPOSE 26656 26657 1317 9090
USER nonroot

ENTRYPOINT ["atomoned"]
CMD ["start"]
