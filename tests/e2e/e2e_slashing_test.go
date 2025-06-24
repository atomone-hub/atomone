package e2e

const jailedValidatorKey = "jailed"

func (s *IntegrationTestSuite) testSlashing(chainEndpoint string) {
	s.Run("unjail validator", func() {
		validators, err := s.queryValidators(chainEndpoint)
		s.Require().NoError(err)

		for _, val := range validators {
			if val.Jailed {
				s.execUnjail(
					s.chainA,
					withKeyValue(flagFrom, jailedValidatorKey),
				)

				valQ, err := s.queryValidator(chainEndpoint, val.OperatorAddress)
				s.Require().NoError(err)
				s.Require().False(valQ.Jailed)
			}
		}
	})
}
