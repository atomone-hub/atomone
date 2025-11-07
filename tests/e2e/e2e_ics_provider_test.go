package e2e

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	providertypes "github.com/allinbits/interchain-security/x/ccv/provider/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

// testProviderModuleInitialization validates that the ICS provider module
// initializes correctly in the e2e environment.
func (s *IntegrationTestSuite) testProviderModuleInitialization() {
	s.T().Log("Verifying ICS provider module initialization in e2e environment")

	endpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))

	// Query provider module params to verify module is initialized
	params := s.queryProviderParams(endpoint)
	s.Require().NotNil(params, "provider params should not be nil")

	s.T().Logf("Provider module initialized successfully with params: %+v", params)
}

// testConsumerRewardsPool verifies the consumer rewards pool module account
// exists and can be queried.
func (s *IntegrationTestSuite) testConsumerRewardsPool() {
	s.T().Log("Verifying consumer rewards pool module account exists")

	endpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))

	// Query the consumer rewards pool account
	poolAddr := authtypes.NewModuleAddress(providertypes.ConsumerRewardsPool).String()
	s.T().Logf("Querying consumer rewards pool account: %s", poolAddr)

	account := s.queryAccount(endpoint, poolAddr)
	s.Require().NotNil(account, "consumer rewards pool account should exist")
	s.T().Logf("Consumer rewards pool account found: %s", account.GetAddress().String())

	// Query balance of the pool
	balance := s.queryBalance(endpoint, poolAddr, uatoneDenom)
	s.T().Logf("Consumer rewards pool balance: %s", balance.String())

	// Balance might be zero initially, but account should exist
	s.Require().NotNil(balance)
}

// testProviderParams tests querying provider module parameters.
func (s *IntegrationTestSuite) testProviderParams() {
	s.T().Log("Testing provider module parameter queries")

	endpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))

	// Query provider params
	params := s.queryProviderParams(endpoint)
	s.Require().NotNil(params, "provider params should not be nil")

	// Verify key parameters exist
	s.T().Logf("TrustingPeriodFraction: %s", params.TrustingPeriodFraction)
	s.T().Logf("CcvTimeoutPeriod: %s", params.CcvTimeoutPeriod)
	s.T().Logf("SlashMeterReplenishPeriod: %s", params.SlashMeterReplenishPeriod)
	s.T().Logf("SlashMeterReplenishFraction: %s", params.SlashMeterReplenishFraction)
	s.T().Logf("ConsumerRewardDenomRegistrationFee: %s", params.ConsumerRewardDenomRegistrationFee)
	s.T().Logf("BlocksPerEpoch: %d", params.BlocksPerEpoch)

	// Basic validation - params should have reasonable values
	s.Require().NotEmpty(params.TrustingPeriodFraction, "trusting period fraction should not be empty")
	s.Require().NotZero(params.CcvTimeoutPeriod, "ccv timeout period should not be zero")

	// Check staking params to verify HistoricalEntries is configured
	body, err := httpGet(fmt.Sprintf("%s/cosmos/staking/v1beta1/params", endpoint))
	s.Require().NoError(err, "failed to query staking params")
	s.T().Logf("Staking params response (first 500 chars): %s", string(body)[:min(500, len(body))])
}

// queryProviderParams queries the provider module parameters via REST API.
func (s *IntegrationTestSuite) queryProviderParams(endpoint string) *providertypes.Params {
	body, err := httpGet(fmt.Sprintf("%s/interchain_security/ccv/provider/params", endpoint))
	s.Require().NoError(err, "failed to query provider params")

	var resp providertypes.QueryParamsResponse
	s.Require().NoError(s.cdc.UnmarshalJSON(body, &resp), "failed to unmarshal provider params response")

	return &resp.Params
}

