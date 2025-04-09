package ante

import (
	"bytes"
	"math"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/feemarket/types"
)

// BankSendGasConsumption is the gas consumption of the bank send that occur during feemarket handler execution.
const BankSendGasConsumption = 385

// FeeMarketCheckDecorator checks sufficient fees from the fee payer based off of the current
// state of the feemarket.
// If the fee payer does not have the funds to pay for the fees, return an InsufficientFunds error.
// Call next AnteHandler if fees successfully checked.
//
// If x/feemarket is disabled (params.Enabled == false), the handler will fall back to the default
// Cosmos SDK fee deduction antehandler.
//
// CONTRACT: Tx must implement FeeTx interface
type FeeMarketCheckDecorator struct {
	feemarketKeeper   FeeMarketKeeper
	bankKeeper        BankKeeper
	feegrantKeeper    FeeGrantKeeper
	accountKeeper     AccountKeeper
	fallbackDecorator sdk.AnteDecorator
}

func NewFeeMarketCheckDecorator(ak AccountKeeper, bk BankKeeper, fk FeeGrantKeeper, fmk FeeMarketKeeper, fallbackDecorator sdk.AnteDecorator) FeeMarketCheckDecorator {
	return FeeMarketCheckDecorator{
		feemarketKeeper:   fmk,
		bankKeeper:        bk,
		feegrantKeeper:    fk,
		accountKeeper:     ak,
		fallbackDecorator: fallbackDecorator,
	}
}

// AnteHandle calls the feemarket internal antehandler if the keeper is enabled.  If disabled, the fallback
// fee antehandler is fallen back to.
func (dfd FeeMarketCheckDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	params, err := dfd.feemarketKeeper.GetParams(ctx)
	if err != nil {
		return ctx, err
	}
	if !params.Enabled {
		// use fallbackDecorator if provided
		if dfd.fallbackDecorator != nil {
			return dfd.fallbackDecorator.AnteHandle(ctx, tx, simulate, next)
		}
		return next(ctx, tx, simulate)
	}

	// GenTx consume no fee
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if !simulate && ctx.BlockHeight() > 0 && feeTx.GetGas() == 0 {
		return ctx, sdkerrors.ErrInvalidGasLimit.Wrapf("must provide positive gas")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas() // use provided gas limit

	if len(feeCoins) == 0 && !simulate {
		return ctx, errorsmod.Wrapf(types.ErrNoFeeCoins, "got length %d", len(feeCoins))
	}
	if len(feeCoins) > 1 {
		return ctx, errorsmod.Wrapf(types.ErrTooManyFeeCoins, "got length %d", len(feeCoins))
	}

	// if simulating - create a dummy zero value for the user
	payCoin := sdk.NewCoin(params.FeeDenom, sdkmath.ZeroInt())
	if !simulate {
		payCoin = feeCoins[0]
	}

	feeGas := int64(feeTx.GetGas())

	minGasPrice, err := dfd.feemarketKeeper.GetMinGasPrice(ctx, payCoin.GetDenom())
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "unable to get min gas price for denom %s", payCoin.GetDenom())
	}

	ctx.Logger().Info("fee deduct ante handle",
		"min gas prices", minGasPrice,
		"fee", feeCoins,
		"gas limit", gas,
	)

	ctx = ctx.WithMinGasPrices(sdk.NewDecCoins(minGasPrice))

	if !simulate {
		err := CheckTxFee(ctx, minGasPrice, payCoin, feeGas)
		if err != nil {
			return ctx, errorsmod.Wrapf(err, "error checking fee")
		}
	}

	// deduct the entire amount that the account provided as fee (payCoin)
	err = dfd.DeductFees(ctx, tx, payCoin)
	if err != nil {
		return ctx, errorsmod.Wrapf(err, "error deducting fee")
	}
	if simulate {
		ctx.GasMeter().ConsumeGas(BankSendGasConsumption, "simulation send gas consumption")
	}

	// Compute tx priority
	if payCoin.Denom == params.FeeDenom {
		// Same denom no conversion needed
		ctx = ctx.WithPriority(GetTxPriority(payCoin, int64(gas), minGasPrice))
	} else {
		// Different denom, payCoin needs to be converted to params.FeeDenom
		// 1. get gas price in params.FeeDenom
		baseGasPrice, err := dfd.feemarketKeeper.GetMinGasPrice(ctx, params.FeeDenom)
		if err != nil {
			return ctx, err
		}
		// 2. compute conversion factor between the 2 denoms
		factor := baseGasPrice.Amount.Quo(minGasPrice.Amount)
		// 3. convert payCoin
		feeCoin := sdk.NewCoin(
			params.FeeDenom,
			payCoin.Amount.ToLegacyDec().Mul(factor).TruncateInt(),
		)
		// 4. compute tx priority
		ctx = ctx.WithPriority(GetTxPriority(feeCoin, int64(gas), baseGasPrice))
	}

	return next(ctx, tx, simulate)
}

