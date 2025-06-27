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

	feemarkettypes "github.com/atomone-hub/atomone/x/feemarket/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
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

		beforeSenderBalance, err := getSpecificBalance(chainAAPIEndpoint, sender, uatoneDenom)
		s.Require().NoError(err)
		beforeRecipientBalance, err := getSpecificBalance(chainAAPIEndpoint, recipient, uatoneDenom)
		s.Require().NoError(err)

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalCommunitySpendFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "CommunityPoolSpend", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		// Check that sender is refunded with the proposal deposit
		s.Require().Eventually(
			func() bool {
				afterSenderBalance, err := getSpecificBalance(chainAAPIEndpoint, sender, uatoneDenom)
				s.Require().NoError(err)

				return afterSenderBalance.IsEqual(beforeSenderBalance)
			},
			10*time.Second,
			time.Second,
		)
		// Check that recipient received the community pool spend
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
	s.Run("community pool spend with number of no votes exceeds threshold", func() {
		s.fundCommunityPool()
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		recipientAddress, _ := s.chainA.validators[1].keyInfo.GetAddress()
		recipient := recipientAddress.String()
		sendAmount := sdk.NewInt64Coin(uatoneDenom, 10_000_000) // 10atone
		s.writeGovCommunitySpendProposal(s.chainA, sendAmount, recipient)

		beforeSenderBalance, err := getSpecificBalance(chainAAPIEndpoint, sender, uatoneDenom)
		s.Require().NoError(err)
		beforeRecipientBalance, err := getSpecificBalance(chainAAPIEndpoint, recipient, uatoneDenom)
		s.Require().NoError(err)

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalCommunitySpendFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "no"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "CommunityPoolSpend", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusRejected)

		// Check that sender is not refunded with the proposal deposit
		s.Require().Eventually(
			func() bool {
				afterSenderBalance, err := getSpecificBalance(chainAAPIEndpoint, sender, uatoneDenom)
				s.Require().NoError(err)

				return afterSenderBalance.Add(depositAmount).Add(initialDepositAmount).
					IsEqual(beforeSenderBalance)
			},
			10*time.Second,
			time.Second,
		)
		// Check that recipient didnt receive the community pool spend since the
		// proposal was rejected
		s.Require().Eventually(
			func() bool {
				afterRecipientBalance, err := getSpecificBalance(chainAAPIEndpoint, recipient, uatoneDenom)
				s.Require().NoError(err)

				return afterRecipientBalance.IsEqual(beforeRecipientBalance)
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
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "cosmos.staking.v1beta1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

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
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.photon.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		newParams := s.queryPhotonParams(chainAAPIEndpoint)
		s.Assert().True(newParams.Params.MintDisabled, "expected photon param mint disabled to be true")

		// Revert change or mint photon test will fail
		params.Params.MintDisabled = false
		proposalCounter++
		depositGovFlags = []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags = []string{strconv.Itoa(proposalCounter), "yes"}
		s.writePhotonParamChangeProposal(s.chainA, params.Params)
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.photon.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		newParams = s.queryPhotonParams(chainAAPIEndpoint)
		s.Require().False(newParams.Params.MintDisabled, "expected photon param mint disabled to be false")
	})
	s.Run("feemarket param change", func() {
		// check existing params
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		params := s.queryFeemarketParams(chainAAPIEndpoint)

		oldAlpha := params.Params.Alpha
		params.Params.Alpha = oldAlpha.Add(sdk.NewDec(1))

		s.writeFeemarketParamChangeProposal(s.chainA, params.Params)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalParamChangeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.feemarket.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		newParams := s.queryFeemarketParams(chainAAPIEndpoint)
		s.Require().Equal(newParams.Params.Alpha, oldAlpha.Add(sdk.NewDec(1)))
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
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "gov/MsgSubmitProposal", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

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

func (s *IntegrationTestSuite) testGovDynamicQuorum() {
	s.Run("dynamic quorum change", func() {

		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		params := s.queryGovParams(chainAAPIEndpoint, "tallying")
		s.T().Logf("Tally Params: %s", params)
		quorums := s.queryGovQuorums(chainAAPIEndpoint)
		// pEma = (Q - Qmin) / (Qmax - Qmin)
		quorumRange := params.GetParams().QuorumRange
		quorumMin := sdk.MustNewDecFromStr(quorumRange.Min)
		quorumMax := sdk.MustNewDecFromStr(quorumRange.Max)
		currentQuorum := sdk.MustNewDecFromStr(quorums.GetQuorum())
		quorumPEma := (currentQuorum.Sub(quorumMin)).Quo(quorumMax.Sub(quorumMin))

		paramsFeemarket := s.queryFeemarketParams(chainAAPIEndpoint)

		oldAlpha := paramsFeemarket.Params.Alpha
		paramsFeemarket.Params.Alpha = oldAlpha.Add(sdk.NewDec(1))

		s.writeFeemarketParamChangeProposal(s.chainA, paramsFeemarket.Params)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalParamChangeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.feemarket.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)
		quorumsAfter := s.queryGovQuorums(chainAAPIEndpoint)
		endQuorum := sdk.MustNewDecFromStr(quorumsAfter.GetQuorum())
		endQuorumPEma := (endQuorum.Sub(quorumMin)).Quo(quorumMax.Sub(quorumMin))

		s.T().Logf("QMin: %s", quorumMin)
		s.T().Logf("QMax: %s", quorumMax)
		s.T().Logf("Quorum Before: %s", currentQuorum)
		s.T().Logf("Quorum PEMA Before: %s", quorumPEma)
		s.T().Logf("Quorum After: %s", endQuorum)
		s.T().Logf("Quorum PEMA Before: %s", endQuorumPEma)
		expectedParticipation := endQuorum.Sub(currentQuorum.Mul(sdk.MustNewDecFromStr("0.8"))).Quo(sdk.MustNewDecFromStr("0.2"))
		s.T().Logf("Expected Participation: %s", expectedParticipation)
		proposal, _ := queryGovProposal(chainAAPIEndpoint, proposalCounter)
		s.T().Logf("Proposal: %s", proposal.Proposal.FinalTallyResult)
		stakingPool := s.queryStakingPool(chainAAPIEndpoint)
		s.T().Logf("Bonded Tokens: %s", stakingPool.Pool.BondedTokens)
		votes := proposal.Proposal.FinalTallyResult.Yes.ToLegacyDec()
		totalVP := stakingPool.Pool.BondedTokens.ToLegacyDec()

		actualParticipation := votes.Quo(totalVP)
		s.T().Logf("Actual Participation: %s", actualParticipation)
		s.Require().True(actualParticipation.Equal(expectedParticipation))
		s.Require().Equal(quorumsAfter.LawQuorum, quorums.LawQuorum)
		s.Require().Equal(quorumsAfter.ConstitutionAmendmentQuorum, quorums.ConstitutionAmendmentQuorum)
	})

	s.Run("dynamic law quorum change", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s",
			s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		params := s.queryGovParams(chainAAPIEndpoint, "tallying")
		s.T().Logf("Tally Params: %s", params)
		quorums := s.queryGovQuorums(chainAAPIEndpoint)
		// pEma = (Q - Qmin) / (Qmax - Qmin)
		lawQuorumRange := params.GetParams().LawQuorumRange
		lawQuorumMin := sdk.MustNewDecFromStr(lawQuorumRange.Min)
		lawQuorumMax := sdk.MustNewDecFromStr(lawQuorumRange.Max)
		currentLawQuorum := sdk.MustNewDecFromStr(quorums.GetLawQuorum())
		lawQuorumPEma := (currentLawQuorum.Sub(lawQuorumMin)).Quo(lawQuorumMax.Sub(lawQuorumMin))

		s.writeGovLawProposal(s.chainA)
		proposalCounter++
		submitGovFlags := []string{configFile(proposalLawFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter,
			"gov/MsgSubmitProposal", submitGovFlags, depositGovFlags, voteGovFlags,
			"vote", govtypesv1beta1.StatusPassed)
		quorumsAfter := s.queryGovQuorums(chainAAPIEndpoint)
		endLawQuorum := sdk.MustNewDecFromStr(quorumsAfter.GetLawQuorum())
		endLawQuorumPEma := (endLawQuorum.Sub(lawQuorumMin)).Quo(lawQuorumMax.Sub(lawQuorumMin))

		s.T().Logf("QMin: %s", lawQuorumMin)
		s.T().Logf("QMax: %s", lawQuorumMax)
		s.T().Logf("lawQuorum Before: %s", currentLawQuorum)
		s.T().Logf("lawQuorum PEMA Before: %s", lawQuorumPEma)
		s.T().Logf("lawQuorum After: %s", endLawQuorum)
		s.T().Logf("lawQuorum PEMA Before: %s", endLawQuorumPEma)
		expectedParticipation := endLawQuorum.Sub(currentLawQuorum.Mul(
			sdk.MustNewDecFromStr("0.8"))).Quo(sdk.MustNewDecFromStr("0.2"))
		s.T().Logf("Expected Participation: %s", expectedParticipation)
		proposal, _ := queryGovProposal(chainAAPIEndpoint, proposalCounter)
		s.T().Logf("Proposal: %s", proposal.Proposal.FinalTallyResult)
		stakingPool := s.queryStakingPool(chainAAPIEndpoint)
		s.T().Logf("Bonded Tokens: %s", stakingPool.Pool.BondedTokens)
		votes := proposal.Proposal.FinalTallyResult.Yes.ToLegacyDec()
		totalVP := stakingPool.Pool.BondedTokens.ToLegacyDec()
		actualParticipation := votes.Quo(totalVP)
		s.T().Logf("Actual Participation: %s", actualParticipation)
		s.Require().True(actualParticipation.Equal(expectedParticipation))
		s.Require().Equal(quorumsAfter.Quorum, quorums.Quorum)
		s.Require().Equal(quorumsAfter.ConstitutionAmendmentQuorum,
			quorums.ConstitutionAmendmentQuorum)
	})

	s.Run("dynamic constitution amendment quorum change", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		params := s.queryGovParams(chainAAPIEndpoint, "tallying")
		s.T().Logf("Tally Params: %s", params)
		quorums := s.queryGovQuorums(chainAAPIEndpoint)
		// pEma = (Q - Qmin) / (Qmax - Qmin)
		constitutionAmendmentQuorumRange := params.GetParams().ConstitutionAmendmentQuorumRange
		constitutionAmendmentQuorumMin := sdk.MustNewDecFromStr(constitutionAmendmentQuorumRange.Min)
		constitutionAmendmentQuorumMax := sdk.MustNewDecFromStr(constitutionAmendmentQuorumRange.Max)
		currentConstitutionAmendmentQuorum := sdk.MustNewDecFromStr(quorums.GetConstitutionAmendmentQuorum())
		constitutionAmendmentQuorumPEma := (currentConstitutionAmendmentQuorum.Sub(constitutionAmendmentQuorumMin)).Quo(constitutionAmendmentQuorumMax.Sub(constitutionAmendmentQuorumMin))

		newConstitution := "New test constitution"
		amendmentMsg := s.generateConstitutionAmendment(s.chainA, newConstitution)

		s.writeGovConstitutionAmendmentProposal(s.chainA, amendmentMsg.Amendment)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalConstitutionAmendmentFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "gov/MsgSubmitProposal", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)
		quorumsAfter := s.queryGovQuorums(chainAAPIEndpoint)
		endConstitutionAmendmentQuorum := sdk.MustNewDecFromStr(quorumsAfter.GetConstitutionAmendmentQuorum())
		endConstitutionAmendmentQuorumPEma := (endConstitutionAmendmentQuorum.Sub(constitutionAmendmentQuorumMin)).Quo(constitutionAmendmentQuorumMax.Sub(constitutionAmendmentQuorumMin))

		s.T().Logf("QMin: %s", constitutionAmendmentQuorumMin)
		s.T().Logf("QMax: %s", constitutionAmendmentQuorumMax)
		s.T().Logf("constitutionAmendmentQuorum Before: %s", currentConstitutionAmendmentQuorum)
		s.T().Logf("constitutionAmendmentQuorum PEMA Before: %s", constitutionAmendmentQuorumPEma)
		s.T().Logf("constitutionAmendmentQuorum After: %s", endConstitutionAmendmentQuorum)
		s.T().Logf("constitutionAmendmentQuorum PEMA Before: %s", endConstitutionAmendmentQuorumPEma)
		expectedParticipation := endConstitutionAmendmentQuorum.Sub(currentConstitutionAmendmentQuorum.Mul(sdk.MustNewDecFromStr("0.8"))).Quo(sdk.MustNewDecFromStr("0.2"))
		s.T().Logf("Expected Participation: %s", expectedParticipation)
		proposal, _ := queryGovProposal(chainAAPIEndpoint, proposalCounter)
		s.T().Logf("Proposal: %s", proposal.Proposal.FinalTallyResult)
		stakingPool := s.queryStakingPool(chainAAPIEndpoint)
		s.T().Logf("Bonded Tokens: %s", stakingPool.Pool.BondedTokens)
		votes := proposal.Proposal.FinalTallyResult.Yes.ToLegacyDec()
		totalVP := stakingPool.Pool.BondedTokens.ToLegacyDec()
		actualParticipation := votes.Quo(totalVP)
		s.T().Logf("Actual Participation: %s", actualParticipation)
		s.Require().True(actualParticipation.Equal(expectedParticipation))
		s.Require().Equal(quorumsAfter.Quorum, quorums.Quorum)
		s.Require().Equal(quorumsAfter.LawQuorum, quorums.LawQuorum)
	})

}

