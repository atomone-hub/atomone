package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/atomone-hub/atomone/app/helpers"
	"github.com/atomone-hub/atomone/x/coredaos/types"
)

// TestGovHookAfterProposalSubmission exercises the coredaos AfterProposalSubmission hook
// directly against a real gov keeper. Proposals are stored via SetProposal (which does not
// fire hooks), so the hook can be invoked in isolation over message shapes that the wired
// submission path would itself reject.
func TestGovHookAfterProposalSubmission(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})
	hooks := app.CoreDaosKeeper.GovHooks()

	current := simtestutil.CreateRandomAccounts(1)[0].String()
	other := simtestutil.CreateRandomAccounts(1)[0].String()
	govAddr := govModuleAddr()
	extDuration := time.Hour

	// changing alters the oversight DAO address; same keeps it unchanged.
	changing := &types.MsgUpdateParams{Authority: govAddr, Params: types.Params{OversightDaoAddress: other, VotingPeriodExtensionDuration: &extDuration}}
	same := &types.MsgUpdateParams{Authority: govAddr, Params: types.Params{OversightDaoAddress: current, VotingPeriodExtensionDuration: &extDuration}}

	setOversight := func(addr string) {
		require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, types.Params{OversightDaoAddress: addr, VotingPeriodExtensionDuration: &extDuration}))
	}
	// store writes a proposal directly (no hook) so the hook can be invoked standalone.
	store := func(id uint64, msgs ...sdk.Msg) {
		anys, err := sdktx.SetMsgs(msgs)
		require.NoError(t, err)
		require.NoError(t, app.GovKeeper.SetProposal(ctx, govv1.Proposal{Id: id, Messages: anys, Status: govv1.StatusVotingPeriod}))
	}

	tests := []struct {
		name      string
		oversight string // "" => oversight DAO unset, hook disabled
		msgs      []sdk.Msg
		wantErr   bool
	}{
		{"disabled when oversight DAO unset", "", []sdk.Msg{changing, same}, false},
		{"single oversight change, not bundled", current, []sdk.Msg{changing}, false},
		{"oversight change bundled with another msg", current, []sdk.Msg{changing, same}, true},
		{"bundled but no oversight change", current, []sdk.Msg{same, same}, false},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setOversight(tt.oversight)
			id := uint64(i + 1)
			store(id, tt.msgs...)
			err := hooks.AfterProposalSubmission(ctx, id)
			if tt.wantErr {
				require.ErrorContains(t, err, "cannot be bundled")
				return
			}
			require.NoError(t, err)
		})
	}

	// A non-existent proposal is a no-op (nothing to enforce).
	setOversight(current)
	require.NoError(t, hooks.AfterProposalSubmission(ctx, 9999))
}

// TestGovHookWiredRejectsBundledOversightChange checks that the coredaos hook is actually
// WIRED into the gov keeper (via SetHooks in app wiring) and fires during a real
// SubmitProposal. Nothing else guards the wiring, so removing the SetHooks call would only
// be caught here. It submits through the SDK gov keeper's SubmitProposal (which invokes
// AfterProposalSubmission and, being the keeper method, skips the msgServer deposit gate).
func TestGovHookWiredRejectsBundledOversightChange(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})

	current := simtestutil.CreateRandomAccounts(1)[0].String()
	other := simtestutil.CreateRandomAccounts(1)[0].String()
	govAddr := govModuleAddr()
	extDuration := time.Hour
	require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, types.Params{OversightDaoAddress: current, VotingPeriodExtensionDuration: &extDuration}))

	// Both messages are signed by the gov module account (required of proposal messages).
	changing := &types.MsgUpdateParams{Authority: govAddr, Params: types.Params{OversightDaoAddress: other, VotingPeriodExtensionDuration: &extDuration}}
	same := &types.MsgUpdateParams{Authority: govAddr, Params: types.Params{OversightDaoAddress: current, VotingPeriodExtensionDuration: &extDuration}}
	proposer := simtestutil.CreateRandomAccounts(1)[0]

	// Bundling an oversight-DAO change with another message must be rejected by the hook.
	_, err := app.GovKeeper.SubmitProposal(ctx, []sdk.Msg{changing, same}, "", "title", "summary", proposer)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot be bundled")

	// A single (non-bundled) oversight change must not trip the hook. It may still error for
	// unrelated reasons, so only assert the hook did not fire.
	_, err = app.GovKeeper.SubmitProposal(ctx, []sdk.Msg{changing}, "", "title", "summary", proposer)
	if err != nil {
		require.NotContains(t, err.Error(), "cannot be bundled")
	}
}

// TestGovHookRejectsUpdateParamsDelegation checks the constitutional invariant that coredaos
// MsgUpdateParams may never be delegated via authz. The check targets the grant creation (the
// only way a gov->X grant for this msg can come to exist) and is unconditional: it holds even
// when no oversight DAO is set, because MsgUpdateParams governs all coredaos params.
func TestGovHookRejectsUpdateParamsDelegation(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})
	hooks := app.CoreDaosKeeper.GovHooks()

	govAddr := govModuleAddr()
	grantee := simtestutil.CreateRandomAccounts(1)[0]
	extDuration := time.Hour

	store := func(id uint64, msgs ...sdk.Msg) {
		anys, err := sdktx.SetMsgs(msgs)
		require.NoError(t, err)
		require.NoError(t, app.GovKeeper.SetProposal(ctx, govv1.Proposal{Id: id, Messages: anys, Status: govv1.StatusVotingPeriod}))
	}

	updateParamsGrant, err := authz.NewMsgGrant(sdk.MustAccAddressFromBech32(govAddr), grantee,
		authz.NewGenericAuthorization(sdk.MsgTypeURL(&types.MsgUpdateParams{})), nil)
	require.NoError(t, err)
	benignGrant, err := authz.NewMsgGrant(sdk.MustAccAddressFromBech32(govAddr), grantee,
		authz.NewGenericAuthorization(sdk.MsgTypeURL(&banktypes.MsgSend{})), nil)
	require.NoError(t, err)

	// Oversight DAO left unset on purpose: the non-delegation rule is unconditional.
	require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, types.Params{VotingPeriodExtensionDuration: &extDuration}))

	store(1, updateParamsGrant)
	require.ErrorContains(t, hooks.AfterProposalSubmission(ctx, 1), "cannot be delegated")

	// Delegating an unrelated message from gov must still be allowed.
	store(2, benignGrant)
	require.NoError(t, hooks.AfterProposalSubmission(ctx, 2))
}

