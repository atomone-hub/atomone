#!/usr/bin/make -f

COMMIT := $(shell git log -1 --format='%H')

VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null)
ifeq (,$(VERSION))
  PREVIOUS_TAG := $(shell git describe --tags --abbrev=0)
  SHORT_COMMIT := $(shell git rev-parse --short HEAD)
  VERSION := $(PREVIOUS_TAG)-$(SHORT_COMMIT)
endif

LEDGER_ENABLED ?= false
TM_VERSION := $(shell go list -f {{.Version}} -m github.com/cometbft/cometbft)
DOCKER := $(shell which docker)
BUILDDIR ?= $(CURDIR)/build
TEST_DOCKER_REPO=cosmos/contrib-atomonetest

GO_SYSTEM_VERSION = $(shell go env GOVERSION | cut -c 3-)
GO_REQUIRED_VERSION = $(shell go list -f {{.GoVersion}} -m)

# command to run dependency utilities
rundep=go run -modfile contrib/devdeps/go.mod

# process build tags
build_tags = netgo
ifeq ($(LEDGER_ENABLED),false)
  export CGO_ENABLED = 0
else
  $(info WARNING: Ledger build involves enabling cgo, which disables the ability to have reproducible builds.)
  export CGO_ENABLED = 1
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(ATOMONE_BUILD_OPTIONS)))
  build_tags += gcc cleveldb
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace := $(whitespace) $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=atomone \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=atomoned \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep) \
			-X github.com/cometbft/cometbft/version.TMCoreSemVer=$(TM_VERSION)

