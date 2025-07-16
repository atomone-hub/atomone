package app

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	dynamicfeepost "github.com/atomone-hub/atomone/x/dynamicfee/post"
)

// PostHandlerOptions are the options required for constructing a Dynamicfee PostHandler.
type HandlerOptions struct {
	DynamicfeeKeeper      dynamicfeepost.DynamicfeeKeeper
	ConsensusParamsKeeper dynamicfeepost.ConsensusParamsKeeper
}

// NewPostHandler returns a PostHandler chain with the fee deduct decorator.
func NewPostHandler(options HandlerOptions) (sdk.PostHandler, error) {
	if options.DynamicfeeKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "dynamicfee keeper is required for post builder")
	}

	postDecorators := []sdk.PostDecorator{
		dynamicfeepost.NewDynamicfeeStateUpdateDecorator(
			options.DynamicfeeKeeper,
			options.ConsensusParamsKeeper,
		),
	}

	return sdk.ChainPostDecorators(postDecorators...), nil
}
