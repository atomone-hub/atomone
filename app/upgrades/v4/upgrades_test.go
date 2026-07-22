package v4_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/app/helpers"
	v4 "github.com/atomone-hub/atomone/app/upgrades/v4"
	govv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestMigrateStakingParams(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewUncachedContext(false, cmtproto.Header{})
	sk := app.StakingKeeper

	require.NoError(t, v4.MigrateStakingParams(ctx, sk))

	got, err := sk.GetParams(ctx)
	require.NoError(t, err)
	require.NoError(t, got.Validate())
	require.Equal(t, sdk.NewCoin(got.BondDenom, math.NewInt(100_000000)), got.KeyRotationFee)
	fivePercent := math.LegacyMustNewDecFromStr("0.05")
	require.Equal(t, fivePercent, got.MinCommissionRate)
	require.Equal(t, fivePercent, got.MaxCommissionRate)
}

// TestUnpackLegacyIBCProposalContent guards against a regression of the v4
// mainnet upgrade failure where migrating gov proposals errored with:
//
//	no concrete type registered for type URL
//	/ibc.core.client.v1.ClientUpdateProposal against interface *v1beta1.Content
//
// Historical mainnet proposals stored a legacy IBC ClientUpdateProposal wrapped
// in a MsgExecLegacyContent. ibc-go registers that content type against the
// SDK's gov Content interface, but the AtomOne gov fork uses its own Content
// interface, so the type must be registered against it manually (see
// x/gov/types/v1beta1/codec.go). This test reproduces the exact unpack chain
// exercised by migrateProposals.
func TestUnpackLegacyIBCProposalContent(t *testing.T) {
	app := helpers.Setup(t)
	cdc := app.AppCodec()

	content := &ibcclienttypes.ClientUpdateProposal{ //nolint:staticcheck
		Title:              "update client",
		Description:        "recover an expired IBC client",
		SubjectClientId:    "07-tendermint-0",
		SubstituteClientId: "07-tendermint-1",
	}
	contentAny, err := codectypes.NewAnyWithValue(content)
	require.NoError(t, err)

	msgAny, err := codectypes.NewAnyWithValue(govv1.NewMsgExecLegacyContent(contentAny, "authority"))
	require.NoError(t, err)

	prop := govv1.Proposal{Id: 1, Messages: []*codectypes.Any{msgAny}}
	bz, err := cdc.Marshal(&prop)
	require.NoError(t, err)

	// Unmarshalling triggers UnpackInterfaces, which resolves the legacy IBC
	// content Any against the AtomOne gov v1beta1.Content interface. This is the
	// exact step that failed during the v4 mainnet upgrade.
	var got govv1.Proposal
	require.NoError(t, cdc.Unmarshal(bz, &got))
}
