package app

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	feemarketpost "github.com/atomone-hub/atomone/x/feemarket/post"
)

// PostHandlerOptions are the options required for constructing a FeeMarket PostHandler.
type HandlerOptions struct {
	FeemarketKeeper       feemarketpost.FeeMarketKeeper
	ConsensusParamsKeeper feemarketpost.ConsensusParamsKeeper
}

// NewPostHandler returns a PostHandler chain with the fee deduct decorator.
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
	if options.FeemarketKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "feemarket keeper is required for post builder")
	}

	postDecorators := []sdk.PostDecorator{
		feemarketpost.NewFeemarketStateUpdateDecorator(
			options.FeemarketKeeper,
			options.ConsensusParamsKeeper,
		),
	}

	return sdk.ChainPostDecorators(postDecorators...), nil
}
