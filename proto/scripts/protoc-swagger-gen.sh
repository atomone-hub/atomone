#!/usr/bin/env bash

set -eo pipefail

# The cosmos-sdk swagger files are generated from the protos of the AtomOne SDK
# module the binary is built against, which the Makefile mounts at
# /cosmos-sdk-proto. The AtomOne SDK does not carry the protos of the modules
# that come from the Cosmos SDK (evidence, feegrant, tx), so those are
# generated from the Cosmos SDK module first and the AtomOne SDK output then
# overwrites the files it provides.
SDK_PROTO_DIR=${SDK_PROTO_DIR:-/cosmos-sdk-proto}

cd proto
# Generate atomone & cosmos-sdk swagger files
# We don't care about filtering query.proto or service.proto files, because
# client/docs/config.json has a white list of the required files.
buf generate --template buf.gen.swagger.yaml
buf generate --template buf.gen.swagger.yaml buf.build/cosmos/cosmos-sdk:v0.50.0
buf generate --template buf.gen.swagger.yaml "$SDK_PROTO_DIR"

# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ../client/docs/config.json -o ../client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

# clean swagger files
rm -rf ./tmp-swagger-gen
