package v1

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetNewMinDeposit(minDepositFloor, lastMinDeposit, maxLastMinDeposit sdk.Coins, alpha sdk.Dec) (newMinDeposit, maxMinDeposit sdk.Coins) {
	newMinDeposit = sdk.NewCoins()
	maxMinDeposit = sdk.NewCoins()
	minDepositFloorDenomsSeen := make(map[string]bool)
	for _, lastMinDepositCoin := range lastMinDeposit {
		minDepositFloorCoinAmt := minDepositFloor.AmountOf(lastMinDepositCoin.Denom)
		if minDepositFloorCoinAmt.IsZero() {
			// minDepositFloor was changed since last update,
			// and this coin was removed.
			// reflect this also in the current min initial deposit,
			// i.e. remove this coin
			continue
		}
		minDepositFloorDenomsSeen[lastMinDepositCoin.Denom] = true
		minDepositCoinAmt, maxMinDepositCoinAmt := calculateMinDepositAmount(
			lastMinDepositCoin,
			minDepositFloorCoinAmt,
			alpha,
			maxLastMinDeposit,
		)
		newMinDeposit = append(newMinDeposit, sdk.NewCoin(lastMinDepositCoin.Denom, minDepositCoinAmt))
		maxMinDeposit = append(maxMinDeposit, sdk.NewCoin(lastMinDepositCoin.Denom, maxMinDepositCoinAmt))
	}

	// make sure any new denoms in minDepositFloor are added to minDeposit
	for _, minDepositFloorCoin := range minDepositFloor {
		if _, seen := minDepositFloorDenomsSeen[minDepositFloorCoin.Denom]; !seen {
			minDepositCoinAmt, maxMinDepositCoinAmt := calculateMinDepositAmount(
				minDepositFloorCoin,
				minDepositFloorCoin.Amount,
				alpha,
				maxLastMinDeposit,
			)
			newMinDeposit = append(newMinDeposit, sdk.NewCoin(minDepositFloorCoin.Denom, minDepositCoinAmt))
			maxMinDeposit = append(maxMinDeposit, sdk.NewCoin(minDepositFloorCoin.Denom, maxMinDepositCoinAmt))
		}
	}

	return newMinDeposit, maxMinDeposit
}

func calculateMinDepositAmount(
	lastMinDepositCoin sdk.Coin,
	minDepositFloorCoinAmt math.Int,
	alpha sdk.Dec,
	maxLastMinDeposit sdk.Coins,
) (minDepositCoinAmt, maxMinDepositCoinAmt math.Int) {
	if alpha.IsPositive() {
		// lastMinDeposit * (1 + alpha)
		percChange := math.LegacyOneDec().Add(alpha)
		minDepositCoinAmt = percChange.MulInt(lastMinDepositCoin.Amount).TruncateInt()
		maxMinDepositCoinAmt = minDepositCoinAmt
	} else {
		// Alpha here is negative, indicating a decrease.
		// Update for decreases is `maxLastMinDeposit - (maxLastMinDeposit - lastMinDeposit) * (1 - alpha)`,
		percChange := math.LegacyOneDec().Sub(alpha)
		maxLastMinDepositCoinAmt := maxLastMinDeposit.AmountOf(lastMinDepositCoin.Denom)
		gap := maxLastMinDepositCoinAmt.Sub(lastMinDepositCoin.Amount)
		if !gap.IsPositive() {
			// a negative gap should actually never happen, but we cover it here anyway.
			// If instead the gap is zero, it indicates the first decrease. We should
			// set a minimum gap to start from or decreases will never actually start.
			// we arbitrarily choose to start from lastMinDepositAmt * alpha^2
			gap = alpha.Power(2).MulInt(lastMinDepositCoin.Amount).TruncateInt()
		}
		minDepositCoinAmt = maxLastMinDepositCoinAmt.Sub(percChange.MulInt(gap).TruncateInt())
		maxMinDepositCoinAmt = maxLastMinDepositCoinAmt
	}
	if minDepositCoinAmt.LT(minDepositFloorCoinAmt) {
		return minDepositFloorCoinAmt, maxMinDepositCoinAmt
	}
	return minDepositCoinAmt, maxMinDepositCoinAmt
}
