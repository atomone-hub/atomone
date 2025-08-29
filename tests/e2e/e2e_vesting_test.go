package e2e

import (
	"encoding/json"
	"math/rand"
	"path/filepath"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	delayedVestingKey    = "delayed_vesting"
	continuousVestingKey = "continuous_vesting"
	lockedVestingKey     = "locker_vesting"
	periodicVestingKey   = "periodic_vesting"

	vestingPeriodFile = "test_period.json"
	vestingTxDelay    = 5
)

type (
	vestingPeriod struct {
		StartTime int64    `json:"start_time"`
		Periods   []period `json:"periods"`
	}
	period struct {
		Coins  string `json:"coins"`
		Length int64  `json:"length_seconds"`
	}
)

var (
	genesisVestingKeys  = []string{continuousVestingKey, delayedVestingKey, lockedVestingKey, periodicVestingKey}
	vestingAmountVested = sdk.NewCoin(uatoneDenom, math.NewInt(99900000000))
	vestingAmount       = sdk.NewCoins(
		sdk.NewInt64Coin(uatoneDenom, 350_000),
		sdk.NewInt64Coin(uphotonDenom, 10_000_000_000),
	)
	vestingBalance          = sdk.NewCoins(vestingAmountVested).Add(vestingAmount...)
	vestingDelegationAmount = sdk.NewCoin(uatoneDenom, math.NewInt(500000000))
)

func (s *IntegrationTestSuite) testDelayedVestingAccount(api string) {
	var (
		valIdx            = 0
		chain             = s.chainA
		val               = chain.validators[valIdx]
		vestingDelayedAcc = chain.genesisVestingAccounts[delayedVestingKey]
	)
	sender, _ := val.keyInfo.GetAddress()
	valOpAddr := sdk.ValAddress(sender).String()

	s.Run("delayed vesting genesis account", func() {
		acc, err := s.queryDelayedVestingAccount(api, vestingDelayedAcc.String())
		s.Require().NoError(err)

		//	Check address balance
		balance := s.queryBalance(api, vestingDelayedAcc.String(), uatoneDenom)
		s.Require().Equal(vestingBalance.AmountOf(uatoneDenom), balance.Amount)

		// Delegate coins should succeed
		s.execDelegate(chain, valIdx, vestingDelegationAmount.String(), valOpAddr,
			vestingDelayedAcc.String(), atomoneHomePath)

		// Validate delegation successful
		s.Require().Eventually(
			func() bool {
				res, err := s.queryDelegation(api, valOpAddr, vestingDelayedAcc.String())
				amt := res.GetDelegationResponse().GetDelegation().GetShares()
				s.Require().NoError(err)

				return amt.Equal(math.LegacyNewDecFromInt(vestingDelegationAmount.Amount))
			},
			20*time.Second,
			time.Second,
		)

		waitTime := acc.EndTime - time.Now().Unix()
		if waitTime > vestingTxDelay {
			//	Transfer coins should fail
			balance := s.queryBalance(api, vestingDelayedAcc.String(), uatoneDenom)
			s.execBankSend(
				chain,
				valIdx,
				vestingDelayedAcc.String(),
				Address(),
				balance.String(),
				true,
			)
			waitTime = acc.EndTime - time.Now().Unix() + vestingTxDelay
			time.Sleep(time.Duration(waitTime) * time.Second)
		}

		//	Transfer coins should succeed
		balance = s.queryBalance(api, vestingDelayedAcc.String(), uatoneDenom)
		s.execBankSend(
			chain,
			valIdx,
			vestingDelayedAcc.String(),
			Address(),
			balance.String(),
			false,
		)
	})
}

func (s *IntegrationTestSuite) testContinuousVestingAccount(api string) {
	s.Run("continuous vesting genesis account", func() {
		var (
			valIdx               = 0
			chain                = s.chainA
			val                  = chain.validators[valIdx]
			continuousVestingAcc = chain.genesisVestingAccounts[continuousVestingKey]
		)
		sender, _ := val.keyInfo.GetAddress()
		valOpAddr := sdk.ValAddress(sender).String()

		acc, err := s.queryContinuousVestingAccount(api, continuousVestingAcc.String())
		s.Require().NoError(err)

		//	Check address balance
		balance := s.queryBalance(api, continuousVestingAcc.String(), uatoneDenom)
		s.Require().Equal(vestingBalance.AmountOf(uatoneDenom), balance.Amount)

		// Delegate coins should succeed
		s.execDelegate(chain, valIdx, vestingDelegationAmount.String(),
			valOpAddr, continuousVestingAcc.String(), atomoneHomePath)

		// Validate delegation successful
		s.Require().Eventually(
			func() bool {
				res, err := s.queryDelegation(api, valOpAddr, continuousVestingAcc.String())
				amt := res.GetDelegationResponse().GetDelegation().GetShares()
				s.Require().NoError(err)

				return amt.Equal(math.LegacyNewDecFromInt(vestingDelegationAmount.Amount))
			},
			20*time.Second,
			time.Second,
		)

		waitStartTime := acc.StartTime - time.Now().Unix()
		if waitStartTime > vestingTxDelay {
			//	Transfer coins should fail
			balance := s.queryBalance(api, continuousVestingAcc.String(), uatoneDenom)
			s.execBankSend(
				chain,
				valIdx,
				continuousVestingAcc.String(),
				Address(),
				balance.String(),
				true,
			)
			waitStartTime = acc.StartTime - time.Now().Unix() + vestingTxDelay
			time.Sleep(time.Duration(waitStartTime) * time.Second)
		}

		waitEndTime := acc.EndTime - time.Now().Unix()
		if waitEndTime > vestingTxDelay {
			//	Transfer coins should fail
			balance := s.queryBalance(api, continuousVestingAcc.String(), uatoneDenom)
			s.execBankSend(
				chain,
				valIdx,
				continuousVestingAcc.String(),
				Address(),
				balance.String(),
				true,
			)
			waitEndTime = acc.EndTime - time.Now().Unix() + vestingTxDelay
			time.Sleep(time.Duration(waitEndTime) * time.Second)
		}

		//	Transfer coins should succeed
		balance = s.queryBalance(api, continuousVestingAcc.String(), uatoneDenom)
		s.execBankSend(
			chain,
			valIdx,
			continuousVestingAcc.String(),
			Address(),
			balance.String(),
			false,
		)
	})
}

