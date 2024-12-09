ARG IMG_TAG=latest

# Compile the atomoned binary
FROM golang:1.22-alpine AS atomoned-builder
WORKDIR /src/app/
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ENV PACKAGES curl make git libc-dev bash gcc linux-headers eudev-dev python3
RUN apk add --no-cache $PACKAGES
RUN CGO_ENABLED=0 make install

# Add to a distroless container
FROM cgr.dev/chainguard/static:$IMG_TAG
ARG IMG_TAG
COPY --from=atomoned-builder /go/bin/atomoned /usr/local/bin/
EXPOSE 26656 26657 1317 9090
USER 0

ENTRYPOINT ["atomoned", "start"]
