package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

/*
testGovSoftwareUpgrade tests passing a gov proposal to upgrade the chain at a given height.
Test Benchmarks:
1. Submission, deposit and vote of message based proposal to upgrade the chain at a height (current height + buffer)
2. Validation that chain halted at upgrade height
3. Teardown & restart chains
4. Reset proposalCounter so subsequent tests have the correct last effective proposal id for chainA
TODO: Perform upgrade in place of chain restart
*/
func (s *IntegrationTestSuite) testGovSoftwareUpgrade() {
	s.Run("software upgrade", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		height := s.getLatestBlockHeight(s.chainA, 0)
		proposalHeight := height + govProposalBlockBuffer
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++

		submitGovFlags := []string{
			"software-upgrade",
			"Upgrade-0",
			"--title='Upgrade V0'",
			"--description='Software Upgrade'",
			"--no-validate",
			fmt.Sprintf("--upgrade-height=%d", proposalHeight),
			"--upgrade-info=my-info",
		}

		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes=0.8,no=0.1,abstain=0.1"}
		s.submitLegacyGovProposal(chainAAPIEndpoint, sender, proposalCounter, upgradetypes.ProposalTypeSoftwareUpgrade, submitGovFlags, depositGovFlags, voteGovFlags, "weighted-vote", true)

		s.verifyChainHaltedAtUpgradeHeight(s.chainA, 0, proposalHeight)
		s.T().Logf("Successfully halted chain at  height %d", proposalHeight)

		s.TearDownSuite()

		s.T().Logf("Restarting containers")
		s.SetupSuite()

		s.Require().Eventually(
			func() bool {
				return s.getLatestBlockHeight(s.chainA, 0) > 0
			},
			30*time.Second,
			time.Second,
		)

		proposalCounter = 0
	})
}

/*
testGovCancelSoftwareUpgrade tests passing a gov proposal that cancels a pending upgrade.
Test Benchmarks:
1. Submission, deposit and vote of message based proposal to upgrade the chain at a height (current height + buffer)
2. Submission, deposit and vote of message based proposal to cancel the pending upgrade
3. Validation that the chain produced blocks past the intended upgrade height
*/
func (s *IntegrationTestSuite) testGovCancelSoftwareUpgrade() {
	s.Run("cancel software upgrade", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()

		sender := senderAddress.String()
		height := s.getLatestBlockHeight(s.chainA, 0)
		proposalHeight := height + 50

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{
			"software-upgrade",
			"Upgrade-1",
			"--title='Upgrade V1'",
			"--description='Software Upgrade'",
			"--no-validate",
			fmt.Sprintf("--upgrade-height=%d", proposalHeight),
		}

		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitLegacyGovProposal(chainAAPIEndpoint, sender, proposalCounter, upgradetypes.ProposalTypeSoftwareUpgrade, submitGovFlags, depositGovFlags, voteGovFlags, "vote", true)

		proposalCounter++
		submitGovFlags = []string{"cancel-software-upgrade", "--title='Upgrade V1'", "--description='Software Upgrade'"}
		depositGovFlags = []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags = []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitLegacyGovProposal(chainAAPIEndpoint, sender, proposalCounter, upgradetypes.ProposalTypeCancelSoftwareUpgrade, submitGovFlags, depositGovFlags, voteGovFlags, "vote", true)

		s.verifyChainPassesUpgradeHeight(s.chainA, 0, proposalHeight)
		s.T().Logf("Successfully canceled upgrade at height %d", proposalHeight)
	})
}

/*
testGovCommunityPoolSpend tests passing a community spend proposal.
Test Benchmarks:
1. Fund Community Pool
2. Submission, deposit and vote of proposal to spend from the community pool to send atoms to a recipient
3. Validation that the recipient balance has increased by proposal amount
*/
func (s *IntegrationTestSuite) testGovCommunityPoolSpend() {
	s.Run("community pool spend", func() {
		s.fundCommunityPool()
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		recipientAddress, _ := s.chainA.validators[1].keyInfo.GetAddress()
		recipient := recipientAddress.String()
		sendAmount := sdk.NewInt64Coin(uatoneDenom, 10_000_000) // 10atone
		s.writeGovCommunitySpendProposal(s.chainA, sendAmount, recipient)

		beforeRecipientBalance, err := getSpecificBalance(chainAAPIEndpoint, recipient, uatoneDenom)
		s.Require().NoError(err)

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalCommunitySpendFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "CommunityPoolSpend", submitGovFlags, depositGovFlags, voteGovFlags, "vote")

		s.Require().Eventually(
			func() bool {
				afterRecipientBalance, err := getSpecificBalance(chainAAPIEndpoint, recipient, uatoneDenom)
				s.Require().NoError(err)

				return afterRecipientBalance.Sub(sendAmount).IsEqual(beforeRecipientBalance)
			},
			10*time.Second,
			time.Second,
		)
	})
}

