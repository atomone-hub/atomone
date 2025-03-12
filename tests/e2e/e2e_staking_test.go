package e2e

import (
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *IntegrationTestSuite) testStaking() {
	s.Run("staking", func() {
		chainEndpoint := fmt.Sprintf("http://%s", s.valResources[s.chainA.id][0].GetHostPort("1317/tcp"))

		validatorA := s.chainA.validators[0]
		validatorB := s.chainA.validators[1]
		validatorAAddr, _ := validatorA.keyInfo.GetAddress()
		validatorBAddr, _ := validatorB.keyInfo.GetAddress()

		validatorAddressA := sdk.ValAddress(validatorAAddr).String()
		validatorAddressB := sdk.ValAddress(validatorBAddr).String()

		delegatorAddress, _ := s.chainA.genesisAccounts[2].keyInfo.GetAddress()

		delegationAmount := sdk.NewInt(500000000)
		delegation := sdk.NewCoin(uatoneDenom, delegationAmount) // 500 atom

		// Alice delegate uatone to Validator A
		s.execDelegate(s.chainA, 0, delegation, validatorAddressA, delegatorAddress.String())

		redelegationAmount := delegationAmount.Quo(sdk.NewInt(2))
		redelegation := sdk.NewCoin(uatoneDenom, redelegationAmount) // 250 atom

		// Alice re-delegate half of her uatone delegation from Validator A to Validator B
		s.execRedelegate(s.chainA, 0, redelegation.String(), validatorAddressA, validatorAddressB, delegatorAddress.String(), atomoneHomePath)

		// Validate re-delegation successful
		s.Require().Eventually(
			func() bool {
				res, err := s.queryDelegation(validatorAddressB, delegatorAddress.String())
				s.Require().NoError(err)
				amt := res.GetDelegationResponse().GetDelegation().GetShares()

				return amt.Equal(sdk.NewDecFromInt(redelegationAmount))
			},
			20*time.Second,
			time.Second,
		)

		var (
			currDelegation       sdk.Coin
			currDelegationAmount math.Int
		)

		// query alice's current delegation from validator A
		s.Require().Eventually(
			func() bool {
				res, err := s.queryDelegation(validatorAddressA, delegatorAddress.String())
				s.Require().NoError(err)
				amt := res.GetDelegationResponse().GetDelegation().GetShares()

				currDelegationAmount = amt.TruncateInt()
				currDelegation = sdk.NewCoin(uatoneDenom, currDelegationAmount)

				return currDelegation.IsValid()
			},
			20*time.Second,
			time.Second,
		)

		// Alice unbonds all her uatone delegation from Validator A
		s.execUnbondDelegation(s.chainA, 0, currDelegation.String(), validatorAddressA, delegatorAddress.String(), atomoneHomePath)

		var ubdDelegationEntry types.UnbondingDelegationEntry

		// validate unbonding delegations
		s.Require().Eventually(
			func() bool {
				res, err := queryUnbondingDelegation(chainEndpoint, validatorAddressA, delegatorAddress.String())
				s.Require().NoError(err)

				s.Require().Len(res.GetUnbond().Entries, 1)
				ubdDelegationEntry = res.GetUnbond().Entries[0]

				return ubdDelegationEntry.Balance.Equal(currDelegationAmount)
			},
			20*time.Second,
			time.Second,
		)

		// cancel the full amount of unbonding delegations from Validator A
		s.execCancelUnbondingDelegation(
			s.chainA,
			0,
			currDelegation.String(),
			validatorAddressA,
			strconv.Itoa(int(ubdDelegationEntry.CreationHeight)),
			delegatorAddress.String(),
			atomoneHomePath,
		)

		// validate that unbonding delegation was successfully canceled
		s.Require().Eventually(
			func() bool {
				resDel, err := s.queryDelegation(validatorAddressA, delegatorAddress.String())
				s.Require().NoError(err)
				amt := resDel.GetDelegationResponse().GetDelegation().GetShares()

				// expect that no unbonding delegations are found for validator A
				_, err = queryUnbondingDelegation(chainEndpoint, validatorAddressA, delegatorAddress.String())
				s.Require().Error(err)

				// expect to get the delegation back
				return amt.Equal(sdk.NewDecFromInt(currDelegationAmount))
			},
			20*time.Second,
			time.Second,
		)
	})
}
