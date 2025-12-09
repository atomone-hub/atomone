package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	dynamicfeetypes "github.com/atomone-hub/atomone/x/dynamicfee/types"
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
		upgradeHeight := height + govProposalBlockBuffer
		s.writeGovSoftwareUpgradeProposal(s.chainA, upgradeHeight)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++

		submitGovFlags := []string{configFile(proposalSoftwareUpgradeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}

		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes=0.8,no=0.1,abstain=0.1"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "SoftwareUpgrade", submitGovFlags, depositGovFlags, voteGovFlags, "weighted-vote", govtypesv1beta1.StatusPassed)

		res := s.queryUpgradePlan(chainAAPIEndpoint)
		s.Require().Equal("v2", res.Plan.Name)
		s.Require().Equal(upgradeHeight, res.Plan.Height)

		s.verifyChainHaltedAtUpgradeHeight(s.chainA, 0, upgradeHeight)
		s.T().Logf("Successfully halted chain at height %d", upgradeHeight)

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
		s.writeGovSoftwareUpgradeProposal(s.chainA, proposalHeight)

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalSoftwareUpgradeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "SoftwareUpgrade", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		proposalCounter++
		s.writeGovCancelUpgradeProposal(s.chainA)
		submitGovFlags = []string{configFile(proposalCancelUpgradeFilename)}
		depositGovFlags = []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags = []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "CancelUpgrade", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

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

		beforeSenderBalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeRecipientBalance := s.queryBalance(chainAAPIEndpoint, recipient, uatoneDenom)

		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalCommunitySpendFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "CommunityPoolSpend", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		// Check that sender is refunded with the proposal deposit
		s.Require().Eventually(
			func() bool {
				afterSenderBalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
				return afterSenderBalance.IsEqual(beforeSenderBalance)
			},
			10*time.Second,
			time.Second,
		)
		// Check that recipient received the community pool spend
		s.Require().Eventually(
			func() bool {
				afterRecipientBalance := s.queryBalance(chainAAPIEndpoint, recipient, uatoneDenom)
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

		beforeSenderBalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
		beforeRecipientBalance := s.queryBalance(chainAAPIEndpoint, recipient, uatoneDenom)

		initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
		deposit := s.queryGovMinDeposit(chainAAPIEndpoint)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalCommunitySpendFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "no"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "CommunityPoolSpend", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusRejected)

		// Check that sender is not refunded with the proposal deposit
		s.Require().Eventually(
			func() bool {
				afterSenderBalance := s.queryBalance(chainAAPIEndpoint, sender, uatoneDenom)
				return afterSenderBalance.Add(deposit).Add(initialDeposit).
					IsEqual(beforeSenderBalance)
			},
			10*time.Second,
			time.Second,
		)
		// Check that recipient didnt receive the community pool spend since the
		// proposal was rejected
		s.Require().Eventually(
			func() bool {
				afterRecipientBalance := s.queryBalance(chainAAPIEndpoint, recipient, uatoneDenom)
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
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
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
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.photon.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		newParams := s.queryPhotonParams(chainAAPIEndpoint)
		s.Assert().True(newParams.Params.MintDisabled, "expected photon param mint disabled to be true")

		// Revert change or mint photon test will fail
		params.Params.MintDisabled = false
		proposalCounter++
		depositGovFlags = []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags = []string{strconv.Itoa(proposalCounter), "yes"}
		s.writePhotonParamChangeProposal(s.chainA, params.Params)
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.photon.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		newParams = s.queryPhotonParams(chainAAPIEndpoint)
		s.Require().False(newParams.Params.MintDisabled, "expected photon param mint disabled to be false")
	})
	s.Run("dynamicfee param change", func() {
		// check existing params
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		params := s.queryDynamicfeeParams(chainAAPIEndpoint)

		oldAlpha := params.Params.Alpha
		params.Params.Alpha = oldAlpha.Add(math.LegacyNewDec(1))

		s.writeDynamicfeeParamChangeProposal(s.chainA, params.Params)
		// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
		proposalCounter++
		submitGovFlags := []string{configFile(proposalParamChangeFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "atomone.dynamicfee.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		newParams := s.queryDynamicfeeParams(chainAAPIEndpoint)
		s.Require().Equal(newParams.Params.Alpha, oldAlpha.Add(math.LegacyNewDec(1)))
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
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
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

func (s *IntegrationTestSuite) testGovTextProposal() {
	s.Run("text proposal pass", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		s.writeTextProposal(s.chainA)
		proposalCounter++
		submitGovFlags := []string{configFile(proposalTextFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "Text", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)
	})
}

func (s *IntegrationTestSuite) testGovDynamicQuorum() {
	s.Run("dynamic quorum change", func() {
		// From the formulae in ADR-005
		// pEma = (Q - Qmin) / (Qmax - Qmin)
		// Participation = (PEma_t - (PEma_{t-1} * 0.8)/0.2
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		params := s.queryGovParams(chainAAPIEndpoint, "tallying")
		quorums := s.queryGovQuorums(chainAAPIEndpoint)
		quorumRange := params.GetParams().QuorumRange
		quorumMin := math.LegacyMustNewDecFromStr(quorumRange.Min)
		quorumMax := math.LegacyMustNewDecFromStr(quorumRange.Max)
		currentQuorum := math.LegacyMustNewDecFromStr(quorums.GetQuorum())
		quorumPEma := (currentQuorum.Sub(quorumMin)).Quo(quorumMax.Sub(quorumMin))
		s.writeTextProposal(s.chainA)
		proposalCounter++
		submitGovFlags := []string{configFile(proposalTextFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "Text", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)
		quorumsAfter := s.queryGovQuorums(chainAAPIEndpoint)
		endQuorum := math.LegacyMustNewDecFromStr(quorumsAfter.GetQuorum())
		endQuorumPEma := (endQuorum.Sub(quorumMin)).Quo(quorumMax.Sub(quorumMin))
		expectedParticipation := endQuorumPEma.Sub(quorumPEma.Mul(math.LegacyMustNewDecFromStr("0.8"))).Quo(math.LegacyMustNewDecFromStr("0.2"))
		proposal, _ := s.queryGovProposal(chainAAPIEndpoint, proposalCounter)
		stakingPool := s.queryStakingPool(chainAAPIEndpoint)
		votes := proposal.Proposal.FinalTallyResult.Yes.ToLegacyDec()
		totalVP := stakingPool.Pool.BondedTokens.ToLegacyDec()
		actualParticipation := votes.Quo(totalVP)

		s.Require().True(actualParticipation.Equal(expectedParticipation))
		s.Require().Equal(quorumsAfter.LawQuorum, quorums.LawQuorum)
		s.Require().Equal(quorumsAfter.ConstitutionAmendmentQuorum, quorums.ConstitutionAmendmentQuorum)
	})

	s.Run("dynamic law quorum change", func() {
		// From the formulae in ADR-005
		// pEma = (Q - Qmin) / (Qmax - Qmin)
		// Participation = (PEma_t - (PEma_{t-1} * 0.8)/0.2
		chainAAPIEndpoint := fmt.Sprintf("http://%s",
			s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		params := s.queryGovParams(chainAAPIEndpoint, "tallying")
		quorums := s.queryGovQuorums(chainAAPIEndpoint)
		lawQuorumRange := params.GetParams().LawQuorumRange
		lawQuorumMin := math.LegacyMustNewDecFromStr(lawQuorumRange.Min)
		lawQuorumMax := math.LegacyMustNewDecFromStr(lawQuorumRange.Max)
		currentLawQuorum := math.LegacyMustNewDecFromStr(quorums.GetLawQuorum())
		lawQuorumPEma := (currentLawQuorum.Sub(lawQuorumMin)).Quo(lawQuorumMax.Sub(lawQuorumMin))
		s.writeGovLawProposal(s.chainA)
		proposalCounter++
		submitGovFlags := []string{configFile(proposalLawFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter,
			"gov/MsgSubmitProposal", submitGovFlags, depositGovFlags, voteGovFlags,
			"vote", govtypesv1beta1.StatusPassed)
		quorumsAfter := s.queryGovQuorums(chainAAPIEndpoint)
		endLawQuorum := math.LegacyMustNewDecFromStr(quorumsAfter.GetLawQuorum())
		endLawQuorumPEma := (endLawQuorum.Sub(lawQuorumMin)).Quo(lawQuorumMax.Sub(lawQuorumMin))
		expectedParticipation := endLawQuorumPEma.Sub(lawQuorumPEma.Mul(math.LegacyMustNewDecFromStr("0.8"))).Quo(math.LegacyMustNewDecFromStr("0.2"))
		proposal, _ := s.queryGovProposal(chainAAPIEndpoint, proposalCounter)
		stakingPool := s.queryStakingPool(chainAAPIEndpoint)
		votes := proposal.Proposal.FinalTallyResult.Yes.ToLegacyDec()
		totalVP := stakingPool.Pool.BondedTokens.ToLegacyDec()
		actualParticipation := votes.Quo(totalVP)

		s.Require().True(actualParticipation.Equal(expectedParticipation), "actual participation does not match expected participation", actualParticipation.String(), expectedParticipation.String())
		s.Require().Equal(quorumsAfter.Quorum, quorums.Quorum, "quorum does not match expected quorum", quorumsAfter.Quorum, quorums.Quorum)
		s.Require().Equal(quorumsAfter.ConstitutionAmendmentQuorum,
			quorums.ConstitutionAmendmentQuorum, "constitution amendment quorum does not match expected quorum", quorumsAfter.ConstitutionAmendmentQuorum, quorums.ConstitutionAmendmentQuorum)
	})

	s.Run("dynamic constitution amendment quorum change", func() {
		// From the formulae in ADR-005
		// pEma = (Q - Qmin) / (Qmax - Qmin)
		// Participation = (PEma_t - (PEma_{t-1} * 0.8)/0.2
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		params := s.queryGovParams(chainAAPIEndpoint, "tallying")
		quorums := s.queryGovQuorums(chainAAPIEndpoint)
		constitutionAmendmentQuorumRange := params.GetParams().ConstitutionAmendmentQuorumRange
		constitutionAmendmentQuorumMin := math.LegacyMustNewDecFromStr(constitutionAmendmentQuorumRange.Min)
		constitutionAmendmentQuorumMax := math.LegacyMustNewDecFromStr(constitutionAmendmentQuorumRange.Max)
		currentConstitutionAmendmentQuorum := math.LegacyMustNewDecFromStr(quorums.GetConstitutionAmendmentQuorum())
		constitutionAmendmentQuorumPEma := (currentConstitutionAmendmentQuorum.Sub(constitutionAmendmentQuorumMin)).Quo(constitutionAmendmentQuorumMax.Sub(constitutionAmendmentQuorumMin))
		newConstitution := "New test constitution 2"
		amendmentMsg := s.generateConstitutionAmendment(s.chainA, newConstitution)
		s.writeGovConstitutionAmendmentProposal(s.chainA, amendmentMsg.Amendment)
		proposalCounter++
		submitGovFlags := []string{configFile(proposalConstitutionAmendmentFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovProposal(chainAAPIEndpoint, sender, proposalCounter, "gov/MsgSubmitProposal", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)
		quorumsAfter := s.queryGovQuorums(chainAAPIEndpoint)
		endConstitutionAmendmentQuorum := math.LegacyMustNewDecFromStr(quorumsAfter.GetConstitutionAmendmentQuorum())
		endConstitutionAmendmentQuorumPEma := (endConstitutionAmendmentQuorum.Sub(constitutionAmendmentQuorumMin)).Quo(constitutionAmendmentQuorumMax.Sub(constitutionAmendmentQuorumMin))
		expectedParticipation := endConstitutionAmendmentQuorumPEma.Sub(constitutionAmendmentQuorumPEma.Mul(math.LegacyMustNewDecFromStr("0.8"))).Quo(math.LegacyMustNewDecFromStr("0.2"))
		proposal, _ := s.queryGovProposal(chainAAPIEndpoint, proposalCounter)
		stakingPool := s.queryStakingPool(chainAAPIEndpoint)
		votes := proposal.Proposal.FinalTallyResult.Yes.ToLegacyDec()
		totalVP := stakingPool.Pool.BondedTokens.ToLegacyDec()
		actualParticipation := votes.Quo(totalVP)

		s.Require().True(actualParticipation.Equal(expectedParticipation), "actual participation does not match expected participation", actualParticipation.String(), expectedParticipation.String())
		s.Require().Equal(quorumsAfter.Quorum, quorums.Quorum, "quorum does not match expected quorum", quorumsAfter.Quorum, quorums.Quorum)
		s.Require().Equal(quorumsAfter.LawQuorum, quorums.LawQuorum, "law quorum does not match expected law quorum", quorumsAfter.LawQuorum, quorums.LawQuorum)
	})
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
	s.T().Logf("Waiting chain halt at height %d", upgradeHeight)
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

func (s *IntegrationTestSuite) submitGovCommand(chainAAPIEndpoint, sender string, proposalID int, govCommand string, proposalFlags []string, expectedStatus govtypesv1beta1.ProposalStatus) {
	s.runGovExec(s.chainA, 0, sender, govCommand, proposalFlags)

	s.T().Logf("Waiting for proposal status %s", expectedStatus.String())
	s.Require().EventuallyWithT(
		func(c *assert.CollectT) {
			res, err := s.queryGovProposal(chainAAPIEndpoint, proposalID)
			require.NoError(c, err)
			assert.Equal(c, res.GetProposal().Status.String(), expectedStatus.String())
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

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
	propMsgBody := fmt.Sprintf(template, govModuleAddress, s.cdc.MustMarshalJSON(&params), initialDeposit)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalParamChangeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) writeCoreDAOsParamChangeProposal(c *chain, params coredaostypes.Params) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	template := `
	{
		"messages": [
		{
		 "@type": "/atomone.coredaos.v1.MsgUpdateParams",
		 "authority": "%s",
		 "params": %s
		}
		],
		"deposit": "%s",
		"metadata": "",
		"title": "Set DAO params",
		"summary": "Set DAO params"
	}
	`
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
	propMsgBody := fmt.Sprintf(template, govModuleAddress, s.cdc.MustMarshalJSON(&params), initialDeposit)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalParamChangeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) writeDynamicfeeParamChangeProposal(c *chain, params dynamicfeetypes.Params) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	template := `
	{
		"messages":[
		  {
			"@type": "/atomone.dynamicfee.v1.MsgUpdateParams",
			"authority": "%s",
			"params": %s
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing dynamicfee param change",
		"metadata": "",
		"title": "Change in dynamicfee params",
		"summary": "summary"
	}
	`
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
	propMsgBody := fmt.Sprintf(template, govModuleAddress, s.cdc.MustMarshalJSON(&params), initialDeposit)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalParamChangeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) writeTextProposal(c *chain) {
	template := `
	{
		"deposit": "%s",
		"metadata": "some metadata",
		"title": "Text Proposal",
		"summary": "summary"
	}
	`
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
	propMsgBody := fmt.Sprintf(template, initialDeposit)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalTextFilename), []byte(propMsgBody))
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
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
	propMsgBody := fmt.Sprintf(template, govModuleAddress, s.cdc.MustMarshalJSON(&params), initialDeposit)
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
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
	propMsgBody := fmt.Sprintf(template, govModuleAddress, amendment, initialDeposit)
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
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)
	propMsgBody := fmt.Sprintf(template, govModuleAddress, initialDeposit)
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

func (s *IntegrationTestSuite) parseGenerateConstitutionAmendmentOutput(msg *govtypesv1.MsgProposeConstitutionAmendment) func([]byte, []byte) error {
	return func(stdOut []byte, stdErr []byte) error {
		if len(stdErr) > 0 {
			return fmt.Errorf("stdErr: %s", string(stdErr))
		}
		return s.cdc.UnmarshalJSON(stdOut, msg)
	}
}

func (s *IntegrationTestSuite) writeGovCommunitySpendProposal(c *chain, amount sdk.Coin, recipient string) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)

	template := `
	{
		"messages":[
		  {
			"@type": "/cosmos.distribution.v1beta1.MsgCommunityPoolSpend",
			"authority": "%s",
			"recipient": "%s",
			"amount": [{
				"denom": "%s",
				"amount": "%s"
			}]
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing validator address",
		"metadata": "Community Pool Spend",
		"title": "Fund Team!",
		"summary": "summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, recipient,
		amount.Denom, amount.Amount.String(), initialDeposit.String())
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalCommunitySpendFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) writeGovSoftwareUpgradeProposal(c *chain, height int64) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)

	template := `
	{
		"messages":[
		  {
			"@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
			"authority": "%s",
			"plan": {
				"name": "v2",
				"time": "0001-01-01T00:00:00Z",
				"height": "%d",
				"info": "",
				"upgraded_client_state": null
			}
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing validator address",
		"metadata": "Software Upgrade Proposal",
		"title": "Upgrade Team!",
		"summary": "summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, height, initialDeposit.String())
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalSoftwareUpgradeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) writeGovCancelUpgradeProposal(c *chain) {
	govModuleAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	initialDeposit := s.queryGovMinInitialDeposit(chainAAPIEndpoint)

	template := `
	{
		"messages":[
		  {
			"@type": "/cosmos.upgrade.v1beta1.MsgCancelUpgrade",
			"authority": "%s"
		  }
		],
		"deposit": "%s",
		"proposer": "Proposing validator address",
		"metadata": "Cancel Upgrade Proposal",
		"title": "Cancel Team!",
		"summary": "summary"
	}
	`
	propMsgBody := fmt.Sprintf(template, govModuleAddress, initialDeposit.String())
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalCancelUpgradeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}

func configFile(filename string) string {
	filepath := filepath.Join(atomoneConfigPath, filename)
	return filepath
}
