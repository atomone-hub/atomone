package spectre

import (
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"
)

var _ exported.ConsensusState = (*ConsensusState)(nil)

// ClientType returns the spectre client type.
func (ConsensusState) ClientType() string {
	return ModuleName
}

// GetRoot returns the execution state root of the finalized header as the
// commitment root.
func (cs *ConsensusState) GetRoot() exported.Root {
	return commitmenttypes.NewMerkleRoot(cs.StateRoot)
}

// GetTimestamp returns the execution timestamp of the finalized header in
// nanoseconds.
func (cs *ConsensusState) GetTimestamp() uint64 {
	return cs.Timestamp * 1_000_000_000
}

// ValidateBasic performs basic validation of the consensus state.
func (cs *ConsensusState) ValidateBasic() error {
	if cs.Slot == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "slot must be non-zero")
	}
	if len(cs.StateRoot) != 32 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "state root must be 32 bytes")
	}
	if cs.Timestamp == 0 {
		return errorsmod.Wrap(clienttypes.ErrInvalidConsensus, "timestamp must be non-zero")
	}
	if err := validateSummarizedSyncCommittee(&cs.CurrentSyncCommittee); err != nil {
		return errorsmod.Wrap(err, "invalid current sync committee")
	}
	if cs.NextSyncCommittee != nil {
		if err := validateSummarizedSyncCommittee(cs.NextSyncCommittee); err != nil {
			return errorsmod.Wrap(err, "invalid next sync committee")
		}
	}
	return nil
}

// validateSummarizedSyncCommittee validates the committee summary fields.
func validateSummarizedSyncCommittee(sc *SummarizedSyncCommittee) error {
	if len(sc.PubkeysHash) != 32 {
		return errorsmod.Wrap(ErrInvalidSyncCommittee, "pubkeys hash must be 32 bytes")
	}
	if len(sc.AggregatePubkey) != blsPubkeySize {
		return errorsmod.Wrap(ErrInvalidSyncCommittee, "aggregate pubkey must be 48 bytes")
	}
	// The aggregate pubkey must deserialize to a valid G1 point.
	if _, err := parsePubkey(sc.AggregatePubkey); err != nil {
		return err
	}
	return nil
}
