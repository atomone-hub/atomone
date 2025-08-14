package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/atomone-hub/atomone/x/coredaos/keeper"
	"github.com/atomone-hub/atomone/x/coredaos/types"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

var initialProposalID = uint64(100000000000000)

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
func WeightedOperations(appParams simtypes.AppParams, cdc codec.JSONCodec, gk types.GovKeeper, sk types.StakingKeeper, ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simulation.WeightedOperations {
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
			SimulateMsgAnnotateProposal(gk, sk, ak, bk, k),
		),
		simulation.NewWeightedOperation(
			weightMsgEndorseProposal,
			SimulateMsgEndorseProposal(gk, sk, ak, bk, k),
		),
		simulation.NewWeightedOperation(
			weightMsgExtendVotingPeriod,
			SimulateMsgExtendVotingPeriod(gk, sk, ak, bk, k),
		),
		simulation.NewWeightedOperation(
			weightMsgVetoProposal,
			SimulateMsgVetoProposal(gk, sk, ak, bk, k),
		),
	}
}

func SimulateMsgAnnotateProposal(gk types.GovKeeper, sk types.StakingKeeper, ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		params := k.GetParams(ctx)
		if params.SteeringDaoAddress == "" {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgAnnotateProposal, "Annotations are disabled"), nil, nil
		}

		proposalID, ok := randomProposalID(r, gk, ctx)
		if !ok {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgAnnotateProposal, "unable to generate proposalID"), nil, nil
		}

		proposal, err := gk.GetProposal(ctx, proposalID)
		if err {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgAnnotateProposal, "cannot get proposal"), nil, nil
		}
		if proposal.Annotation != "" {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgAnnotateProposal, "proposal already annotated"), nil, nil
		}

		msg := types.NewMsgAnnotateProposal(
			SteeringDaoAccount.Address,
			proposalID,
			simtypes.RandStringOfLength(r, 100),
		)
		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         moduletestutil.MakeTestEncodingConfig().TxConfig,
			Cdc:           nil,
			Msg:           msg,
			MsgType:       TypeMsgAnnotateProposal,
			Context:       ctx,
			SimAccount:    SteeringDaoAccount,
			AccountKeeper: ak,
			Bankkeeper:    bk,
			ModuleName:    types.ModuleName,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

func SimulateMsgEndorseProposal(gk types.GovKeeper, sk types.StakingKeeper, ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		params := k.GetParams(ctx)
		if params.SteeringDaoAddress == "" {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgEndorseProposal, "Endorsements are disabled"), nil, nil
		}

		proposalID, ok := randomProposalID(r, gk, ctx)
		if !ok {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgEndorseProposal, "unable to generate proposalID"), nil, nil
		}

		proposal, err := gk.GetProposal(ctx, proposalID)
		if err {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgEndorseProposal, "cannot get proposal"), nil, nil
		}

		if proposal.Endorsed {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgEndorseProposal, "proposal already endorsed"), nil, nil
		}

		msg := types.NewMsgEndorseProposal(
			SteeringDaoAccount.Address,
			proposalID,
		)
		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         moduletestutil.MakeTestEncodingConfig().TxConfig,
			Cdc:           nil,
			Msg:           msg,
			MsgType:       TypeMsgEndorseProposal,
			Context:       ctx,
			SimAccount:    SteeringDaoAccount,
			AccountKeeper: ak,
			Bankkeeper:    bk,
			ModuleName:    types.ModuleName,
		}
		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

func SimulateMsgExtendVotingPeriod(gk types.GovKeeper, sk types.StakingKeeper, ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		params := k.GetParams(ctx)
		if params.SteeringDaoAddress == "" && params.OversightDaoAddress == "" {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgExtendVotingPeriod, "Voting period extensions are disabled"), nil, nil
		}

		var fromAccount simtypes.Account
		if params.SteeringDaoAddress == "" && params.OversightDaoAddress == "" {
			randInt := r.Intn(2)
			if randInt%2 == 0 {
				fromAccount = SteeringDaoAccount
			} else {
				fromAccount = OversightDaoAccount
			}
		} else {
			if params.SteeringDaoAddress != "" {
				fromAccount = SteeringDaoAccount
			} else {
				fromAccount = OversightDaoAccount
			}
		}

		proposalID, ok := randomProposalID(r, gk, ctx)
		if !ok {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgExtendVotingPeriod, "unable to generate proposalID"), nil, nil
		}

		proposal, err := gk.GetProposal(ctx, proposalID)
		if err {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgExtendVotingPeriod, "cannot get proposal"), nil, nil
		}

		if proposal.TimesVotingPeriodExtended >= params.VotingPeriodExtensionsLimit {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgExtendVotingPeriod, "proposal voting period extended too many times"), nil, nil
		}

		msg := types.NewMsgExtendVotingPeriod(
			fromAccount.Address,
			proposalID,
		)
		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         moduletestutil.MakeTestEncodingConfig().TxConfig,
			Cdc:           nil,
			Msg:           msg,
			MsgType:       TypeMsgExtendVotingPeriod,
			Context:       ctx,
			SimAccount:    fromAccount,
			AccountKeeper: ak,
			Bankkeeper:    bk,
			ModuleName:    types.ModuleName,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

func SimulateMsgVetoProposal(gk types.GovKeeper, sk types.StakingKeeper, ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		params := k.GetParams(ctx)
		if params.OversightDaoAddress == "" {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgVetoProposal, "Voting period extensions are disabled"), nil, nil
		}

		proposalID, ok := randomProposalID(r, gk, ctx)
		if !ok {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgVetoProposal, "unable to generate proposalID"), nil, nil
		}

		_, err := gk.GetProposal(ctx, proposalID)
		if err {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgVetoProposal, "cannot get proposal"), nil, nil
		}

		var burnDeposit bool
		randInt := r.Intn(2)
		if randInt%2 == 0 {
			burnDeposit = true
		} else {
			burnDeposit = false
		}
		msg := types.NewMsgVetoProposal(
			OversightDaoAccount.Address,
			proposalID,
			burnDeposit,
		)

		txCtx := simulation.OperationInput{
			R:             r,
			App:           app,
			TxGen:         moduletestutil.MakeTestEncodingConfig().TxConfig,
			Cdc:           nil,
			Msg:           msg,
			MsgType:       TypeMsgVetoProposal,
			Context:       ctx,
			SimAccount:    OversightDaoAccount,
			AccountKeeper: ak,
			Bankkeeper:    bk,
			ModuleName:    types.ModuleName,
		}

		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}

// Pick a random proposal ID between the initial proposal ID
// (defined in gov GenesisState) and the latest proposal ID
// that has voting period status
// It does not provide a default ID.
func randomProposalID(r *rand.Rand, k types.GovKeeper, ctx sdk.Context) (proposalID uint64, found bool) {
	proposalID, _ = k.GetProposalID(ctx)

	switch {
	case proposalID > initialProposalID:
		// select a random ID between [initialProposalID, proposalID]
		proposalID = uint64(simtypes.RandIntBetween(r, int(initialProposalID), int(proposalID)))

	default:
		// This is called on the first call to this funcion
		// in order to update the global variable
		initialProposalID = proposalID
	}

	proposal, ok := k.GetProposal(ctx, proposalID)
	if !ok || proposal.Status != govv1.StatusVotingPeriod {
		return proposalID, false
	}

	return proposalID, true
}
