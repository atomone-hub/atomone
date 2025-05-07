package e2e

import (
	"fmt"

	"github.com/atomone-hub/atomone/x/feemarket/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// "github.com/cosmos/cosmos-sdk/types/tx"
	"time"

	"cosmossdk.io/math"
)

/*
Test Feemarket Queries:
- params
- state
- gas_price/{denom}
- gas_prices
*/
func (s *IntegrationTestSuite) testFeemarketQuery() {
	s.Run("feemarket test params", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		params := s.queryFeemarketParams(chainEndpoint)
		var maxBlockUtilization uint64 = 30000000
		var window_size uint64 = 8
		s.Require().Equal("0.025000000000000000", params.Params.Alpha.String())
		s.Require().Equal("0.950000000000000000", params.Params.Beta.String())
		s.Require().Equal("0.250000000000000000", params.Params.Gamma.String())
		s.Require().Equal("0.000010000000000000", params.Params.MinBaseGasPrice.String())
		s.Require().Equal("0.010000000000000000", params.Params.MinLearningRate.String())
		s.Require().Equal("0.500000000000000000", params.Params.MaxLearningRate.String())
		s.Require().Equal(maxBlockUtilization, params.Params.MaxBlockUtilization)
		s.Require().Equal(window_size, params.Params.Window)
		s.Require().Equal("uphoton", params.Params.FeeDenom)
		s.Require().True(params.Params.Enabled)
	})

	s.Run("feemarket test state", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		params := s.queryFeemarketState(chainEndpoint)
		s.Require().Equal("0.000010000000000000", params.State.BaseGasPrice.String())
		for i := range params.State.Window {
			s.Require().Equal(uint64(0), params.State.Window[i])
		}
		s.Require().True(params.State.Index >= 0)
		s.Require().True(int(params.State.Index) < len(params.State.Window))
		s.Require().True(params.State.LearningRate.IsPositive())
	})

	s.Run("feemarket test get price", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		atoneGasPrice := s.queryFeemarketGasPrice(chainEndpoint, "uatone")
		s.Require().Equal("uatone", atoneGasPrice.Price.Denom)
		s.Require().True(atoneGasPrice.Price.Amount.IsPositive())
		photonGasPrice := s.queryFeemarketGasPrice(chainEndpoint, "uphoton")
		s.Require().Equal("uphoton", photonGasPrice.Price.Denom)
		s.Require().True(photonGasPrice.Price.Amount.IsPositive())
	})

	s.Run("feemarket test get prices", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		gasPrices := s.queryFeemarketGasPrices(chainEndpoint)
		atoneAmount := gasPrices.Prices.AmountOf("uatone")
		photonAmount := gasPrices.Prices.AmountOf("uphoton")
		s.Require().True(atoneAmount.IsPositive())
		s.Require().True(photonAmount.IsPositive())
	})
}

func toLegacyDec(num uint64) math.LegacyDec {
	return math.LegacyNewDecFromInt(math.NewIntFromUint64(num))
}

/*
Test Gas Price change
*/

