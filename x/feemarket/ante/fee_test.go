package ante_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/atomone-hub/atomone/x/feemarket/ante"
	"github.com/atomone-hub/atomone/x/feemarket/testutil"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

func TestAnteHandle(t *testing.T) {
	var (
		addrs  = simtestutil.CreateIncrementalAccounts(3)
		acc1   = authtypes.NewBaseAccountWithAddress(addrs[0])
		acc2   = authtypes.NewBaseAccountWithAddress(addrs[1])
		acc3   = authtypes.NewBaseAccountWithAddress(addrs[2])
		txBody = &tx.TxBody{
			Messages: []*codectypes.Any{
				codectypes.UnsafePackAny(testdata.NewTestMsg(addrs[0], addrs[1])),
			},
		}
	)
	tests := []struct {
		name                 string
		tx                   sdk.Tx
		genTx                bool
		simulate             bool
		disableFeeGrant      bool
		disableFeemarket     bool
		denomResolver        types.DenomResolver
		setup                func(testutil.Mocks)
		expectedMinGasPrices string
		expectedTxPriority   int64
		expectedError        string
	}{
		{
			name: "ok: gentx requires no gas",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{},
				},
				Body: txBody,
			},
			genTx:                true,
			expectedMinGasPrices: "",
			expectedTxPriority:   0,
		},
		{
			name: "fail: 0 gas given",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{},
				},
				Body: txBody,
			},
			expectedError: "must provide positive gas: invalid gas limit",
		},
		{
			name: "ok: 0 gas given with disabled feemarket",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{},
				},
				Body: txBody,
			},
			disableFeemarket: true,
		},
		{
			name: "ok: simulate --gas=auto behavior",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{},
				},
				Body: txBody,
			},
			simulate: true,
			setup: func(m testutil.Mocks) {
				m.AccountKeeper.EXPECT().
					GetAccount(gomock.Any(), addrs[0]).Return(acc1)
				m.BankKeeper.EXPECT().
					SendCoinsFromAccountToModule(gomock.Any(), addrs[0],
						authtypes.FeeCollectorName, sdk.NewCoins())
			},
			expectedMinGasPrices: "1.000000000000000000stake",
			expectedTxPriority:   0,
		},
		{
			name: "fail: 0 fee given",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
					},
				},
				Body: txBody,
			},
			expectedError: "got length 0: no fee coin provided. Must provide one.",
		},
		{
			name: "fail: too many fee coins given",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 1),
							sdk.NewInt64Coin("photon", 2),
						),
					},
				},
				Body: txBody,
			},
			expectedError: "got length 2: too many fee coins provided. Only one fee coin may be provided",
		},
		{
			name: "fail: unresolvable denom",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin("photon", 2),
						),
					},
				},
				Body: txBody,
			},
			expectedError: "unable to get min gas price for denom photon: error resolving denom",
		},
		{
			name: "fail: not enough fee",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 2),
						),
					},
				},
				Body: txBody,
			},
			expectedError: "error checking fee: got: 2stake required: 42stake, minGasPrice: 1.000000000000000000stake: insufficient fee",
		},
		{
			name: "fail: unknown account",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 42),
						),
					},
				},
				Body: txBody,
			},
			setup: func(m testutil.Mocks) {
				m.AccountKeeper.EXPECT().
					GetAccount(gomock.Any(), addrs[0]).Return(nil)
			},
			expectedError: fmt.Sprintf(
				"error escrowing funds: fee payer address: %s does not exist: unknown address",
				addrs[0],
			),
		},
		{
			name: "ok: enough fee",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 42),
						),
					},
				},
				Body: txBody,
			},
			setup: func(m testutil.Mocks) {
				m.AccountKeeper.EXPECT().
					GetAccount(gomock.Any(), addrs[0]).Return(acc1)
				m.BankKeeper.EXPECT().
					SendCoinsFromAccountToModule(gomock.Any(), addrs[0],
						authtypes.FeeCollectorName,
						sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 42)))
			},
			expectedMinGasPrices: "1.000000000000000000stake",
			expectedTxPriority:   1000000,
		},
		{
			name:          "ok: enough fee with resolvable denom",
			denomResolver: &types.TestDenomResolver{},
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin("photon", 42),
						),
					},
				},
				Body: txBody,
			},
			setup: func(m testutil.Mocks) {
				m.AccountKeeper.EXPECT().
					GetAccount(gomock.Any(), addrs[0]).Return(acc1)
				m.BankKeeper.EXPECT().
					SendCoinsFromAccountToModule(gomock.Any(), addrs[0],
						authtypes.FeeCollectorName,
						sdk.NewCoins(sdk.NewInt64Coin("photon", 42)))
			},
			expectedMinGasPrices: "1.000000000000000000photon",
			expectedTxPriority:   1000000,
		},
		{
			name: "ok: enough fee with named payer",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 42),
						),
						// payer is the second signer
						Payer: acc2.Address,
					},
				},
				Body: txBody,
			},
			setup: func(m testutil.Mocks) {
				m.AccountKeeper.EXPECT().
					GetAccount(gomock.Any(), addrs[1]).Return(acc2)
				m.BankKeeper.EXPECT().
					SendCoinsFromAccountToModule(gomock.Any(), addrs[1],
						authtypes.FeeCollectorName,
						sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 42)))
			},
			expectedMinGasPrices: "1.000000000000000000stake",
			expectedTxPriority:   1000000,
		},
		{
			name: "fail: enough fee with not enough funds",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 42),
						),
					},
				},
				Body: txBody,
			},
			setup: func(m testutil.Mocks) {
				m.AccountKeeper.EXPECT().
					GetAccount(gomock.Any(), addrs[0]).Return(acc1)
				m.BankKeeper.EXPECT().
					SendCoinsFromAccountToModule(gomock.Any(), addrs[0],
						authtypes.FeeCollectorName,
						sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 42))).
					Return(errors.New("NOPE"))
			},
			expectedError: "error escrowing funds: NOPE",
		},
		{
			name: "ok: enough fee with granter",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 42),
						),
						Granter: acc3.Address,
					},
				},
				Body: txBody,
			},
			setup: func(m testutil.Mocks) {
				m.FeeGrantKeeper.EXPECT().UseGrantedFees(gomock.Any(), addrs[2], addrs[0],
					sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 42)),
					gomock.Any(),
				)
				m.AccountKeeper.EXPECT().
					GetAccount(gomock.Any(), addrs[2]).Return(acc3)
				m.BankKeeper.EXPECT().
					SendCoinsFromAccountToModule(gomock.Any(), addrs[2],
						authtypes.FeeCollectorName,
						sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 42)))
			},
			expectedMinGasPrices: "1.000000000000000000stake",
			expectedTxPriority:   1000000,
		},
		{
			name: "fail: enough fee with granter but feegrant disabled",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 42),
						),
						Granter: acc3.Address,
					},
				},
				Body: txBody,
			},
			disableFeeGrant: true,
			expectedError:   "error escrowing funds: fee grants are not enabled: invalid request",
		},
		{
			name: "fail: enough fee with granter but not granted",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
						Amount: sdk.NewCoins(
							sdk.NewInt64Coin(sdk.DefaultBondDenom, 42),
						),
						Granter: acc3.Address,
					},
				},
				Body: txBody,
			},
			setup: func(m testutil.Mocks) {
				m.FeeGrantKeeper.EXPECT().UseGrantedFees(gomock.Any(), addrs[2], addrs[0],
					sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 42)),
					gomock.Any(),
				).Return(errors.New("NOPE"))
			},
			expectedError: fmt.Sprintf(
				"error escrowing funds: %s does not allow to pay fees for %s: NOPE",
				acc3.Address, acc1.Address),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, m, ctx := testutil.SetupFeemarketKeeper(t)
			// set default params and state
			params := types.DefaultParams()
			params.Enabled = !tt.disableFeemarket
			err := k.SetParams(ctx, params)
			require.NoError(t, err)
			err = k.SetState(ctx, types.DefaultState())
			require.NoError(t, err)
			if tt.denomResolver != nil {
				k.SetDenomResolver(tt.denomResolver)
			}
			var (
				nextInvoked bool
				next        = func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
					nextInvoked = true
					return ctx, nil
				}
			)
			var feeGrantKeeper types.FeeGrantKeeper
			if !tt.disableFeeGrant {
				feeGrantKeeper = m.FeeGrantKeeper
			}
			fmd := ante.NewFeeMarketCheckDecorator(m.AccountKeeper, m.BankKeeper, feeGrantKeeper, k, nil)
			if tt.genTx {
				ctx = ctx.WithBlockHeight(0)
			} else {
				ctx = ctx.WithBlockHeight(1)
			}
			if tt.setup != nil {
				tt.setup(m)
			}

			newCtx, err := fmd.AnteHandle(ctx, tt.tx, tt.simulate, next)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			assert.True(t, nextInvoked, "next is not invoked")
			assert.Equal(t, tt.expectedMinGasPrices, newCtx.MinGasPrices().String())
			assert.Equal(t, tt.expectedTxPriority, newCtx.Priority())
		})
	}
}
