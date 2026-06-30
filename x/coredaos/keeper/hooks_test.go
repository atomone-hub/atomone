package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
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