// testGovParamChange tests passing a param change proposal.
func (s *IntegrationTestSuite) testGovParamChange() {
	s.Run("staking param change", func() {
		// check existing params
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		params := s.queryStakingParams(chainAAPIEndpoint)
		oldMaxValidator := params.Params.MaxValidators
		// add 10 to actual max validators
		params.Params.MaxValidators = oldMaxValidator + 10

		s.writeStakingParamChangeProposal(s.chainA, params.Params)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalParamChangeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "cosmos.staking.v1beta1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote")

		newParams := s.queryStakingParams(chainAAPIEndpoint)
		s.Assert().NotEqual(oldMaxValidator, newParams.Params.MaxValidators)
	})

	s.Run("photon param change", func() {
		// check existing params
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		params := s.queryPhotonParams(chainAAPIEndpoint)
		// toggle param mint_disabled
		oldMintDisabled := params.Params.MintDisabled
		s.Require().False(oldMintDisabled, "expected photon param mint disabled to be false")
		params.Params.MintDisabled = true

		s.writePhotonParamChangeProposal(s.chainA, params.Params)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalParamChangeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.photon.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote")

		newParams := s.queryPhotonParams(chainAAPIEndpoint)
		s.Assert().True(newParams.Params.MintDisabled, "expected photon param mint disabled to be true")

		// Revert change or mint photon test will fail
		params.Params.MintDisabled = false
		proposalCounter++
		depositGovFlags = []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags = []string{strconv.Itoa(proposalCounter), "yes"}
		s.writePhotonParamChangeProposal(s.chainA, params.Params)
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.photon.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote")

		newParams = s.queryPhotonParams(chainAAPIEndpoint)
		s.Require().False(newParams.Params.MintDisabled, "expected photon param mint disabled to be false")
	})
}

func (s *IntegrationTestSuite) testGovConstitutionAmendment() {
	s.Run("constitution amendment", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()

		newConstitution := "New test constitution"
		amendmentMsg := s.generateConstitutionAmendment(s.chainA, newConstitution)

		s.writeGovConstitutionAmendmentProposal(s.chainA, amendmentMsg.Amendment)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalConstitutionAmendmentFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "gov/MsgSubmitProposal", submitGovFlags, depositGovFlags, voteGovFlags, "vote")

		s.Require().Eventually(
			func() bool {
				res := s.queryConstitution(chainAAPIEndpoint)
				return res.Constitution == newConstitution
			},
			10*time.Second,
			time.Second,
		)
	})
}

