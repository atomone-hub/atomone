package ante

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/atomone-hub/atomone/x/photon/testutil"
	"github.com/atomone-hub/atomone/x/photon/types"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

func TestValidateFeeDecorator(t *testing.T) {
	tests := []struct {
		name            string
		tx              sdk.Tx
		txFeeExceptions []string
		expectedError   string
	}{
		{
			name: "fail: multiple fee denoms",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin("uatone", 1),
							sdk.NewInt64Coin("uphoton", 1),
						),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
			expectedError: "too many fee coins, only accepts fees in one denom",
		},
		{
			name: "ok: MsgMintPhoton fee uatone",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin("uatone", 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
		},
		{
			name: "ok: MsgMintPhoton fee uphoton",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin("uphoton", 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
		},
		{
			name: "fail: MsgMintPhoton fee xxx",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin("xxx", 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgMintPhoton{}),
					},
				},
			},
			expectedError: "expected 1uatone,1uphoton got 1xxx: invalid fee token",
		},
		{
			name: "ok: MsgUpdateParams fee uphoton",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin("uphoton", 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
		},
		{
			name: "fail: MsgUpdateParams fee uatone",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						Amount: sdk.NewCoins(sdk.NewInt64Coin("uatone", 1)),
					},
				},
				Body: &tx.TxBody{
					Messages: []*codectypes.Any{
						codectypes.UnsafePackAny(&types.MsgUpdateParams{}),
					},
				},
			},
			expectedError: "expected 1uphoton got 1uatone: invalid fee token",
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
			expectedError: "expected 1uphoton got 1xxx: invalid fee token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, _, ctx := testutil.SetupPhotonKeeper(t)
			k.SetParams(ctx, types.DefaultParams())
			var (
				nextInvoked bool
				next        = func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
					nextInvoked = true
					return ctx, nil
				}
				vfd = NewValidateFeeDecorator(k)
			)

			_, err := vfd.AnteHandle(ctx, tt.tx, false, next)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			require.True(t, nextInvoked, "next is not invoked")
		})
	}
}

func TestIsTxFeeExcepted(t *testing.T) {
	tests := []struct {
		name            string
		tx              sdk.Tx
		txFeeExceptions []string
		expectedRes     bool
	}{
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
			res := isTxFeeExcepted(tt.tx, tt.txFeeExceptions)

			assert.Equal(t, tt.expectedRes, res)
		})
	}
}