package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func createValidators(powers ...int64) ([]stakingtypes.Validator, error) {
	vals := make([]stakingtypes.Validator, len(powers))
	for i, p := range powers {
		vals[i] = stakingtypes.Validator{
			OperatorAddress: sdk.ValAddress([]byte{byte(i)}).String(),
			Tokens:          math.NewInt(p),
			Status:          stakingtypes.Bonded,
			Commission:      stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		}
	}
	return vals, nil
}

func TestAdjustEta_NoInterval(t *testing.T) {
	initEta := math.LegacyNewDecWithPrec(3, 2)
	s := setupTestKeeper(t, initEta, 119_999)

	err := s.distrKeeper.AdjustEta(s.ctx)
	require.NoError(t, err)
	gotParams, _ := s.distrKeeper.Params.Get(s.ctx)
	require.Equal(t, initEta, gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_NotEnoughValidators(t *testing.T) {
	initEta := math.LegacyNewDecWithPrec(3, 2)
	s := setupTestKeeper(t, initEta, 120_000)

	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(10, 10)).AnyTimes()

	err := s.distrKeeper.AdjustEta(s.ctx)
	require.NoError(t, err)
	gotParams, _ := s.distrKeeper.Params.Get(s.ctx)
	require.Equal(t, initEta, gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_Increase(t *testing.T) {
	initEta := math.LegacyNewDecWithPrec(3, 2)
	s := setupTestKeeper(t, initEta, 120_000)

	// highAvg = 100, lowAvg = 10, ratio = 10 >= 3, should increase
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(100, 100, 10)).AnyTimes()

	err := s.distrKeeper.AdjustEta(s.ctx)
	require.NoError(t, err)
	gotParams, _ := s.distrKeeper.Params.Get(s.ctx)
	require.Equal(t, initEta.Add(math.LegacyNewDecWithPrec(3, 2)), gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_Decrease(t *testing.T) {
	initEta := math.LegacyNewDecWithPrec(3, 2)
	s := setupTestKeeper(t, initEta, 120_000)

	// highAvg = 20, lowAvg = 10, ratio = 2 < 3, should decrease
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(20, 20, 10)).AnyTimes()

	err := s.distrKeeper.AdjustEta(s.ctx)
	require.NoError(t, err)
	gotParams, _ := s.distrKeeper.Params.Get(s.ctx)
	require.Equal(t, math.LegacyZeroDec(), gotParams.NakamotoBonusCoefficient)
}

func TestAdjustEta_ClampZero(t *testing.T) {
	initEta := math.LegacyZeroDec()
	s := setupTestKeeper(t, initEta, 120_000)

	// highAvg = 20, lowAvg = 10, ratio = 2 < 3, should decrease, and clamp at 0
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(20, 20, 10)).AnyTimes()

	err := s.distrKeeper.AdjustEta(s.ctx)
	require.NoError(t, err)
	gotParams, _ := s.distrKeeper.Params.Get(s.ctx)
	require.True(t, gotParams.NakamotoBonusCoefficient.GTE(math.LegacyZeroDec()))
}

func TestAdjustEta_ClampOne(t *testing.T) {
	initEta := math.LegacyOneDec()
	s := setupTestKeeper(t, initEta, 120_000)

	// highAvg = 100, lowAvg = 10, ratio = 10 >= 3, should increase
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(100, 100, 10)).AnyTimes()

	err := s.distrKeeper.AdjustEta(s.ctx)
	require.NoError(t, err)
	gotParams, _ := s.distrKeeper.Params.Get(s.ctx)
	require.True(t, gotParams.NakamotoBonusCoefficient.LTE(math.LegacyOneDec()))
}
