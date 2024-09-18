package simulation_test

import (
	"math/rand"
	"testing"

	"gotest.tools/v3/assert"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/atomone-hub/atomone/x/gov/simulation"
)

func TestProposalMsgs(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)

	ctx := sdk.NewContext(nil, tmproto.Header{}, true, nil)
	accounts := simtypes.RandomAccounts(r, 3)

	// execute ProposalMsgs function
	weightedProposalMsgs := simulation.ProposalMsgs()
	assert.Assert(t, len(weightedProposalMsgs) == 1)

	w0 := weightedProposalMsgs[0]

	// tests w0 interface:
	assert.Equal(t, simulation.OpWeightSubmitTextProposal, w0.AppParamsKey())
	assert.Equal(t, simulation.DefaultWeightTextProposal, w0.DefaultWeight())

	msg := w0.MsgSimulatorFn()(r, ctx, accounts)
	assert.Assert(t, msg == nil)
}

func TestProposalContents(t *testing.T) {
	// initialize parameters
	s := rand.NewSource(1)
	r := rand.New(s)

	ctx := sdk.NewContext(nil, tmproto.Header{}, true, nil)
	accounts := simtypes.RandomAccounts(r, 3)

	// execute ProposalContents function
	weightedProposalContent := simulation.ProposalContents()
	assert.Assert(t, len(weightedProposalContent) == 3)

	for _, w := range weightedProposalContent {
		// tests w interface:
		assert.Equal(t, simulation.OpWeightMsgDeposit, w.AppParamsKey())
		assert.Equal(t, simulation.DefaultWeightTextProposal, w.DefaultWeight())

		content := w.ContentSimulatorFn()(r, ctx, accounts)

		assert.Assert(t, content != nil)
		assert.Equal(t, "gov", content.ProposalRoute())
		assert.Assert(t, content.ProposalType() == "Text" || content.ProposalType() == "Law" || content.ProposalType() == "ConstitutionAmendment")
	}
}
