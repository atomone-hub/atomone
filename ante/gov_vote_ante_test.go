package ante_test

import (
	"testing"

	"github.com/atomone-hub/atomone/ante"
	"github.com/atomone-hub/atomone/app/helpers"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"

	"cosmossdk.io/math"
	"github.com/atomone-hub/atomone/crypto/keys/ed25519"
	sdk "github.com/atomone-hub/atomone/types"
	govv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	stakingtypes "github.com/atomone-hub/atomone/x/staking/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"
)

// Test that the GovVoteDecorator rejects v1beta1 vote messages from accounts with less than 1 atom staked
// Submitting v1beta1.VoteMsg should not be possible through the CLI, but it's still possible to craft a transaction
func TestVoteSpamDecoratorGovV1Beta1(t *testing.T) {
	atomoneApp := helpers.Setup(t)
	ctx := atomoneApp.NewUncachedContext(true, tmproto.Header{})
	decorator := ante.NewGovVoteDecorator(atomoneApp.AppCodec(), atomoneApp.StakingKeeper)
	stakingKeeper := atomoneApp.StakingKeeper

	// Get validator
	valAddr1 := stakingKeeper.GetAllValidators(ctx)[0].GetOperator()

	// Create one more validator
	pk := ed25519.GenPrivKeyFromSecret([]byte{uint8(13)}).PubKey()
	validator2, err := stakingtypes.NewValidator(
		sdk.ValAddress(pk.Address()),
		pk,
		stakingtypes.Description{},
	)
	valAddr2 := validator2.GetOperator()
	require.NoError(t, err)
	// Make sure the validator is bonded so it's not removed on Undelegate
	validator2.Status = stakingtypes.Bonded
	stakingKeeper.SetValidator(ctx, validator2)
	err = stakingKeeper.SetValidatorByConsAddr(ctx, validator2)
	require.NoError(t, err)
	stakingKeeper.SetNewValidatorByPowerIndex(ctx, validator2)
	err = stakingKeeper.Hooks().AfterValidatorCreated(ctx, validator2.GetOperator())
	require.NoError(t, err)

	// Get delegator (this account was created during setup)
	addr := atomoneApp.AccountKeeper.GetAccountAddressByID(ctx, 0)
	delegator, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)

	tests := []struct {
		name       string
		bondAmt    math.Int
		validators []sdk.ValAddress
		expectPass bool
	}{
		{
			name:       "delegate 0 atone",
			bondAmt:    sdk.ZeroInt(),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: false,
		},
		{
			name:       "delegate 0.1 atone",
			bondAmt:    sdk.NewInt(100000),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: false,
		},
		{
			name:       "delegate 1 atone",
			bondAmt:    sdk.NewInt(1000000),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: true,
		},
		{
			name:       "delegate 1 atom to two validators",
			bondAmt:    sdk.NewInt(1000000),
			validators: []sdk.ValAddress{valAddr1, valAddr2},
			expectPass: true,
		},
		{
			name:       "delegate 0.9 atom to two validators",
			bondAmt:    sdk.NewInt(900000),
			validators: []sdk.ValAddress{valAddr1, valAddr2},
			expectPass: false,
		},
		{
			name:       "delegate 10 atone",
			bondAmt:    sdk.NewInt(10000000),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: true,
		},
	}

	for _, tc := range tests {
		// Unbond all tokens for this delegator
		delegations := stakingKeeper.GetAllDelegatorDelegations(ctx, delegator)
		for _, del := range delegations {
			_, err := stakingKeeper.Undelegate(ctx, delegator, del.GetValidatorAddr(), del.GetShares())
			require.NoError(t, err)
		}

		// Delegate tokens
		if !tc.bondAmt.IsZero() {
			amt := tc.bondAmt.Quo(sdk.NewInt(int64(len(tc.validators))))
			for _, valAddr := range tc.validators {
				val, found := stakingKeeper.GetValidator(ctx, valAddr)
				require.True(t, found)
				_, err := stakingKeeper.Delegate(ctx, delegator, amt, stakingtypes.Unbonded, val, true)
				require.NoError(t, err)
			}
		}

		// Create vote message
		msg := govv1beta1.NewMsgVote(
			delegator,
			0,
			govv1beta1.OptionYes,
		)

		// Validate vote message
		err := decorator.ValidateVoteMsgs(ctx, []sdk.Msg{msg})
		if tc.expectPass {
			require.NoError(t, err, "expected %v to pass", tc.name)
		} else {
			require.Error(t, err, "expected %v to fail", tc.name)
		}
	}
}

