package e2e

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
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

func (s *IntegrationTestSuite) transferIBCv2(c *chain, clientID, sender,
	recipient string, senderKeying keyring.Keyring, token sdk.Coin,
) {
	s.T().Logf("transfering v2 %s from %s (%s) to %s (%s) using %s", token, s.chainA.id, sender, s.chainB.id, recipient, clientID)
	// NOTE(tb): There is currently no CLI command for the transfer app in IBCv2
	// We have to forge everything by hand.
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
	endpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	acc, err := s.queryAccount(endpoint, sender)
	s.Require().NoError(err)

	// TODO externalize signMsg so it can be used by multiple account type?
	tx, err := s.chainA.validators[0].signMsg(acc.GetAccountNumber(), acc.GetSequence(), msg)
	s.Require().NoError(err)
	bz, err = tx.Marshal()
	s.Require().NoError(err)
	res, err := s.rpcClient(c, 0).BroadcastTxSync(context.Background(), bz)
	s.Require().NoError(err)
	err = s.waitAtomOneTx(endpoint, res.Hash.String(), nil)
	s.Require().NoError(err)
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

		// Determine ibc denom trace which is "ibc/"+HEX(SHA256({port}/{channel}/{denom}))
		bz := sha256.Sum256([]byte(fmt.Sprintf("transfer/%s/%s", channelIdA, uatoneDenom)))
		ibcDenom := fmt.Sprintf("ibc/%X", bz)

		tokenChainA := sdk.NewInt64Coin(uatoneDenom, 1_000_000_000) // 1,000 atone
		tokenChainB := sdk.NewCoin(ibcDenom, tokenChainA.Amount)    // 1,000 ibc/{port}/{channel}/{denom}

		// Get balance before test
		beforeChainABalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeChainBBalance := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)

		s.transferIBC(s.chainA, 0, channelIdA, sender, recipient, tokenChainA.String(), "")

		if s.hermesResource != nil {
			// Test is using hermes relayer, call the required function
			pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, channelIdA)
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

		// Now try to send back the tokens to chainA (unwind)
		s.Run("transfer_back_to_chainA", func() {
			// Get balance before test
			beforeChainABalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
			beforeChainBBalance := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)

			s.transferIBC(s.chainB, 0, channelIdB, recipient, sender, tokenChainB.String(), "")

			if s.hermesResource != nil {
				// Test is using hermes relayer, call the required function
				pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, channelIdB)
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
	})
}

func (s *IntegrationTestSuite) testIBCv2TokenTransfer(clientIdA, clientIdB string) {
	s.Run("transferv2_to_chainB", func() {
		address, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := address.String()
		senderName := s.chainA.validators[0].keyInfo.Name
		address, _ = s.chainB.validators[0].keyInfo.GetAddress()
		recipient := address.String()

		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))

		// Determine ibc denom trace which is "ibc/"+HEX(SHA256({port}/{channel}/{denom}))
		bz := sha256.Sum256([]byte(fmt.Sprintf("transfer/%s/%s", clientIdA, uatoneDenom)))
		ibcDenom := fmt.Sprintf("ibc/%X", bz)

		tokenChainA := sdk.NewInt64Coin(uatoneDenom, 1_000_000_000) // 1,000 atone
		tokenChainB := sdk.NewCoin(ibcDenom, tokenChainA.Amount)    // 1,000 ibc/{port}/{channel}/{denom}

		// Get balance before test
		beforeChainABalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeChainBBalance := s.queryBalance(chainBAPIEndpoint, recipient, ibcDenom)
		senderKeyring := s.chainA.validators[0].keyring()

		_ = senderName
		s.transferIBCv2(s.chainA, clientIdA, sender, recipient, senderKeyring, tokenChainA)

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
	})
}
