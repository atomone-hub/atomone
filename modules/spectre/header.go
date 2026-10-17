package spectre

import (
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"
)

var (
	_ exported.ClientMessage = (*Header)(nil)
	_ exported.ClientMessage = (*Misbehaviour)(nil)
)

// ClientType returns the spectre client type.
func (Header) ClientType() string {
	return ModuleName
}

// ValidateBasic performs stateless validation of the header.
func (h *Header) ValidateBasic() error {
	if h.TrustedSlot == 0 {
		return errorsmod.Wrap(ErrInvalidHeader, "trusted slot must be non-zero")
	}
	if h.ActiveSyncCommittee == nil {
		return errorsmod.Wrap(ErrInvalidHeader, "active sync committee is required")
	}
	if err := validateFullSyncCommittee(h.ActiveSyncCommittee); err != nil {
		return errorsmod.Wrap(err, "invalid active sync committee")
	}
	return validateLightClientUpdateBasic(&h.ConsensusUpdate)
}

// ClientType returns the spectre client type.
func (Misbehaviour) ClientType() string {
	return ModuleName
}

// ValidateBasic performs stateless validation of the misbehaviour.
func (m *Misbehaviour) ValidateBasic() error {
	if m.TrustedSlot == 0 {
		return errorsmod.Wrap(ErrInvalidMisbehaviour, "trusted slot must be non-zero")
	}
	if m.ActiveSyncCommittee == nil {
		return errorsmod.Wrap(ErrInvalidMisbehaviour, "active sync committee is required")
	}
	if err := validateFullSyncCommittee(m.ActiveSyncCommittee); err != nil {
		return errorsmod.Wrap(err, "invalid active sync committee")
	}
	if err := validateLightClientUpdateBasic(&m.Update_1); err != nil {
		return errorsmod.Wrap(err, "invalid update 1")
	}
	if err := validateLightClientUpdateBasic(&m.Update_2); err != nil {
		return errorsmod.Wrap(err, "invalid update 2")
	}
	return nil
}

// validateFullSyncCommittee validates the structure of a full sync committee.
func validateFullSyncCommittee(sc *SyncCommittee) error {
	if len(sc.AggregatePubkey) != blsPubkeySize {
		return errorsmod.Wrapf(ErrInvalidSyncCommittee, "aggregate pubkey must be %d bytes", blsPubkeySize)
	}
	if _, err := parsePubkey(sc.AggregatePubkey); err != nil {
		return err
	}
	if len(sc.Pubkeys) == 0 {
		return errorsmod.Wrap(ErrInvalidSyncCommittee, "pubkeys must not be empty")
	}
	// Pubkey validation (point deserialization) is expensive; it is done
	// during consensus state verification. Here only the encoding length is
	// checked.
	for i, pk := range sc.Pubkeys {
		if len(pk) != blsPubkeySize {
			return errorsmod.Wrapf(ErrInvalidSyncCommittee, "pubkey %d must be %d bytes", i, blsPubkeySize)
		}
	}
	return nil
}

// validateLightClientUpdateBasic performs structural validation of a light
// client update, deferring cryptographic checks to the state machine.
func validateLightClientUpdateBasic(update *LightClientUpdate) error {
	if update.SignatureSlot == 0 {
		return errorsmod.Wrap(ErrInvalidHeader, "signature slot must be non-zero")
	}
	if len(update.SyncAggregate.SyncCommitteeSignature) != blsSignatureSize {
		return errorsmod.Wrapf(ErrInvalidHeader, "sync signature must be %d bytes", blsSignatureSize)
	}
	if len(update.SyncAggregate.SyncCommitteeBits) == 0 {
		return errorsmod.Wrap(ErrInvalidHeader, "sync committee bits must not be empty")
	}
	if err := validateBranchLengths(update.FinalityBranch, floorLog2(finalizedRootGindex), "finality"); err != nil {
		return err
	}
	if update.NextSyncCommittee != nil && len(update.NextSyncCommitteeBranch) == 0 {
		return errorsmod.Wrap(ErrInvalidHeader, "next sync committee requires a branch")
	}
	if update.NextSyncCommittee == nil && len(update.NextSyncCommitteeBranch) > 0 {
		return errorsmod.Wrap(ErrInvalidHeader, "next sync committee branch requires a committee")
	}
	if len(update.NextSyncCommitteeBranch) > 0 {
		if err := validateBranchLengths(update.NextSyncCommitteeBranch, floorLog2(nextSyncCommitteeGindex), "next sync committee"); err != nil {
			return err
		}
	}
	return nil
}

// validateBranchLengths checks that a branch has at most the given depth and
// every node is 32 bytes.
func validateBranchLengths(branch [][]byte, depth int, name string) error {
	if len(branch) > depth {
		return errorsmod.Wrapf(ErrInvalidHeader, "%s branch has %d nodes, maximum %d", name, len(branch), depth)
	}
	for i, node := range branch {
		if len(node) != 32 {
			return errorsmod.Wrapf(ErrInvalidHeader, "%s branch node %d must be 32 bytes", name, i)
		}
	}
	return nil
}