// testGovGovernors tests passing a text proposal and vote with governors.
func (s *IntegrationTestSuite) testGovGovernors() {
	s.Run("governors", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.writeGovTextProposal(s.chainA)

		// create a governor
		acc1Addr, _ := s.chainA.genesisAccounts[1].keyInfo.GetAddress()
		governorAddr := govtypes.GovernorAddress(acc1Addr).String()
		// a governor must have a delegation of at least 10atone
		valAddr, _ := s.chainA.validators[0].keyInfo.GetAddress()
		validatorAddr := sdk.ValAddress(valAddr).String()
		govDelegatorAddr := acc1Addr.String()
		govDelegation := sdk.NewInt64Coin("uatone", 10_000_000)
		s.execDelegate(s.chainA, 0, govDelegation, validatorAddr, govDelegatorAddr)
		// run create-governor
		s.runGovExec(s.chainA, 0, govDelegatorAddr, "create-governor", []string{
			govDelegatorAddr, "moniker", "identity", "website", "security-contact", "details",
		})
		// check governor is created
		s.Require().Eventually(
			func() bool {
				governor, err := queryGovGovernor(chainAAPIEndpoint, governorAddr)
				s.Require().NoError(err)
				return governor.Governor != nil
			},
			15*time.Second,
			time.Second,
		)

		// delegate to this governor
		// first create a delegator
		acc2Addr, _ := s.chainA.genesisAccounts[2].keyInfo.GetAddress()
		delegatorAddr := acc2Addr.String()
		delDelegation := sdk.NewInt64Coin("uatone", 10_000_000)
		s.execDelegate(s.chainA, 0, delDelegation, validatorAddr, delegatorAddr)
		// then delegate to governor
		s.runGovExec(s.chainA, 0, delegatorAddr, "delegate-governor", []string{
			delegatorAddr, governorAddr,
		})
		// check governor delegation is created
		s.Require().Eventually(
			func() bool {
				resp, err := queryGovGovernorDelegation(chainAAPIEndpoint, delegatorAddr)
				s.Require().NoError(err)
				return resp.GovernorAddress == governorAddr
			},
			15*time.Second,
			time.Second,
		)
		// assert governor valshares
		resp, err := queryGovGovernorValShares(chainAAPIEndpoint, governorAddr)
		s.Require().NoError(err)
		s.Require().Len(resp.ValShares, 1, "expected 1 valshare")
		s.Require().Equal(governorAddr, resp.ValShares[0].GovernorAddress)
		s.Require().Equal(validatorAddr, resp.ValShares[0].ValidatorAddress)
		validator, err := queryValidator(chainAAPIEndpoint, validatorAddr)
		s.Require().NoError(err)
		totalDelegations := delDelegation.Add(govDelegation)
		expectedShares, err := validator.SharesFromTokens(totalDelegations.Amount)
		s.Require().NoError(err)
		s.Require().True(expectedShares.Equal(resp.ValShares[0].Shares), "want shares %s, got %s", expectedShares, resp.ValShares[0].Shares)

		// Create a governance proposal
		proposalCounter++
		submitGovFlags := []string{configFile(proposalTextFilename)}
		s.submitGovCommand(chainAAPIEndpoint, sender, proposalCounter, "submit-proposal", submitGovFlags, govtypesv1.StatusVotingPeriod)

		// Vote with governor
		voteFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovCommand(chainAAPIEndpoint, govDelegatorAddr, proposalCounter, "vote", voteFlags, govtypesv1.StatusPassed)

		// assert tally result
		prop, err := queryGovProposal(chainAAPIEndpoint, proposalCounter)
		s.Require().NoError(err)
		expectedTally := &govtypesv1.TallyResult{
			YesCount:     totalDelegations.Amount.String(),
			NoCount:      "0",
			AbstainCount: "0",
		}
		s.Require().Equal(expectedTally, prop.Proposal.FinalTallyResult)
	})
}

