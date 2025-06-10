package post_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/feemarket/post"
	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

type mocks struct {
	ctx                   sdk.Context
	FeeMarketKeeper       *MockFeeMarketKeeper
	ConsensusParamsKeeper *MockConsensusParamsKeeper
}

func setupMocks(t *testing.T) mocks {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mocks{
		ctx:                   sdk.NewContext(nil, tmproto.Header{}, false, log.NewTestLogger(t)),
		FeeMarketKeeper:       NewMockFeeMarketKeeper(ctrl),
		ConsensusParamsKeeper: NewMockConsensusParamsKeeper(ctrl),
	}
}

func TestPostHandle(t *testing.T) {
	tests := []struct {
		name             string
		genTx            bool
		simulate         bool
		disableFeemarket bool
		setup            func(mocks)
	}{
		{
			name:  "ok: skip gentx",
			genTx: true,
		},
		{
			name:             "ok: feemarket disabled",
			disableFeemarket: true,
			setup: func(m mocks) {
				params := types.DefaultParams()
				params.Enabled = false
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).Return(params, nil)
			},
		},
		{
			name: "ok: enabled height not reached",
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
				m.FeeMarketKeeper.EXPECT().GetEnabledHeight(m.ctx).Return(int64(2), nil)
			},
		},
		{
			name: "ok: state updated",
			setup: func(m mocks) {
				m.ConsensusParamsKeeper.EXPECT().Get(m.ctx).
					Return(&tmproto.ConsensusParams{
						Block: &tmproto.BlockParams{MaxGas: testutil.MaxBlockGas},
					}, nil)
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
				m.FeeMarketKeeper.EXPECT().GetEnabledHeight(m.ctx).Return(int64(0), nil)
				m.FeeMarketKeeper.EXPECT().GetState(m.ctx).
					Return(types.DefaultState(), nil)

				gasConsumed := storetypes.Gas(1000)
				m.ctx.GasMeter().ConsumeGas(gasConsumed, "")

				expectedState := types.DefaultState()
				expectedState.Window[0] = gasConsumed
				m.FeeMarketKeeper.EXPECT().SetState(m.ctx, expectedState)
			},
		},
		{
			name:     "ok: simulate && state updated",
			simulate: true,
			setup: func(m mocks) {
				m.ConsensusParamsKeeper.EXPECT().Get(m.ctx).
					Return(&tmproto.ConsensusParams{
						Block: &tmproto.BlockParams{MaxGas: testutil.MaxBlockGas},
					}, nil)
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
				m.FeeMarketKeeper.EXPECT().GetEnabledHeight(m.ctx).Return(int64(0), nil)
				m.FeeMarketKeeper.EXPECT().GetState(m.ctx).
					Return(types.DefaultState(), nil)

				gasConsumed := storetypes.Gas(1000)
				m.ctx.GasMeter().ConsumeGas(gasConsumed, "")

				expectedState := types.DefaultState()
				expectedState.Window[0] = gasConsumed
				m.FeeMarketKeeper.EXPECT().SetState(m.ctx, expectedState)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				m           = setupMocks(t)
				dfd         = post.NewFeemarketStateUpdateDecorator(m.FeeMarketKeeper, m.ConsensusParamsKeeper)
				nextInvoked bool
				next        = func(ctx sdk.Context, tx sdk.Tx, simulate, success bool) (sdk.Context, error) {
					nextInvoked = true
					return ctx, nil
				}
			)
			if tt.genTx {
				m.ctx = m.ctx.WithBlockHeight(0)
			} else {
				m.ctx = m.ctx.WithBlockHeight(1)
			}
			if tt.setup != nil {
				tt.setup(m)
			}

			_, err := dfd.PostHandle(m.ctx, nil, tt.simulate, true, next)

			require.NoError(t, err)
			assert.True(t, nextInvoked, "next is not invoked")
		})
	}
}
