package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func TestGetSetParticipationEma(t *testing.T) {
	k, _, _, ctx := setupGovKeeper(t)
	assert := assert.New(t)

	assert.Equal(v1.DefaultParticipationEma, k.GetParticipationEMA(ctx).String())
	assert.Equal(v1.DefaultParticipationEma, k.GetConstitutionAmendmentParticipationEMA(ctx).String())
	assert.Equal(v1.DefaultParticipationEma, k.GetLawParticipationEMA(ctx).String())

	k.SetParticipationEMA(ctx, math.LegacyNewDecWithPrec(1, 2))
	k.SetConstitutionAmendmentParticipationEMA(ctx, math.LegacyNewDecWithPrec(2, 2))
	k.SetLawParticipationEMA(ctx, math.LegacyNewDecWithPrec(3, 2))

	assert.Equal(math.LegacyNewDecWithPrec(1, 2).String(), k.GetParticipationEMA(ctx).String())
	assert.Equal(math.LegacyNewDecWithPrec(2, 2).String(), k.GetConstitutionAmendmentParticipationEMA(ctx).String())
	assert.Equal(math.LegacyNewDecWithPrec(3, 2).String(), k.GetLawParticipationEMA(ctx).String())

	assert.Equal(math.LegacyNewDecWithPrec(104, 3).String(), k.GetQuorum(ctx).String())
	assert.Equal(math.LegacyNewDecWithPrec(108, 3).String(), k.GetConstitutionAmendmentQuorum(ctx).String())
	assert.Equal(math.LegacyNewDecWithPrec(112, 3).String(), k.GetLawQuorum(ctx).String())
}

func TestUpdateParticipationEma(t *testing.T) {
	tests := []struct {
		name                                        string
		proposal                                    v1.Proposal
		expectedParticipationEma                    string
		expectedConstitutionAmdmentParticipationEma string
		expectedLawParticipationEma                 string
	}{
		{
			name:                     "proposal w/o message",
			proposal:                 v1.Proposal{},
			expectedParticipationEma: math.LegacyNewDecWithPrec(41, 2).String(),
			expectedConstitutionAmdmentParticipationEma: v1.DefaultParticipationEma,
			expectedLawParticipationEma:                 v1.DefaultParticipationEma,
		},
		{
			name:                     "proposal with propose law message",
			proposal:                 v1.Proposal{Messages: setMsgs(t, []sdk.Msg{&v1.MsgProposeLaw{}})},
			expectedParticipationEma: v1.DefaultParticipationEma,
			expectedConstitutionAmdmentParticipationEma: v1.DefaultParticipationEma,
			expectedLawParticipationEma:                 math.LegacyNewDecWithPrec(41, 2).String(),
		},
		{
			name:                     "proposal with propose constitution amendment message",
			proposal:                 v1.Proposal{Messages: setMsgs(t, []sdk.Msg{&v1.MsgProposeConstitutionAmendment{}})},
			expectedParticipationEma: v1.DefaultParticipationEma,
			expectedConstitutionAmdmentParticipationEma: math.LegacyNewDecWithPrec(41, 2).String(),
			expectedLawParticipationEma:                 v1.DefaultParticipationEma,
		},
		{
			name: "proposal with all kinds of messages",
			proposal: v1.Proposal{Messages: setMsgs(t, []sdk.Msg{
				&v1.MsgProposeConstitutionAmendment{},
				&v1.MsgProposeLaw{},
				&banktypes.MsgSend{},
			})},
			expectedParticipationEma:                    math.LegacyNewDecWithPrec(41, 2).String(),
			expectedConstitutionAmdmentParticipationEma: math.LegacyNewDecWithPrec(41, 2).String(),
			expectedLawParticipationEma:                 math.LegacyNewDecWithPrec(41, 2).String(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			k, _, _, ctx := setupGovKeeper(t)
			assert.Equal(v1.DefaultParticipationEma, k.GetParticipationEMA(ctx).String())
			assert.Equal(v1.DefaultParticipationEma, k.GetConstitutionAmendmentParticipationEMA(ctx).String())
			assert.Equal(v1.DefaultParticipationEma, k.GetLawParticipationEMA(ctx).String())
			newParticipation := math.LegacyNewDecWithPrec(5, 2) // 5% participation

			k.UpdateParticipationEMA(ctx, tt.proposal, newParticipation)

			assert.Equal(tt.expectedParticipationEma, k.GetParticipationEMA(ctx).String())
			assert.Equal(tt.expectedConstitutionAmdmentParticipationEma, k.GetConstitutionAmendmentParticipationEMA(ctx).String())
			assert.Equal(tt.expectedLawParticipationEma, k.GetLawParticipationEMA(ctx).String())
		})
	}
}
