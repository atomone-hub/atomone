package gno

import (
	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	errorsmod "cosmossdk.io/errors"
)

// ConvertToGnoValidatorSet converts a protobuf ValidatorSet to a bfttypes.ValidatorSet.
// It returns an error if any validator has a non-ed25519 public key, an invalid address,
// non-positive or out-of-bounds voting power, if any address is duplicated, if the total
// voting power exceeds the allowed bound, or if the resulting validator set is nil or empty.
//
// Unlike Gno's NewValidatorSet constructor (which sorts validators by address), this function
// preserves the input order because GetByIndex-based commit verification relies on the proto
// validator set ordering matching the commit precommit ordering. The validation below therefore
// replicates the safety checks performed by Gno's updateWithChangeSet/processChanges without
// reordering the set. Skipping these checks would allow a relayer-supplied set with negative
// voting power to produce a negative total, making the +2/3 commit threshold negative and
// satisfiable by a single signature.
func ConvertToGnoValidatorSet(valSet *ValidatorSet) (*bfttypes.ValidatorSet, error) {
	if valSet == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator set is nil")
	}

	gnoValset := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(valSet.Validators)),
		Proposer:   nil,
	}

	seen := make(map[string]struct{}, len(valSet.Validators))
	totalVotingPower := int64(0)
	for i, val := range valSet.Validators {
		key := val.PubKey
		if key.GetEd25519() == nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		address, err := crypto.AddressFromString(val.Address)
		if err != nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "invalid validator address")
		}
		if _, ok := seen[val.Address]; ok {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "duplicate validator address %s", val.Address)
		}
		seen[val.Address] = struct{}{}

		// Reject non-positive voting power: a real Gno validator set never contains
		// negative- or zero-power members (zero-power entries are removed during updates).
		// A negative power would corrupt the total and the +2/3 commit threshold.
		if val.VotingPower <= 0 {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "validator %s has non-positive voting power %d", val.Address, val.VotingPower)
		}
		if val.VotingPower > bfttypes.MaxTotalVotingPower {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "validator %s voting power %d exceeds max %d", val.Address, val.VotingPower, bfttypes.MaxTotalVotingPower)
		}
		totalVotingPower += val.VotingPower
		if totalVotingPower > bfttypes.MaxTotalVotingPower {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "total voting power exceeds max %d", bfttypes.MaxTotalVotingPower)
		}

		gnoValset.Validators[i] = &bfttypes.Validator{
			Address:          address,
			PubKey:           ed25519.PubKeyEd25519(key.GetEd25519()),
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}

	gnoValset.TotalVotingPower() // ensure TotalVotingPower is computed and cached

	if gnoValset.IsNilOrEmpty() {
		return nil, errorsmod.Wrap(ErrInvalidValidatorSet, "validator set is nil or empty")
	}

	return &gnoValset, nil
}

// ConvertToGnoCommit converts a protobuf Commit to a bfttypes.Commit.
func ConvertToGnoCommit(commit *Commit) (*bfttypes.Commit, error) {
	if commit == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "commit is nil")
	}

	if commit.BlockId == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "commit block ID is nil")
	}
	if commit.BlockId.PartsHeader == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "commit block ID parts header is nil")
	}

	gnoCommit := bfttypes.Commit{
		BlockID: bfttypes.BlockID{
			Hash: commit.BlockId.Hash,
			PartsHeader: bfttypes.PartSetHeader{
				Total: int(commit.BlockId.PartsHeader.Total),
				Hash:  commit.BlockId.PartsHeader.Hash,
			},
		},
		Precommits: make([]*bfttypes.CommitSig, len(commit.Precommits)),
	}

	for i, sig := range commit.Precommits {
		// Proto3 repeated message fields always deserialize as non-nil pointers,
		// so absent validators appear as zero-value CommitSig structs rather than
		// nil entries. Detect absent validators by checking for an empty signature.
		if sig == nil || len(sig.Signature) == 0 {
			continue
		}
		if sig.BlockId == nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "precommit block ID is nil")
		}
		if sig.BlockId.PartsHeader == nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "precommit block ID parts header is nil")
		}
		address, err := crypto.AddressFromString(sig.ValidatorAddress)
		if err != nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "invalid validator address")
		}
		gnoCommit.Precommits[i] = &bfttypes.CommitSig{
			ValidatorIndex: int(sig.ValidatorIndex),
			Signature:      sig.Signature,
			BlockID: bfttypes.BlockID{
				Hash: sig.BlockId.Hash,
				PartsHeader: bfttypes.PartSetHeader{
					Total: int(sig.BlockId.PartsHeader.Total),
					Hash:  sig.BlockId.PartsHeader.Hash,
				},
			},
			Type:             bfttypes.SignedMsgType(sig.Type),
			Height:           sig.Height,
			Round:            int(sig.Round),
			Timestamp:        sig.Timestamp,
			ValidatorAddress: address,
		}
	}

	return &gnoCommit, nil
}

