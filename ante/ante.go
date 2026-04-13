package ante

import (
	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	atomoneerrors "github.com/atomone-hub/atomone/types/errors"
	dynamicfeeante "github.com/atomone-hub/atomone/x/dynamicfee/ante"
	dynamicfeekeeper "github.com/atomone-hub/atomone/x/dynamicfee/keeper"
	photonante "github.com/atomone-hub/atomone/x/photon/ante"
	photonkeeper "github.com/atomone-hub/atomone/x/photon/keeper"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	Codec            codec.BinaryCodec
	IBCkeeper        *ibckeeper.Keeper
	StakingKeeper    *stakingkeeper.Keeper
	PhotonKeeper     *photonkeeper.Keeper
	TxFeeChecker     ante.TxFeeChecker
	DynamicfeeKeeper *dynamicfeekeeper.Keeper
}

func NewAnteHandler(opts HandlerOptions) (sdk.AnteHandler, error) {
	if opts.AccountKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if opts.BankKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if opts.SignModeHandler == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if opts.IBCkeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrLogic, "IBC keeper is required for AnteHandler")
	}
	if opts.StakingKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrNotFound, "staking param store is required for AnteHandler")
	}
	if opts.PhotonKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrNotFound, "photon keeper is required for AnteHandler")
	}
	if opts.DynamicfeeKeeper == nil {
		return nil, errorsmod.Wrap(atomoneerrors.ErrNotFound, "dynamicfee keeper is required for AnteHandler")
	}

	sigGasConsumer := opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}
	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(opts.AccountKeeper),
		ante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		ante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		ante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		photonante.NewValidateFeeDecorator(opts.PhotonKeeper),
		dynamicfeeante.NewDynamicfeeCheckDecorator(
			opts.AccountKeeper,
			opts.BankKeeper,
			opts.FeegrantKeeper,
			opts.DynamicfeeKeeper,
			ante.NewDeductFeeDecorator( // legacy fee deduct decorator used as fallback if dynamicfee is disabled
				opts.AccountKeeper,
				opts.BankKeeper,
				opts.FeegrantKeeper,
				opts.TxFeeChecker,
			),
		),
		NewGovVoteDecorator(opts.Codec, opts.StakingKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCkeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