// Test that the GovVoteDecorator rejects v1 vote messages from accounts with less than 1 atom staked
// Usually, only v1.VoteMsg can be submitted using the CLI.
func TestVoteSpamDecoratorGovV1(t *testing.T) {
	atomoneApp := helpers.Setup(t)
	ctx := atomoneApp.NewUncachedContext(true, tmproto.Header{})
	decorator := ante.NewGovVoteDecorator(atomoneApp.AppCodec(), atomoneApp.StakingKeeper)
	stakingKeeper := atomoneApp.StakingKeeper

	// Get validator
	valAddr1 := stakingKeeper.GetAllValidators(ctx)[0].GetOperator()

	// Create one more validator
	pk := ed25519.GenPrivKeyFromSecret([]byte{uint8(13)}).PubKey()
	validator2, err := stakingtypes.NewValidator(
		sdk.ValAddress(pk.Address()),
		pk,
		stakingtypes.Description{},
	)
	valAddr2 := validator2.GetOperator()
	require.NoError(t, err)
	// Make sure the validator is bonded so it's not removed on Undelegate
	validator2.Status = stakingtypes.Bonded
	stakingKeeper.SetValidator(ctx, validator2)
	err = stakingKeeper.SetValidatorByConsAddr(ctx, validator2)
	require.NoError(t, err)
	stakingKeeper.SetNewValidatorByPowerIndex(ctx, validator2)
	err = stakingKeeper.Hooks().AfterValidatorCreated(ctx, validator2.GetOperator())
	require.NoError(t, err)

	// Get delegator (this account was created during setup)
	addr := atomoneApp.AccountKeeper.GetAccountAddressByID(ctx, 0)
	delegator, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)

	tests := []struct {
		name       string
		bondAmt    math.Int
		validators []sdk.ValAddress
		expectPass bool
	}{
		{
			name:       "delegate 0 atone",
			bondAmt:    sdk.ZeroInt(),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: false,
		},
		{
			name:       "delegate 0.1 atone",
			bondAmt:    sdk.NewInt(100000),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: false,
		},
		{
			name:       "delegate 1 atone",
			bondAmt:    sdk.NewInt(1000000),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: true,
		},
		{
			name:       "delegate 1 atom to two validators",
			bondAmt:    sdk.NewInt(1000000),
			validators: []sdk.ValAddress{valAddr1, valAddr2},
			expectPass: true,
		},
		{
			name:       "delegate 0.9 atom to two validators",
			bondAmt:    sdk.NewInt(900000),
			validators: []sdk.ValAddress{valAddr1, valAddr2},
			expectPass: false,
		},
		{
			name:       "delegate 10 atone",
			bondAmt:    sdk.NewInt(10000000),
			validators: []sdk.ValAddress{valAddr1},
			expectPass: true,
		},
	}

	for _, tc := range tests {
		// Unbond all tokens for this delegator
		delegations := stakingKeeper.GetAllDelegatorDelegations(ctx, delegator)
		for _, del := range delegations {
			_, err := stakingKeeper.Undelegate(ctx, delegator, del.GetValidatorAddr(), del.GetShares())
			require.NoError(t, err)
		}

		// Delegate tokens
		if !tc.bondAmt.IsZero() {
			amt := tc.bondAmt.Quo(sdk.NewInt(int64(len(tc.validators))))
			for _, valAddr := range tc.validators {
				val, found := stakingKeeper.GetValidator(ctx, valAddr)
				require.True(t, found)
				_, err := stakingKeeper.Delegate(ctx, delegator, amt, stakingtypes.Unbonded, val, true)
				require.NoError(t, err)
			}
		}

		// Create vote message
		msg := govv1.NewMsgVote(
			delegator,
			0,
			govv1.VoteOption_VOTE_OPTION_YES,
			"new-v1-vote-message-test",
		)

		// Validate vote message
		err := decorator.ValidateVoteMsgs(ctx, []sdk.Msg{msg})
		if tc.expectPass {
			require.NoError(t, err, "expected %v to pass", tc.name)
		} else {
			require.Error(t, err, "expected %v to fail", tc.name)
		}
	}
}
