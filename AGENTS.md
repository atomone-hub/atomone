# AGENTS.md

## Project Overview

AtomOne is a Cosmos SDK-based blockchain node, forked from Cosmos Hub (v15.2.0). It uses a custom Cosmos SDK fork (`atomone-hub/cosmos-sdk`) and adds custom modules for governance DAOs, dynamic fees, governance overrides, and photon tokens. The binary is `atomoned`.

## Build & Development Commands

```bash
make build              # Compile binary to build/
make install            # Install to $GOPATH/bin
make lint               # Run golangci-lint (10m timeout)
make format             # Run gofumpt (required before committing)
make lint-fix           # Auto-fix lint issues

make test-unit          # Unit tests (5m timeout)
make test-race          # Race condition tests
make test-e2e           # E2E tests (25m timeout, uses Docker)
make run-tests          # All tests with optional tparse

make proto-all          # Format, lint, and generate protobuf code
make proto-gen          # Generate Go code from .proto files

make localnet-start     # Start single-node local network
```

Run a single test:
```bash
go test ./x/photon/keeper/... -run TestMintPhoton -v
```

## Architecture

**App wiring:** `app/app.go` defines `AtomOneApp`. Keepers are instantiated in `app/keepers/`, modules registered in `app/modules.go`, upgrade handlers in `app/upgrades/`.

**Custom modules (`x/`):**
- `x/gov` — Governance wrapper augmenting Cosmos SDK gov (restricts validator voting via delegation)
- `x/photon` — Photon token minting/burning with fee exceptions
- `x/dynamicfee` — Dynamic fee market calculations (ante + post handlers)
- `x/coredaos` — Core DAO address management (Oversight, Photon DAOs)

Each module follows standard Cosmos SDK structure: `keeper/`, `types/`, `client/cli/`, `ante/` or `post/`, `testutil/`, `module.go`.

**Ante/Post handlers:** `ante/` contains the main ante handler chain including a gov vote filter that prevents delegated voting. `post/` handles fee/photon deductions after tx execution.

**Protobuf:** Definitions in `proto/atomone/`. Generated via Docker (`proto-builder`). Generated files (`*.pb.go`, `*.pb.gw.go`) are committed.

## Code Conventions

- **Import order** (enforced by linter): standard → default → blank → dot → cometbft → cosmos → cosmossdk → cosmos-sdk → atomone-hub
- **Commit messages:** `type(scope): description` (types: `chore`, `fix`, `feat`, `build`, `refactor`, `test`, `docs`)
- **Testing:** Table-driven tests with `require` assertions (testify). E2E tests use Docker-based CometBFT networks.
- **Linting:** `.golangci.yml` enables ~20 linters. Skips test files and protobuf generated code.
- Always run `make format` before committing.

## Migrations & Upgrades

Two distinct mechanisms exist and must not be confused:

**App-level upgrade handlers** (`app/upgrades/`): Callbacks triggered by on-chain governance upgrade proposals. Registered in `app/app.go` via `Upgrades` slice. At the proposal height, `PreBlocker` calls `ApplyUpgrade` which invokes the handler. Without a governance proposal, no handler runs.

**Module consensus version migrations** (`RegisterMigration` in SDK modules): Migration functions registered inside each module (e.g. staking `Migrate5to6`). These only execute when `mm.RunMigrations()` is called — typically from inside an app-level upgrade handler.

**Key behaviors:**
- There is **no automatic check** at startup comparing a module's code consensus version vs stored version. A version mismatch alone does not cause a panic.
- If a new binary is deployed without a governance upgrade proposal, `RunMigrations` never runs. The new module code operates against old store data. For protobuf fields added by a migration, they deserialize as zero values, which can cause silent functional breakage (not crashes).
- `RunMigrations` diffs the stored `VersionMap` against each module's `ConsensusVersion()` and runs registered migration handlers sequentially. If a handler is missing for any version step, it errors.
- To set chain-specific param values (e.g. fixed commission rates), add logic in the upgrade handler **after** `RunMigrations`, or use a governance `MsgUpdateParams` proposal after the upgrade.

**Adding a new upgrade:** Create a new package in `app/upgrades/` (see `v4/` as reference), add it to the `Upgrades` slice in `app/app.go`. The handler should call `mm.RunMigrations(ctx, configurator, vm)` to trigger all pending module migrations.

## Key Dependencies

- Cosmos SDK: custom fork at `github.com/atomone-hub/cosmos-sdk`
- CometBFT: v0.38.x
- IBC-go: v10.x
- Bech32 prefix: `atone` (not `cosmos`)
