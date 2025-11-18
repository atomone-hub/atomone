package e2e

import (
	"context"
	"path/filepath"
)

// testConsumerChainLaunch tests the complete consumer chain launch lifecycle:
// 1. Wait for spawn time (proposal was submitted in testConsumerProposalSubmission)
// 2. Query consumer genesis from provider
// 3. Create and initialize consumer chain infrastructure
// 4. Merge CCV genesis with consumer genesis
// 5. Start consumer chain with ICS consumer daemon
// 6. Verify consumer chain is producing blocks
func (s *IntegrationTestSuite) testConsumerChainLaunch() {
	s.T().Log("Testing complete consumer chain launch lifecycle")

	// Consumer chain will be launched by chainA (provider)
	providerChain := s.chainA
	consumerChainID := "test-consumer-1"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Step 1: Wait for spawn time (proposal was already submitted in testConsumerProposalSubmission)
	s.T().Log("Waiting for consumer chain spawn time...")
	err := s.waitForConsumerChainSpawnTime(ctx, providerChain, consumerChainID)
	s.Require().NoError(err, "failed to wait for consumer spawn time")

	// Step 2: Query consumer genesis from provider
	s.T().Log("Querying consumer genesis from provider...")
	ccvGenesis, err := s.queryConsumerGenesis(providerChain, consumerChainID)
	s.Require().NoError(err, "failed to query consumer genesis")
	s.Require().NotEmpty(ccvGenesis, "consumer genesis should not be empty")

	// Step 3: Create consumer chain
	s.T().Log("Creating consumer chain...")
	consumerChain, err := s.createConsumerChain(consumerChainID)
	s.Require().NoError(err, "failed to create consumer chain")

	// Step 4: Initialize consumer chain infrastructure
	s.T().Log("Initializing consumer chain infrastructure...")
	s.Require().NoError(consumerChain.createAndInitValidators(2))

	// Copy provider validator keys to consumer validators
	// Consumer chains use the same validator set as the provider
	for i, consumerVal := range consumerChain.validators {
		providerVal := providerChain.validators[i]
		providerKeyPath := filepath.Join(providerVal.configDir(), "config", "priv_validator_key.json")
		consumerKeyPath := filepath.Join(consumerVal.configDir(), "config", "priv_validator_key.json")
		_, err := copyFile(providerKeyPath, consumerKeyPath)
		s.Require().NoError(err)
	}

	// Copy genesis file to all validators
	val0ConfigDir := consumerChain.validators[0].configDir()
	for _, val := range consumerChain.validators[1:] {
		_, err := copyFile(
			filepath.Join(val0ConfigDir, "config", "genesis.json"),
			filepath.Join(val.configDir(), "config", "genesis.json"),
		)
		s.Require().NoError(err)
	}

	// Step 5: Merge CCV genesis with consumer genesis
	s.T().Log("Merging CCV genesis into consumer chain...")
	err = s.mergeConsumerGenesis(consumerChain, ccvGenesis)
	s.Require().NoError(err, "failed to merge consumer genesis")

	// Step 6: Configure validator P2P networking
	s.T().Log("Configuring consumer validator P2P networking...")
	s.initValidatorConfigs(consumerChain)

	// Step 7: Start consumer chain
	s.T().Log("Starting consumer chain...")
	err = s.startConsumerChain(consumerChain)
	s.Require().NoError(err, "failed to start consumer chain")

	// Step 8: Verify consumer chain is producing blocks
	s.T().Log("Verifying consumer chain is producing blocks...")
	s.assertConsumerChainHeight(consumerChain, 3)
}
