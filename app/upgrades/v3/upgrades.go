package v3

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/atomone-hub/atomone/app/keepers"
	govkeeper "github.com/atomone-hub/atomone/x/gov/keeper"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v3
func CreateUpgradeHandler(
	mm *module.Manager,
	_ codec.Codec,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		sdkCtx.Logger().Info("Starting module migrations...")
		// RunMigrations will detect the add of the feemarket module, will initiate
		// its genesis and will fill the versionMap with its consensus version.
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		if err := initGovDynamicQuorum(sdkCtx, keepers.GovKeeperWrapper); err != nil {
			return vm, err
		}

		if err := convertFundsToPhoton(sdkCtx, keepers); err != nil {
			return vm, err
		}

		return vm, nil
	}
}

// initGovDynamicQuorum initialized the gov module for the dynamic quorum
// features, which means setting the new parameters min/max quorums and the
// participation ema.
func initGovDynamicQuorum(ctx sdk.Context, govKeeper *govkeeper.Keeper) error {
	ctx.Logger().Info("Initializing gov module for dynamic quorum...")
	params, err := govKeeper.Params.Get(ctx)
	if err != nil {
		return err
	}
	defaultParams := v1.DefaultParams()
	params.QuorumRange = &sdkgovv1.QuorumRange{
		Min: defaultParams.QuorumRange.Min,
		Max: defaultParams.QuorumRange.Max,
	}
	params.ConstitutionAmendmentQuorumRange = &sdkgovv1.QuorumRange{
		Min: defaultParams.ConstitutionAmendmentQuorumRange.Min,
		Max: defaultParams.ConstitutionAmendmentQuorumRange.Max,
	}
	params.LawQuorumRange = &sdkgovv1.QuorumRange{
		Min: defaultParams.LawQuorumRange.Min,
		Max: defaultParams.LawQuorumRange.Max,
	}
	if err := govKeeper.Params.Set(ctx, params); err != nil {
		return fmt.Errorf("set gov params: %w", err)
	}
	// NOTE(tb): Disregarding whales' votes, the current participation is less than 12%
	initParticipationEma := math.LegacyNewDecWithPrec(12, 2)
	if err := govKeeper.ParticipationEMA.Set(ctx, initParticipationEma); err != nil {
		return fmt.Errorf("set participation EMA: %w", err)
	}
	if err := govKeeper.ConstitutionAmendmentParticipationEMA.Set(ctx, initParticipationEma); err != nil {
		return fmt.Errorf("set constitution amendment participation EMA: %w", err)
	}
	if err := govKeeper.LawParticipationEMA.Set(ctx, initParticipationEma); err != nil {
		return fmt.Errorf("set law participation EMA: %w", err)
	}
	ctx.Logger().Info("Gov module initialized for dynamic quorum")
	return nil
}

// convertFundsToPhoton converts 50% of the bond denom (uatone) funds from
// community pool and 90% from treasury DAO to photons (uphoton)
func convertFundsToPhoton(ctx sdk.Context, keepers *keepers.AppKeepers) error {
	ctx.Logger().Info("Converting 50% of bond denom funds from community pool and 90% from treasury DAO to photons...")

	// Get bond denom
	bondDenom, err := keepers.StakingKeeper.BondDenom(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bond denom: %w", err)
	}

	// Get treasury DAO address
	treasuryAddrBz := []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xbd\xa0")
	treasuryAddr := sdk.MustBech32ifyAddressBytes(sdk.GetConfig().GetBech32AccountAddrPrefix(), treasuryAddrBz)

	// Process community pool funds
	if err := convertCommunityPoolFundsToPhoton(ctx, keepers, bondDenom); err != nil {
		return fmt.Errorf("failed to convert community pool funds: %w", err)
	}

	// Process treasury DAO funds
	if err := convertTreasuryDAOFundsToPhoton(ctx, keepers, bondDenom, treasuryAddr); err != nil {
		return fmt.Errorf("failed to convert treasury DAO funds: %w", err)
	}

	ctx.Logger().Info("Successfully converted funds to photons")
	return nil
}

// convertCommunityPoolFundsToPhoton converts 50% of the community pool bond denom funds to photons
func convertCommunityPoolFundsToPhoton(ctx sdk.Context, keepers *keepers.AppKeepers, bondDenom string) error {
	feePool, err := keepers.DistrKeeper.FeePool.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to get fee pool: %w", err)
	} else if errors.Is(err, collections.ErrNotFound) {
		return nil
	}

	// Get community pool funds
	communityPoolCoins := feePool.CommunityPool
	bondDenomCoin := communityPoolCoins.AmountOf(bondDenom)

	if bondDenomCoin.IsZero() {
		ctx.Logger().Info("No bond denom funds in community pool to convert")
		return nil
	}

	// Calculate 50% of funds to convert
	amountToConvert := bondDenomCoin.Mul(math.LegacyNewDecWithPrec(50, 2))
	coinToConvert := sdk.NewCoin(bondDenom, amountToConvert.TruncateInt())

	// Send funds from community pool to photon module account
	if err := sendCommunityPoolFundsToModuleAccount(ctx, keepers.BankKeeper, keepers.DistrKeeper, sdk.NewCoins(coinToConvert), photontypes.ModuleName); err != nil {
		return fmt.Errorf("failed to distribute from community pool: %w", err)
	}

	bondDenomSupply := keepers.BankKeeper.GetSupply(ctx, bondDenom).Amount.ToLegacyDec()
	uphotonSupply := keepers.BankKeeper.GetSupply(ctx, photontypes.Denom).Amount.ToLegacyDec()
	conversionRate := keepers.PhotonKeeper.PhotonConversionRate(ctx, bondDenomSupply, uphotonSupply)

	// Calculate photons to mint
	photonAmtToMint := coinToConvert.Amount.ToLegacyDec().Mul(conversionRate).TruncateInt()
	photonToMint := sdk.NewCoin(photontypes.Denom, photonAmtToMint)

	// Mint photons
	err = keepers.BankKeeper.MintCoins(ctx, photontypes.ModuleName, sdk.NewCoins(photonToMint))
	if err != nil {
		return fmt.Errorf("failed to mint photons: %w", err)
	}

	// Burn the bond denom coins
	err = keepers.BankKeeper.BurnCoins(ctx, photontypes.ModuleName, sdk.NewCoins(coinToConvert))
	if err != nil {
		return fmt.Errorf("failed to burn bond denom coins: %w", err)
	}

	// Send minted photons back to community pool
	moduleAddr := keepers.AccountKeeper.GetModuleAddress(photontypes.ModuleName)
	err = keepers.DistrKeeper.FundCommunityPool(ctx, sdk.NewCoins(photonToMint), moduleAddr)
	if err != nil {
		return fmt.Errorf("failed to fund community pool with photons: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"convert_community_pool_to_photon",
			sdk.NewAttribute("bond_denom_converted", coinToConvert.String()),
			sdk.NewAttribute("photons_minted", photonToMint.String()),
			sdk.NewAttribute("conversion_rate", conversionRate.String()),
		),
	)

	return nil
}

