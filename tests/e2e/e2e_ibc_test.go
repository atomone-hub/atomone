package e2e

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
)

//nolint:unparam
func (s *IntegrationTestSuite) transferIBC(c *chain, valIdx int, channelID, sender, recipient, token, note string) {
	s.T().Logf("transfering %s from %s (%s) to %s (%s) using %s", token, s.chainA.id, sender, s.chainB.id, recipient, channelID)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ibcCmd := []string{
		atomonedBinary,
		txCommand,
		"ibc-transfer",
		"transfer",
		"transfer",
		channelID,
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
	s.executeAtomoneTxCommand(ctx, c, ibcCmd, valIdx, s.defaultExecValidation(c, valIdx, nil))
	s.T().Log("successfully transfered IBC tokens")
}

func (s *IntegrationTestSuite) transferIBCv2(c *chain, clientID, sender, recipient string, token sdk.Coin) {
	s.T().Logf("transfering v2 %s from %s (%s) to %s (%s) using %s", token, s.chainA.id, sender, s.chainB.id, recipient, clientID)
	// NOTE: There is currently no CLI command for the transfer app in IBCv2 so
	// we have to forge everything by hand.
	packetData := transfertypes.NewFungibleTokenPacketData(
		token.Denom, token.Amount.String(), sender, recipient, "",
	)
	bz := s.cdc.MustMarshal(&packetData)
	payload := channeltypesv2.NewPayload(
		transfertypes.PortID, transfertypes.PortID, transfertypes.V1,
		transfertypes.EncodingProtobuf, bz,
	)
	timeoutTimestamp := uint64(time.Now().Add(time.Hour).Unix())
	msg := channeltypesv2.NewMsgSendPacket(
		clientID, timeoutTimestamp, sender, payload,
	)
	s.signAndBroadcastMsg(c, c.validators[0].keyInfo, msg)
	s.T().Log("successfully transfered IBCv2 tokens")
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

func (s *IntegrationTestSuite) testIBCTokenTransfer(channelIdA, channelIdB string) {
	s.Run("transfer_to_chainB", func() {
		address, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := address.String()

		address, _ = s.chainB.validators[0].keyInfo.GetAddress()
		recipient := address.String()

		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))

		// Determine ibc denom which is "ibc/"+HEX(SHA256({port}/{channel}/{denom}))
		ibcTrace := fmt.Sprintf("transfer/%s/%s", channelIdA, uatoneDenom)
		ibcDenom := fmt.Sprintf("ibc/%X", sha256.Sum256([]byte(ibcTrace)))

		tokenChainA := sdk.NewInt64Coin(uatoneDenom, 1_000_000_000) // 1,000 atone
		tokenChainB := sdk.NewCoin(ibcDenom, tokenChainA.Amount)    // 1,000 ibc/{port}/{channel}/{denom}

		// Get balance before test
		beforeBalanceChainA := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeBalanceChainB := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)

		s.transferIBC(s.chainA, 0, channelIdA, sender, recipient, tokenChainA.String(), "")

		if s.hermesResource != nil {
			// Test is using hermes relayer, call the required function
			pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, channelIdA)
			s.Require().True(pass)
		}
		// Assert new balance of chainA to be equal to beforeBalance-tokenChainA
		s.assertCoinBalance(s.chainA, sender, beforeBalanceChainA.Sub(tokenChainA))
		// Assert new balance of chainB to be equal to beforeBalance+tokenChainB
		s.assertCoinBalance(s.chainB, recipient, beforeBalanceChainB.Add(tokenChainB))

		// Now try to send back the tokens to chainA (unwind)
		s.Run("transfer_back_to_chainA", func() {
			// Get balance before test
			beforeBalanceChainA := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
			beforeBalanceChainB := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)

			s.transferIBC(s.chainB, 0, channelIdB, recipient, sender, tokenChainB.String(), "")

			if s.hermesResource != nil {
				// Test is using hermes relayer, call the required function
				pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, channelIdB)
				s.Require().True(pass)
			}
			// Assert new balance of chainA to be equal to beforeBalance+tokenChainA
			s.assertCoinBalance(s.chainA, sender, beforeBalanceChainA.Add(tokenChainA))
			// Assert new balance of chainB to be equal to beforeBalance-tokenChainB
			s.assertCoinBalance(s.chainB, recipient, beforeBalanceChainB.Sub(tokenChainB))
		})
	})
}

func (s *IntegrationTestSuite) testIBCv2TokenTransfer(clientIdA, clientIdB string) {
	defer s.saveTsRelayerLogs() // TODO remove when flakyness is fixed
	s.Run("transferv2_to_chainB", func() {
		address, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := address.String()
		address, _ = s.chainB.validators[0].keyInfo.GetAddress()
		recipient := address.String()

		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))

		// Determine ibc denom which is "ibc/"+HEX(SHA256({port}/{channel}/{denom}))
		ibcTrace := fmt.Sprintf("transfer/%s/%s", clientIdA, uatoneDenom)
		ibcDenom := fmt.Sprintf("ibc/%X", sha256.Sum256([]byte(ibcTrace)))

		tokenChainA := sdk.NewInt64Coin(uatoneDenom, 1_000_000_000) // 1,000 atone
		tokenChainB := sdk.NewCoin(ibcDenom, tokenChainA.Amount)    // 1,000 ibc/{port}/{channel}/{denom}

		// Get balance before test
		beforeBalanceChainA := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeBalanceChainB := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)

		s.transferIBCv2(s.chainA, clientIdA, sender, recipient, tokenChainA)

		// Assert new balance of chainA to be equal to beforeBalance-tokenChainA
		s.assertCoinBalance(s.chainA, sender, beforeBalanceChainA.Sub(tokenChainA))
		// Assert new balance of chainB to be equal to beforeBalance+tokenChainB
		s.assertCoinBalance(s.chainB, recipient, beforeBalanceChainB.Add(tokenChainB))

		// Now try to send back the tokens to chainA (unwind)
		s.Run("transferv2_back_to_chainA", func() {
			// Get balance before test
			beforeBalanceChainA := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
			beforeBalanceChainB := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)
			// Unwinding in IBCv2 requires to us the trace in plain text as denom
			tokenTransfer := sdk.NewCoin(ibcTrace, tokenChainB.Amount)

			s.transferIBCv2(s.chainB, clientIdB, recipient, sender, tokenTransfer)

			// Assert new balance of chainA to be equal to beforeBalance+tokenChainA
			s.assertCoinBalance(s.chainA, sender, beforeBalanceChainA.Add(tokenChainA))
			// Assert new balance of chainB to be equal to beforeBalance-tokenChainB
			s.assertCoinBalance(s.chainB, recipient, beforeBalanceChainB.Sub(tokenChainB))
		})
	})
}

func (s *IntegrationTestSuite) assertCoinBalance(c *chain, addr string, expected sdk.Coin) {
	s.T().Logf("asserting balance of %s has %s ", addr, expected)
	endpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	s.Require().EventuallyWithT(
		func(c *assert.CollectT) {
			// TODO use s.queryBalance when flakyness is fixed
			balances, err := s.queryAllBalances(endpoint, addr)
			if err != nil {
				panic(err)
			}
			// current := s.queryBalance(endpoint, addr, expected.Denom)
			ok, current := balances.Find(expected.Denom)
			if !ok {
				current = sdk.NewCoin(expected.Denom, math.ZeroInt())
			}
			assert.Equal(c,
				expected.String(), current.String(),
				"wrong coin balance for %s: %s", addr, balances,
			)
		},
		time.Minute,
		time.Second,
	)
}
