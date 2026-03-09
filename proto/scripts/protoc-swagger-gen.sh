#!/usr/bin/env bash

set -eo pipefail

cd proto
# Generate atomone & cosmos-sdj swagger files
# We don't care about filtering query.proto or service.proto files, because
# client/docs/config.json has a white list of the required files.
buf generate --template buf.gen.swagger.yaml
buf generate --template buf.gen.swagger.yaml buf.build/cosmos/cosmos-sdk

# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ../client/docs/config.json -o ../client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

# clean swagger files
rm -rf ./tmp-swagger-gen
