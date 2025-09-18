package e2e

import (
	"fmt"
	"strconv"

	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
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
	oversiteDAOAddress, err := s.chainA.multiSigAccounts[1].keyInfo.GetAddress()
	s.Require().NoError(err)
	params.Params.SteeringDaoAddress = steeringDAOAddress.String()
	params.Params.OversightDaoAddress = oversiteDAOAddress.String()
	s.writeCoreDAOsParamChangeProposal(s.chainA, params.Params)
	// Gov tests may be run in arbitrary order, each test must increment proposalCounter to have the correct proposal id to submit and query
	proposalCounter++
	submitGovFlags := []string{configFile(proposalParamChangeFilename)}
	depositGovFlags := []string{strconv.Itoa(proposalCounter), s.queryGovMinDeposit(chainAAPIEndpoint).String()}
	voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
	s.submitGovProposal(chainAAPIEndpoint, senderAddress.String(), proposalCounter, "atomone.coredaos.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

	newParams := s.queryCoreDAOsParams(chainAAPIEndpoint)
	s.Require().Equal(newParams.Params.SteeringDaoAddress, steeringDAOAddress.String())
	s.Require().Equal(newParams.Params.OversightDaoAddress, oversiteDAOAddress.String())

	s.execBankMultiSend(s.chainA, valIdx, senderAddress.String(),
		[]string{steeringDAOAddress.String(), oversiteDAOAddress.String()},
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
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalCounter, "submit-proposal", submitGovFlags, govtypesv1beta1.StatusDepositPeriod)
	s.submitGovCommand(chainAAPIEndpoint, sender, proposalCounter, "deposit", depositGovFlags, govtypesv1beta1.StatusVotingPeriod)
	return proposalCounter
}
