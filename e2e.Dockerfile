ARG IMG_TAG=latest

# Compile the atomoned binary
FROM golang:1.21-alpine AS atomoned-builder
WORKDIR /src/app/
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ENV PACKAGES curl make git libc-dev bash gcc linux-headers eudev-dev python3
RUN apk add --no-cache $PACKAGES
RUN CGO_ENABLED=0 make install

# Add to a distroless container
FROM alpine:$IMG_TAG
RUN adduser -D nonroot
ARG IMG_TAG
COPY --from=atomoned-builder /go/bin/atomoned /usr/local/bin/
EXPOSE 26656 26657 1317 9090
USER nonroot

# TODO: Why is min gas price not being read from the config?
ENTRYPOINT ["atomoned", "start", "--minimum-gas-prices", "0.00001uatone"]
