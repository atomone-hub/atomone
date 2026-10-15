package e2e

import (
	"fmt"
	"strconv"
	"time"

	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) testCoreDAOs() {
	valIdx := 0
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	senderAddress, _ := s.chainA.validators[valIdx].keyInfo.GetAddress()

	params := s.queryCoreDAOsParams(chainAAPIEndpoint)

	steeringDAOAddress, err := s.chainA.multiSigAccounts[0].keyInfo.GetAddress()
	s.Require().NoError(err)
	oversightDAOAddress, err := s.chainA.multiSigAccounts[1].keyInfo.GetAddress()
	s.Require().NoError(err)
	params.Params.SteeringDaoAddress = steeringDAOAddress.String()
	params.Params.OversightDaoAddress = oversightDAOAddress.String()
	s.writeCoreDAOsParamChangeProposal(s.chainA, params.Params)
	// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
	proposalCounter++
	submitGovFlags := []string{configFile(proposalParamChangeFilename)}
	depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
	voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
	s.submitGovProposal(chainAAPIEndpoint, senderAddress.String(), proposalCounter, "atomone.coredaos.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1.StatusPassed)

	newParams := s.queryCoreDAOsParams(chainAAPIEndpoint)
	s.Require().Equal(newParams.Params.SteeringDaoAddress, steeringDAOAddress.String())
	s.Require().Equal(newParams.Params.OversightDaoAddress, oversightDAOAddress.String())

	s.execBankMultiSend(s.chainA, valIdx, senderAddress.String(),
		[]string{steeringDAOAddress.String(), oversightDAOAddress.String()},
		sdk.NewCoins(
			tokenAmount,
			sdk.NewInt64Coin(uphotonDenom, 100_000_000),
		).String(),
		false,
	)

	s.Run("coredaos annotation", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		steeringDAOAccount := s.chainA.multiSigAccounts[0]

		proposalID := s.submitVotingPeriodLawProposal(s.chainA)

		atomoneCommand := []string{
			atomonedBinary,
			txCommand,
			coredaostypes.ModuleName,
			"annotate",
			strconv.FormatInt(int64(proposalID), 10),
			"Proposal Annotation",
		}
		s.executeMultiSigTxCommand(s.chainA, atomoneCommand, valIdx, steeringDAOAccount, false)
		proposal := s.queryGovV1Proposal(chainAAPIEndpoint, proposalCounter)
		s.Require().Equal("Proposal Annotation", proposal.Proposal.Annotation)
	})

	s.Run("coredaos extend voting period", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		steeringDAOAccount := s.chainA.multiSigAccounts[0]

		proposalID := s.submitVotingPeriodLawProposal(s.chainA)
		proposalBeforeExtension := s.queryGovV1Proposal(chainAAPIEndpoint, proposalID)

		atomoneCommand := []string{
			atomonedBinary,
			txCommand,
			coredaostypes.ModuleName,
			"extend-voting-period",
			strconv.FormatInt(int64(proposalID), 10),
		}
		s.executeMultiSigTxCommand(s.chainA, atomoneCommand, valIdx, steeringDAOAccount, false)
		proposalAfterExtension := s.queryGovV1Proposal(chainAAPIEndpoint, proposalID)

		extendedVotingPeriod := proposalBeforeExtension.Proposal.VotingEndTime.Before(*proposalAfterExtension.Proposal.VotingEndTime)
		s.Require().True(extendedVotingPeriod)

		s.Require().Equal(uint32(0), proposalBeforeExtension.Proposal.TimesVotingPeriodExtended)
		s.Require().Equal(uint32(1), proposalAfterExtension.Proposal.TimesVotingPeriodExtended)
	})

	s.Run("coredaos endorse", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		steeringDAOAccount := s.chainA.multiSigAccounts[0]

		proposalID := s.submitVotingPeriodLawProposal(s.chainA)
		proposalBeforeEndorsement := s.queryGovV1Proposal(chainAAPIEndpoint, proposalID)

		atomoneCommand := []string{
			atomonedBinary,
			txCommand,
			coredaostypes.ModuleName,
			"endorse",
			strconv.FormatInt(int64(proposalID), 10),
		}
		s.executeMultiSigTxCommand(s.chainA, atomoneCommand, valIdx, steeringDAOAccount, false)
		proposalAfterEndorsement := s.queryGovV1Proposal(chainAAPIEndpoint, proposalID)

		s.Require().False(proposalBeforeEndorsement.Proposal.Endorsed)
		s.Require().True(proposalAfterEndorsement.Proposal.Endorsed)
	})

	s.Run("coredaos veto", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		oversiteDAOAccount := s.chainA.multiSigAccounts[1]

		proposalID := s.submitVotingPeriodLawProposal(s.chainA)
		proposalBeforeVeto := s.queryGovV1Proposal(chainAAPIEndpoint, proposalID)

		// A distinct depositor tops up the proposal during its voting period.
		// The veto's deposit cleanup is deferred to the coredaos EndBlocker, and
		// this deposit must be refunded there — this exercises the deferred
		// cleanup path on a live chain (see the gas-exhaustion fix).
		depositor, err := s.chainA.genesisAccounts[3].keyInfo.GetAddress()
		s.Require().NoError(err)
		minDeposit := s.queryGovMinDeposit(chainAAPIEndpoint)
		s.submitGovCommand(chainAAPIEndpoint, depositor.String(), proposalID, "deposit",
			[]string{strconv.Itoa(proposalID), minDeposit.String()}, govtypesv1.StatusVotingPeriod)

		// submitGovCommand only returns after the deposit tx is committed, so the
		// depositor's balance already reflects the locked deposit (and its fee).
		balanceAfterDeposit := s.queryBalance(chainAAPIEndpoint, depositor.String(), uatoneDenom)

		atomoneCommand := []string{
			atomonedBinary,
			txCommand,
			coredaostypes.ModuleName,
			"veto",
			strconv.FormatInt(int64(proposalID), 10),
			"false",
		}
		s.executeMultiSigTxCommand(s.chainA, atomoneCommand, valIdx, oversiteDAOAccount, false)
		proposalAfterVeto := s.queryGovV1Proposal(chainAAPIEndpoint, proposalID)

		s.Require().Equal(govtypesv1.StatusVotingPeriod, proposalBeforeVeto.Proposal.Status)
		s.Require().Equal(govtypesv1.StatusVetoed, proposalAfterVeto.Proposal.Status)

		// The deferred EndBlocker cleanup must refund the depositor in full.
		s.Require().Eventually(func() bool {
			bal := s.queryBalance(chainAAPIEndpoint, depositor.String(), uatoneDenom)
			return bal.Equal(balanceAfterDeposit.Add(minDeposit))
		}, 30*time.Second, 2*time.Second)
	})

	s.Run("coredaos cannot stake", func() {
		oversiteDAOAccount := s.chainA.multiSigAccounts[1]
		validatorA := s.chainA.validators[0]
		validatorAAddr, _ := validatorA.keyInfo.GetAddress()
		validatorAddressA := sdk.ValAddress(validatorAAddr).String()

		atomoneCommand := []string{
			atomonedBinary,
			txCommand,
			stakingtypes.ModuleName,
			"delegate",
			validatorAddressA,
			tokenAmount.String(),
		}
		s.executeMultiSigTxCommand(s.chainA, atomoneCommand, valIdx, oversiteDAOAccount, true)
	})
}

// Submits a law proposal that stays in voting period
func (s *IntegrationTestSuite) submitVotingPeriodLawProposal(c *chain) int {
	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	s.writeGovLawProposal(s.chainA)
	proposalCounter++
	submitGovFlags := []string{configFile(proposalLawFilename)}
	depositGovFlags := []string{strconv.Itoa(proposalCounter)}
	deposit := s.queryGovMinDeposit(chainAAPIEndpoint)
	depositString := deposit.String()
	depositGovFlags = append(depositGovFlags, depositString)
	senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
	sender := senderAddress.String()
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalCounter, "submit-proposal", submitGovFlags, govtypesv1.StatusDepositPeriod)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalCounter, "deposit", depositGovFlags, govtypesv1.StatusVotingPeriod)
	return proposalCounter
}
