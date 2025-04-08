package ante_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"

	appparams "github.com/atomone-hub/atomone/app/params"
	"github.com/atomone-hub/atomone/x/photon/ante"
	"github.com/atomone-hub/atomone/x/photon/types"
)

type mocks struct {
	ctx          sdk.Context
	PhotonKeeper *MockPhotonKeeper
}

func setupMocks(t *testing.T) mocks {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mocks{
		ctx:          sdk.NewContext(nil, tmproto.Header{}, false, log.TestingLogger()),
		PhotonKeeper: NewMockPhotonKeeper(ctrl),
	}
}

func TestValidateFeeDecorator(t *testing.T) {
	tests := []struct {
		name          string
		tx            sdk.Tx
		setup         func(mocks)
		expectedError string
	}{
		{
			name: "ok: no fee",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
		},
		{
			name: "ok: tx MsgMintPhoton accepts any fee denom bc declared in txFeeExceptions",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(appparams.BondDenom, 1),
							sdk.NewInt64Coin("xxx", 1),
						),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
			setup: func(m mocks) {
				m.PhotonKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams())
			},
		},
		{
			name: "ok: MsgUpdateParams fee uphoton",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin(types.Denom, 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
			setup: func(m mocks) {
				m.PhotonKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams())
			},
		},
		{
			name: "fail: MsgUpdateParams fee uatone",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin(appparams.BondDenom, 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
			setup: func(m mocks) {
				m.PhotonKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams())
			},
			expectedError: "fee denom uatone not allowed: invalid fee token",
		},
		{
			name: "fail: MsgUpdateParams fee xxx",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin("xxx", 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
			setup: func(m mocks) {
				m.PhotonKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams())
			},
			expectedError: "fee denom xxx not allowed: invalid fee token",
		},
		{
			name: "fail: MsgUpdateParams multiple fee denom",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(appparams.BondDenom, 1),
							sdk.NewInt64Coin("xxx", 1),
						),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
			setup: func(m mocks) {
				m.PhotonKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams())
			},
			expectedError: "too many fee coins, only accepts fees in one denom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				m           = setupMocks(t)
				nextInvoked bool
				next        = func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
					nextInvoked = true
					return ctx, nil
				}
			)
			if tt.setup != nil {
				tt.setup(m)
			}

			vfd := ante.NewValidateFeeDecorator(m.PhotonKeeper)
			_, err := vfd.AnteHandle(m.ctx, tt.tx, false, next)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			require.True(t, nextInvoked, "next is not invoked")
		})
	}
}

func TestAllowsAnyTxFee(t *testing.T) {
	tests := []struct {
		name            string
		tx              sdk.Tx
		txFeeExceptions []string
		expectedRes     bool
	}{
		{
			name: "wildcard fee execptions",
			tx: &tx.Tx{
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
			txFeeExceptions: []string{"*"},
			expectedRes:     true,
		},
		{
			name: "empty fee execptions",
			tx: &tx.Tx{
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
			txFeeExceptions: nil,
			expectedRes:     false,
		},
		{
			name: "one message match txFeeExceptions",
			tx: &tx.Tx{
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
			txFeeExceptions: []string{sdk.MsgTypeURL(&types.MsgMintPhoton{})},
			expectedRes:     true,
		},
		{
			name: "multiple messages not all match txFeeExceptions",
			tx: &tx.Tx{
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
			txFeeExceptions: []string{sdk.MsgTypeURL(&types.MsgMintPhoton{})},
			expectedRes:     false,
		},
		{
			name: "multiple same messages match txFeeExceptions",
			tx: &tx.Tx{
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
			txFeeExceptions: []string{sdk.MsgTypeURL(&types.MsgMintPhoton{})},
			expectedRes:     true,
		},
		{
			name: "multiple different messages match txFeeExceptions",
			tx: &tx.Tx{
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
			txFeeExceptions: []string{
				sdk.MsgTypeURL(&types.MsgMintPhoton{}),
				sdk.MsgTypeURL(&types.MsgUpdateParams{}),
			},
			expectedRes: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := ante.AllowsAnyTxFee(tt.tx, tt.txFeeExceptions)

			assert.Equal(t, tt.expectedRes, res)
		})
	}
}
