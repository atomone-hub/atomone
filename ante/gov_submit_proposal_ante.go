package ante

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	atomoneerrors "github.com/atomone-hub/atomone/types/errors"
	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// CoredaosParamsGetter is the interface required by GovSubmitProposalDecorator to
// retrieve the current coredaos params.
type CoredaosParamsGetter interface {
	GetParams(ctx context.Context) coredaostypes.Params
}

// GovSubmitProposalDecorator rejects any governance proposal whose message list
// contains a coredaos MsgUpdateParams that changes the oversight DAO address
// when that message is bundled together with other proposal messages.
type GovSubmitProposalDecorator struct {
	cdc            codec.BinaryCodec
	coredaosKeeper CoredaosParamsGetter
}

// NewGovSubmitProposalDecorator creates a new GovSubmitProposalDecorator.
func NewGovSubmitProposalDecorator(cdc codec.BinaryCodec, coredaosKeeper CoredaosParamsGetter) GovSubmitProposalDecorator {
	return GovSubmitProposalDecorator{
		cdc:            cdc,
		coredaosKeeper: coredaosKeeper,
	}
}

func (g GovSubmitProposalDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx,
	simulate bool, next sdk.AnteHandler,
) (newCtx sdk.Context, err error) {
	msgs := tx.GetMsgs()
	if err = g.ValidateSubmitProposalMsgs(ctx, msgs); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}

// ValidateSubmitProposalMsgs checks that no submit-proposal message bundles a
// coredaos MsgUpdateParams that changes the oversight DAO address together with
// other proposal messages. It also inspects authz.MsgExec wrappers
func (g GovSubmitProposalDecorator) ValidateSubmitProposalMsgs(ctx sdk.Context, msgs []sdk.Msg) error {
	validateMsg := func(m sdk.Msg) error {
		var proposalMsgs []*codectypes.Any
		switch msg := m.(type) {
		case *govv1.MsgSubmitProposal:
			proposalMsgs = msg.GetMessages()
		case *sdkgovv1.MsgSubmitProposal:
			proposalMsgs = msg.GetMessages()
		default:
			return nil
		}
		return g.validateProposalMessages(ctx, proposalMsgs)
	}

	for _, m := range msgs {
		if execMsg, ok := m.(*authz.MsgExec); ok {
			for _, anyInner := range execMsg.Msgs {
				var inner sdk.Msg
				if err := g.cdc.UnpackAny(anyInner, &inner); err != nil {
					continue
				}
				if err := validateMsg(inner); err != nil {
					return err
				}
			}
			continue
		}
		if err := validateMsg(m); err != nil {
			return err
		}
	}
	return nil
}

// addressChanged returns true if newAddr and currentAddr refer to
// different accounts. Comparison is done on the decoded bytes to
// make it case-insensitive
func addressChanged(newAddr, currentAddr string) (bool, error) {
	if newAddr == "" && currentAddr == "" {
		return false, nil
	}
	if newAddr == "" || currentAddr == "" {
		return true, nil
	}
	newAccAddr, err := sdk.AccAddressFromBech32(newAddr)
	if err != nil {
		return false, err
	}
	currentAccAddr, err := sdk.AccAddressFromBech32(currentAddr)
	if err != nil {
		return false, err
	}
	return !newAccAddr.Equals(currentAccAddr), nil
}

// flattenProposalMsgs recursively unpacks a list of Any-encoded messages,
// expanding authz.MsgExec wrappers so that all leaf messages are returned.
// Entries that cannot be unpacked are represented as nil (they still count
// towards the total message count for bundling detection).
func (g GovSubmitProposalDecorator) flattenProposalMsgs(anyMsgs []*codectypes.Any) []sdk.Msg {
	var result []sdk.Msg
	for _, anyMsg := range anyMsgs {
		var msg sdk.Msg
		if err := g.cdc.UnpackAny(anyMsg, &msg); err != nil {
			// Cannot unpack — count as an opaque message.
			result = append(result, nil)
			continue
		}
		if execMsg, ok := msg.(*authz.MsgExec); ok {
			result = append(result, g.flattenProposalMsgs(execMsg.Msgs)...)
		} else {
			result = append(result, msg)
		}
	}
	return result
}

// validateProposalMessages inspects a list of proposal messages and returns an
// error if a coredaos MsgUpdateParams that changes the oversight DAO address is
// bundled with other messages. authz.MsgExec wrappers are expanded recursively
// so that nested oversight-DAO changes are also detected.
func (g GovSubmitProposalDecorator) validateProposalMessages(ctx sdk.Context, anyMsgs []*codectypes.Any) error {
	effectiveMsgs := g.flattenProposalMsgs(anyMsgs)

	// Bundling is only possible when there is more than one effective message.
	if len(effectiveMsgs) <= 1 {
		return nil
	}

	currentParams := g.coredaosKeeper.GetParams(ctx)

	for _, msg := range effectiveMsgs {
		if msg == nil {
			// Unpackable message — cannot be MsgUpdateParams, skip.
			continue
		}
		updateParams, ok := msg.(*coredaostypes.MsgUpdateParams)
		if !ok {
			continue
		}
		changed, err := addressChanged(updateParams.Params.OversightDaoAddress, currentParams.OversightDaoAddress)
		if err != nil {
			return errorsmod.Wrap(
				atomoneerrors.ErrUnauthorized,
				"failed to compare Oversight DAO addresses: "+err.Error(),
			)
		}
		if changed {
			return errorsmod.Wrap(
				atomoneerrors.ErrUnauthorized,
				"proposal that changes the Oversight DAO address cannot be bundled with other messages",
			)
		}
	}
	return nil
}