ifeq (cleveldb,$(findstring cleveldb,$(ATOMONE_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq (,$(findstring nostrip,$(ATOMONE_BUILD_OPTIONS)))
  ldflags += -w -s
endif
ifneq ($(strip $(MIN_VOTING_PERIOD)),)
	ldflags += -X github.com/atomone-hub/atomone/x/gov/types/v1.MinVotingPeriod=$(MIN_VOTING_PERIOD)
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(ATOMONE_BUILD_OPTIONS)))
#  BUILD_FLAGS += -trimpath
endif

###############################################################################
###                              Build                                      ###
###############################################################################

all: install lint run-tests test-e2e vulncheck

print_tm_version:
	@echo $(TM_VERSION)

check_go_version:
ifneq ($(GO_SYSTEM_VERSION), $(GO_REQUIRED_VERSION))
	@echo 'ERROR: Go version $(GO_REQUIRED_VERSION) is required for building AtomOne'
	@echo '--> You can install it using:'
	@echo 'go install golang.org/dl/go$(GO_REQUIRED_VERSION)@latest && go$(GO_REQUIRED_VERSION) download'
	@echo '--> Then prefix your make command with:'
	@echo 'GOROOT=$$(go$(GO_REQUIRED_VERSION) env GOROOT) PATH=$$GOROOT/bin:$$PATH'
	exit 1
endif

check_ledger:
ifeq ($(LEDGER_ENABLED),false)
	$(info Building without Ledger support. Set LEDGER_ENABLED=true or use build-ledger target to build with Ledger support.)
endif

BUILD_TARGETS := build install

build: BUILD_ARGS=-o $(BUILDDIR)/

$(BUILD_TARGETS): check_go_version check_ledger go.sum $(BUILDDIR)/
	go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

build-ledger: # Kept for convenience
	$(MAKE) build LEDGER_ENABLED=true

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

vulncheck:
	$(rundep) golang.org/x/vuln/cmd/govulncheck ./...

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

clean:
	rm -rf $(BUILDDIR)/ artifacts/

.PHONY: all build build-ledger install vulncheck clean clean print_tm_version go-mod-cache

###############################################################################
###                                Release                                  ###
###############################################################################

# create tag and run goreleaser without publishing
create-release-dry-run: check_go_version
ifneq ($(strip $(TAG)),)
	@echo "--> Dry running release for tag: $(TAG)"
	@echo "--> Create tag: $(TAG) dry run"
	git tag -s $(TAG) -m $(TAG)
	git push origin $(TAG) --dry-run
	@echo "--> Running goreleaser"
	TM_VERSION=$(TM_VERSION) $(rundep) github.com/goreleaser/goreleaser release --clean --skip=publish
	@echo "--> Done create-release-dry-run for tag: $(TAG)"
	cat dist/SHA256SUMS-$(TAG).txt
	@echo "--> Delete local tag: $(TAG)"
	@git tag -d $(TAG)
else
	@echo "--> No tag specified, skipping tag release"
endif

# create tag and publish it
create-release:
ifneq ($(strip $(TAG)),)
	@echo "--> Running release for tag: $(TAG)"
	@echo "--> Create release tag: $(TAG)"
	git tag -s $(TAG) -m $(TAG)
	git push origin $(TAG)
	@echo "--> Done creating release tag: $(TAG)"
else
	@echo "--> No tag specified, skipping create-release"
endif

.PHONY: create-release-dry-run create-release

###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

include sims.mk

PACKAGES_UNIT=$(shell go list ./... | grep -v -e '/tests/e2e')
PACKAGES_E2E=$(shell cd tests/e2e && go list ./... | grep '/e2e')
TEST_PACKAGES=./...
TEST_TARGETS := test-unit test-unit-cover test-race test-e2e

test-unit: ARGS=-timeout=5m -tags='norace'
test-unit: TEST_PACKAGES=$(PACKAGES_UNIT)
test-unit-cover: ARGS=-timeout=5m -tags='norace' -coverprofile=coverage.txt -covermode=atomic
test-unit-cover: TEST_PACKAGES=$(PACKAGES_UNIT)
test-race: ARGS=-timeout=5m -race
test-race: TEST_PACKAGES=$(PACKAGES_UNIT)
test-e2e: ARGS=-timeout=25m -v
test-e2e: TEST_PACKAGES=$(PACKAGES_E2E)
$(TEST_TARGETS): run-tests

run-tests:
ifneq (,$(shell which tparse 2>/dev/null))
	@echo "--> Running tests"
	@go test -mod=readonly -json $(ARGS) $(TEST_PACKAGES) | tparse
else
	@echo "--> Running tests"
	@go test -mod=readonly $(ARGS) $(TEST_PACKAGES)
endif

.PHONY: run-tests $(TEST_TARGETS)

docker-build-debug:
	@docker build -t cosmos/atomoned-e2e -f e2e.Dockerfile --build-arg GO_VERSION=$(GO_REQUIRED_VERSION) .

docker-build-hermes:
	@cd tests/e2e/docker; docker build -t ghcr.io/cosmos/hermes-e2e:1.0.0 -f hermes.Dockerfile .

docker-build-all: docker-build-debug docker-build-hermes

mockgen_cmd=$(rundep) github.com/golang/mock/mockgen

mocks-gen:
	$(mockgen_cmd) -source=x/gov/testutil/expected_keepers.go -package testutil -destination x/gov/testutil/expected_keepers_mocks.go
	$(mockgen_cmd) -source=x/photon/types/expected_keepers.go -package testutil -destination x/photon/testutil/expected_keepers_mocks.go
	$(mockgen_cmd) -source=x/photon/ante/expected_keepers.go -package ante_test -destination x/photon/ante/expected_keepers_mocks_test.go
	$(mockgen_cmd) -source=x/dynamicfee/ante/expected_keepers.go -package ante_test -destination x/dynamicfee/ante/expected_keepers_mocks_test.go
	$(mockgen_cmd) -source=x/dynamicfee/post/expected_keepers.go -package post_test -destination x/dynamicfee/post/expected_keepers_mocks_test.go
	$(mockgen_cmd) -source=x/coredaos/types/expected_keepers.go -package testutil -destination x/coredaos/testutil/expected_keepers_mocks.go

.PHONY: docker-build-debug docker-build-hermes docker-build-all mocks-gen

###############################################################################
###                                Linting                                  ###
###############################################################################

golangci_lint_cmd=$(rundep) github.com/golangci/golangci-lint/cmd/golangci-lint

# golangci might not work properly when run with newer versions of go, so we
# add a restriction by adding check_go_version as a dependency.
lint: check_go_version
	@echo "--> Running linter"
	@$(golangci_lint_cmd) run --timeout=10m

lint-fix: check_go_version
	@echo "--> Running linter fix"
	@$(golangci_lint_cmd) run --fix --out-format=tab --issues-exit-code=0

format: lint-fix
	@echo "--> Running gofumpt"
	@find . -name '*.go' -type f \
		-not -path "*.git*" \
		-not -name "*.pb.go" \
		-not -name "*.pb.gw.go" \
		| xargs $(rundep) mvdan.cc/gofumpt -w -l

.PHONY: format lint lint-fix

###############################################################################
###                                Localnet                                 ###
###############################################################################

localnet_home=~/.atomone-localnet
localnetd=./build/atomoned --home $(localnet_home)

localnet-start: build
	rm -rf ~/.atomone-localnet
	$(localnetd) init localnet --default-denom uatone --chain-id localnet
	$(localnetd) config set client chain-id localnet
	$(localnetd) config set client keyring-backend test
	$(localnetd) keys add val
	$(localnetd) genesis add-genesis-account val 1000000000000uatone,1000000000uphoton
	$(localnetd) keys add user
	$(localnetd) genesis add-genesis-account user 1000000000uatone,1000000000uphoton
	$(localnetd) genesis gentx val 1000000000uatone
	$(localnetd) genesis collect-gentxs
	# Add treasury DAO address
	$(localnetd) genesis add-genesis-account atone1qqqqqqqqqqqqqqqqqqqqqqqqqqqqp0dqtalx52 5388766663072uatone
	# Add CP funds
	$(localnetd) genesis add-genesis-account atone1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8flcml8 5388766663072uatone
	jq '.app_state.distribution.fee_pool.community_pool = [ { "denom": "uatone", "amount": "5388766663072.000000000000000000" }]' $(localnet_home)/config/genesis.json > /tmp/gen
	mv /tmp/gen $(localnet_home)/config/genesis.json
	# Previous add-genesis-account call added the auth module account as a BaseAccount, we need to remove it
	jq 'del(.app_state.auth.accounts[] | select(.address == "atone1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8flcml8"))' $(localnet_home)/config/genesis.json > /tmp/gen
	mv /tmp/gen $(localnet_home)/config/genesis.json
	# Set validator gas prices
	sed -i.bak 's#^minimum-gas-prices = .*#minimum-gas-prices = "0.01uatone,0.01uphoton"#g' $(localnet_home)/config/app.toml
	# enable REST API
	$(localnetd) config set app api.enable true
	# Decrease voting period to 5min
	jq '.app_state.gov.params.voting_period = "300s"' $(localnet_home)/config/genesis.json > /tmp/gen
	mv /tmp/gen $(localnet_home)/config/genesis.json
	jq --rawfile data contrib/localnet/constitution-mock.md '.app_state.gov.constitution=$$data' $(localnet_home)/config/genesis.json > /tmp/gen
	mv /tmp/gen $(localnet_home)/config/genesis.json
	$(localnetd) start

localnet-restart: build
	$(localnetd) start

localnet-submit-upgrade-proposal:
	$(localnetd) tx gov submit-proposal --from user contrib/localnet/proposal_upgrade.json -y --gas-prices 0.02uphoton
	sleep 5
	$(localnetd) tx gov vote 1 yes --from val -y --gas-prices 0.02uphoton

localnet-submit-text-proposal:
	$(localnetd) tx gov submit-proposal --from user contrib/localnet/proposal_text.json -y --gas-prices 0.02uphoton

.PHONY: localnet-start localnet-restart localnet-submit-upgrade-proposal localnet-submit-text-proposal

###############################################################################
###                                Docker                                   ###
###############################################################################

test-docker:
	@docker build -f contrib/Dockerfile.test -t ${TEST_DOCKER_REPO}:$(shell git rev-parse --short HEAD) .
	@docker tag ${TEST_DOCKER_REPO}:$(shell git rev-parse --short HEAD) ${TEST_DOCKER_REPO}:$(shell git rev-parse --abbrev-ref HEAD | sed 's#/#_#g')
	@docker tag ${TEST_DOCKER_REPO}:$(shell git rev-parse --short HEAD) ${TEST_DOCKER_REPO}:latest

test-docker-push: test-docker
	@docker push ${TEST_DOCKER_REPO}:$(shell git rev-parse --short HEAD)
	@docker push ${TEST_DOCKER_REPO}:$(shell git rev-parse --abbrev-ref HEAD | sed 's#/#_#g')
	@docker push ${TEST_DOCKER_REPO}:latest

.PHONY: test-docker test-docker-push

###############################################################################
###                                Protobuf                                 ###
###############################################################################
protoVer=0.17.0
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "--> Generating Protobuf files"
	@$(protoImage) sh ./proto/scripts/protocgen.sh

proto-swagger-gen:
	@echo "--> Generating Protobuf Swagger"
	@$(protoImage) sh ./proto/scripts/protoc-swagger-gen.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json

proto-check-breaking:
	@$(protoImage) buf breaking --against $(HTTPS_GIT)#branch=main

proto-update-deps:
	@echo "--> Updating Protobuf dependencies"
	$(DOCKER) run --rm -v $(CURDIR)/proto:/workspace --workdir /workspace $(protoImageName) buf mod update

.PHONY: proto-all proto-gen proto-swagger-gen proto-format proto-lint proto-check-breaking proto-update-deps
