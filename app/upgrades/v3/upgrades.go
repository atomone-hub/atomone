package v3

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/atomone-hub/atomone/app/keepers"
	appparams "github.com/atomone-hub/atomone/app/params"
	govkeeper "github.com/atomone-hub/atomone/x/gov/keeper"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v3
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("Starting module migrations...")
		// RunMigrations will detect the add of the feemarket module, will initiate
		// its genesis and will fill the versionMap with its consensus version.
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}
		if err := initGovDynamicQuorum(ctx, keepers.GovKeeper); err != nil {
			return vm, err
		}
		if err := convertFundsToPhoton(ctx, keepers); err != nil {
			return vm, err
		}

		ctx.Logger().Info("Upgrade complete")
		return vm, nil
	}
}

// initGovDynamicQuorum initialized the gov module for the dynamic quorum
// features, which means setting the new parameters min/max quorums and the
// participation ema.
func initGovDynamicQuorum(ctx sdk.Context, govKeeper *govkeeper.Keeper) error {
	ctx.Logger().Info("Initializing gov module for dynamic quorum...")
	params := govKeeper.GetParams(ctx)
	defaultParams := v1.DefaultParams()
	params.QuorumRange = defaultParams.QuorumRange
	params.ConstitutionAmendmentQuorumRange = defaultParams.ConstitutionAmendmentQuorumRange
	params.LawQuorumRange = defaultParams.LawQuorumRange
	if err := govKeeper.SetParams(ctx, params); err != nil {
		return fmt.Errorf("set gov params: %w", err)
	}
	// NOTE(tb): Disregarding whales' votes, the current participation is less than 12%
	initParticipationEma := sdk.NewDecWithPrec(12, 2)
	govKeeper.SetParticipationEMA(ctx, initParticipationEma)
	govKeeper.SetConstitutionAmendmentParticipationEMA(ctx, initParticipationEma)
	govKeeper.SetLawParticipationEMA(ctx, initParticipationEma)
	ctx.Logger().Info("Gov module initialized for dynamic quorum")
	return nil
}

// convertFundsToPhoton converts 90% of the bond denom (uatone) funds from community pool and treasury DAO to photons (uphoton)
func convertFundsToPhoton(ctx sdk.Context, keepers *keepers.AppKeepers) error {
	ctx.Logger().Info("Converting 90% of bond denom funds to photons...")

	// Get bond denom
	bondDenom := keepers.StakingKeeper.BondDenom(ctx)

	// Get treasury DAO address
	treasuryAddrBz := []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xbd\xa0")
	treasuryAddr := sdk.MustBech32ifyAddressBytes(appparams.Bech32PrefixAccAddr, treasuryAddrBz)

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

// convertCommunityPoolFundsToPhoton converts 90% of the community pool bond denom funds to photons
func convertCommunityPoolFundsToPhoton(ctx sdk.Context, keepers *keepers.AppKeepers, bondDenom string) error {
	// Get community pool funds
	communityPoolCoins := keepers.DistrKeeper.GetFeePoolCommunityCoins(ctx)
	bondDenomCoin := communityPoolCoins.AmountOf(bondDenom)

	if bondDenomCoin.IsZero() {
		ctx.Logger().Info("No bond denom funds in community pool to convert")
		return nil
	}

	// Calculate 90% of funds to convert
	amountToConvert := bondDenomCoin.Mul(sdk.NewDecWithPrec(90, 2))
	coinToConvert := sdk.NewCoin(bondDenom, amountToConvert.TruncateInt())

	// First, distribute from community pool to photon module account
	moduleAddr := keepers.AccountKeeper.GetModuleAddress(photontypes.ModuleName)
	err := keepers.DistrKeeper.DistributeFromFeePool(ctx, sdk.NewCoins(coinToConvert), moduleAddr)
	if err != nil {
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
	amountToConvert := balance.Amount.ToLegacyDec().Mul(sdk.NewDecWithPrec(90, 2)).TruncateInt()
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
