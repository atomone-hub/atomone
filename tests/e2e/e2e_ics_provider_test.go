package e2e

import (
	"fmt"

	providertypes "github.com/allinbits/interchain-security/x/ccv/provider/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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
}

// queryProviderParams queries the provider module parameters via REST API.
func (s *IntegrationTestSuite) queryProviderParams(endpoint string) *providertypes.Params {
	body, err := httpGet(fmt.Sprintf("%s/interchain_security/ccv/provider/params", endpoint))
	s.Require().NoError(err, "failed to query provider params")

	var resp providertypes.QueryParamsResponse
	s.Require().NoError(s.cdc.UnmarshalJSON(body, &resp), "failed to unmarshal provider params response")

	return &resp.Params
}

