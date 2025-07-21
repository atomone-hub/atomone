package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/atomone-hub/atomone/x/coredaos/keeper"
	"github.com/atomone-hub/atomone/x/coredaos/types"
)

// CoreDaos message types
var (
	TypeMsgAnnotateProposal   = sdk.MsgTypeURL(&types.MsgAnnotateProposal{})
	TypeMsgEndorseProposal    = sdk.MsgTypeURL(&types.MsgEndorseProposal{})
	TypeMsgExtendVotingPeriod = sdk.MsgTypeURL(&types.MsgExtendVotingPeriod{})
	TypeMsgVetoProposal       = sdk.MsgTypeURL(&types.MsgVetoProposal{})
)

// Simulation operation weights for CoreDaos module
//
//nolint:gosec // these are not hard-coded credentials.
const (
	OpWeightMsgAnnotateProposal        = "op_weight_msg_annotate_proposal"
	DefaultWeightMsgAnnotateProposal   = 100
	OpWeightMsgEndorseProposal         = "op_weight_msg_endorse_proposal"
	DefaultWeightMsgEndorseProposal    = 100
	OpWeightMsgExtendVotingPeriod      = "op_weight_msg_extend_voting_period"
	DefaultWeightMsgExtendVotingPeriod = 100
	OpWeightMsgVetoProposal            = "op_weight_msg_veto_proposal"
	DefaultWeightMsgVetoProposal       = 100
)

// WeightedOperations returns all the operations from the CoreDaos module with their respective weights
func WeightedOperations(appParams simtypes.AppParams, cdc codec.JSONCodec, k keeper.Keeper) simulation.WeightedOperations {
	var weightMsgAnnotateProposal int
	appParams.GetOrGenerate(cdc, OpWeightMsgAnnotateProposal, &weightMsgAnnotateProposal, nil,
		func(_ *rand.Rand) {
			weightMsgAnnotateProposal = DefaultWeightMsgAnnotateProposal
		},
	)

	var weightMsgEndorseProposal int
	appParams.GetOrGenerate(cdc, OpWeightMsgEndorseProposal, &weightMsgEndorseProposal, nil,
		func(_ *rand.Rand) {
			weightMsgEndorseProposal = DefaultWeightMsgEndorseProposal
		},
	)

	var weightMsgExtendVotingPeriod int
	appParams.GetOrGenerate(cdc, OpWeightMsgExtendVotingPeriod, &weightMsgExtendVotingPeriod, nil,
		func(_ *rand.Rand) {
			weightMsgExtendVotingPeriod = DefaultWeightMsgExtendVotingPeriod
		},
	)

	var weightMsgVetoProposal int
	appParams.GetOrGenerate(cdc, OpWeightMsgVetoProposal, &weightMsgVetoProposal, nil,
		func(_ *rand.Rand) {
			weightMsgVetoProposal = DefaultWeightMsgVetoProposal
		},
	)

	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			weightMsgAnnotateProposal,
			SimulateMsgAnnotateProposal(k),
		),
		simulation.NewWeightedOperation(
			weightMsgEndorseProposal,
			SimulateMsgEndorseProposal(k),
		),
		simulation.NewWeightedOperation(
			weightMsgExtendVotingPeriod,
			SimulateMsgExtendVotingPeriod(k),
		),
		simulation.NewWeightedOperation(
			weightMsgVetoProposal,
			SimulateMsgVetoProposal(k),
		),
	}
}

func SimulateMsgAnnotateProposal(k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// TODO

		return simtypes.NoOpMsg(types.ModuleName, TypeMsgAnnotateProposal, "MsgAnnotateProposal simulation not implemented yet"), nil, nil
	}
}

func SimulateMsgEndorseProposal(k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// TODO

		return simtypes.NoOpMsg(types.ModuleName, TypeMsgEndorseProposal, "MsgEndorseProposal simulation not implemented yet"), nil, nil
	}
}

func SimulateMsgExtendVotingPeriod(k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// TODO

		return simtypes.NoOpMsg(types.ModuleName, TypeMsgExtendVotingPeriod, "MsgExtendVotingPeriod simulation not implemented yet"), nil, nil
	}
}

func SimulateMsgVetoProposal(k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		// TODO

		return simtypes.NoOpMsg(types.ModuleName, TypeMsgVetoProposal, "MsgVetoProposal simulation not implemented yet"), nil, nil
	}
}
