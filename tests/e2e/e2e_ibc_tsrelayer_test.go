package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// runIBCTSRelayer bootstraps an IBC ts-relayer by creating an IBC connection and
// a transfer channel between chainA and chainB.
// Return the channelID.
func (s *IntegrationTestSuite) runIBCTSRelayer() string {
	s.T().Log("starting ts-relayer container")

	tmpDir, err := os.MkdirTemp("", "atomone-e2e-testnet-ts-relayer-")
	s.Require().NoError(err)
	s.tmpDirs = append(s.tmpDirs, tmpDir)

	rlyA := s.chainA.genesisAccounts[relayerAccountIndex]
	rlyB := s.chainB.genesisAccounts[relayerAccountIndex]

	cfgPath := path.Join(tmpDir, "ts-relayer")

	s.Require().NoError(os.MkdirAll(cfgPath, 0o755))

	s.tsRelayerResource, err = s.dkrPool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       fmt.Sprintf("%s-%s-ts-relayer", s.chainA.id, s.chainB.id),
			Repository: "ts-relayer",
			Tag:        "latest",
			NetworkID:  s.dkrNet.Network.ID,
			PortBindings: map[docker.Port][]docker.PortBinding{
				"3031/tcp": {{HostIP: "", HostPort: "3031"}},
			},
			User:   "root",
			CapAdd: []string{"IPC_LOCK"},
		},
		noRestart,
	)
	s.Require().NoError(err)

	s.T().Logf("started ts-relayer container: %s", s.tsRelayerResource.Container.ID)

	// XXX: Give time to both networks to start, otherwise we might see gRPC
	// transport errors.
	time.Sleep(10 * time.Second)

	s.tsRelayerAddMnemonic(s.chainA.id, rlyA.mnemonic)
	s.tsRelayerAddMnemonic(s.chainB.id, rlyB.mnemonic)
	s.tsRelayerAddGasPrice(s.chainA.id, "0.0001uphoton")
	s.tsRelayerAddGasPrice(s.chainB.id, "0.0001uphoton")

	// create the client between the two AtomOne chains
	return s.tsRelayerAddPath(
		s.chainA.id, s.valResources[s.chainA.id][0].Container.Name[1:],
		s.chainB.id, s.valResources[s.chainB.id][0].Container.Name[1:],
	)
}

func (s *IntegrationTestSuite) tearDownTsRelayer() {
	if s.tsRelayerResource != nil {
		s.T().Log("tearing down TS relayer...")
		s.Require().NoError(s.dkrPool.Purge(s.tsRelayerResource))
		s.tsRelayerResource = nil
	}
}

func (s *IntegrationTestSuite) executeTsRelayerCommand(ctx context.Context, args []string) string {
	cmd := append(tsRelayerBinary, args...)
	exec, err := s.dkrPool.Client.CreateExec(docker.CreateExecOptions{
		Context:      ctx,
		AttachStdout: true,
		AttachStderr: true,
		Container:    s.tsRelayerResource.Container.ID,
		User:         "root",
		Cmd:          cmd,
	})
	s.Require().NoError(err)

	var out bytes.Buffer
	err = s.dkrPool.Client.StartExec(exec.ID, docker.StartExecOptions{
		Context:      ctx,
		Detach:       false,
		OutputStream: &out,
		ErrorStream:  &out,
	})
	s.Require().NoError(err, "ts-relayer startExec error: %s", out.String())

	exitCode := -1
	for {
		inspectExec, err := s.dkrPool.Client.InspectExec(exec.ID)
		s.Require().NoError(err, "ts-relayer inspectExec error: %s", out.String())

		if !inspectExec.Running {
			exitCode = inspectExec.ExitCode
			break
		}
	}
	s.Require().Equal(0, exitCode, "error in ts-relayer cmd '%s', err=%v, exitCode=%d, out=%s", strings.Join(cmd, " "), exitCode, err, out.String())
	return out.String()
}

func (s *IntegrationTestSuite) tsRelayerAddMnemonic(chainID, mnemonic string) {
	s.T().Logf("ts-relayer: adding mnemonic for chain %s", chainID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	args := []string{
		"add-mnemonic",
		"-c", chainID,
		mnemonic,
	}
	s.executeTsRelayerCommand(ctx, args)

	s.T().Logf("ts-relayer: mnemonic added for chain %s", chainID)
}

func (s *IntegrationTestSuite) tsRelayerAddGasPrice(chainID, gasPrice string) {
	s.T().Logf("ts-relayer: adding gas-price for chain %s", chainID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	args := []string{
		"add-gas-price",
		"-c", chainID,
		gasPrice,
	}
	s.executeTsRelayerCommand(ctx, args)

	s.T().Logf("ts-relayer: gas-price added for chain %s", chainID)
}

func (s *IntegrationTestSuite) tsRelayerAddPath(chainAID, containerAID, chainBID, containerBID string) string {
	s.T().Logf("ts-relayer: adding path between chains %s and %s", chainAID, chainBID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	args := []string{
		"add-path",
		"-s", chainAID,
		"-d", chainBID,
		"--surl", "http://" + containerAID + ":26657",
		"--durl", "http://" + containerBID + ":26657",
	}
	out := s.executeTsRelayerCommand(ctx, args)

	// Parsing channelId from logs, format is:
	// "Chanel open confirm for port transfer: channel-0 => channel-0"
	regex := regexp.MustCompile(`Chanel open confirm for port transfer: channel-(\d+) => channel-`)
	matches := regex.FindStringSubmatch(out)
	s.Require().NotNil(matches, "unable to parse ts-relayer output")
	channelId := "channel-" + matches[1]

	s.T().Logf("ts-relayer: path added between chains %s and %s (channel %s)", chainAID, chainBID, channelId)
	return channelId
}
