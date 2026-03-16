package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/ante"
	"github.com/atomone-hub/atomone/app/helpers"
	govv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

func TestVoteSpamDecoratorAuthz(t *testing.T) {
	atomoneApp := helpers.Setup(t)
	ctx := atomoneApp.NewUncachedContext(true, tmproto.Header{})
	decorator := ante.NewGovVoteDecorator(atomoneApp.AppCodec(), atomoneApp.StakingKeeper)
	stakingKeeper := atomoneApp.StakingKeeper

	// Get validator
	valAddrs, err := stakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	valAddr1, err := stakingKeeper.ValidatorAddressCodec().StringToBytes(valAddrs[0].OperatorAddress)
	require.NoError(t, err)

	// Get delegator (this account was created during setup)
	delegator, err := atomoneApp.AccountKeeper.Accounts.Indexes.Number.MatchExact(ctx, 0)
	require.NoError(t, err)

	// Create another account for grantee
	granteeAddr := sdk.AccAddress(ed25519.GenPrivKeyFromSecret([]byte{uint8(14)}).PubKey().Address())

	// Delegate 1 atone so delegator has enough stake to vote
	val, err := stakingKeeper.GetValidator(ctx, valAddr1)
	require.NoError(t, err)
	_, err = stakingKeeper.Delegate(ctx, delegator, math.NewInt(1000000), stakingtypes.Unbonded, val, true)
	require.NoError(t, err)

	// Create inner vote message
	innerMsg := govv1beta1.NewMsgVote(
		nil, // nil addr
		0,
		govv1beta1.OptionYes,
	)

	// Wrap in authz.MsgExec
	execMsg := authz.NewMsgExec(granteeAddr, []sdk.Msg{innerMsg})
	err = decorator.ValidateVoteMsgs(ctx, []sdk.Msg{&execMsg})
	require.Error(t, err)

	// Create nested authz.MsgExec
	nestedExecMsg := authz.NewMsgExec(granteeAddr, []sdk.Msg{&execMsg})
	err = decorator.ValidateVoteMsgs(ctx, []sdk.Msg{&nestedExecMsg})
	require.Error(t, err)
}