func (s *IntegrationTestSuite) testPeriodicVestingAccount(api string) { //nolint:unused

	s.Run("test periodic vesting genesis account", func() {
		var (
			valIdx              = 0
			chain               = s.chainA
			val                 = chain.validators[valIdx]
			periodicVestingAddr = chain.genesisVestingAccounts[periodicVestingKey].String()
		)
		sender, _ := val.keyInfo.GetAddress()
		valOpAddr := sdk.ValAddress(sender).String()

		s.execCreatePeriodicVestingAccount(
			chain,
			periodicVestingAddr,
			filepath.Join(atomoneHomePath, vestingPeriodFile),
			withKeyValue(flagFrom, sender.String()),
		)

		acc, err := s.queryPeriodicVestingAccount(api, periodicVestingAddr)
		s.Require().NoError(err)

		//	Check address balance
		balance := s.queryBalance(api, periodicVestingAddr, uatoneDenom)

		expectedBalance := sdk.NewCoin(uatoneDenom, math.NewInt(0))
		for _, period := range acc.VestingPeriods {
			// _, coin := ante.Find(period.Amount, uatoneDenom)
			_, coin := period.Amount.Find(uatoneDenom)
			expectedBalance = expectedBalance.Add(coin)
		}
		s.Require().Equal(expectedBalance, balance)

		waitStartTime := acc.StartTime - time.Now().Unix()
		if waitStartTime > vestingTxDelay {
			//	Transfer coins should fail
			balance = s.queryBalance(api, periodicVestingAddr, uatoneDenom)
			s.execBankSend(
				chain,
				valIdx,
				periodicVestingAddr,
				Address(),
				balance.String(),
				true,
			)
			waitStartTime = acc.StartTime - time.Now().Unix() + vestingTxDelay
			time.Sleep(time.Duration(waitStartTime) * time.Second)
		}

		firstPeriod := acc.StartTime + acc.VestingPeriods[0].Length
		waitFirstPeriod := firstPeriod - time.Now().Unix()
		if waitFirstPeriod > vestingTxDelay {
			//	Transfer coins should fail
			balance = s.queryBalance(api, periodicVestingAddr, uatoneDenom)
			s.execBankSend(
				chain,
				valIdx,
				periodicVestingAddr,
				Address(),
				balance.String(),
				true,
			)
			waitFirstPeriod = firstPeriod - time.Now().Unix() + vestingTxDelay
			time.Sleep(time.Duration(waitFirstPeriod) * time.Second)
		}

		// Delegate coins should succeed
		s.execDelegate(chain, valIdx, vestingDelegationAmount.String(), valOpAddr,
			periodicVestingAddr, atomoneHomePath)

		// Validate delegation successful
		s.Require().Eventually(
			func() bool {
				res, err := s.queryDelegation(api, valOpAddr, periodicVestingAddr)
				amt := res.GetDelegationResponse().GetDelegation().GetShares()
				s.Require().NoError(err)

				return amt.Equal(math.LegacyNewDecFromInt(vestingDelegationAmount.Amount))
			},
			20*time.Second,
			time.Second,
		)

		//	Transfer coins should succeed
		balance = s.queryBalance(api, periodicVestingAddr, uatoneDenom)
		s.execBankSend(
			chain,
			valIdx,
			periodicVestingAddr,
			Address(),
			balance.String(),
			false,
		)

		secondPeriod := firstPeriod + acc.VestingPeriods[1].Length
		waitSecondPeriod := secondPeriod - time.Now().Unix()
		if waitSecondPeriod > vestingTxDelay {
			time.Sleep(time.Duration(waitSecondPeriod) * time.Second)

			//	Transfer coins should succeed
			balance = s.queryBalance(api, periodicVestingAddr, uatoneDenom)
			s.execBankSend(
				chain,
				valIdx,
				periodicVestingAddr,
				Address(),
				balance.String(),
				false,
			)
		}
	})
}

// generateVestingPeriod generate the vesting period file
func generateVestingPeriod() ([]byte, error) {
	p := vestingPeriod{
		StartTime: time.Now().Add(time.Duration(rand.Intn(20)+95) * time.Second).Unix(),
		Periods: []period{
			{
				Coins:  "850000000" + uatoneDenom,
				Length: 35,
			},
			{
				Coins:  "2000000000" + uatoneDenom,
				Length: 35,
			},
		},
	}
	return json.Marshal(p)
}
