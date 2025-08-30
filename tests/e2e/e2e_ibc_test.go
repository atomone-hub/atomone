package e2e

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"

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
	// require the recipient account receives the IBC tokens (IBC packets ACKd)
	address, _ := s.chainA.validators[0].keyInfo.GetAddress()
	sender := address.String()

	address, _ = s.chainB.validators[0].keyInfo.GetAddress()
	recipient := address.String()

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))

	// Determine ibc denom trace which is "ibc/"+HEX(SHA256({port}/{channel}/{denom}))
	bz := sha256.Sum256([]byte("transfer/channel-0/" + uatoneDenom))
	ibcDenom := fmt.Sprintf("ibc/%X", bz)

	tokenChainA := sdk.NewInt64Coin(uatoneDenom, 1_000_000_000) // 1,000 atone
	tokenChainB := sdk.NewCoin(ibcDenom, tokenChainA.Amount)    // 1,000 ibc/{port}/{channel}/{denom}

	s.Run("send_to_chainB", func() {
		address, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := address.String()

		address, _ = s.chainB.validators[0].keyInfo.GetAddress()
		recipient := address.String()

		// Get balance before test
		beforeChainABalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeChainBBalance := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)

		s.sendIBC(s.chainA, 0, sender, recipient, tokenChainA.String(), "")

		if s.hermesResource != nil {
			// Test is using hermes relayer, call the required function
			pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, transferChannel)
			s.Require().True(pass)
		}
		s.Require().EventuallyWithT(
			func(c *assert.CollectT) {
				// Assert new balance of chainA to be equal to beforeBalance-tokenChainA
				newChainABalance := s.queryBalance(chainAAPIEndpoint, sender, tokenChainA.Denom)
				assert.Equal(c,
					beforeChainABalance.Sub(tokenChainA).String(), newChainABalance.String(),
					"wrong chainA balance: before(%s) - transfered(%s) != %s", beforeChainABalance, tokenChainA, newChainABalance,
				)
				// Assert new balance of chainB to be equal to beforeBalance+tokenChainB
				newChainBBalance := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)
				assert.Equal(c,
					beforeChainBBalance.Add(tokenChainB).String(), newChainBBalance.String(),
					"wrong chainB balance: before(%s) + transfered(%s) != %s", beforeChainBBalance, tokenChainB, newChainBBalance,
				)
			},
			time.Minute,
			time.Second,
		)
		beforeChainBBalance = s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)
	})

	// NOTE: following test depends on previous test
	s.Run("send_back_to_chainA", func() {
		// Get balance before test
		beforeChainABalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeChainBBalance := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)

		s.sendIBC(s.chainB, 0, recipient, sender, tokenChainB.String(), "")

		if s.hermesResource != nil {
			// Test is using hermes relayer, call the required function
			pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, transferChannel)
			s.Require().True(pass)
		}
		s.Require().EventuallyWithT(
			func(c *assert.CollectT) {
				// Assert new balance of chainA to be equal to beforeBalance+tokenChainA
				newChainABalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
				assert.Equal(c,
					beforeChainABalance.Add(tokenChainA).String(), newChainABalance.String(),
					"wrong chainA balance: before(%s) + transfered(%s) != %s", beforeChainABalance, tokenChainA, newChainABalance,
				)
				// Assert new balance of chainB to be equal to beforeBalance-tokenChainB
				newChainBBalance := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)
				assert.Equal(c,
					beforeChainBBalance.Sub(tokenChainB).String(), newChainBBalance.String(),
					"wrong chainB balancebefore(%s) + transfered(%s) != %s", beforeChainBBalance, tokenChainB, newChainBBalance,
				)
			},
			time.Minute,
			time.Second,
		)
	})
}
