package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/address"

	sdk "github.com/atomone-hub/atomone/types"
	simtypes "github.com/atomone-hub/atomone/types/simulation"
	"github.com/atomone-hub/atomone/x/auth/types"
	"github.com/atomone-hub/atomone/x/simulation"
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
	params.MaxMemoCharacters = uint64(simtypes.RandIntBetween(r, 1, 1000))
	params.TxSigLimit = uint64(simtypes.RandIntBetween(r, 1, 1000))
	params.TxSizeCostPerByte = uint64(simtypes.RandIntBetween(r, 1, 1000))
	params.SigVerifyCostED25519 = uint64(simtypes.RandIntBetween(r, 1, 1000))
	params.SigVerifyCostSecp256k1 = uint64(simtypes.RandIntBetween(r, 1, 1000))

	return &types.MsgUpdateParams{
		Authority: authority.String(),
		Params:    params,
	}
}
