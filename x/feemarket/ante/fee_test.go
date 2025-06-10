package ante_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	xtxsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/atomone-hub/atomone/x/feemarket/ante"
	"github.com/atomone-hub/atomone/x/feemarket/types"
)

type mocks struct {
	ctx             sdk.Context
	AccountKeeper   *MockAccountKeeper
	BankKeeper      *MockBankKeeper
	FeeGrantKeeper  *MockFeeGrantKeeper
	FeeMarketKeeper *MockFeeMarketKeeper
}

func setupMocks(t *testing.T) mocks {
	t.Helper()
	ctrl := gomock.NewController(t)
	return mocks{
		ctx:             sdk.NewContext(nil, tmproto.Header{}, false, log.NewTestLogger(t)),
		AccountKeeper:   NewMockAccountKeeper(ctrl),
		BankKeeper:      NewMockBankKeeper(ctrl),
		FeeGrantKeeper:  NewMockFeeGrantKeeper(ctrl),
		FeeMarketKeeper: NewMockFeeMarketKeeper(ctrl),
	}
}

func TestAnteHandle(t *testing.T) {
	interfaceRegistry, _ := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: xtxsigning.Options{
			AddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32AccountAddrPrefix(),
			},
			ValidatorAddressCodec: address.Bech32Codec{
				Bech32Prefix: sdk.GetConfig().GetBech32ValidatorAddrPrefix(),
			},
		},
	})

	txConfig := authtx.NewTxConfig(
		codec.NewProtoCodec(interfaceRegistry),
		[]signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT},
	)

	var (
		addrs = simtestutil.CreateIncrementalAccounts(3)
		acc1  = authtypes.NewBaseAccountWithAddress(addrs[0])
		acc2  = authtypes.NewBaseAccountWithAddress(addrs[1])
		acc3  = authtypes.NewBaseAccountWithAddress(addrs[2])
	)

	tests := []struct {
		name                 string
		tx                   func() sdk.Tx
		genTx                bool
		simulate             bool
		disableFeeGrant      bool
		setup                func(mocks)
		expectedConsumedGas  int
		expectedMinGasPrices string
		expectedTxPriority   int64
		expectedError        string
	}{
		{
			name: "ok: gentx requires no gas",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				return txBuilder.GetTx()
			},
			genTx: true,
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
			},
			expectedMinGasPrices: "",
			expectedTxPriority:   0,
		},
		{
			name: "fail: 0 gas given",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil)
			},
			expectedError: "must provide positive gas: invalid gas limit",
		},
		{
			name: "ok: 0 gas given with disabled feemarket",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				params := types.DefaultParams()
				params.Enabled = false
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).Return(params, nil)
			},
		},
		{
			name: "ok: simulate --gas=auto behavior",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				return txBuilder.GetTx()
			},
			simulate: true,
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[0]).
					Return(acc1)
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(),
					addrs[0], authtypes.FeeCollectorName, sdk.NewCoins())
			},
			expectedConsumedGas:  ante.BankSendGasConsumption,
			expectedMinGasPrices: "1.000000000000000000uphoton",
			expectedTxPriority:   0,
		},
		{
			name: "fail: 0 fee given",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
			},
			expectedError: "got length 0: no fee coin provided. Must provide one.",
		},
		{
			name: "fail: too many fee coins given",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 1),
					sdk.NewInt64Coin("photon", 2),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
			},
			expectedError: "got length 2: too many fee coins provided. Only one fee coin may be provided",
		},
		{
			name: "fail: getMinGasPrice returns an error",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 2),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.DecCoin{}, errors.New("OUPS"))
			},
			expectedError: "unable to get min gas price for denom uphoton: OUPS",
		},
		{
			name: "fail: not enough fee",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 1),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
			},
			expectedError: "error checking fee: got: 1uphoton required: 42uphoton, minGasPrice: 1.000000000000000000uphoton: insufficient fee",
		},
		{
			name: "fail: unknown account",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 42),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[0]).Return(nil)
			},
			expectedError: fmt.Sprintf(
				"error deducting fee: fee payer address: %s does not exist: unknown address",
				addrs[0],
			),
		},
		{
			name: "ok: enough fee",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 42),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[0]).
					Return(acc1)
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(),
					addrs[0], authtypes.FeeCollectorName,
					sdk.NewCoins(sdk.NewInt64Coin(types.DefaultFeeDenom, 42)))
			},
			expectedMinGasPrices: "1.000000000000000000uphoton",
			expectedTxPriority:   1000000,
		},
		{
			name: "ok: more fee than gas limit increases tx priority",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 420),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[0]).
					Return(acc1)
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(),
					addrs[0], authtypes.FeeCollectorName,
					sdk.NewCoins(sdk.NewInt64Coin(types.DefaultFeeDenom, 420)))
			},
			expectedMinGasPrices: "1.000000000000000000uphoton",
			expectedTxPriority:   10000000,
		},
		{
			name: "ok: enough fee with different denom",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin("uatone", 420),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, "uatone").
					Return(sdk.NewInt64DecCoin("uatone", 10), nil)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[0]).
					Return(acc1)
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(),
					addrs[0], authtypes.FeeCollectorName,
					sdk.NewCoins(sdk.NewInt64Coin("uatone", 420)))
				// second call to GetMinGasPrice for tx priority computation
				ctx := m.ctx.WithMinGasPrices(sdk.NewDecCoins(
					sdk.NewInt64DecCoin("uatone", 10),
				))
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
			},
			expectedMinGasPrices: "10.000000000000000000uatone",
			expectedTxPriority:   1000000,
		},
		{
			name: "ok: enough fee with named payer",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 42),
				))
				txBuilder.SetFeePayer(addrs[1])
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[1]).
					Return(acc2)
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(),
					addrs[1], authtypes.FeeCollectorName,
					sdk.NewCoins(sdk.NewInt64Coin(types.DefaultFeeDenom, 42)))
			},
			expectedMinGasPrices: "1.000000000000000000uphoton",
			expectedTxPriority:   1000000,
		},
		{
			name: "fail: enough fee with not enough funds",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 42),
				))
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[0]).Return(acc1)
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(),
					addrs[0], authtypes.FeeCollectorName,
					sdk.NewCoins(sdk.NewInt64Coin(types.DefaultFeeDenom, 42))).
					Return(errors.New("NOPE"))
			},
			expectedError: "error deducting fee: NOPE: insufficient funds",
		},
		{
			name: "ok: enough fee with granter",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 42),
				))
				txBuilder.SetFeeGranter(addrs[2])
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.FeeGrantKeeper.EXPECT().UseGrantedFees(gomock.Any(), addrs[2],
					addrs[0], sdk.NewCoins(sdk.NewInt64Coin(types.DefaultFeeDenom, 42)),
					gomock.Any(),
				)
				m.AccountKeeper.EXPECT().GetAccount(gomock.Any(), addrs[2]).
					Return(acc3)
				m.BankKeeper.EXPECT().SendCoinsFromAccountToModule(gomock.Any(),
					addrs[2], authtypes.FeeCollectorName,
					sdk.NewCoins(sdk.NewInt64Coin(types.DefaultFeeDenom, 42)))
			},
			expectedMinGasPrices: "1.000000000000000000uphoton",
			expectedTxPriority:   1000000,
		},
		{
			name: "fail: enough fee with granter but feegrant disabled",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 42),
				))
				txBuilder.SetFeeGranter(addrs[2])
				return txBuilder.GetTx()
			},
			disableFeeGrant: true,
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
			},
			expectedError: "error deducting fee: fee grants are not enabled: invalid request",
		},
		{
			name: "fail: enough fee with granter but not granted",
			tx: func() sdk.Tx {
				txBuilder := txConfig.NewTxBuilder()
				txBuilder.SetMsgs(testdata.NewTestMsg(addrs[0], addrs[1]))
				txBuilder.SetGasLimit(42)
				txBuilder.SetFeeAmount(sdk.NewCoins(
					sdk.NewInt64Coin(types.DefaultFeeDenom, 42),
				))
				txBuilder.SetFeeGranter(addrs[2])
				return txBuilder.GetTx()
			},
			setup: func(m mocks) {
				m.FeeMarketKeeper.EXPECT().GetParams(m.ctx).
					Return(types.DefaultParams(), nil).Times(2)
				m.FeeMarketKeeper.EXPECT().GetMinGasPrice(m.ctx, types.DefaultFeeDenom).
					Return(sdk.NewInt64DecCoin(types.DefaultFeeDenom, 1), nil)
				m.FeeGrantKeeper.EXPECT().UseGrantedFees(gomock.Any(), addrs[2],
					addrs[0], sdk.NewCoins(sdk.NewInt64Coin(types.DefaultFeeDenom, 42)),
					gomock.Any()).
					Return(errors.New("NOPE"))
			},
			expectedError: fmt.Sprintf(
				"error deducting fee: %s does not allow to pay fees for %s: NOPE",
				acc3.Address, acc1.Address),
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
				feeGrantKeeper ante.FeeGrantKeeper
			)
			if !tt.disableFeeGrant {
				feeGrantKeeper = m.FeeGrantKeeper
			}
			fmd := ante.NewFeeMarketCheckDecorator(m.AccountKeeper, m.BankKeeper, feeGrantKeeper, m.FeeMarketKeeper, nil)
			if tt.genTx {
				m.ctx = m.ctx.WithBlockHeight(0)
			} else {
				m.ctx = m.ctx.WithBlockHeight(1)
			}
			if tt.setup != nil {
				tt.setup(m)
			}

			newCtx, err := fmd.AnteHandle(m.ctx, tt.tx(), tt.simulate, next)

			if tt.expectedError != "" {
				require.EqualError(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			assert.True(t, nextInvoked, "next is not invoked")
			assert.EqualValues(t, tt.expectedConsumedGas, newCtx.GasMeter().GasConsumed())
			assert.Equal(t, tt.expectedMinGasPrices, newCtx.MinGasPrices().String(), "wrong min gas price")
			assert.Equal(t, tt.expectedTxPriority, newCtx.Priority(), "wrong tx priority")
		})
	}
}
