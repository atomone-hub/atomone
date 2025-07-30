package e2e

import (
	"context"
	"fmt"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (s *IntegrationTestSuite) testCoreDAOs() {
	valIdx := 0
	s.Run("dao parameter change", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[valIdx].keyInfo.GetAddress()
		accountsNumber := 2
		signersNumber := 3
		thereshold := 2
		s.chainA.addMultiSigAccountFromMnemonic(accountsNumber, signersNumber, thereshold)

		params, err := queryCoreDAOsParams(chainAAPIEndpoint)
		s.Require().NoError(err)

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
		depositGovFlags := []string{strconv.Itoa(proposalCounter), depositAmount.String()}
		voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		s.submitGovProposal(chainAAPIEndpoint, senderAddress.String(), proposalCounter, "atomone.coredaos.v1.MsgUpdateParams", submitGovFlags, depositGovFlags, voteGovFlags, "vote", govtypesv1beta1.StatusPassed)

		newParams, err := queryCoreDAOsParams(chainAAPIEndpoint)
		s.Require().NoError(err)
		s.Require().Equal(newParams.Params.SteeringDaoAddress, steeringDAOAddress.String())
		s.Require().Equal(newParams.Params.OversightDaoAddress, oversiteDAOAddress.String())

		s.execBankSend(s.chainA, 0, senderAddress.String(), steeringDAOAddress.String(), sdk.NewInt64Coin(uphotonDenom, 100_000_000).String(), false)
		s.execBankSend(s.chainA, 0, senderAddress.String(), oversiteDAOAddress.String(), tokenAmount.String(), false)

	})
	s.Run("dao annotation", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		//Create a proposal that stays in voting period
		s.writeTextProposal(s.chainA)
		proposalCounter++
		submitGovFlags := []string{configFile(proposalTextFilename)}
		depositGovFlags := []string{strconv.Itoa(proposalCounter), sdk.NewInt64Coin(uatoneDenom, 1_000_000_000).String()}
		//voteGovFlags := []string{strconv.Itoa(proposalCounter), "yes"}
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
		sender := senderAddress.String()
		s.submitGovCommand(chainAAPIEndpoint, sender, proposalCounter, "submit-proposal", submitGovFlags, govtypesv1beta1.StatusDepositPeriod)
		s.submitGovCommand(chainAAPIEndpoint, sender, proposalCounter, "deposit", depositGovFlags, govtypesv1beta1.StatusVotingPeriod)

		msgAnnotationFile := "msgAnnotation.json"
		steeringDAOAddress, err := s.chainA.multiSigAccounts[0].keyInfo.GetAddress()
		steeringDAOKeyName := s.chainA.multiSigAccounts[0].keyInfo.Name
		s.Require().NoError(err)

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		// Generate offline coredaos annotation transaction
		opt := []flagOption{withKeyValue(flagFrom, steeringDAOKeyName)}
		opts := applyOptions(s.chainA.id, opt)
		atomoneCommand := []string{
			atomonedBinary,
			txCommand,
			coredaostypes.ModuleName,
			"annotate",
			strconv.FormatInt(int64(proposalCounter), 10),
			"Proposal Annotation",
		}
		for flag, value := range opts {
			atomoneCommand = append(atomoneCommand, fmt.Sprintf("--%s=%v", flag, value))
		}
		atomoneCommand = append(atomoneCommand, "--generate-only")
		//atomoneCommand = append(atomoneCommand, fmt.Sprintf("> %s", msgAnnotationFile))
		fmt.Println("Command sent: ", strings.Join(atomoneCommand, " "))
		s.executeAtomoneTxCommand(ctx, s.chainA, atomoneCommand, valIdx, s.noValidationStoreOutput(s.chainA, valIdx, msgAnnotationFile))

		// Sign offline tx
		for idx, signer := range s.chainA.multiSigAccounts[0].signers {
			signerAddress, err := signer.keyInfo.GetAddress()
			s.Require().NoError(err)
			opt = []flagOption{withKeyValue(flagMultisig, steeringDAOAddress.String())}
			opt = append(opt, withKeyValue(flagFrom, signerAddress.String()))
			opt = append(opt, withKeyValue(flagOutputDocument, configFile(fmt.Sprintf("signed_%v_%s", idx, msgAnnotationFile))))
			opts = applyOptions(s.chainA.id, opt)
			atomoneCommand = []string{
				atomonedBinary,
				txCommand,
				"sign",
				configFile(msgAnnotationFile),
			}
			for flag, value := range opts {
				atomoneCommand = append(atomoneCommand, fmt.Sprintf("--%s=%v", flag, value))
			}
			fmt.Println("Command sent: ", strings.Join(atomoneCommand, " "))
			s.executeAtomoneTxCommand(ctx, s.chainA, atomoneCommand, valIdx, s.noValidationStoreOutput(s.chainA, valIdx, ""))
		}

		// Combine Signatures
		atomoneCommand = []string{
			atomonedBinary,
			txCommand,
			"multisign",
			configFile(msgAnnotationFile),
			s.chainA.multiSigAccounts[0].keyInfo.Name,
		}
		for idx := range s.chainA.multiSigAccounts[0].signers {
			atomoneCommand = append(atomoneCommand, configFile(fmt.Sprintf("signed_%v_%s", idx, msgAnnotationFile)))
		}
		atomoneCommand = append(atomoneCommand, fmt.Sprintf("--%s=%v", flagKeyringBackend, "test"))
		atomoneCommand = append(atomoneCommand, fmt.Sprintf("--%s=%v", flagChainID, s.chainA.id))
		atomoneCommand = append(atomoneCommand, fmt.Sprintf("--%s=%v", flagHome, atomoneHomePath))
		// atomoned tx multisign msgAnnotation.json  multisig-0 signed_0_msgAnnotation.json signed_1_msgAnnotation.json signed_2_msgAnnotation.json --keyring-backend=test --output=json --chain-id=chain-QH2e0a --home=/home/nonroot/.atomone
		fmt.Println("Command sent: ", strings.Join(atomoneCommand, " "))
		s.executeAtomoneTxCommand(ctx, s.chainA, atomoneCommand, valIdx, s.noValidationStoreOutput(s.chainA, valIdx, fmt.Sprintf("signed_%s", msgAnnotationFile)))

		// Broadcast tx
		opt = []flagOption{withKeyValue(flagFrom, steeringDAOKeyName)}
		opts = applyOptions(s.chainA.id, opt)
		atomoneCommand = []string{
			atomonedBinary,
			txCommand,
			"broadcast",
			configFile(fmt.Sprintf("signed_%s", msgAnnotationFile)),
		}
		for flag, value := range opts {
			atomoneCommand = append(atomoneCommand, fmt.Sprintf("--%s=%v", flag, value))
		}
		s.executeAtomoneTxCommand(ctx, s.chainA, atomoneCommand, valIdx, s.expectErrExecValidation(s.chainA, valIdx, false))

		proposal, err := queryGovProposal(chainAAPIEndpoint, proposalCounter)
		s.Require().NoError(err)
		s.Require().Equal("Proposal Annotation", proposal.Proposal.Annotation)
	})
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

	propMsgBody := fmt.Sprintf(template, govModuleAddress, cdc.MustMarshalJSON(&params), initialDepositAmount)
	err := writeFile(filepath.Join(c.validators[0].configDir(), "config", proposalParamChangeFilename), []byte(propMsgBody))
	s.Require().NoError(err)
}
