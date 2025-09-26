package e2e

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) testBankTokenTransfer() {
	s.Run("send tokens between accounts", func() {
		var (
			valIdx        = 0
			c             = s.chainA
			chainEndpoint = fmt.Sprintf("http://%s", s.valResources[c.id][valIdx].GetHostPort("1317/tcp"))
		)

		// define one sender and two recipient accounts
		alice, _ := c.genesisAccounts[1].keyInfo.GetAddress()
		bob, _ := c.genesisAccounts[2].keyInfo.GetAddress()
		charlie, _ := c.genesisAccounts[3].keyInfo.GetAddress()

		var beforeAliceUAtoneBalance,
			beforeBobUAtoneBalance,
			beforeCharlieUAtoneBalance,
			afterAliceUAtoneBalance,
			afterBobUAtoneBalance,
			afterCharlieUAtoneBalance sdk.Coin

		// get balances of sender and recipient accounts
		s.Require().Eventually(
			func() bool {
				beforeAliceUAtoneBalance = s.queryBalance(chainEndpoint, alice.String(), uatoneDenom)
				beforeBobUAtoneBalance = s.queryBalance(chainEndpoint, bob.String(), uatoneDenom)
				beforeCharlieUAtoneBalance = s.queryBalance(chainEndpoint, charlie.String(), uatoneDenom)

				return beforeAliceUAtoneBalance.IsValid() && beforeBobUAtoneBalance.IsValid() && beforeCharlieUAtoneBalance.IsValid()
			},
			10*time.Second,
			time.Second,
		)

		// alice sends tokens to bob
		s.execBankSend(s.chainA, valIdx, alice.String(), bob.String(), tokenAmount.String(), false)

		// check that the transfer was successful
		s.Require().Eventually(
			func() bool {
				afterAliceUAtoneBalance = s.queryBalance(chainEndpoint, alice.String(), uatoneDenom)
				afterBobUAtoneBalance = s.queryBalance(chainEndpoint, bob.String(), uatoneDenom)

				decremented := beforeAliceUAtoneBalance.Sub(tokenAmount).IsEqual(afterAliceUAtoneBalance)
				incremented := beforeBobUAtoneBalance.Add(tokenAmount).IsEqual(afterBobUAtoneBalance)

				return decremented && incremented
			},
			10*time.Second,
			time.Second,
		)

		// save the updated account balances of alice and bob
		beforeAliceUAtoneBalance, beforeBobUAtoneBalance = afterAliceUAtoneBalance, afterBobUAtoneBalance

		// alice sends tokens to bob and charlie, at once
		s.execBankMultiSend(s.chainA, valIdx, alice.String(),
			[]string{bob.String(), charlie.String()}, tokenAmount.String(), false)

		s.Require().Eventually(
			func() bool {
				afterAliceUAtoneBalance = s.queryBalance(chainEndpoint, alice.String(), uatoneDenom)
				afterBobUAtoneBalance = s.queryBalance(chainEndpoint, bob.String(), uatoneDenom)
				afterCharlieUAtoneBalance = s.queryBalance(chainEndpoint, charlie.String(), uatoneDenom)

				// assert alice's account gets decremented the amount of tokens twice
				decremented := beforeAliceUAtoneBalance.Sub(tokenAmount).Sub(tokenAmount).IsEqual(afterAliceUAtoneBalance)
				incremented := beforeBobUAtoneBalance.Add(tokenAmount).IsEqual(afterBobUAtoneBalance) &&
					beforeCharlieUAtoneBalance.Add(tokenAmount).IsEqual(afterCharlieUAtoneBalance)

				return decremented && incremented
			},
			10*time.Second,
			time.Second,
		)
	})

	s.Run("send tokens with atone fees", func() {
		var (
			valIdx = 0
			c      = s.chainA
		)
		alice, _ := c.genesisAccounts[1].keyInfo.GetAddress()
		bob, _ := c.genesisAccounts[2].keyInfo.GetAddress()

		// alice sends tokens to bob should fail because doesn't use photons for the fees.
		atoneFees := sdk.NewCoin(uatoneDenom, standardFees.Amount)
		s.execBankSend(s.chainA, valIdx, alice.String(), bob.String(),
			tokenAmount.String(), true, withKeyValue(flagFees, atoneFees))
	})
}
