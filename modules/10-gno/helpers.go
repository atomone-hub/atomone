package gno

import (
	"math"

	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	errorsmod "cosmossdk.io/errors"
)

// ConvertToGnoValidatorSet converts a protobuf ValidatorSet to a bfttypes.ValidatorSet.
// It returns an error if any validator has a non-ed25519 or malformed public key, an
// invalid address, an address that is not derived from its public key, non-positive or
// out-of-bounds voting power, if any address is duplicated, if the total voting power
// exceeds the allowed bound, or if the resulting validator set is nil or empty.
//
// Unlike Gno's NewValidatorSet constructor (which sorts validators by address), this function
// preserves the input order because GetByIndex-based commit verification relies on the proto
// validator set ordering matching the commit precommit ordering. The validation below therefore
// replicates the safety checks performed by Gno's updateWithChangeSet/processChanges without
// reordering the set. Skipping these checks would allow a relayer-supplied set with negative
// voting power to produce a negative total, making the +2/3 commit threshold negative and
// satisfiable by a single signature.
//
// The address-to-pubkey binding matters because Validator.Bytes() (and therefore
// ValidatorSet.Hash()) excludes the address, and commit sign bytes exclude the validator
// address and index. Without the binding, a relayer could attach distinct addresses to a
// single public key and have one signature counted in several validator slots.
func ConvertToGnoValidatorSet(valSet *ValidatorSet) (*bfttypes.ValidatorSet, error) {
	if valSet == nil {
		return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator set is nil")
	}

	gnoValset := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(valSet.Validators)),
		Proposer:   nil,
	}

	seen := make(map[crypto.Address]struct{}, len(valSet.Validators))
	totalVotingPower := int64(0)
	for i, val := range valSet.Validators {
		if val == nil {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "validator at index %d is nil", i)
		}
		keyBytes := val.PubKey.GetEd25519()
		if keyBytes == nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		// Converting a slice to [32]byte panics when the slice is shorter, so
		// check the length before building the key.
		if len(keyBytes) != ed25519.PubKeyEd25519Size {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "validator pubkey has length %d, expected %d", len(keyBytes), ed25519.PubKeyEd25519Size)
		}
		pubKey := ed25519.PubKeyEd25519(keyBytes)
		address, err := crypto.AddressFromString(val.Address)
		if err != nil {
			return nil, errorsmod.Wrap(clienttypes.ErrInvalidHeader, "invalid validator address")
		}
		// Same binding Gno enforces in its own validator set constructor.
		if address != pubKey.Address() {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "validator address %s does not match pubkey", val.Address)
		}
		if _, ok := seen[address]; ok {
			return nil, errorsmod.Wrapf(ErrInvalidValidatorSet, "duplicate validator address %s", val.Address)
		}
		seen[address] = struct{}{}

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
			PubKey:           pubKey,
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

	partsHeader, err := convertPartSetHeader(commit.BlockId.PartsHeader)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid commit block ID")
	}
	gnoCommit := bfttypes.Commit{
		BlockID: bfttypes.BlockID{
			Hash:        commit.BlockId.Hash,
			PartsHeader: partsHeader,
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
		sigPartsHeader, err := convertPartSetHeader(sig.BlockId.PartsHeader)
		if err != nil {
			return nil, errorsmod.Wrapf(err, "invalid block ID in precommit %d", i)
		}
		// SignedMsgType is a byte. Reject anything the conversion would truncate,
		// so gno's own precommit type check sees the actual wire value.
		if sig.Type > math.MaxUint8 {
			return nil, errorsmod.Wrapf(clienttypes.ErrInvalidHeader, "precommit %d type %d out of range", i, sig.Type)
		}
		gnoCommit.Precommits[i] = &bfttypes.CommitSig{
			ValidatorIndex: int(sig.ValidatorIndex),
			Signature:      sig.Signature,
			BlockID: bfttypes.BlockID{
				Hash:        sig.BlockId.Hash,
				PartsHeader: sigPartsHeader,
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
	lastPartsHeader, err := convertPartSetHeader(header.LastBlockId.PartsHeader)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid header last block ID")
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
			Hash:        header.LastBlockId.Hash,
			PartsHeader: lastPartsHeader,
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

// ConvertToGnoBlockID converts a protobuf BlockID to a bfttypes.BlockID. A nil block
// ID or parts header converts to the zero value. Callers that require a present block
// ID should check IsComplete on the result, since ValidateBasic accepts the zero value.
func ConvertToGnoBlockID(blockID *BlockID) (bfttypes.BlockID, error) {
	if blockID == nil {
		return bfttypes.BlockID{}, nil
	}
	partsHeader, err := convertPartSetHeader(blockID.PartsHeader)
	if err != nil {
		return bfttypes.BlockID{}, err
	}
	return bfttypes.BlockID{
		Hash:        blockID.Hash,
		PartsHeader: partsHeader,
	}, nil
}

// convertPartSetHeader converts a protobuf PartSetHeader to a bfttypes.PartSetHeader,
// enforcing the bounds gno's PartSetHeader.ValidateBasic applies. A nil parts header
// converts to the zero value.
//
// gno's CanonicalizePartSetHeader panics on a Total outside the uint32 range
// while computing vote sign bytes, and relies on PartSetHeader.ValidateBasic having
// run first. None of the light client's ValidateBasic paths reach that check, so a
// relayer-supplied Total has to be bounded here, at conversion, before it can reach
// commit verification.
func convertPartSetHeader(psh *PartSetHeader) (bfttypes.PartSetHeader, error) {
	if psh == nil {
		return bfttypes.PartSetHeader{}, nil
	}
	if psh.Total < 0 || psh.Total > bfttypes.MaxBlockPartsCount {
		return bfttypes.PartSetHeader{}, errorsmod.Wrapf(clienttypes.ErrInvalidHeader, "parts header total %d out of range [0, %d]", psh.Total, bfttypes.MaxBlockPartsCount)
	}
	if err := bfttypes.ValidateHash(psh.Hash); err != nil {
		return bfttypes.PartSetHeader{}, errorsmod.Wrapf(clienttypes.ErrInvalidHeader, "invalid parts header hash: %v", err)
	}
	return bfttypes.PartSetHeader{
		Total: int(psh.Total),
		Hash:  psh.Hash,
	}, nil
}
