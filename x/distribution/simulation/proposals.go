package simulation

import (
	"math/rand"

	sdk "github.com/atomone-hub/atomone/types"
	simtypes "github.com/atomone-hub/atomone/types/simulation"
	"github.com/atomone-hub/atomone/x/distribution/types"
	"github.com/atomone-hub/atomone/x/simulation"
	"github.com/cosmos/cosmos-sdk/types/address"
)

// Simulation operation weights constants
const (
	DefaultWeightMsgUpdateParams int = 50

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
	params.CommunityTax = simtypes.RandomDecAmount(r, sdk.NewDec(1))
	params.WithdrawAddrEnabled = r.Intn(2) == 0

	return &types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}
