package e2e

import (
	"fmt"
	"path/filepath"
	"strconv"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	coredaostypes "github.com/atomone-hub/atomone/x/coredaos/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	govtypesv1beta1 "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

func (s *IntegrationTestSuite) testCoreDAOs() {
	s.Run("dao parameter change", func() {
		chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
		senderAddress, _ := s.chainA.validators[0].keyInfo.GetAddress()
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
