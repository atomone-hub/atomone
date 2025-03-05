package ante_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	_ "github.com/cosmos/cosmos-sdk/x/auth"

	antesuite "github.com/skip-mev/feemarket/x/feemarket/ante/suite"
	"github.com/skip-mev/feemarket/x/feemarket/types"
)

func TestAnteHandle(t *testing.T) {
	// Same data for every test case
	gasLimit := antesuite.NewTestGasLimit()

	validFeeAmount := types.DefaultMinBaseGasPrice.MulInt64(int64(gasLimit))
	validFee := sdk.NewCoins(sdk.NewCoin("stake", validFeeAmount.TruncateInt()))
	validFeeDifferentDenom := sdk.NewCoins(sdk.NewCoin("atom", math.Int(validFeeAmount)))

	testCases := []antesuite.TestCase{
		{
			Name: "0 gas given should fail",
			Malleate: func(s *antesuite.TestSuite) antesuite.TestCaseArgs {
				accs := s.CreateTestAccounts(1)

				return antesuite.TestCaseArgs{
					Msgs:      []sdk.Msg{testdata.NewTestMsg(accs[0].Account.GetAddress())},
					GasLimit:  0,
					FeeAmount: validFee,
				}
			},
			RunAnte:  true,
			RunPost:  false,
			Simulate: false,
			ExpPass:  false,
			ExpErr:   sdkerrors.ErrOutOfGas,
		},
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
			RunPost:  false,
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
			RunPost:  false,
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
			RunPost:  false,
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
			RunPost:  false,
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
			RunPost:  false,
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
			RunPost:  false,
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
			RunPost:  true,
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
			RunPost:  true,
			Simulate: false,
			ExpPass:  false,
			ExpErr:   sdkerrors.ErrOutOfGas,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.Name), func(t *testing.T) {
			s := antesuite.SetupTestSuite(t, true)
			s.TxBuilder = s.ClientCtx.TxConfig.NewTxBuilder()
			args := tc.Malleate(s)

			s.RunTestCase(t, tc, args)
		})
	}
}
