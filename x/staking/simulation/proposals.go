package simulation

import (
	"math/rand"
	"time"

	sdk "github.com/atomone-hub/atomone/types"
	simtypes "github.com/atomone-hub/atomone/types/simulation"
	"github.com/atomone-hub/atomone/x/simulation"
	"github.com/atomone-hub/atomone/x/staking/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

// Simulation operation weights constants
const (
	DefaultWeightMsgUpdateParams int = 100

	OpWeightMsgUpdateParams = "op_weight_msg_update_params" //nolint:gosec
)

// ProposalMsgs defines the module weighted proposals' contents
func ProposalMsgs() []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			OpWeightMsgUpdateParams,
			DefaultWeightMsgUpdateParams,
			SimulateMsgUpdateParams,
		),
	}
}

// SimulateMsgUpdateParams returns a random MsgUpdateParams
func SimulateMsgUpdateParams(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	// use the default gov module account address as authority
	var authority sdk.AccAddress = address.Module("gov")

	params := types.DefaultParams()
	params.BondDenom = simtypes.RandStringOfLength(r, 10)
	params.HistoricalEntries = uint32(simtypes.RandIntBetween(r, 0, 1000))
	params.MaxEntries = uint32(simtypes.RandIntBetween(r, 1, 1000))
	params.MaxValidators = uint32(simtypes.RandIntBetween(r, 1, 1000))
	params.UnbondingTime = time.Duration(simtypes.RandTimestamp(r).UnixNano())
	params.MinCommissionRate = simtypes.RandomDecAmount(r, sdk.NewDec(1))

	return &types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}
