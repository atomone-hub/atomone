package keepers

import (
	"context"

	"cosmossdk.io/math"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	photonkeeper "github.com/atomone-hub/atomone/x/photon/keeper"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

// VAASPhotonRate presents x/photon's conversion rate to the VAAS provider,
// which prices the consumer fees in photons: the rate the ConversionRate
// query reports, computed from the bond denom and photon supplies.
type VAASPhotonRate struct {
	Photon  *photonkeeper.Keeper
	Bank    bankkeeper.Keeper
	Staking *stakingkeeper.Keeper
}

// ConversionRate returns how many photons one bond token converts to.
func (r VAASPhotonRate) ConversionRate(ctx context.Context) (math.LegacyDec, error) {
	bondDenom, err := r.Staking.BondDenom(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	bondSupply := r.Bank.GetSupply(ctx, bondDenom).Amount.ToLegacyDec()
	photonSupply := r.Bank.GetSupply(ctx, photontypes.Denom).Amount.ToLegacyDec()
	return r.Photon.PhotonConversionRate(ctx, bondSupply, photonSupply), nil
}
