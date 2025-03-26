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
	percChange := math.LegacyOneDec().Add(alpha)

	if alpha.IsPositive() {
		minDepositCoinAmt = lastMinDepositCoin.Amount.ToLegacyDec().Mul(percChange).TruncateInt()
		maxMinDepositCoinAmt = minDepositCoinAmt
	} else {
		// Since update for decreases is `maxLastMinDeposit - (maxLastMinDeposit - lastMinDeposit) * (1 + alpha)`,
		// we can simplify this to `lastMinDeposit * (1 + alpha) - maxLastMinDeposit * alpha`.
		// Alpha here is negative so the term `- maxLastMinDeposit * alpha` is in fact positive.
		maxLastMinDepositCoinAmt := maxLastMinDeposit.AmountOf(lastMinDepositCoin.Denom)
		if maxLastMinDepositCoinAmt.IsZero() {
			panic("maxLastMinDeposit should have all the same denoms as lastMinDeposit")
		}
		minDepositCoinAmt = lastMinDepositCoin.Amount.ToLegacyDec().Mul(percChange).Sub(maxLastMinDepositCoinAmt.ToLegacyDec().Mul(alpha)).TruncateInt()
		maxMinDepositCoinAmt = maxLastMinDepositCoinAmt
	}
	if minDepositCoinAmt.LT(minDepositFloorCoinAmt) {
		return minDepositFloorCoinAmt, maxMinDepositCoinAmt
	}
	return minDepositCoinAmt, maxMinDepositCoinAmt
}
