package e2e

import (
	"fmt"
	"strconv"
	"time"

	feemarkettypes "github.com/atomone-hub/atomone/x/feemarket/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
		state := s.queryFeemarketState(chainEndpoint)
		params := s.queryFeemarketParams(chainEndpoint)
		s.Require().Equal("0.000010000000000000", state.State.BaseGasPrice.String())
		s.Require().Equal(uint64(len(state.State.Window)), params.Params.Window)
		s.Require().True(state.State.Index >= 0)
		s.Require().True(int(state.State.Index) < len(state.State.Window))
		s.Require().True(state.State.LearningRate.IsPositive())
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

func (s *IntegrationTestSuite) testFeemarketGasPriceChange() {
	s.Run("gas price change", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		params := s.queryFeemarketParams(chainEndpoint)
		// define one sender
		sender, _ := c.genesisAccounts[0].keyInfo.GetAddress()
		// Initialize array of recipients account
		var destAccounts []string
		tokenAmount := sdk.NewInt64Coin(uatoneDenom, 100_000) // 0.1atone
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

		// wait until the current LR is less than max LR
		var StateBeforeMultisendTx feemarkettypes.StateResponse
		s.Require().Eventually(
			func() bool {
				StateBeforeMultisendTx = s.queryFeemarketState(chainEndpoint)
				return StateBeforeMultisendTx.State.LearningRate.LT(params.Params.MaxLearningRate)
			},
			10*time.Second,
			time.Second,
		)

		txHeight := s.execBankMultiSend(s.chainA, valIdx, sender.String(),
			destAccountsMultisend, tokenAmount.String(), false)
		StateAfterMultisendTx := s.queryFeemarketStateAtHeight(chainEndpoint, strconv.Itoa(txHeight))

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
}
