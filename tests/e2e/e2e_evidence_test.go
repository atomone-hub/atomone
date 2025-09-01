package e2e

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/x/evidence/exported"
	evidencetypes "cosmossdk.io/x/evidence/types"
)

func (s *IntegrationTestSuite) testEvidenceQueries() {
	s.Run("queries", func() {
		var (
			valIdx   = 0
			chain    = s.chainA
			chainAPI = fmt.Sprintf("http://%s", s.valResources[chain.id][valIdx].GetHostPort("1317/tcp"))
		)
		res, err := s.queryAllEvidence(chainAPI)
		s.Require().NoError(err)
		s.Require().Equal(numberOfEvidences, len(res.Evidence))
		for _, evidence := range res.Evidence {
			var exportedEvidence exported.Evidence
			err := s.cdc.UnpackAny(evidence, &exportedEvidence)
			s.Require().NoError(err)
			eq, ok := exportedEvidence.(*evidencetypes.Equivocation)
			s.Require().True(ok)
			s.execQueryEvidence(chain, valIdx, string(eq.Hash())) // TODO: check this string conversion was good
		}
	})
}

func (s *IntegrationTestSuite) execQueryEvidence(c *chain, valIdx int, hash string) (res evidencetypes.Equivocation) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s.T().Logf("querying evidence %X on chain %s", hash, c.id)

	atomoneCommand := []string{
		atomonedBinary,
		queryCommand,
		evidencetypes.ModuleName,
		hash,
	}

	s.executeAtomoneTxCommand(ctx, c, atomoneCommand, valIdx, func(stdOut []byte, stdErr []byte) error {
		// TODO parse evidence after fix the SDK
		// https://github.com/cosmos/cosmos-sdk/issues/13444
		// s.Require().NoError(yaml.Unmarshal(stdOut, &res))
		return nil
	})
	return res
}