// TestGovHookRejectsNestedAuthzOversightTakeover is the end-to-end regression test for the
// nested-authz.MsgExec bypass: a proposal that (1) grants the attacker gov's authority over
// coredaos.MsgUpdateParams and (2) hides a malicious MsgUpdateParams two authz.MsgExec layers
// deep. The nested change never appears as a top-level MsgUpdateParams, so it evaded the
// bundling guard; it is now rejected at submission because the required grant (message 1)
// cannot be created.
func TestGovHookRejectsNestedAuthzOversightTakeover(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})

	govAddr := govModuleAddr()
	extDuration := time.Hour
	original := simtestutil.CreateRandomAccounts(1)[0]
	attacker := simtestutil.CreateRandomAccounts(1)[0]
	require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, types.Params{
		OversightDaoAddress: original.String(), VotingPeriodExtensionDuration: &extDuration,
	}))

	// Message 1: gov -> attacker grant for coredaos.MsgUpdateParams (the un-hideable linchpin).
	govGrantsAttacker, err := authz.NewMsgGrant(sdk.MustAccAddressFromBech32(govAddr), attacker,
		authz.NewGenericAuthorization(sdk.MsgTypeURL(&types.MsgUpdateParams{})), nil)
	require.NoError(t, err)

	// Message 2: MsgExec(gov) -> MsgExec(attacker) -> MsgUpdateParams{OversightDaoAddress: attacker}.
	malicious := &types.MsgUpdateParams{Authority: govAddr, Params: types.Params{OversightDaoAddress: attacker.String(), VotingPeriodExtensionDuration: &extDuration}}
	maliciousAny, err := codectypes.NewAnyWithValue(malicious)
	require.NoError(t, err)
	innerExec := &authz.MsgExec{Grantee: attacker.String(), Msgs: []*codectypes.Any{maliciousAny}}
	innerAny, err := codectypes.NewAnyWithValue(innerExec)
	require.NoError(t, err)
	outerExec := &authz.MsgExec{Grantee: govAddr, Msgs: []*codectypes.Any{innerAny}}

	_, err = app.GovKeeper.SubmitProposal(ctx, []sdk.Msg{govGrantsAttacker, outerExec}, "", "title", "summary", attacker)
	require.Error(t, err, "nested-authz oversight takeover must be rejected at submission")
	require.ErrorContains(t, err, "cannot be delegated")

	// The oversight DAO must be untouched.
	require.Equal(t, original.String(), app.CoreDaosKeeper.GetParams(ctx).OversightDaoAddress)
}

// TestSelfExecWrappedGrantRejectedUpstream covers the one path the coredaos hook does NOT
// inspect directly: the delegating MsgGrant hidden inside a self-executing authz.MsgExec
// (grantee == gov at every layer). The coredaos hook only checks top-level MsgGrant messages,
// so completeness here relies on gov's SubmitProposal rejecting any self-executing MsgExec that
// reaches a gov-signed leaf (ContainsSelfExecAsAuthority) — and the leaf MsgGrant is gov-signed
// (granter == gov). This test pins that the upstream guard fires for both single and double
// wrapping, which is what makes invariant (1) in AfterProposalSubmission sufficient.
func TestSelfExecWrappedGrantRejectedUpstream(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})

	govAddr := govModuleAddr()
	grantee := simtestutil.CreateRandomAccounts(1)[0]
	proposer := simtestutil.CreateRandomAccounts(1)[0]

	grant, err := authz.NewMsgGrant(sdk.MustAccAddressFromBech32(govAddr), grantee,
		authz.NewGenericAuthorization(sdk.MsgTypeURL(&types.MsgUpdateParams{})), nil)
	require.NoError(t, err)
	grantAny, err := codectypes.NewAnyWithValue(grant)
	require.NoError(t, err)

	// Single self-exec wrap: MsgExec(gov){ MsgGrant(gov->grantee) }.
	selfExec := &authz.MsgExec{Grantee: govAddr, Msgs: []*codectypes.Any{grantAny}}
	_, err = app.GovKeeper.SubmitProposal(ctx, []sdk.Msg{selfExec}, "", "title", "summary", proposer)
	require.Error(t, err, "self-exec-wrapped grant must be rejected at submission")
	require.ErrorContains(t, err, "self-executing")

	// Double self-exec wrap: MsgExec(gov){ MsgExec(gov){ MsgGrant(gov->grantee) } }.
	innerAny, err := codectypes.NewAnyWithValue(selfExec)
	require.NoError(t, err)
	doubleExec := &authz.MsgExec{Grantee: govAddr, Msgs: []*codectypes.Any{innerAny}}
	_, err = app.GovKeeper.SubmitProposal(ctx, []sdk.Msg{doubleExec}, "", "title", "summary", proposer)
	require.Error(t, err, "double self-exec-wrapped grant must be rejected at submission")
	require.ErrorContains(t, err, "self-executing")
}
