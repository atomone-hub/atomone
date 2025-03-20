package simulation

import (
	"math/rand"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

const (
	// OpWeightSubmitTextProposal app params key for text proposal
	OpWeightSubmitTextProposal = "op_weight_submit_text_proposal"
	// OpWeightSubmitConstitutionAmendmentProposal app params key for constitution amendment proposal
	OpWeightSubmitConstitutionAmendmentProposal = "op_weight_submit_constitution_amendment_proposal"
	// OpWeightSubmitLawProposal app params key for law proposal
	OpWeightSubmitLawProposal = "op_weight_submit_law_proposal"
)

// ProposalMsgs defines the module weighted proposals' contents
func ProposalMsgs() []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{
		simulation.NewWeightedProposalMsg(
			OpWeightSubmitTextProposal,
			DefaultWeightTextProposal,
			SimulateTextProposal,
		),
		simulation.NewWeightedProposalMsg(
			OpWeightSubmitConstitutionAmendmentProposal,
			DefaultWeightConstitutionAmendment,
			SimulateConstitutionAmendmentProposal,
		),
		simulation.NewWeightedProposalMsg(
			OpWeightSubmitLawProposal,
			DefaultWeightLawProposal,
			SimulateLawProposal,
		),
	}
}

// SimulateTextProposal returns a random text proposal content.
// A text proposal is a proposal that contains no msgs.
func SimulateTextProposal(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	return nil
}

// ProposalContents defines the module weighted proposals' contents
//

func ProposalContents() []simtypes.WeightedProposalContent {
	return []simtypes.WeightedProposalContent{
		simulation.NewWeightedProposalContent(
			OpWeightMsgDeposit,
			DefaultWeightTextProposal,
			SimulateLegacyTextProposalContent,
		),
	}
}

// SimulateTextProposalContent returns a random text proposal content.
//

func SimulateLegacyTextProposalContent(r *rand.Rand, _ sdk.Context, _ []simtypes.Account) simtypes.Content {
	return v1beta1.NewTextProposal(
		simtypes.RandStringOfLength(r, 140),
		simtypes.RandStringOfLength(r, 5000),
	)
}

// SimulateConstitutionAmendmentProposal returns a random constitution amendment proposal.
func SimulateConstitutionAmendmentProposal(_ *rand.Rand, ctx sdk.Context, _ []simtypes.Account) sdk.Msg {
	emptyAmendment := "@@ -1 +1 @@\n-\n+\n" // valid diff with no changes
	return v1.NewMsgProposeConstitutionAmendment(authtypes.NewModuleAddress(govtypes.ModuleName), emptyAmendment)
}

// SimulateLawProposal returns a random law proposal.
func SimulateLawProposal(_ *rand.Rand, _ sdk.Context, _ []simtypes.Account) sdk.Msg {
	return &v1.MsgProposeLaw{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	}
}
