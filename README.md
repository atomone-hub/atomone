# AtomOne

AtomOne is built using the [Cosmos SDK](https://github.com/cosmos/cosmos-sdk) as a fork of the
[Cosmos Hub](https://github.com/cosmos/gaia) at version [v15.2.0](https://github.com/cosmos/gaia/releases/tag/v15.2.0) (common commit hash 7281c9b).

The following modifications have been made to the Cosmos Hub software to create AtomOne:

1. Removed x/globalfee module and revert to older and simpler fee decorator
2. Removed Packet Forwarding Middleware
3. Removed Interchain Security module
4. Reverted to standard Cosmos SDK v0.47.10 without the Liquid Staking Module (LSM)
5. Changed Bech32 prefixes to `atone` (see `cmd/atomoned/cmd/config.go`)
6. Removed ability for validators to vote on proposals with delegations, they can only use their own stake

## Reproducible builds (TODO)

An effort has been made to make it possible to build the exact same binary
locally as the Github Release section. To do this, checkout to the expected
version and then simply run `make build` (which will output the binary to the
`build` directory) or `make install`. The resulted binary should have the same
sha256 hash than the one from the Github Release section.