// DeductFees deducts the provided fee from the payer account during tx execution.
func (dfd FeeMarketCheckDecorator) DeductFees(ctx sdk.Context, sdkTx sdk.Tx, providedFee sdk.Coin) error {
	feeTx, ok := sdkTx.(sdk.FeeTx)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()
	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if dfd.feegrantKeeper == nil {
			return sdkerrors.ErrInvalidRequest.Wrap("fee grants are not enabled")
		}
		if !bytes.Equal(feeGranter, feePayer) {
			if !providedFee.IsNil() {
				err := dfd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, sdk.NewCoins(providedFee), sdkTx.GetMsgs())
				if err != nil {
					return errorsmod.Wrapf(err, "%s does not allow to pay fees for %s", feeGranter, feePayer)
				}
			}
		}
		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := dfd.accountKeeper.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return sdkerrors.ErrUnknownAddress.Wrapf("fee payer address: %s does not exist", deductFeesFrom)
	}

	err := dfd.bankKeeper.SendCoinsFromAccountToModule(ctx, deductFeesFromAcc.GetAddress(), authtypes.FeeCollectorName, sdk.NewCoins(providedFee))
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	events := sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, providedFee.String()),
			sdk.NewAttribute(sdk.AttributeKeyFeePayer, deductFeesFrom.String()),
		),
	}
	ctx.EventManager().EmitEvents(events)

	return nil
}

// CheckTxFee implements the logic for the fee market to check if a Tx has provided sufficient
// fees given the current state of the fee market. Returns an error if insufficient fees.
func CheckTxFee(ctx sdk.Context, gasPrice sdk.DecCoin, feeCoin sdk.Coin, feeGas int64) error {
	// Ensure that the provided fees meet the minimum
	if !gasPrice.IsZero() {
		var requiredFee sdk.Coin

		glDec := sdkmath.LegacyNewDec(feeGas)
		limitFee := gasPrice.Amount.Mul(glDec)
		requiredFee = sdk.NewCoin(gasPrice.Denom, limitFee.Ceil().RoundInt())

		if !feeCoin.IsGTE(requiredFee) {
			return sdkerrors.ErrInsufficientFee.Wrapf(
				"got: %s required: %s, minGasPrice: %s",
				feeCoin,
				requiredFee,
				gasPrice,
			)
		}
	}

	return nil
}

const (
	// gasPricePrecision is the amount of digit precision to scale the gas prices to.
	gasPricePrecision = 6
)

// GetTxPriority returns a naive tx priority based on the amount of gas price provided in a transaction.
//
// The fee amount is divided by the gasLimit to calculate "Effective Gas Price".
// This value is then normalized and scaled into an integer, so it can be used as a priority.
//
//	effectiveGasPrice = feeAmount / gas limit (denominated in fee per gas)
//	normalizedGasPrice = effectiveGasPrice / currentGasPrice (floor is 1.  The minimum effective gas price can ever be is current gas price)
//	scaledGasPrice = normalizedGasPrice * 10 ^ gasPricePrecision (amount of decimal places in the normalized gas price to consider when converting to int64).
func GetTxPriority(fee sdk.Coin, gasLimit int64, currentGasPrice sdk.DecCoin) int64 {
	// protections from dividing by 0
	if gasLimit == 0 {
		return 0
	}

	// if the gas price is 0, just use a raw amount
	if currentGasPrice.IsZero() {
		return fee.Amount.Int64()
	}

	effectiveGasPrice := fee.Amount.ToLegacyDec().QuoInt64(gasLimit)
	normalizedGasPrice := effectiveGasPrice.Quo(currentGasPrice.Amount)
	scaledGasPrice := normalizedGasPrice.MulInt64(int64(math.Pow10(gasPricePrecision)))

	// overflow panic protection
	if scaledGasPrice.GTE(sdkmath.LegacyNewDec(math.MaxInt64)) {
		return math.MaxInt64
	} else if scaledGasPrice.LTE(sdkmath.LegacyOneDec()) {
		return 0
	}

	return scaledGasPrice.TruncateInt64()
}