// convertTreasuryDAOFundsToPhoton converts 90% of the treasury DAO's bond denom funds to photons
func convertTreasuryDAOFundsToPhoton(ctx sdk.Context, keepers *keepers.AppKeepers, bondDenom string, treasuryAddr string) error {
	treasuryAccAddr, err := sdk.AccAddressFromBech32(treasuryAddr)
	if err != nil {
		return fmt.Errorf("invalid treasury DAO address: %w", err)
	}

	// Get treasury balance
	balance := keepers.BankKeeper.GetBalance(ctx, treasuryAccAddr, bondDenom)

	if balance.IsZero() {
		ctx.Logger().Info("No bond denom funds in treasury DAO to convert")
		return nil
	}

	// Calculate 90% of funds
	amountToConvert := balance.Amount.ToLegacyDec().Mul(math.LegacyNewDecWithPrec(90, 2)).TruncateInt()
	coinToConvert := sdk.NewCoin(bondDenom, amountToConvert)

	// Send bond denom to photon module account for burning
	err = keepers.BankKeeper.SendCoinsFromAccountToModule(ctx, treasuryAccAddr, photontypes.ModuleName, sdk.NewCoins(coinToConvert))
	if err != nil {
		return fmt.Errorf("failed to send coins from treasury to module: %w", err)
	}

	bondDenomSupply := keepers.BankKeeper.GetSupply(ctx, bondDenom).Amount.ToLegacyDec()
	uphotonSupply := keepers.BankKeeper.GetSupply(ctx, photontypes.Denom).Amount.ToLegacyDec()
	conversionRate := keepers.PhotonKeeper.PhotonConversionRate(ctx, bondDenomSupply, uphotonSupply)

	// Calculate photons to mint
	photonAmtToMint := coinToConvert.Amount.ToLegacyDec().Mul(conversionRate).RoundInt()
	photonToMint := sdk.NewCoin(photontypes.Denom, photonAmtToMint)

	// Mint photons
	err = keepers.BankKeeper.MintCoins(ctx, photontypes.ModuleName, sdk.NewCoins(photonToMint))
	if err != nil {
		return fmt.Errorf("failed to mint photons: %w", err)
	}

	// Burn the bond denom coins
	err = keepers.BankKeeper.BurnCoins(ctx, photontypes.ModuleName, sdk.NewCoins(coinToConvert))
	if err != nil {
		return fmt.Errorf("failed to burn bond denom coins: %w", err)
	}

	// Send minted photons back to treasury
	err = keepers.BankKeeper.SendCoinsFromModuleToAccount(ctx, photontypes.ModuleName, treasuryAccAddr, sdk.NewCoins(photonToMint))
	if err != nil {
		return fmt.Errorf("failed to send photons to treasury: %w", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"convert_treasury_to_photon",
			sdk.NewAttribute("bond_denom_converted", coinToConvert.String()),
			sdk.NewAttribute("photons_minted", photonToMint.String()),
			sdk.NewAttribute("conversion_rate", conversionRate.String()),
		),
	)

	return nil
}

// sendCommunityPoolFundsToModuleAccount sends funds from the community pool to the specified module account.
// It is a modified version of the original `DistributeFromFeePool` function that uses `SendCoinsFromModuleToModule`
// instead of `SendCoinsFromModuleToAccount` to send the funds to the photon module.
func sendCommunityPoolFundsToModuleAccount(ctx sdk.Context, bankKeeper distrtypes.BankKeeper, distrKeeper distrkeeper.Keeper, amount sdk.Coins, receiverModuleName string) error {
	feePool, err := distrKeeper.FeePool.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return fmt.Errorf("failed to get fee pool: %w", err)
	} else if errors.Is(err, collections.ErrNotFound) {
		return nil
	}

	// NOTE the community pool isn't a module account, however its coins
	// are held in the distribution module account. Thus the community pool
	// must be reduced separately from the SendCoinsFromModuleToAccount call
	newPool, negative := feePool.CommunityPool.SafeSub(sdk.NewDecCoinsFromCoins(amount...))
	if negative {
		return distrtypes.ErrBadDistribution
	}

	feePool.CommunityPool = newPool

	if err := bankKeeper.SendCoinsFromModuleToModule(ctx, distrtypes.ModuleName, receiverModuleName, amount); err != nil {
		return err
	}

	if err := distrKeeper.FeePool.Set(ctx, feePool); err != nil {
		return err
	}

	return nil
}
