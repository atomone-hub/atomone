package gno

import (
	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	errorsmod "cosmossdk.io/errors"
)

// ConvertToGnoValidatorSet converts a protobuf ValidatorSet to a bfttypes.ValidatorSet.
// It returns an error if any validator has a non-ed25519 public key or if the resulting
// validator set is nil or empty.
func ConvertToGnoValidatorSet(valSet *ValidatorSet) (*bfttypes.ValidatorSet, error) {
	if valSet == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator set is nil")
	}

	gnoValset := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(valSet.Validators)),
		Proposer:   nil,
	}

	for i, val := range valSet.Validators {
		key := val.PubKey
		if key.GetEd25519() == nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		gnoValset.Validators[i] = &bfttypes.Validator{
			Address:          crypto.MustAddressFromString(val.Address),
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
		if sig == nil {
			continue
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
			ValidatorAddress: crypto.MustAddressFromString(sig.ValidatorAddress),
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

	gnoHeader := bfttypes.Header{
		Version:  header.Version,
		ChainID:  header.ChainId,
		Height:   header.Height,
		Time:     header.Time,
		NumTxs:   header.NumTxs,
		TotalTxs: header.TotalTxs,
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
		ProposerAddress:    crypto.MustAddressFromString(header.ProposerAddress),
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
	return bfttypes.BlockID{
		Hash: blockID.Hash,
		PartsHeader: bfttypes.PartSetHeader{
			Total: int(blockID.PartsHeader.Total),
			Hash:  blockID.PartsHeader.Hash,
		},
	}
}
