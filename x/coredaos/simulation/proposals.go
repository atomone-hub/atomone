package simulation

import (
	"math/rand"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/atomone-hub/atomone/x/coredaos/types"
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

	params.VotingPeriodExtensionsLimit = uint32(simtypes.RandIntBetween(r, 0, 10))                       // Random limit between 0 and 9
	votingPeriodExtensionDuration := time.Duration(simtypes.RandIntBetween(r, 1, 60*60*6)) * time.Second // Random duration between 1 second and 6 hours
	params.VotingPeriodExtensionDuration = &votingPeriodExtensionDuration

	return &types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}