func (s *IntegrationTestSuite) submitLegacyGovProposal(chainAAPIEndpoint, sender string, proposalID int, proposalType string, submitFlags []string, depositFlags []string, voteFlags []string, voteCommand string, withDeposit bool) {
	s.T().Logf("Submitting Gov Proposal: %s", proposalType)
	// min deposit of 1000uatone is required in e2e tests, otherwise the gov antehandler causes the proposal to be dropped
	sflags := submitFlags
	if withDeposit {
		sflags = append(sflags, "--deposit="+initialDepositAmount.String())
	}
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "submit-legacy-proposal", sflags, govtypesv1beta1.StatusDepositPeriod)
	s.T().Logf("Depositing Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "deposit", depositFlags, govtypesv1beta1.StatusVotingPeriod)
	s.T().Logf("Voting Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, voteCommand, voteFlags, govtypesv1beta1.StatusPassed)
}

// NOTE: in SDK >= v0.47 the submit-proposal does not have a --deposit flag
// Instead, the deposit is added to the "deposit" field of the proposal JSON (usually stored as a file)
// you can use `atomoned tx gov draft-proposal` to create a proposal file that you can use
// min initial deposit of 100uatone is required in e2e tests, otherwise the proposal would be dropped
func (s *IntegrationTestSuite) submitGovProposal(chainAAPIEndpoint, sender string, proposalID int, proposalType string, submitFlags []string, depositFlags []string, voteFlags []string, voteCommand string, expectedStatusAfterVote govtypesv1beta1.ProposalStatus) {
	s.T().Logf("Submitting Gov Proposal: %s", proposalType)
	sflags := submitFlags
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "submit-proposal", sflags, govtypesv1beta1.StatusDepositPeriod)
	s.T().Logf("Depositing Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, "deposit", depositFlags, govtypesv1beta1.StatusVotingPeriod)
	s.T().Logf("Voting Gov Proposal: %s", proposalType)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalID, voteCommand, voteFlags, expectedStatusAfterVote)
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

func (s *IntegrationTestSuite) submitGovCommand(chainAAPIEndpoint, sender string, proposalID int, govCommand string, proposalFlags []string, expectedSuccessStatus govtypesv1beta1.ProposalStatus) {
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

func (s *IntegrationTestSuite) writeFeemarketParamChangeProposal(c *chain, params feemarkettypes.Params) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	template := `
	{
		"messages":[
		  {
			"@type": "/atomone.feemarket.v1.MsgUpdateParams",
			"authority": "%s",
			"params": %s
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing feemarket param change",
		"metadata": "",
		"title": "Change in feemarket params",
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

func (s *IntegrationTestSuite) writeGovLawProposal(c *chain) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	template := `
	{
	 "messages": [
		{
		 "@type": "/atomone.gov.v1.MsgProposeLaw",
		 "authority": "%s"
		}
	 ],
	 "deposit": "%s",
	 "metadata": "New Law",
	 "title": "New Law",
	 "summary": "This is the summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, initialDepositAmount)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalLawFilename), []byte(propMsgBody))
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
