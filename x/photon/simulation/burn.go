package simulation

import (
	"math/rand"

	"github.com/atomone-hub/atomone/x/photon/keeper"
	"github.com/atomone-hub/atomone/x/photon/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgBurn(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	sk types.StakingKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgBurn{
			ToAddress: simAccount.Address.String(),
		}

		// TODO: Handling the Burn simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "Burn simulation not implemented"), nil, nil
	}
}