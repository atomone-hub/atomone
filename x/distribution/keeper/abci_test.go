package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/x/distribution/types"
)

func TestBeginBlocker_NakamotoBonusEtaChange(t *testing.T) {
	s := setupTestKeeper(t, math.LegacyNewDecWithPrec(3, 2), types.EtaUpdateInterval)

	// Use η = 0.03, block height triggers adjustment
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(100, 100, 10))
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(634195840)))
	s.bankKeeper.EXPECT().GetAllBalances(gomock.Any(), s.feeCollectorAcc.GetAddress()).Return(fees).AnyTimes()
	s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", types.ModuleName, fees)

	// Simulate BeginBlocker
	err := s.distrKeeper.BeginBlocker(s.ctx)
	require.NoError(t, err)

	// Check η was increased (high/low avg ratio >= 3)
	params, err := s.distrKeeper.Params.Get(s.ctx)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(6, 2), params.NakamotoBonusCoefficient)
}

func TestBeginBlocker_NakamotoBonusEtaDecrease(t *testing.T) {
	s := setupTestKeeper(t, math.LegacyNewDecWithPrec(3, 2), types.EtaUpdateInterval)

	// Use η = 0.03, block height triggers adjustment, but ratio < 3 (should decrease to 0)
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(20, 20, 10))
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(634195840)))
	s.bankKeeper.EXPECT().GetAllBalances(gomock.Any(), s.feeCollectorAcc.GetAddress()).Return(fees).AnyTimes()
	s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", types.ModuleName, fees)

	// Simulate BeginBlocker
	err := s.distrKeeper.BeginBlocker(s.ctx)
	require.NoError(t, err)

	// Check η was decreased and clamped at 0
	params, err := s.distrKeeper.Params.Get(s.ctx)
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), params.NakamotoBonusCoefficient)
}

func TestAllocateTokens_NakamotoBonusClampEta(t *testing.T) {
	s := setupTestKeeper(t, math.LegacyOneDec(), types.EtaUpdateInterval)

	// η = 1.0, should clamp to 1.0 even if increase requested
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(100, 100, 10))
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(634195840)))
	s.bankKeeper.EXPECT().GetAllBalances(gomock.Any(), s.feeCollectorAcc.GetAddress()).Return(fees).AnyTimes()
	s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", types.ModuleName, fees)

	// Simulate BeginBlocker
	err := s.distrKeeper.BeginBlocker(s.ctx)
	require.NoError(t, err)

	// Should stay at 1
	params, err := s.distrKeeper.Params.Get(s.ctx)
	require.NoError(t, err)
	require.Equal(t, math.LegacyOneDec(), params.NakamotoBonusCoefficient)
}

func TestAllocateTokens_NakamotoBonusClampEtaZero(t *testing.T) {
	s := setupTestKeeper(t, math.LegacyZeroDec(), types.EtaUpdateInterval)

	// η = 0.0, should clamp to 0.0 even if decrease requested
	s.stakingKeeper.EXPECT().GetBondedValidatorsByPower(s.ctx).Return(createValidators(20, 20, 10))
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(634195840)))
	s.bankKeeper.EXPECT().GetAllBalances(gomock.Any(), s.feeCollectorAcc.GetAddress()).Return(fees).AnyTimes()
	s.bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", types.ModuleName, fees)

	// Simulate BeginBlocker
	err := s.distrKeeper.BeginBlocker(s.ctx)
	require.NoError(t, err)

	// Should stay at 0
	params, err := s.distrKeeper.Params.Get(s.ctx)
	require.NoError(t, err)
	require.Equal(t, math.LegacyZeroDec(), params.NakamotoBonusCoefficient)
}
