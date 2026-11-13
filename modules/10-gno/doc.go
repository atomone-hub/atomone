/*
Package gno implements a concrete LightClientModule, ClientState, ConsensusState,
Header, Misbehaviour and types for the Tendermint 2 / GNO consensus light client.
This implementation is based off the ICS 07 specification for Tendermint consensus
(https://github.com/cosmos/ibc/tree/main/spec/client/ics-007-tendermint-client)

Validators of the counterparty chain may sign with ed25519 or secp256k1 consensus
keys, and a validator set may hold both. A secp256k1 signature is 64 bytes, r||s
with a low s, over the SHA-256 of the vote sign bytes. It is verified by the Gno
tm2 library, which has a pure Go implementation and, under the libsecp256k1 build
tag, a cgo one. Both accept the same signatures, but nodes are expected to build
without that tag so that all of them run the same verification code.

Note that client identifiers are expected to be in the form: 10-gno-{N}.
Client identifiers are generated and validated by core IBC, unexpected client identifiers will result in errors.
*/
package gno
