/*
DONE
package gno implements a concrete LightClientModule, ClientState, ConsensusState,
Header, Misbehaviour and types for the Tendermint 2 / GNO consensus light client.
This implementation is based off the ICS 07 specification for Tendermint consensus
(https://github.com/cosmos/ibc/tree/main/spec/client/ics-007-tendermint-client)

Note that client identifiers are expected to be in the form: 10-gno-{N}.
Client identifiers are generated and validated by core IBC, unexpected client identifiers will result in errors.
*/
package gno
