package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// runIBCHermesRelayer bootstraps an IBC Hermes relayer by creating an IBC connection and
// a transfer channel between chainA and chainB.
// Returns the channel-ids of both side.
func (s *IntegrationTestSuite) runIBCHermesRelayer() (string, string) {
	s.T().Log("starting Hermes relayer container")

	tmpDir, err := os.MkdirTemp("", "atomone-e2e-testnet-hermes-")
	s.Require().NoError(err)
	s.tmpDirs = append(s.tmpDirs, tmpDir)

	rlyA := s.chainA.genesisAccounts[relayerAccountIndex]
	rlyB := s.chainB.genesisAccounts[relayerAccountIndex]

	hermesCfgPath := path.Join(tmpDir, "hermes")

	s.Require().NoError(os.MkdirAll(hermesCfgPath, 0o755))
	_, err = copyFile(
		filepath.Join("./scripts/", "hermes_bootstrap.sh"),
		filepath.Join(hermesCfgPath, "hermes_bootstrap.sh"),
	)
	s.Require().NoError(err)

	s.hermesResource, err = s.dkrPool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       fmt.Sprintf("%s-%s-relayer", s.chainA.id, s.chainB.id),
			Repository: "ghcr.io/cosmos/hermes-e2e",
			Tag:        "1.0.0",
			NetworkID:  s.dkrNet.Network.ID,
			Mounts: []string{
				fmt.Sprintf("%s/:/root/hermes", hermesCfgPath),
			},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"3031/tcp": {{HostIP: "", HostPort: "3031"}},
			},
			Env: []string{
				fmt.Sprintf("ATOMONE_A_E2E_CHAIN_ID=%s", s.chainA.id),
				fmt.Sprintf("ATOMONE_B_E2E_CHAIN_ID=%s", s.chainB.id),
				fmt.Sprintf("ATOMONE_A_E2E_RLY_MNEMONIC=%s", rlyA.mnemonic),
				fmt.Sprintf("ATOMONE_B_E2E_RLY_MNEMONIC=%s", rlyB.mnemonic),
				fmt.Sprintf("ATOMONE_A_E2E_VAL_HOST=%s", s.valResources[s.chainA.id][0].Container.Name[1:]),
				fmt.Sprintf("ATOMONE_B_E2E_VAL_HOST=%s", s.valResources[s.chainB.id][0].Container.Name[1:]),
			},
			User: "root",
			Entrypoint: []string{
				"sh",
				"-c",
				"chmod +x /root/hermes/hermes_bootstrap.sh && /root/hermes/hermes_bootstrap.sh && tail -f /dev/null",
			},
		},
		noRestart,
	)
	s.Require().NoError(err)

	s.T().Logf("started Hermes relayer container: %s", s.hermesResource.Container.ID)

	// XXX: Give time to both networks to start, otherwise we might see gRPC
	// transport errors.
	time.Sleep(10 * time.Second)

	// create the client, connection and channel between the two AtomOne chains
	connectionID := s.hermesCreateConnection()
	return s.hermesCreateChannel(connectionID)
}

func (s *IntegrationTestSuite) tearDownHermesRelayer() {
	if s.hermesResource != nil {
		s.T().Log("tearing down Hermes relayer...")
		s.Require().NoError(s.dkrPool.Purge(s.hermesResource))
		s.hermesResource = nil
	}
}

func (s *IntegrationTestSuite) executeHermesCommand(ctx context.Context, hermesCmd []string) ([]byte, error) {
	var outBuf bytes.Buffer
	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.hermesResource.Container.ID,
		User:         "root",
		Cmd:          hermesCmd,
	})
	s.Require().NoError(err)

	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &outBuf,
	})
	s.Require().NoError(err)

	// Check that the stdout output contains the expected status
	// and look for errors, e.g "insufficient fees"
	stdOut := []byte{}
	scanner := bufio.NewScanner(&outBuf)
	for scanner.Scan() {
		stdOut = scanner.Bytes()
		var out map[string]interface{}
		err = json.Unmarshal(stdOut, &out)
		s.Require().NoError(err)
		if err != nil {
			return nil, fmt.Errorf("hermes relayer command returned failed with error: %s", err)
		}
		// errors are catched by observing the logs level in the stderr output
		if lvl := out["level"]; lvl != nil && strings.ToLower(lvl.(string)) == "error" {
			fields := out["fields"].(map[string]any)
			errMsg := fields["message"]
			resp := fields["response"]
			return nil, fmt.Errorf("hermes relayer command failed: %s: %s", errMsg, resp)
		}
		if s := out["status"]; s != nil && s != "success" {
			return nil, fmt.Errorf("hermes relayer command returned failed with status: %s", s)
		}
	}

	return stdOut, nil
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