// testConsumerProposalSubmission tests submitting a consumer addition proposal via governance.
func (s *IntegrationTestSuite) testConsumerProposalSubmission() {
	s.T().Log("Testing consumer chain proposal submission")

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
	sender := senderAddress.String()

	// Define test consumer chain parameters
	testConsumerChainID := "test-consumer-1"

	// Write consumer addition proposal
	s.writeConsumerAdditionProposal(s.chainA, testConsumerChainID)

	// Increment proposal counter for this test
	proposalCounter++
	currentProposalID := proposalCounter

	// Submit proposal
	submitGovFlags := []string{configFile(proposalConsumerAdditionFilename)}
	depositGovFlags := []string{strconv.Itoa(currentProposalID), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
	voteGovFlags := []string{strconv.Itoa(currentProposalID), "yes"}

	s.T().Logf("Submitting consumer addition proposal for chain %s (proposal #%d)", testConsumerChainID, currentProposalID)
	s.T().Logf("Proposal deposit: %s", s.queryGovMinDeposit(chainAAPIEndpoint).String())
	s.T().Logf("Initial deposit: %s", s.queryGovMinInitialDeposit(chainAAPIEndpoint).String())

	s.submitGovProposal(chainAAPIEndpoint, sender, currentProposalID, "ConsumerAddition", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

	// Verify proposal passed
	s.T().Logf("Consumer addition proposal #%d passed successfully", currentProposalID)

	// Verify the proposal appears in pending consumer chain start proposals
	s.Require().Eventually(
		func() bool {
			body, err := httpGet(fmt.Sprintf("%s/interchain_security/ccv/provider/consumer_chain_start_proposals", chainAAPIEndpoint))
			if err != nil {
				return false
			}

			var resp providertypes.QueryConsumerChainStartProposalsResponse
			if err := s.cdc.UnmarshalJSON(body, &resp); err != nil {
				return false
			}

			if resp.Proposals != nil {
				for _, prop := range resp.Proposals.Pending {
					if prop.ChainId == testConsumerChainID {
						s.T().Logf("Consumer chain %s found in pending start proposals (spawn time: %s)", testConsumerChainID, prop.SpawnTime)
						return true
					}
				}
			}
			return false
		},
		10*time.Second,
		1*time.Second,
		"consumer chain should appear in pending consumer chain start proposals",
	)
}

// writeConsumerAdditionProposal writes a consumer addition proposal JSON file.
func (s *IntegrationTestSuite) writeConsumerAdditionProposal(c *chain, consumerChainID string) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)

	// Get provider params to use as defaults for consumer
	providerParams := s.queryProviderParams(chainAAPIEndpoint)

	// Set spawn time in the future to allow governance to complete
	// (governance typically takes 20-30 seconds for deposit + voting periods)
	spawnTime := time.Now().Add(45 * time.Second)

	template := `
	{
		"messages":[
		  {
			"@type": "/interchain_security.ccv.provider.v1.MsgConsumerAddition",
			"authority": "%s",
			"chain_id": "%s",
			"initial_height": {
				"revision_number": 1,
				"revision_height": 1
			},
			"genesis_hash": "Z2VuX2hhc2g=",
			"binary_hash": "YmluX2hhc2g=",
			"spawn_time": "%s",
			"unbonding_period": "1814400s",
			"ccv_timeout_period": "%s",
			"transfer_timeout_period": "3600s",
			"consumer_redistribution_fraction": "0.75",
			"blocks_per_distribution_transmission": 1000,
			"historical_entries": 10000,
			"distribution_transmission_channel": "",
			"top_N": 0,
			"validators_power_cap": 0,
			"validator_set_cap": 0,
			"allowlist": [],
			"denylist": []
		  }
		],
		"deposit": "%s",
		"proposer": "Test proposer",
		"metadata": "Consumer Chain Addition",
		"title": "Add %s consumer chain",
		"summary": "Proposal to add %s as a consumer chain for e2e testing"
	}
	`

	propMsgBody := fmt.Sprintf(template,
		govModuleAddress,
		consumerChainID,
		spawnTime.Format(time.RFC3339),
		providerParams.CcvTimeoutPeriod.String(),
		initialDeposit.String(),
		consumerChainID,
		consumerChainID,
	)

	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalConsumerAdditionFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

