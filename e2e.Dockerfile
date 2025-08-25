ARG GO_VERSION

# Compile the atomoned binary
# we should always pass in a GO_VERSION, but we should also set a default
# to prevent breakage. Our `make install` also runs a sanity check, so this
# is safe and please the linter.
FROM golang:${GO_VERSION:-1.24.5}-alpine AS atomoned-builder
WORKDIR /src/app/
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN apk add --no-cache curl make git libc-dev bash gcc linux-headers eudev-dev python3
RUN CGO_ENABLED=0 make install

# Add to a distroless container
FROM alpine:latest
RUN adduser -D nonroot
COPY --from=atomoned-builder /go/bin/atomoned /usr/local/bin/
EXPOSE 26656 26657 1317 9090
USER nonroot

ENTRYPOINT ["atomoned", "start"]