type hermesRelayerPacketsOutput struct {
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

	var relayerPacketsOutput hermesRelayerPacketsOutput
	err = json.Unmarshal(stdout, &relayerPacketsOutput)
	s.Require().NoError(err)

	// Check if "unreceived_packets" exists in "src"
	return len(relayerPacketsOutput.Result.Src.UnreceivedPackets) != 0
}

// hermesCreateConnection returns the connectionID.
func (s *IntegrationTestSuite) hermesCreateConnection() string {
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

	out, err := s.executeHermesCommand(ctx, hermesCmd)
	s.Require().NoError(err, "failed to connect chains: %s", err)
	// Output: {"result":{"a_side":{"client_id":"07-tendermint-0","connection_id":"connection-0"},"b_side":{"client_id":"07-tendermint-0","connection_id":"connection-0"},"delay_period":{"nanos":0,"secs":0}},"status":"success"}
	var res struct {
		Result struct {
			ASide struct {
				ClientID     string `json:"client_id"`
				ConnectionID string `json:"connection_id"`
			} `json:"a_side"`
			BSide struct {
				ClientID     string `json:"client_id"`
				ConnectionID string `json:"connection_id"`
			} `json:"b_side"`
		} `json:"result"`
		Status string `json:"status"`
	}
	err = json.Unmarshal(out, &res)
	s.Require().NoError(err, "failed to parse hermes create connection output %s: %s", string(out), err)

	s.T().Logf("IBC connection %s created between chain %s and %s", res.Result.ASide.ConnectionID, s.chainA.id, s.chainB.id)
	return res.Result.ASide.ConnectionID
}

// Returns the channel-ids of both side
func (s *IntegrationTestSuite) hermesCreateChannel(connectionID string) (string, string) {
	s.T().Logf("creating IBC transfer channel created between chains %s and %s", s.chainA.id, s.chainB.id)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	hermesCmd := []string{
		hermesBinary,
		"--json",
		"create",
		"channel",
		"--a-chain", s.chainA.id,
		"--a-connection", connectionID,
		"--a-port", "transfer",
		"--b-port", "transfer",
		"--channel-version", "ics20-1",
		"--order", "unordered",
	}

	out, err := s.executeHermesCommand(ctx, hermesCmd)
	s.Require().NoError(err, "failed to create IBC transfer channel between chains: %s", err)
	// Output {"result":{"a_side":{"channel_id":"channel-0","client_id":"07-tendermint-0","connection_id":"connection-0","port_id":"transfer","version":"ics20-1"},"b_side":{"channel_id":"channel-0","client_id":"07-tendermint-0","connection_id":"connection-0","port_id":"transfer","version":"ics20-1"},"connection_delay":{"nanos":0,"secs":0},"ordering":"Unordered"},"status":"success"}
	var res struct {
		Result struct {
			ASide struct {
				ChannelID    string `json:"channel_id"`
				ClientID     string `json:"client_id"`
				ConnectionID string `json:"connection_id"`
				PortID       string `json:"port_id"`
				Version      string `json:"version"`
			} `json:"a_side"`
			BSide struct {
				ChannelID    string `json:"channel_id"`
				ClientID     string `json:"client_id"`
				ConnectionID string `json:"connection_id"`
				PortID       string `json:"port_id"`
				Version      string `json:"version"`
			} `json:"b_side"`
		} `json:"result"`
		Status string `json:"status"`
	}
	err = json.Unmarshal(out, &res)
	s.Require().NoError(err, "failed to parse hermes create channel output %s: %s", string(out), err)

	s.T().Logf("IBC transfer channel %s created between chains %s and %s", res.Result.ASide.ChannelID, s.chainA.id, s.chainB.id)
	return res.Result.ASide.ChannelID, res.Result.BSide.ChannelID
}
