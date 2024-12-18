package e2e

import (
	"context"
	"encoding/json"
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

func (s *IntegrationTestSuite) hermesTransfer(configPath, srcChainID, dstChainID, srcChannelID, denom string, sendAmt, timeOutOffset, numMsg int) (success bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	hermesCmd := []string{
		hermesBinary,
		"--json",
		fmt.Sprintf("--config=%s", configPath),
		"tx",
		"ft-transfer",
		fmt.Sprintf("--dst-chain=%s", dstChainID),
		fmt.Sprintf("--src-chain=%s", srcChainID),
		fmt.Sprintf("--src-channel=%s", srcChannelID),
		fmt.Sprintf("--src-port=%s", "transfer"),
		fmt.Sprintf("--amount=%v", sendAmt),
		fmt.Sprintf("--denom=%s", denom),
		fmt.Sprintf("--timeout-height-offset=%v", timeOutOffset),
		fmt.Sprintf("--number-msgs=%v", numMsg),
	}

	if _, err := s.executeHermesCommand(ctx, hermesCmd); err != nil {
		return false
	}

	return true
}

func (s *IntegrationTestSuite) hermesClearPacket(configPath, chainID, channelID string) (success bool) { //nolint:unparam
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	hermesCmd := []string{
		hermesBinary,
		"--json",
		fmt.Sprintf("--config=%s", configPath),
		"clear",
		"packets",
		fmt.Sprintf("--chain=%s", chainID),
		fmt.Sprintf("--channel=%s", channelID),
		fmt.Sprintf("--port=%s", "transfer"),
	}

	if _, err := s.executeHermesCommand(ctx, hermesCmd); err != nil {
		return false
	}

	return true
}

type RelayerPacketsOutput struct {
	Result struct {
		Dst struct {
			UnreceivedPackets []uint64 `json:"unreceived_packets"`
		} `json:"dst"`
		Src struct {
			UnreceivedPackets []uint64 `json:"unreceived_packets"`
		} `json:"src"`
	} `json:"result"`
	Status string `json:"status"`
}

func (s *IntegrationTestSuite) hermesPendingPackets(chainID, channelID string) (pendingPackets bool) { //nolint:unparam
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	hermesCmd := []string{
		hermesBinary,
		"--json",
		"query",
		"packet",
		"pending",
		fmt.Sprintf("--chain=%s", chainID),
		fmt.Sprintf("--channel=%s", channelID),
		fmt.Sprintf("--port=%s", "transfer"),
	}

	stdout, err := s.executeHermesCommand(ctx, hermesCmd)
	s.Require().NoError(err)

	var relayerPacketsOutput RelayerPacketsOutput
	err = json.Unmarshal(stdout, &relayerPacketsOutput)
	s.Require().NoError(err)

	// Check if "unreceived_packets" exists in "src"
	return len(relayerPacketsOutput.Result.Src.UnreceivedPackets) != 0
}

func (s *IntegrationTestSuite) queryRelayerWalletsBalances() (sdk.Coin, sdk.Coin) {
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	acctAddrChainA, _ := s.chainA.genesisAccounts[relayerAccountIndexHermes].keyInfo.GetAddress()
	scrRelayerBalance, err := getSpecificBalance(
		chainAAPIEndpoint,
		acctAddrChainA.String(),
		uatoneDenom)
	s.Require().NoError(err)

	chainBAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))
	acctAddrChainB, _ := s.chainB.genesisAccounts[relayerAccountIndexHermes].keyInfo.GetAddress()
	dstRelayerBalance, err := getSpecificBalance(
		chainBAPIEndpoint,
		acctAddrChainB.String(),
		uatoneDenom)
	s.Require().NoError(err)

	return scrRelayerBalance, dstRelayerBalance
}

func (s *IntegrationTestSuite) createConnection() {
	s.T().Logf("connecting %s and %s chains via IBC", s.chainA.id, s.chainB.id)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	hermesCmd := []string{
		hermesBinary,
		"--json",
		"create",
		"connection",
		"--a-chain",
		s.chainA.id,
		"--b-chain",
		s.chainB.id,
	}

	_, err := s.executeHermesCommand(ctx, hermesCmd)
	s.Require().NoError(err, "failed to connect chains: %s", err)

	s.T().Logf("connected %s and %s chains via IBC", s.chainA.id, s.chainB.id)
}

func (s *IntegrationTestSuite) createChannel() {
	s.T().Logf("creating IBC transfer channel created between chains %s and %s", s.chainA.id, s.chainB.id)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	hermesCmd := []string{
		hermesBinary,
		"--json",
		"create",
		"channel",
		"--a-chain", s.chainA.id,
		"--a-connection", "connection-0",
		"--a-port", "transfer",
		"--b-port", "transfer",
		"--channel-version", "ics20-1",
		"--order", "unordered",
	}

	_, err := s.executeHermesCommand(ctx, hermesCmd)
	s.Require().NoError(err, "failed to create IBC transfer channel between chains: %s", err)

	s.T().Logf("IBC transfer channel created between chains %s and %s", s.chainA.id, s.chainB.id)
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
				balances, err = queryAtomOneAllBalances(chainBAPIEndpoint, recipient)
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

		pass := s.hermesClearPacket(hermesConfigWithGasPrices, s.chainA.id, transferChannel)
		s.Require().True(pass)

		s.Require().Eventually(
			func() bool {
				balances, err = queryAtomOneAllBalances(chainBAPIEndpoint, recipient)
				s.Require().NoError(err)
				return balances.Len() != 0
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