func (s *IntegrationTestSuite) testFeemarketGasPriceChange() {
	s.Run("gas price change lite", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		// define one sender and two recipient accounts
		sender, _ := c.genesisAccounts[0].keyInfo.GetAddress()
		// Initialize array of recipients account
		var destAccounts []string
		tokenAmount = sdk.NewInt64Coin(uatoneDenom, 100_000) // 0.1atone
		for i := range len(c.genesisAccounts) {
			address, _ := c.genesisAccounts[i].keyInfo.GetAddress()
			destAccounts = append(destAccounts, address.String())
		}

		var destAccountsMultisend []string
		for range 600 {
			for j := range len(destAccounts) {
				destAccountsMultisend = append(destAccountsMultisend, destAccounts[j])
			}
		}

		StateBeforeMultisendTx := s.queryFeemarketState(chainEndpoint)
		s.execBankMultiSend(s.chainA, valIdx, sender.String(),
			destAccountsMultisend, tokenAmount.String(), false)
		StateAfterMultisendTx := s.queryFeemarketState(chainEndpoint)

		oldFee := StateBeforeMultisendTx.State.BaseGasPrice
		newFee := StateAfterMultisendTx.State.BaseGasPrice

		s.Require().True(newFee.GT(oldFee),
			"Expected new Fee (%s) higher than old fee (%s)",
			newFee, oldFee)
		oldLearningRate := StateBeforeMultisendTx.State.LearningRate
		newLearningRate := StateAfterMultisendTx.State.LearningRate

		s.Require().True(newLearningRate.GT(oldLearningRate),
			"Expected newLearningRate (%s) higher than currentLearningRate (%s)",
			newLearningRate, oldLearningRate)
	})

	s.Run("gas price change full", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		// define one sender and two recipient accounts
		sender, _ := c.genesisAccounts[0].keyInfo.GetAddress()

		var beforeAccountBalances,
			afterAccountBalances []sdk.Coin

		feemarketParams := s.queryFeemarketParams(chainEndpoint)

		// Get balances of sender and recipient accounts
		s.Require().Eventually(
			func() bool {
				for i := range len(c.genesisAccounts) {
					accountID := i
					address, _ := c.genesisAccounts[accountID].keyInfo.GetAddress()
					addressUAtoneBalance, err := getSpecificBalance(chainEndpoint, address.String(), uatoneDenom)
					beforeAccountBalances = append(beforeAccountBalances, addressUAtoneBalance)
					s.Require().NoError(err)
				}

				balanceValid := beforeAccountBalances[0].IsValid()

				for i := range len(c.genesisAccounts) {
					balanceValid = balanceValid && beforeAccountBalances[i].IsValid()
				}
				return balanceValid
			},
			10*time.Second,
			time.Second,
		)

		// Initialize array of recipients account
		var destAccounts []string
		tokenAmount = sdk.NewInt64Coin(uatoneDenom, 100_000) // 0.1atone
		for i := range len(c.genesisAccounts) {
			address, _ := c.genesisAccounts[i].keyInfo.GetAddress()
			destAccounts = append(destAccounts, address.String())
		}

		var destAccountsMultisend []string

		for range 600 {
			for j := range len(destAccounts) {
				destAccountsMultisend = append(destAccountsMultisend, destAccounts[j])
			}
		}
		var StateBeforeMultisendTx,
			StateAfterMultisendTx types.StateResponse

		// Makes sure we obtain the state of two consecutive blocks
		// One before the multisend tx and one after
		s.Require().Eventually(
			func() bool {
				StateBeforeMultisendTx = s.queryFeemarketState(chainEndpoint)
				s.execBankMultiSend(s.chainA, valIdx, sender.String(),
					destAccountsMultisend, tokenAmount.String(), false)
				StateAfterMultisendTx = s.queryFeemarketState(chainEndpoint)
				if StateAfterMultisendTx.State.Index == StateBeforeMultisendTx.State.Index+1 {
					return true
				} else {
					return false
				}
			},
			40*time.Second,
			time.Second,
		)

		// Get balances after mutlisend tx
		s.Require().Eventually(
			func() bool {
				for i := range len(c.genesisAccounts) {
					accountID := i
					address, _ := c.genesisAccounts[accountID].keyInfo.GetAddress()
					addressUAtoneBalance, err := getSpecificBalance(chainEndpoint, address.String(), uatoneDenom)
					afterAccountBalances = append(afterAccountBalances, addressUAtoneBalance)
					s.Require().NoError(err)
				}

				balanceValid := afterAccountBalances[0].IsValid()
				for i := range len(c.genesisAccounts) {
					balanceValid = balanceValid && afterAccountBalances[i].IsValid()
				}

				balancesAreDifferent := false
				for i := range len(c.genesisAccounts) {
					balancesAreDifferent = balancesAreDifferent ||
						!afterAccountBalances[i].Amount.Equal(beforeAccountBalances[i].Amount)
				}

				return balanceValid && balancesAreDifferent
			},
			10*time.Second,
			2*time.Second,
		)

		// Compute expected LearningRate and GasPrice
		targetBlockSize := toLegacyDec(feemarketParams.Params.MaxBlockUtilization / 2)
		currentBlockSize := toLegacyDec(StateAfterMultisendTx.State.Window[StateBeforeMultisendTx.State.Index])

		maxBlockUtilization := toLegacyDec(feemarketParams.Params.MaxBlockUtilization)
		windowSize := toLegacyDec(feemarketParams.Params.Window)
		one := toLegacyDec(1)

		windowGasSum := uint64(0)
		for i := uint64(0); i < feemarketParams.Params.Window; i++ {
			windowGasSum += StateAfterMultisendTx.State.Window[i]
		}
		// We know that there has been only 1 multisend transaction, so we know that the window
		// has zeroes at all indexes exept at the current index where it has a value of currentBlockSize
		blockConsumption := toLegacyDec(windowGasSum).Quo(windowSize.Mul(maxBlockUtilization))

		alpha := feemarketParams.Params.Alpha
		beta := feemarketParams.Params.Beta
		gamma := feemarketParams.Params.Gamma
		currentLearningRate := StateBeforeMultisendTx.State.LearningRate
		newLearningRate := StateAfterMultisendTx.State.LearningRate
		expectedLearningRate := toLegacyDec(0)
		maxLearningRate := feemarketParams.Params.MaxLearningRate
		minLearningRate := feemarketParams.Params.MinLearningRate
		currentBaseFee := StateBeforeMultisendTx.State.BaseGasPrice
		newBaseFee := StateAfterMultisendTx.State.BaseGasPrice
		minBaseFee := feemarketParams.Params.MinBaseGasPrice

		if blockConsumption.LTE(gamma) || blockConsumption.GTE(one.Sub(gamma)) {
			expectedLearningRate = alpha.Add(currentLearningRate)
			if expectedLearningRate.GT(maxLearningRate) {
				expectedLearningRate = maxLearningRate
			}
		} else {
			expectedLearningRate = beta.Mul(currentLearningRate)
			if expectedLearningRate.LT(minLearningRate) {
				expectedLearningRate = minLearningRate
			}
		}
		expectedNewBaseFee := currentBaseFee.Mul(
			one.Add(expectedLearningRate.Mul((currentBlockSize.Sub(targetBlockSize)).Quo(targetBlockSize))))

		if expectedNewBaseFee.LT(minBaseFee) {
			expectedNewBaseFee = minBaseFee
		}

		s.Require().True(expectedLearningRate.Equal(newLearningRate),
			"Expected Learning Rate: %s Actual: %s", expectedLearningRate, newLearningRate)
		s.Require().True(expectedNewBaseFee.Equal(newBaseFee),
			"Expected Base Fee: %s Actual: %s", expectedNewBaseFee, newBaseFee)
	})

}
