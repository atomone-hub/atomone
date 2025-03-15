package ante_test

import (
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
	// Same data for every test case
	// gasLimit := antesuite.NewTestGasLimit()
	//
	// validFeeAmount := types.DefaultMinBaseGasPrice.MulInt64(int64(gasLimit))
	// validFee := sdk.NewCoins(sdk.NewCoin("stake", validFeeAmount.TruncateInt()))
	// validFeeDifferentDenom := sdk.NewCoins(sdk.NewCoin("atom", math.Int(validFeeAmount)))

	var (
		addrs  = simtestutil.CreateIncrementalAccounts(1)
		acc1   = authtypes.NewBaseAccountWithAddress(addrs[0])
		txBody = &tx.TxBody{
			Messages: []*codectypes.Any{
				codectypes.UnsafePackAny(testdata.NewTestMsg(addrs[0])),
			},
		}
	)
	tests := []struct {
		name                 string
		tx                   sdk.Tx
		genTx                bool
		simulate             bool
		setup                func(testutil.Mocks)
		expectedMinGasPrices string
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
			genTx: true,
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
			name: "ok: simulate --gas=auto behavior",
			tx: &tx.Tx{
				AuthInfo: &tx.AuthInfo{
					Fee: &tx.Fee{
						GasLimit: 42,
					},
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
		},
	}
	/*
		testCases := []antesuite.TestCase{
			// test --gas=auto flag settings
			// when --gas=auto is set, cosmos-sdk sets gas=0 and simulate=true
			{
				Name: "--gas=auto behaviour test",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)
					s.MockBankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, accs[0].Account.GetAddress(),
						types.FeeCollectorName, mock.Anything).Return(nil)
					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  0,
						FeeAmount: validFee,
					}
				},
				RunAnte:  true,
				Simulate: true,
				ExpPass:  true,
			},
			{
				Name: "0 gas given should fail with resolvable denom",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)

					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  0,
						FeeAmount: validFeeDifferentDenom,
					}
				},
				RunAnte:  true,
				Simulate: false,
				ExpPass:  false,
				ExpErr:   sdkerrors.ErrOutOfGas,
			},
			{
				Name: "0 gas given should pass in simulate - no fee",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)
					s.MockBankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, accs[0].Account.GetAddress(),
						types.FeeCollectorName, mock.Anything).Return(nil).Once()
					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  0,
						FeeAmount: nil,
					}
				},
				RunAnte:  true,
				Simulate: true,
				ExpPass:  true,
				ExpErr:   nil,
			},
			{
				Name: "0 gas given should pass in simulate - fee",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)
					s.MockBankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, accs[0].Account.GetAddress(),
						types.FeeCollectorName, mock.Anything).Return(nil).Once()
					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  0,
						FeeAmount: validFee,
					}
				},
				RunAnte:  true,
				Simulate: true,
				ExpPass:  true,
				ExpErr:   nil,
			},
			{
				Name: "signer has enough funds, should pass",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)
					s.MockBankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, accs[0].Account.GetAddress(),
						types.FeeCollectorName, mock.Anything).Return(nil)
					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  gasLimit,
						FeeAmount: validFee,
					}
				},
				RunAnte:  true,
				Simulate: false,
				ExpPass:  true,
				ExpErr:   nil,
			},
			{
				Name: "signer has enough funds in resolvable denom, should pass",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)
					s.MockBankKeeper.On("SendCoinsFromAccountToModule", mock.Anything, accs[0].Account.GetAddress(),
						types.FeeCollectorName, mock.Anything).Return(nil).Once()
					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  gasLimit,
						FeeAmount: validFeeDifferentDenom,
					}
				},
				RunAnte:  true,
				Simulate: false,
				ExpPass:  true,
				ExpErr:   nil,
			},
			{
				Name: "no fee - fail",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)

					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  1000000000,
						FeeAmount: nil,
					}
				},
				RunAnte:  true,
				Simulate: false,
				ExpPass:  false,
				ExpErr:   types.ErrNoFeeCoins,
			},
			{
				Name: "no gas limit - fail",
				Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
					accs := s.CreateTestAccounts(1)

					return antesuite.TestCaseArgs{
						Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
						GasLimit:  0,
						FeeAmount: nil,
					}
				},
				RunAnte:  true,
				Simulate: false,
				ExpPass:  false,
				ExpErr:   sdkerrors.ErrOutOfGas,
			},
		}
	*/

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, m, ctx := testutil.SetupFeemarketKeeper(t)
			// set default params and state
			err := k.SetParams(ctx, types.DefaultParams())
			require.NoError(t, err)
			err = k.SetState(ctx, types.DefaultState())
			require.NoError(t, err)
			var (
				nextInvoked bool
				next        = func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
					nextInvoked = true
					return ctx, nil
				}
				fmd = ante.NewFeeMarketCheckDecorator(m.AccountKeeper, m.BankKeeper, m.FeeGrantKeeper, k, nil)
			)
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
		})
		// t.Run(fmt.Sprintf("Case %s", tc.Name), func(t *testing.T) {
		// s := antesuite.SetupTestSuite(t, true)
		// s.TxBuilder = s.ClientCtx.TxConfig.NewTxBuilder()
		// args := tc.Malleate(s)
		//
		// s.RunTestCase(t, tc, args)
		// })
	}
}
