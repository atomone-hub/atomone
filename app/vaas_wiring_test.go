package atomone_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	providerkeeper "github.com/allinbits/vaas/x/vaas/provider/keeper"
	providertypes "github.com/allinbits/vaas/x/vaas/provider/types"
	vaastypes "github.com/allinbits/vaas/x/vaas/types"
	"github.com/atomone-hub/atomone/app/helpers"
	"github.com/atomone-hub/atomone/app/keepers"
)

// TestVAASProviderWiring pins what the application must hand the VAAS
// provider for the protocol to hold: the module runs last in EndBlock, after
// staking, so the validator set it hands CometBFT is the one staking settled
// in the same block; its staking hook is registered, so a consensus-key
// rotation onto an assigned consumer key is refused; its send restriction is
// on the bank keeper, so nothing reaches a consumer fee pool outside the
// module's own paths; its IBC v2 route is registered under the provider
// application id, so consumer packets are routed to it; and the photon rate
// it prices consumer fees with is x/photon's.
func TestVAASProviderWiring(t *testing.T) {
	app := helpers.Setup(t)
	ctx := app.NewContext(false)

	// The orders.
	end := app.ModuleManager().OrderEndBlockers
	require.Equal(t, providertypes.ModuleName, end[len(end)-1], "the provider runs last in EndBlock")
	require.Less(t, slices.Index(end, stakingtypes.ModuleName), slices.Index(end, providertypes.ModuleName),
		"staking settles the bonded set before the provider reads it")
	begin := app.ModuleManager().OrderBeginBlockers
	require.Contains(t, begin, providertypes.ModuleName)
	require.Contains(t, app.ModuleManager().OrderInitGenesis, providertypes.ModuleName)

	// The staking hook.
	hooks, ok := app.StakingKeeper.Hooks().(stakingtypes.MultiStakingHooks)
	require.True(t, ok, "the staking keeper carries the multi hooks the app sets")
	require.True(t, slices.ContainsFunc(hooks, func(h stakingtypes.StakingHooks) bool {
		_, isProvider := h.(providerkeeper.Hooks)
		return isProvider
	}), "the provider's staking hooks are registered")

	// The send restriction: a consumer fee pool address refuses a direct send.
	const consumerID = uint64(0)
	pool := app.ProviderKeeper.GetConsumerFeePoolAddress(consumerID)
	require.NoError(t, app.ProviderKeeper.FeePoolAddressToConsumerId.Set(ctx, pool, consumerID))
	coins := sdk.NewCoins(sdk.NewInt64Coin("uatone", 1))
	sender := sdk.AccAddress([]byte("vaas-wiring-sender__"))
	require.NoError(t, app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, sender, coins))
	err := app.BankKeeper.SendCoins(ctx, sender, pool, coins)
	require.ErrorContains(t, err, "not permitted")

	// The IBC v2 route.
	require.True(t, app.IBCKeeper.ChannelKeeperV2.Router.HasRoute(vaastypes.ProviderAppID),
		"the provider is routed under its application id")

	// The photon rate the provider prices consumer fees with is x/photon's.
	rate, err := keepers.VAASPhotonRate{Photon: app.PhotonKeeper, Bank: app.BankKeeper, Staking: app.StakingKeeper}.ConversionRate(ctx)
	require.NoError(t, err)
	require.True(t, rate.IsPositive(), "the conversion rate is positive: %s", rate)

	// The default genesis carries the provider's section and validates.
	genesis := app.DefaultGenesis()
	raw, ok := genesis[providertypes.ModuleName]
	require.True(t, ok, "the default genesis has a %s section", providertypes.ModuleName)
	var state providertypes.GenesisState
	require.NoError(t, app.AppCodec().UnmarshalJSON(raw, &state))
	require.NoError(t, state.Validate())
}
