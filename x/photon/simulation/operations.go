package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/atomone-hub/atomone/x/photon/keeper"
	"github.com/atomone-hub/atomone/x/photon/types"
)

// Photon message types
var (
	TypeMsgMintPhoton = sdk.MsgTypeURL(&types.MsgMintPhoton{})
)

// Simulation operation weights constants
//
//nolint:gosec // these are not hard-coded credentials.
const (
	OpWeightMsgMintPhoton = "op_weight_msg_mint_photon"

	DefaultWeightMsgMintPhoton = 100
)

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(appParams simtypes.AppParams, cdc codec.JSONCodec,
	ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper, k keeper.Keeper,
) simulation.WeightedOperations {
	var weightMsgMintPhoton int
	appParams.GetOrGenerate(cdc, OpWeightMsgMintPhoton, &weightMsgMintPhoton, nil,
		func(_ *rand.Rand) {
			weightMsgMintPhoton = DefaultWeightMsgMintPhoton
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgMintPhoton,
			SimulateMsgMintPhoton(ak, bk, sk, k),
		),
	}
}

func SimulateMsgMintPhoton(
	ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper, k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// Check if mint is disabled
		if k.GetParams(ctx).MintDisabled {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgMintPhoton, "mint is disabled"), nil, nil
		}
		toAddress, _ := simtypes.RandomAcc(r, accs)
		acc := ak.GetAccount(ctx, toAddress.Address)
		spendable := bk.SpendableCoins(ctx, acc.GetAddress())
		if len(spendable) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgMintPhoton, "no spendable coins"), nil, nil
		}
		bondDenom := sk.BondDenom(ctx)
		ok, amount := spendable.Find(bondDenom)
		if !ok {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgMintPhoton, "no bond denom in spendable coins"), nil, nil
		}

		msg := types.NewMsgMintPhoton(toAddress.Address, amount)
		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           moduletestutil.MakeTestEncodingConfig().TxConfig,
			Cdc:             nil,
			Msg:             msg,
			MsgType:         TypeMsgMintPhoton,
			Context:         ctx,
			SimAccount:      toAddress,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: spendable,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
