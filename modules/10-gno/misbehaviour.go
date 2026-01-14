package gno

import (
	"time"

	bfttypes "github.com/gnolang/gno/tm2/pkg/bft/types"
	"github.com/gnolang/gno/tm2/pkg/crypto"
	"github.com/gnolang/gno/tm2/pkg/crypto/ed25519"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"
)

var _ exported.ClientMessage = (*Misbehaviour)(nil)

// FrozenHeight is same for all misbehaviour
var FrozenHeight = clienttypes.NewHeight(0, 1)

// NewMisbehaviour creates a new Misbehaviour instance.
func NewMisbehaviour(clientID string, header1, header2 *Header) *Misbehaviour {
	return &Misbehaviour{
		ClientId: clientID,
		Header1:  header1,
		Header2:  header2,
	}
}

// ClientType is Gno light client
func (Misbehaviour) ClientType() string {
	return Gno
}

// GetTime returns the timestamp at which misbehaviour occurred. It uses the
// maximum value from both headers to prevent producing an invalid header outside
// of the misbehaviour age range.
func (misbehaviour Misbehaviour) GetTime() time.Time {
	t1, t2 := misbehaviour.Header1.GetTime(), misbehaviour.Header2.GetTime()
	if t1.After(t2) {
		return t1
	}
	return t2
}

// ValidateBasic implements Misbehaviour interface
func (misbehaviour Misbehaviour) ValidateBasic() error {
	if misbehaviour.Header1 == nil {
		return errorsmod.Wrap(ErrInvalidHeader, "misbehaviour Header1 cannot be nil")
	}
	if misbehaviour.Header2 == nil {
		return errorsmod.Wrap(ErrInvalidHeader, "misbehaviour Header2 cannot be nil")
	}
	if misbehaviour.Header1.TrustedHeight.RevisionHeight == 0 {
		return errorsmod.Wrapf(ErrInvalidHeaderHeight, "misbehaviour Header1 cannot have zero revision height")
	}
	if misbehaviour.Header2.TrustedHeight.RevisionHeight == 0 {
		return errorsmod.Wrapf(ErrInvalidHeaderHeight, "misbehaviour Header2 cannot have zero revision height")
	}
	if misbehaviour.Header1.TrustedValidators == nil {
		return errorsmod.Wrap(ErrInvalidValidatorSet, "trusted validator set in Header1 cannot be empty")
	}
	if misbehaviour.Header2.TrustedValidators == nil {
		return errorsmod.Wrap(ErrInvalidValidatorSet, "trusted validator set in Header2 cannot be empty")
	}
	if misbehaviour.Header1.SignedHeader.Header.ChainId != misbehaviour.Header2.SignedHeader.Header.ChainId {
		return errorsmod.Wrap(clienttypes.ErrInvalidMisbehaviour, "headers must have identical chainIDs")
	}

	if err := host.ClientIdentifierValidator(misbehaviour.ClientId); err != nil {
		return errorsmod.Wrap(err, "misbehaviour client ID is invalid")
	}

	// ValidateBasic on both validators
	if err := misbehaviour.Header1.ValidateBasic(); err != nil {
		return errorsmod.Wrap(
			clienttypes.ErrInvalidMisbehaviour,
			errorsmod.Wrap(err, "header 1 failed validation").Error(),
		)
	}
	if err := misbehaviour.Header2.ValidateBasic(); err != nil {
		return errorsmod.Wrap(
			clienttypes.ErrInvalidMisbehaviour,
			errorsmod.Wrap(err, "header 2 failed validation").Error(),
		)
	}
	// Ensure that Height1 is greater than or equal to Height2
	if misbehaviour.Header1.GetHeight().LT(misbehaviour.Header2.GetHeight()) {
		return errorsmod.Wrapf(clienttypes.ErrInvalidMisbehaviour, "Header1 height is less than Header2 height (%s < %s)", misbehaviour.Header1.GetHeight(), misbehaviour.Header2.GetHeight())
	}

	blockId1 := bfttypes.BlockID{
		Hash:        misbehaviour.Header1.SignedHeader.Header.LastBlockId.Hash,
		PartsHeader: bfttypes.PartSetHeader{Total: int(misbehaviour.Header1.SignedHeader.Header.LastBlockId.PartsHeader.Total), Hash: misbehaviour.Header1.SignedHeader.Header.LastBlockId.PartsHeader.Hash},
	}
	err := blockId1.ValidateBasic()
	if err != nil {
		return errorsmod.Wrap(err, "invalid block ID from header 1 in misbehaviour")
	}
	blockId2 := bfttypes.BlockID{
		Hash:        misbehaviour.Header2.SignedHeader.Header.LastBlockId.Hash,
		PartsHeader: bfttypes.PartSetHeader{Total: int(misbehaviour.Header2.SignedHeader.Header.LastBlockId.PartsHeader.Total), Hash: misbehaviour.Header2.SignedHeader.Header.LastBlockId.PartsHeader.Hash},
	}
	err = blockId2.ValidateBasic()
	if err != nil {
		return errorsmod.Wrap(err, "invalid block ID from header 2 in misbehaviour")
	}

	if err := validCommit(misbehaviour.Header1.SignedHeader.Header.ChainId, blockId1,
		*misbehaviour.Header1.SignedHeader.Commit, misbehaviour.Header1.ValidatorSet); err != nil {
		return err
	}
	return validCommit(misbehaviour.Header2.SignedHeader.Header.ChainId, blockId2,
		*misbehaviour.Header2.SignedHeader.Commit, misbehaviour.Header2.ValidatorSet)
}

