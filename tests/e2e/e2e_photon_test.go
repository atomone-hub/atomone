package e2e

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) testMintPhoton() {
	subtest := func(fees sdk.Coin) func() {
		return func() {
			var (
				c             = s.chainA
				valIdx        = 0
				chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
			)
			alice, _ := c.genesisAccounts[1].keyInfo.GetAddress()
			beforeBalance, err := queryAtomOneAllBalances(chainEndpoint, alice.String())
			s.Require().NoError(err)
			beforeSupply := s.queryBankSupply(chainEndpoint)
			conversionRate := s.queryPhotonConversionRate(chainEndpoint)
			s.Require().Positive(conversionRate.MustFloat64())
			burnedAtoneAmt := sdk.NewInt64Coin(uatoneDenom, 1_000_000)

			resp := s.execPhotonMint(s.chainA, valIdx, alice.String(), burnedAtoneAmt.String(),
				false, withKeyValue(flagFees, fees),
			)

			expectedBalance := beforeBalance.
				Sub(burnedAtoneAmt). // remove burned atones
				Add(resp.Minted).    // add minted photons
				Sub(fees)            // remove fees

			afterBalance, err := queryAtomOneAllBalances(chainEndpoint, alice.String())
			s.Require().NoError(err)
			s.Require().Equal(expectedBalance.String(), afterBalance.String())
			var (
				_, beforeUphotonSupply = beforeSupply.Find(uphotonDenom)
				expectedUphotonSupply  = beforeUphotonSupply.Add(resp.Minted)
				afterSupply            = s.queryBankSupply(chainEndpoint)
				_, afterUphotonSupply  = afterSupply.Find(uphotonDenom)
			)
			s.Require().Equal(expectedUphotonSupply.String(), afterUphotonSupply.String())
			// For atone supply assertion we must take into account inflation and so
			// we except the final supply to be greater or equal than the initial
			// supply - the burned atones.
			var (
				_, beforeUatoneSupply = beforeSupply.Find(uatoneDenom)
				_, afterUatoneSupply  = afterSupply.Find(uatoneDenom)
			)
			s.Require().True(afterUatoneSupply.IsGTE(beforeUatoneSupply.Sub(burnedAtoneAmt)),
				"after supply should be >= than initial %s supply", uatoneDenom)
		}
	}
	s.Run("mint photon", subtest(standardFees))
	atoneFees := sdk.NewCoin(uatoneDenom, standardFees.Amount)
	s.Run("mint photon with atone fees", subtest(atoneFees))
	s.Run("mint photon wrong denom does not deduct fees", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		alice, _ := c.genesisAccounts[1].keyInfo.GetAddress()

		beforeBalance, err := queryAtomOneAllBalances(chainEndpoint, alice.String())
		s.Require().NoError(err)

		// Issue an incorrect transaction with wrong Denom
		_ = s.execPhotonMint(s.chainA, valIdx, alice.String(), "1000wrongDenom",
			true, withKeyValue(flagGas, "200000"))

		time.Sleep(1 * time.Second)
		afterBalance, err := queryAtomOneAllBalances(chainEndpoint, alice.String())

		var (
			_, beforeUphotonBalance = beforeBalance.Find(uphotonDenom)
			_, afterUphotonBalance  = afterBalance.Find(uphotonDenom)
			_, beforeUatoneBalance  = beforeBalance.Find(uatoneDenom)
			_, afterUatoneBalance   = afterBalance.Find(uatoneDenom)
		)

		s.Require().True(beforeUatoneBalance.IsEqual(afterUatoneBalance),
			"Fees should not be deducted for a malformed tx\n"+
				"Balance before tx: %s != Balance after tx: %s",
			beforeUatoneBalance, afterUatoneBalance)
		s.Require().True(beforeUphotonBalance.IsEqual(afterUphotonBalance),
			"Fees should not be deducted for a malformed tx\n"+
				"Balance before tx: %s != Balance after tx: %s",
			beforeUphotonBalance, afterUphotonBalance)
	})
}
