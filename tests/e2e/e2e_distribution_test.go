package e2e

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func (s *IntegrationTestSuite) testDistribution() {
	s.Run("distribution", func() {
		chainEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))

		validatorB := s.chainA.validators[1]
		validatorBAddr, _ := validatorB.keyInfo.GetAddress()

		valOperAddressA := sdk.ValAddress(validatorBAddr).String()

		delegatorAddress, _ := s.chainA.genesisAccounts[2].keyInfo.GetAddress()

		newWithdrawalAddress, _ := s.chainA.genesisAccounts[3].keyInfo.GetAddress()

		beforeBalance := s.queryBalance(chainEndpoint, newWithdrawalAddress.String(), uatoneDenom)

		s.execSetWithdrawAddress(s.chainA, 0, delegatorAddress.String(), newWithdrawalAddress.String(), atomoneHomePath)

		// Verify
		s.Require().Eventually(
			func() bool {
				res, err := s.queryDelegatorWithdrawalAddress(chainEndpoint, delegatorAddress.String())
				s.Require().NoError(err)

				return res.WithdrawAddress == newWithdrawalAddress.String()
			},
			10*time.Second,
			time.Second,
		)

		s.execWithdrawReward(s.chainA, 0, delegatorAddress.String(), valOperAddressA, atomoneHomePath)
		s.Require().Eventually(
			func() bool {
				afterBalance := s.queryBalance(chainEndpoint, newWithdrawalAddress.String(), uatoneDenom)
				return afterBalance.IsGTE(beforeBalance)
			},
			10*time.Second,
			time.Second,
		)
	})
}

/*
fundCommunityPool tests the funding of the community pool on behalf of the distribution module.
Test Benchmarks:
1. Validation that balance of the distribution module account before funding
2. Execution funding the community pool
3. Verification that correct funds have been deposited to distribution module account
*/
func (s *IntegrationTestSuite) fundCommunityPool() {
	distModuleAddress := authtypes.NewModuleAddress(distrtypes.ModuleName).String()

	chainAAPIEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))
	sender, _ := s.chainA.validators[0].keyInfo.GetAddress()

	beforeDistUatoneBalance := s.queryBalance(chainAAPIEndpoint, distModuleAddress, tokenAmount.Denom)

	s.execDistributionFundCommunityPool(s.chainA, 0, sender.String(), tokenAmount.String())

	s.Require().Eventually(
		func() bool {
			afterDistUatoneBalance := s.queryBalance(chainAAPIEndpoint, distModuleAddress, tokenAmount.Denom)
			// check if the balance is increased by the tokenAmount
			return beforeDistUatoneBalance.Add(tokenAmount).IsLT(afterDistUatoneBalance)
		},
		15*time.Second,
		time.Second,
	)
}
