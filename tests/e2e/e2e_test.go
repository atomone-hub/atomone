package e2e

import "fmt"

var (
	runBankTest                   = true
	runEncodeTest                 = true
	runEvidenceTest               = true
	runFeeGrantTest               = true
	runGovTest                    = true
	runIBCTest                    = true
	runSlashingTest               = true
	runStakingAndDistributionTest = true
	runVestingTest                = true
	runRestInterfacesTest         = true
	runPhotonTest                 = true
	runDynamicfeeTest             = true
	runCoreDAOsTest               = true
	runICSProviderTest            = true
)

func (s *IntegrationTestSuite) TestRestInterfaces() {
	if !runRestInterfacesTest {
		s.T().Skip()
	}
	s.testRestInterfaces()
}

func (s *IntegrationTestSuite) TestBank() {
	if !runBankTest {
		s.T().Skip()
	}
	s.testBankTokenTransfer()
}

func (s *IntegrationTestSuite) TestEncode() {
	if !runEncodeTest {
		s.T().Skip()
	}
	s.testEncode()
	s.testDecode()
}

func (s *IntegrationTestSuite) TestEvidence() {
	if !runEvidenceTest {
		s.T().Skip()
	}
	s.testEvidenceQueries()
}

func (s *IntegrationTestSuite) TestFeeGrant() {
	if !runFeeGrantTest {
		s.T().Skip()
	}
	s.testFeeGrant()
}

func (s *IntegrationTestSuite) TestGov() {
	if !runGovTest {
		s.T().Skip()
	}
	s.testGovSoftwareUpgrade()
	s.testGovCancelSoftwareUpgrade()
	s.testGovCommunityPoolSpend()
	s.testGovParamChange()
	s.testGovConstitutionAmendment()
	s.testGovDynamicQuorum()
	s.testGovTextProposal()
}

func (s *IntegrationTestSuite) TestIBC_hermesRelayer() {
	if !runIBCTest {
		s.T().Skip()
	}
	s.ensureIBCSetup()
	channelIdA, channelIdB := s.runIBCHermesRelayer()
	defer s.tearDownHermesRelayer()

	s.testIBCTokenTransfer(channelIdA, channelIdB)
}

func (s *IntegrationTestSuite) TestIBC_tsRelayer() {
	if !runIBCTest {
		s.T().Skip()
	}
	s.ensureIBCSetup()
	ibcV1Path, ibcV2Path := s.runIBCTSRelayer()
	defer s.tearDownTsRelayer()

	s.testIBCTokenTransfer(ibcV1Path.ChannelA, ibcV1Path.ChannelB)
	s.testIBCv2TokenTransfer(ibcV2Path.ClientA, ibcV2Path.ClientB)
}

func (s *IntegrationTestSuite) TestSlashing() {
	if !runSlashingTest {
		s.T().Skip()
	}
	chainAPI := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	s.testSlashing(chainAPI)
}

// todo add fee test with wrong denom order
func (s *IntegrationTestSuite) TestStakingAndDistribution() {
	if !runStakingAndDistributionTest {
		s.T().Skip()
	}
	s.testStaking()
	s.testDistribution()
}

func (s *IntegrationTestSuite) TestVesting() {
	if !runVestingTest {
		s.T().Skip()
	}
	chainAAPI := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	s.testDelayedVestingAccount(chainAAPI)
	s.testContinuousVestingAccount(chainAAPI)
	// s.testPeriodicVestingAccount(chainAAPI) TODO: add back when v0.45 adds the missing CLI command.
}

func (s *IntegrationTestSuite) TestPhoton() {
	if !runPhotonTest {
		s.T().Skip()
	}
	s.testMintPhoton()
}

func (s *IntegrationTestSuite) TestDynamicfee() {
	if !runDynamicfeeTest {
		s.T().Skip()
	}
	s.testDynamicfeeQuery()
	s.testDynamicfeeGasPriceChange()
}

func (s *IntegrationTestSuite) TestCoreDAOs() {
	if !runCoreDAOsTest {
		s.T().Skip()
	}
	s.testCoreDAOs()
}

func (s *IntegrationTestSuite) TestICSProvider() {
	if !runICSProviderTest {
		s.T().Skip()
	}
	s.ensureICSSetup()
	s.testProviderModuleInitialization()
	s.testConsumerRewardsPool()
	s.testProviderParams()
	s.testConsumerProposalSubmission()
	s.testConsumerChainLaunch()
}
