package e2e

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) testMintPhoton() {
	s.Run("mint photon", func() {
		var (
			c             = s.chainA
			valIdx        = 0
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)
		alice, _ := c.genesisAccounts[1].keyInfo.GetAddress()
		beforeBalance, err := queryAtomOneAllBalances(chainEndpoint, alice.String())
		s.Require().NoError(err)
		s.Require().True(beforeBalance.AmountOf(uphotonDenom).IsZero(), "expected 0 balance of photon")
		beforeSupply := s.queryBankSupply(chainEndpoint)
		s.Require().True(beforeSupply.AmountOf(uphotonDenom).IsZero(), "expected 0 balance of photon in total supply")
		conversionRate := s.queryPhotonConversionRate(chainEndpoint)
		s.Require().Positive(conversionRate.MustFloat64())

		burnedAtoneAmt := sdk.NewInt64Coin(uatoneDenom, 1_000_000)
		resp := s.execPhotonMint(s.chainA, valIdx, alice.String(), burnedAtoneAmt.String(), standardFees.String())

		expectedBalance := beforeBalance.
			Sub(burnedAtoneAmt). // remove burned atones
			Add(resp.Minted).    // add minted photons
			Sub(standardFees)    // remove fees
		afterBalance, err := queryAtomOneAllBalances(chainEndpoint, alice.String())
		s.Require().NoError(err)
		s.Require().Equal(expectedBalance.String(), afterBalance.String())
		var (
			expectedUphotonSupply = resp.Minted
			afterSupply           = s.queryBankSupply(chainEndpoint)
			_, afterUphotonSupply = afterSupply.Find(uphotonDenom)
		)
		s.Require().Equal(expectedUphotonSupply.String(), afterUphotonSupply.String())
		// For atone supply assertion we must take into account inflation and so
		// we except the final supply to be greater or equal than the initial
		// supply - the burned atones.
		var (
			_, beforeUatoneSupply = beforeSupply.Find(uatoneDenom)
			_, afterUatoneSupply  = afterSupply.Find(uatoneDenom)
		)
		fmt.Println("BEFORE", beforeUatoneSupply.Sub(burnedAtoneAmt))
		fmt.Println("AFTER ", afterUatoneSupply)
		s.Require().True(afterUatoneSupply.IsGTE(beforeUatoneSupply.Sub(burnedAtoneAmt)),
			"after supply should be >= than initial %s supply", uatoneDenom)
	})
}