// validCommit checks if the given commit is a valid commit from the passed-in validatorset
func validCommit(chainID string, blockID bfttypes.BlockID, commit Commit, valSet *ValidatorSet) (err error) {
	err = blockID.ValidateBasic()
	if err != nil {
		return errorsmod.Wrap(err, "block ID is not gno block ID type")
	}
	gnoCommit := bfttypes.Commit{
		BlockID:    bfttypes.BlockID{Hash: commit.BlockId.Hash, PartsHeader: bfttypes.PartSetHeader{Total: int(commit.BlockId.PartsHeader.Total), Hash: commit.BlockId.PartsHeader.Hash}},
		Precommits: make([]*bfttypes.CommitSig, len(commit.Precommits)),
	}
	for i, sig := range commit.Precommits {
		if sig == nil {
			continue
		}
		gnoCommit.Precommits[i] = &bfttypes.CommitSig{
			ValidatorIndex:   int(sig.ValidatorIndex),
			Signature:        sig.Signature,
			BlockID:          bfttypes.BlockID{Hash: sig.BlockId.Hash, PartsHeader: bfttypes.PartSetHeader{Total: int(sig.BlockId.PartsHeader.Total), Hash: sig.BlockId.PartsHeader.Hash}},
			Type:             bfttypes.SignedMsgType(sig.Type),
			Height:           sig.Height,
			Round:            int(sig.Round),
			Timestamp:        sig.Timestamp,
			ValidatorAddress: crypto.MustAddressFromString(sig.ValidatorAddress),
		}
	}
	err = gnoCommit.ValidateBasic()
	if err != nil {
		return errorsmod.Wrap(err, "commit is not gno commit type")
	}

	gnoValset := bfttypes.ValidatorSet{
		Validators: make([]*bfttypes.Validator, len(valSet.Validators)),
		Proposer:   nil,
	}
	for i, val := range valSet.Validators {
		key := val.PubKey
		if (key.GetEd25519()) == nil {
			return errorsmod.Wrap(clienttypes.ErrInvalidHeader, "validator pubkey is not ed25519")
		}
		gnoValset.Validators[i] = &bfttypes.Validator{
			Address:          crypto.MustAddressFromString(val.Address),
			PubKey:           ed25519.PubKeyEd25519(key.GetEd25519()),
			VotingPower:      val.VotingPower,
			ProposerPriority: val.ProposerPriority,
		}
	}
	gnoValset.TotalVotingPower() // ensure TotalVotingPower is set
	empty := gnoValset.IsNilOrEmpty()
	if empty {
		return errorsmod.Wrap(err, "validator set is not gno validator set type")
	}

	if err := gnoValset.VerifyCommit(chainID, blockID, gnoCommit.Height(), &gnoCommit); err != nil {
		return errorsmod.Wrap(clienttypes.ErrInvalidMisbehaviour, "validator set did not commit to header")
	}

	return nil
}
