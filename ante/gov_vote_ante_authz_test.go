package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
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

	// Double nested authz.MsgExec
	innerGoodMsg := govv1beta1.NewMsgVote(
		delegator,
		0,
		govv1beta1.OptionYes,
	)
	execGoodMsg := authz.NewMsgExec(granteeAddr, []sdk.Msg{innerGoodMsg})
	nestedExecMsg = authz.NewMsgExec(granteeAddr, []sdk.Msg{&execGoodMsg, &execMsg})
	err = decorator.ValidateVoteMsgs(ctx, []sdk.Msg{&nestedExecMsg})
	require.Error(t, err)
}

// maxAuthzNestingDepth mirrors the unexported constant in gov_ante_utils.go:
// iterateMsg rejects authz.MsgExec nestings strictly deeper than this value.
const maxAuthzNestingDepth = 8

// nestInMsgExec wraps leaf in `depth` nested authz.MsgExec layers and returns the
// outermost wrapper as an sdk.Msg.
func nestInMsgExec(t *testing.T, grantee sdk.AccAddress, leaf sdk.Msg, depth int) sdk.Msg {
	t.Helper()
	require.GreaterOrEqual(t, depth, 1)
	cur, err := codectypes.NewAnyWithValue(leaf)
	require.NoError(t, err)
	var exec *authz.MsgExec
	for i := 0; i < depth; i++ {
		exec = &authz.MsgExec{Grantee: grantee.String(), Msgs: []*codectypes.Any{cur}}
		cur, err = codectypes.NewAnyWithValue(exec)
		require.NoError(t, err)
	}
	return exec
}

// TestVoteSpamDecoratorAuthzNestingDepth checks that authz.MsgExec wrappers are
// only expanded up to maxAuthzNestingDepth levels, so a deeply-nested wrapper
// cannot be used to exhaust resources or bypass the vote check.
func TestVoteSpamDecoratorAuthzNestingDepth(t *testing.T) {
	atomoneApp := helpers.Setup(t)
	ctx := atomoneApp.NewUncachedContext(true, tmproto.Header{})
	decorator := ante.NewGovVoteDecorator(atomoneApp.AppCodec(), atomoneApp.StakingKeeper)
	stakingKeeper := atomoneApp.StakingKeeper

	valAddrs, err := stakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	valAddr1, err := stakingKeeper.ValidatorAddressCodec().StringToBytes(valAddrs[0].OperatorAddress)
	require.NoError(t, err)

	// This account was created during setup.
	delegator, err := atomoneApp.AccountKeeper.Accounts.Indexes.Number.MatchExact(ctx, 0)
	require.NoError(t, err)

	granteeAddr := sdk.AccAddress(ed25519.GenPrivKeyFromSecret([]byte{uint8(14)}).PubKey().Address())

	// Delegate enough stake so the inner vote passes the stake check: the only
	// reason a deeply-nested wrapper gets rejected is the nesting-depth limit.
	val, err := stakingKeeper.GetValidator(ctx, valAddr1)
	require.NoError(t, err)
	_, err = stakingKeeper.Delegate(ctx, delegator, math.NewInt(1000000), stakingtypes.Unbonded, val, true)
	require.NoError(t, err)

	validVote := govv1beta1.NewMsgVote(delegator, 0, govv1beta1.OptionYes)

	// At the limit: the inner vote is reached and validates successfully.
	atLimit := nestInMsgExec(t, granteeAddr, validVote, maxAuthzNestingDepth)
	require.NoError(t, decorator.ValidateVoteMsgs(ctx, []sdk.Msg{atLimit}))

	// At the limit with an invalid inner vote: rejected, but for the inner vote's
	// own reason (empty voter) rather than the depth limit. This proves the inner
	// message is actually reached and checked at the limit.
	invalidVote := govv1beta1.NewMsgVote(nil, 0, govv1beta1.OptionYes)
	atLimitInvalid := nestInMsgExec(t, granteeAddr, invalidVote, maxAuthzNestingDepth)
	err = decorator.ValidateVoteMsgs(ctx, []sdk.Msg{atLimitInvalid})
	require.Error(t, err)
	require.NotContains(t, err.Error(), "authz nesting depth exceeded")

	// One layer beyond the limit: rejected before reaching the inner vote.
	beyondLimit := nestInMsgExec(t, granteeAddr, validVote, maxAuthzNestingDepth+1)
	err = decorator.ValidateVoteMsgs(ctx, []sdk.Msg{beyondLimit})
	require.Error(t, err)
	require.ErrorContains(t, err, "authz nesting depth exceeded")
}