func (s *IntegrationTestSuite) submitLegacyGovProposal(chainAAPIEndpoint, sender string, proposalID int, proposalType string, submitFlags []string, depositFlags []string, voteFlags []string, voteCommand string, withDeposit bool) {
	s.T().Logf("Submitting Gov Proposal: %s", proposalType)
	// min deposit of 1000uatone is required in e2e tests, otherwise the gov antehandler causes the proposal to be dropped
	sflags := submitFlags
	if withDeposit {
		sflags = append(sflags, "--deposit="+initialDepositAmount.String())
	}
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "submit-legacy-proposal", sflags, govtypesv1.StatusDepositPeriod)
	s.T().Logf("Depositing Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "deposit", depositFlags, govtypesv1.StatusVotingPeriod)
	s.T().Logf("Voting Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, voteCommand, voteFlags, govtypesv1.StatusPassed)
}

// NOTE: in SDK >= v0.47 the submit-proposal does not have a --deposit flag
// Instead, the deposit is added to the "deposit" field of the proposal JSON (usually stored as a file)
// you can use `atomoned tx gov draft-proposal` to create a proposal file that you can use
// min initial deposit of 100uatone is required in e2e tests, otherwise the proposal would be dropped
func (s *IntegrationTestSuite) submitGovProposal(chainAAPIEndpoint, sender string, proposalID int, proposalType string, submitFlags []string, depositFlags []string, voteFlags []string, voteCommand string) {
	s.T().Logf("Submitting Gov Proposal: %s", proposalType)
	sflags := submitFlags
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "submit-proposal", sflags, govtypesv1.StatusDepositPeriod)
	s.T().Logf("Depositing Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "deposit", depositFlags, govtypesv1.StatusVotingPeriod)
	s.T().Logf("Voting Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, voteCommand, voteFlags, govtypesv1.StatusPassed)
}

func (s *IntegrationTestSuite) verifyChainHaltedAtUpgradeHeight(c *chain, valIdx int, upgradeHeight int64) {
	s.Require().Eventually(
		func() bool {
			currentHeight := s.getLatestBlockHeight(c, valIdx)

			return currentHeight == upgradeHeight
		},
		30*time.Second,
		time.Second,
	)

	counter := 0
	s.Require().Eventually(
		func() bool {
			currentHeight := s.getLatestBlockHeight(c, valIdx)

			if currentHeight > upgradeHeight {
				return false
			}
			if currentHeight == upgradeHeight {
				counter++
			}
			return counter >= 2
		},
		8*time.Second,
		time.Second,
	)
}

func (s *IntegrationTestSuite) verifyChainPassesUpgradeHeight(c *chain, valIdx int, upgradeHeight int64) {
	var currentHeight int64
	s.Require().Eventually(
		func() bool {
			currentHeight = s.getLatestBlockHeight(c, valIdx)
			return currentHeight > upgradeHeight
		},
		30*time.Second,
		time.Second,
		"expected chain height greater than %d: got %d", upgradeHeight, currentHeight,
	)
}

func (s *IntegrationTestSuite) submitGovCommand(chainAAPIEndpoint, sender string, proposalID int, govCommand string, proposalFlags []string, expectedSuccessStatus govtypesv1.ProposalStatus) {
	s.runGovExec(s.chainA, 0, sender, govCommand, proposalFlags)

	s.Require().Eventually(
		func() bool {
			proposal, err := queryGovProposal(chainAAPIEndpoint, proposalID)
			s.Require().NoError(err)
			return proposal.GetProposal().Status == expectedSuccessStatus
		},
		15*time.Second,
		time.Second,
	)
}

func (s *IntegrationTestSuite) writeStakingParamChangeProposal(c *chain, params stakingtypes.Params) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	template := `
	{
		"messages":[
		  {
			"@type": "/cosmos.staking.v1beta1.MsgUpdateParams",
			"authority": "%s",
			"params": %s
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing staking param change",
		"metadata": "",
		"title": "Change in staking params",
		"summary": "summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, cdc.MustMarshalJSON(&params), initialDepositAmount)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalParamChangeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) writePhotonParamChangeProposal(c *chain, params photontypes.Params) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	template := `
	{
		"messages":[
		  {
			"@type": "/atomone.photon.v1.MsgUpdateParams",
			"authority": "%s",
			"params": %s
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing photon param change",
		"metadata": "",
		"title": "Change in photon params",
		"summary": "summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, cdc.MustMarshalJSON(&params), initialDepositAmount)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalParamChangeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

// writeGovTextProposal creates a text proposal JSON file with full required deposit.
func (s *IntegrationTestSuite) writeGovTextProposal(c *chain) {
	template := `
	{
		"deposit": "%s",
		"metadata": "The metadata",
		"title": "A text proposal",
		"summary": "The summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, depositAmount)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalTextFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) writeGovConstitutionAmendmentProposal(c *chain, amendment string) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	// escape newlines in amendment
	amendment = strings.ReplaceAll(amendment, "\n", "\\n")
	template := `
	{
		"messages":[
		  {
			"@type": "/atomone.gov.v1.MsgProposeConstitutionAmendment",
			"authority": "%s",
			"amendment": "%s"
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing validator address",
		"metadata": "Constitution Amendment",
		"title": "Constitution Amendment",
		"summary": "summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, amendment, initialDepositAmount)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalConstitutionAmendmentFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) generateConstitutionAmendment(c *chain, newConstitution string) govtypesv1.MsgProposeConstitutionAmendment {
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", newConstitutionFilename), []byte(newConstitution))
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	govCommand := "generate-constitution-amendment"
	cmd := []string{
		atomonedBinary,
		txCommand,
		govtypes.ModuleName,
		govCommand,
		configFile(newConstitutionFilename),
	}

	s.T().Logf("Executing atomoned tx gov %s on chain %s", govCommand, c.id)
	var msg govtypesv1.MsgProposeConstitutionAmendment
	s.executeAtomoneTxCommand(ctx, c, cmd, 0, s.parseGenerateConstitutionAmendmentOutput(&msg))
	s.T().Logf("Successfully executed %s", govCommand)

	s.Require().NoError(err)
	return msg
}

func (s *IntegrationTestSuite) parseGenerateConstitutionAmendmentOutput(msg *govtypesv1.MsgProposeConstitutionAmendment) func([]byte, []byte) bool {
	return func(stdOut []byte, stdErr []byte) bool {
		if len(stdErr) > 0 {
			s.T().Logf("Error: %s", string(stdErr))
			return false
		}

		err := cdc.UnmarshalJSON(stdOut, msg)
		s.Require().NoError(err)

		return true
	}
}