// ConvertToGnoHeader converts a protobuf GnoHeader to a bfttypes.Header.
func ConvertToGnoHeader(header *GnoHeader) (*bfttypes.Header, error) {
	if header == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "header is nil")
	}

	var dataHash []byte
	var lastResultsHash []byte
	if len(header.DataHash) > 0 {
		dataHash = header.DataHash
	}
	if len(header.LastResultsHash) > 0 {
		lastResultsHash = header.LastResultsHash
	}

	if header.LastBlockId == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "header last block ID is nil")
	}
	if header.LastBlockId.PartsHeader == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "header last block ID parts header is nil")
	}

	address, err := crypto.AddressFromString(header.ProposerAddress)
	if err != nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "invalid validator address")
	}
	gnoHeader := bfttypes.Header{
		Version:    header.Version,
		ChainID:    header.ChainId,
		Height:     header.Height,
		Time:       header.Time,
		NumTxs:     header.NumTxs,
		TotalTxs:   header.TotalTxs,
		AppVersion: header.AppVersion,
		LastBlockID: bfttypes.BlockID{
			Hash: header.LastBlockId.Hash,
			PartsHeader: bfttypes.PartSetHeader{
				Total: int(header.LastBlockId.PartsHeader.Total),
				Hash:  header.LastBlockId.PartsHeader.Hash,
			},
		},
		LastCommitHash:     header.LastCommitHash,
		DataHash:           dataHash,
		ValidatorsHash:     header.ValidatorsHash,
		NextValidatorsHash: header.NextValidatorsHash,
		ConsensusHash:      header.ConsensusHash,
		AppHash:            header.AppHash,
		LastResultsHash:    lastResultsHash,
		ProposerAddress:    address,
	}

	return &gnoHeader, nil
}

// ConvertToGnoSignedHeader converts a protobuf SignedHeader to a bfttypes.SignedHeader.
func ConvertToGnoSignedHeader(signedHeader *SignedHeader) (*bfttypes.SignedHeader, error) {
	if signedHeader == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "signed header is nil")
	}

	gnoHeader, err := ConvertToGnoHeader(signedHeader.Header)
	if err != nil {
		return nil, err
	}

	gnoCommit, err := ConvertToGnoCommit(signedHeader.Commit)
	if err != nil {
		return nil, err
	}

	return &bfttypes.SignedHeader{
		Header: gnoHeader,
		Commit: gnoCommit,
	}, nil
}

// ConvertToGnoBlockID converts a protobuf BlockID to a bfttypes.BlockID.
func ConvertToGnoBlockID(blockID *BlockID) bfttypes.BlockID {
	if blockID == nil {
		return bfttypes.BlockID{}
	}
	if blockID.PartsHeader == nil {
		return bfttypes.BlockID{
			Hash: blockID.Hash,
		}
	}
	return bfttypes.BlockID{
		Hash: blockID.Hash,
		PartsHeader: bfttypes.PartSetHeader{
			Total: int(blockID.PartsHeader.Total),
			Hash:  blockID.PartsHeader.Hash,
		},
	}
}
