package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	IBCv1 = "1"
	IBCv2 = "2"
)

type tsRelayerPath struct {
	ID         int
	Version    int
	ChainIdA   string
	ChainIdB   string
	ChainTypeA string
	ChainTypeB string
	ClientB    string
	ClientA    string
	ChannelA   string // empty for IBCv2 path
	ChannelB   string // empty for IBCv2 path
	NodeA      string
	NodeB      string
}

// runIBCTSRelayer bootstraps an IBC ts-relayer by creating an IBC connection and
// a transfer channel between chainA and chainB.
// Returns the paths for ibcV1 and ibcV2.
func (s *IntegrationTestSuite) runIBCTSRelayer() (ibcV1Path tsRelayerPath, ibcV2Path tsRelayerPath) {
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
			Repository: "ghcr.io/allinbits/ibc-v2-ts-relayer",
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

	s.tsRelayerAddMnemonic(s.chainA.id, rlyA.mnemonic)
	s.tsRelayerAddMnemonic(s.chainB.id, rlyB.mnemonic)
	s.tsRelayerAddGasPrice(s.chainA.id, "0.0001uphoton")
	s.tsRelayerAddGasPrice(s.chainB.id, "0.0001uphoton")

	// create IBCv1 path between the two AtomOne chains
	s.tsRelayerAddPath(
		s.chainA.id, s.valResources[s.chainA.id][0].Container.Name[1:],
		s.chainB.id, s.valResources[s.chainB.id][0].Container.Name[1:],
		IBCv1,
	)
	// create IBCv2 path between the two AtomOne chains
	s.tsRelayerAddPath(
		s.chainA.id, s.valResources[s.chainA.id][0].Container.Name[1:],
		s.chainB.id, s.valResources[s.chainB.id][0].Container.Name[1:],
		IBCv2,
	)
	paths := s.tsRelayerDumpPaths()
	ibcV1Path = paths[0] // created first
	ibcV2Path = paths[1]

	// Populate channel-ids for ibcV1Path
	chainAAPI := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	res := s.queryIBCConnectionChannels(chainAAPI, ibcV1Path.ClientA)
	ibcV1Path.ChannelA = res.Channels[0].ChannelId
	chainBAPI := fmt.Sprintf("http://%s", s.valResources[s.chainB.id][0].GetHostPort("1317/tcp"))
	res = s.queryIBCConnectionChannels(chainBAPI, ibcV1Path.ClientB)
	ibcV1Path.ChannelB = res.Channels[0].ChannelId

	return
}

func (s *IntegrationTestSuite) tearDownTsRelayer() {
	if s.tsRelayerResource != nil {
		s.T().Log("tearing down TS relayer...")
		s.Require().NoError(s.dkrPool.Purge(s.tsRelayerResource))
		s.tsRelayerResource = nil
	}
}

func (s *IntegrationTestSuite) executeTsRelayerCommand(ctx context.Context, args []string) []byte {
	tsRelayerBinary := []string{"/bin/with_keyring", "ibc-v2-ts-relayer"}
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
	return out.Bytes()
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

func (s *IntegrationTestSuite) tsRelayerAddPath(chainAID, containerAID, chainBID, containerBID, ibcVersion string) {
	s.T().Logf("ts-relayer: adding IBCv%s path between chains %s and %s", ibcVersion, chainAID, chainBID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	args := []string{
		"add-path",
		"-s", chainAID,
		"-d", chainBID,
		"--surl", "http://" + containerAID + ":26657",
		"--durl", "http://" + containerBID + ":26657",
		"--ibc-version", ibcVersion,
	}
	s.executeTsRelayerCommand(ctx, args)
	s.T().Logf("ts-relayer: IBCv%s path added between chains %s and %s", ibcVersion, chainAID, chainBID)
}

func (s *IntegrationTestSuite) tsRelayerDumpPaths() []tsRelayerPath {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	args := []string{"dump-paths"}
	out := s.executeTsRelayerCommand(ctx, args)
	var paths []tsRelayerPath
	err := json.Unmarshal(out, &paths)
	s.Require().NoError(err)
	return paths
}
