package post_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/dynamicfee/post"
	"github.com/atomone-hub/atomone/x/dynamicfee/testutil"
	"github.com/atomone-hub/atomone/x/dynamicfee/types"
)

type mocks struct {
	ctx              sdk.Context
	DynamicfeeKeeper *MockDynamicfeeKeeper
}

func setupMocks(t *testing.T) mocks {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mocks{
		ctx:              sdk.NewContext(nil, tmproto.Header{}, false, log.TestingLogger()),
		DynamicfeeKeeper: NewMockDynamicfeeKeeper(ctrl),
	}
}

func TestPostHandle(t *testing.T) {
	tests := []struct {
		name              string
		genTx             bool
		simulate          bool
		disableDynamicfee bool
		setup             func(mocks)
	}{
		{
			name:  "ok: skip gentx",
			genTx: true,
		},
		{
			name:              "ok: dynamicfee disabled",
			disableDynamicfee: true,
			setup: func(m mocks) {
				params := types.DefaultParams()
				params.Enabled = false
				m.DynamicfeeKeeper.EXPECT().GetParams(m.ctx).Return(params, nil)
			},
		},
		{
			name: "ok: enabled height not reached",
			setup: func(m mocks) {
				m.DynamicfeeKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
				m.DynamicfeeKeeper.EXPECT().GetEnabledHeight(m.ctx).Return(int64(2), nil)
			},
		},
		{
			name: "ok: state updated",
			setup: func(m mocks) {
				m.DynamicfeeKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
				m.DynamicfeeKeeper.EXPECT().GetEnabledHeight(m.ctx).Return(int64(0), nil)
				m.DynamicfeeKeeper.EXPECT().GetState(m.ctx).
					Return(types.DefaultState(), nil)

				gasConsumed := sdk.Gas(1000)
				m.ctx.GasMeter().ConsumeGas(gasConsumed, "")

				expectedState := types.DefaultState()
				expectedState.Window[0] = gasConsumed
				m.DynamicfeeKeeper.EXPECT().SetState(m.ctx, expectedState)
				m.DynamicfeeKeeper.EXPECT().GetMaxGas(m.ctx).Return(uint64(30_000_000), nil)
			},
		},
		{
			name:     "ok: simulate && state updated",
			simulate: true,
			setup: func(m mocks) {
				m.DynamicfeeKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
				m.DynamicfeeKeeper.EXPECT().GetEnabledHeight(m.ctx).Return(int64(0), nil)
				m.DynamicfeeKeeper.EXPECT().GetState(m.ctx).
					Return(types.DefaultState(), nil)

				gasConsumed := sdk.Gas(1000)
				m.ctx.GasMeter().ConsumeGas(gasConsumed, "")

				expectedState := types.DefaultState()
				expectedState.Window[0] = gasConsumed
				m.DynamicfeeKeeper.EXPECT().SetState(m.ctx, expectedState)
				m.DynamicfeeKeeper.EXPECT().GetMaxGas(m.ctx).Return(uint64(30_000_000), nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				m           = setupMocks(t)
				dfd         = post.NewDynamicfeeStateUpdateDecorator(m.DynamicfeeKeeper)
				nextInvoked bool
				next        = func(ctx sdk.Context, tx sdk.Tx, simulate, success bool) (sdk.Context, error) {
					nextInvoked = true
					return ctx, nil
				}
			)
			maxBlockGas := testutil.MaxBlockGas
			m.ctx = m.ctx.WithConsensusParams(&tmproto.ConsensusParams{Block: &tmproto.BlockParams{MaxGas: int64(maxBlockGas)}})
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
