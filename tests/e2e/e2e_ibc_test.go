package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

//nolint:unparam
func (s *IntegrationTestSuite) sendIBC(c *chain, valIdx int, sender, recipient, token, note string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ibcCmd := []string{
		atomonedBinary,
		txCommand,
		"ibc-transfer",
		"transfer",
		"transfer",
		"channel-0",
		recipient,
		token,
		fmt.Sprintf("--from=%s", sender),
		fmt.Sprintf("--%s=%s", flags.FlagFees, standardFees.String()),
		fmt.Sprintf("--%s=%s", flags.FlagChainID, c.id),
		// fmt.Sprintf("--%s=%s", flags.FlagNote, note),
		fmt.Sprintf("--memo=%s", note),
		"--keyring-backend=test",
		"--broadcast-mode=sync",
		"--output=json",
		"-y",
	}
	s.T().Logf("sending %s from %s (%s) to %s (%s) with memo %s", token, s.chainA.id, sender, s.chainB.id, recipient, note)
	s.executeAtomoneTxCommand(ctx, c, ibcCmd, valIdx, s.defaultExecValidation(c, valIdx, nil))
	s.T().Log("successfully sent IBC tokens")
}

func (s *IntegrationTestSuite) queryRelayerWalletsBalances() (sdk.Coins, sdk.Coins) {
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	acctAddrChainA, _ := s.chainA.genesisAccounts[relayerAccountIndex].keyInfo.GetAddress()
	scrRelayerBalance, err := s.queryAllBalances(
		chainAAPIEndpoint,
		acctAddrChainA.String())
	s.Require().NoError(err)

	chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))
	acctAddrChainB, _ := s.chainB.genesisAccounts[relayerAccountIndex].keyInfo.GetAddress()
	dstRelayerBalance, err := s.queryAllBalances(
		chainBAPIEndpoint,
		acctAddrChainB.String())
	s.Require().NoError(err)

	return scrRelayerBalance, dstRelayerBalance
}

func (s *IntegrationTestSuite) testIBCTokenTransfer() {
	s.Run("send_uatom_to_chainB", func() {
		// require the recipient account receives the IBC tokens (IBC packets ACKd)
		var (
			balances      sdk.Coins
			err           error
			beforeBalance int64
			ibcStakeDenom string
		)

		address, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := address.String()

		address, _ = s.chainB.validators[0].keyInfo.GetAddress()
		recipient := address.String()

		chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))

		s.Require().Eventually(
			func() bool {
				balances, err = s.queryAllBalances(chainBAPIEndpoint, recipient)
				s.Require().NoError(err)
				return balances.Len() != 0
			},
			time.Minute,
			5*time.Second,
		)
		for _, c := range balances {
			if strings.Contains(c.Denom, "ibc/") {
				beforeBalance = c.Amount.Int64()
				break
			}
		}

		s.sendIBC(s.chainA, 0, sender, recipient, tokenAmount.String(), "")

		if s.hermesResource != nil {
			// Test is using hermes relayer, call the required function
			pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, transferChannel)
			s.Require().True(pass)
		}

		s.Require().Eventually(
			func() bool {
				balances, err = s.queryAllBalances(chainBAPIEndpoint, recipient)
				s.Require().NoError(err)
				return balances.Len() > 2 // wait for atone, photon and the ibc denom
			},
			time.Minute,
			5*time.Second,
		)
		for _, c := range balances {
			if strings.Contains(c.Denom, "ibc/") {
				ibcStakeDenom = c.Denom
				s.Require().Equal(tokenAmount.Amount.Int64()+beforeBalance, c.Amount.Int64())
				break
			}
		}

		s.Require().NotEmpty(ibcStakeDenom)
	})
}
