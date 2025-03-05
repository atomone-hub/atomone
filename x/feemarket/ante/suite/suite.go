package suite

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	testkeeper "github.com/skip-mev/feemarket/testutils/keeper"
	feemarketante "github.com/skip-mev/feemarket/x/feemarket/ante"
	"github.com/skip-mev/feemarket/x/feemarket/ante/mocks"
	feemarketkeeper "github.com/skip-mev/feemarket/x/feemarket/keeper"
	feemarketpost "github.com/skip-mev/feemarket/x/feemarket/post"
	feemarkettypes "github.com/skip-mev/feemarket/x/feemarket/types"
)

type TestSuite struct {
	suite.Suite

	Ctx         sdk.Context
	AnteHandler sdk.AnteHandler
	PostHandler sdk.PostHandler
	ClientCtx   client.Context
	TxBuilder   client.TxBuilder

	AccountKeeper   feemarketante.AccountKeeper
	FeeMarketKeeper *feemarketkeeper.Keeper
	BankKeeper      feemarketante.BankKeeper
	FeeGrantKeeper  feemarketante.FeeGrantKeeper

	MockBankKeeper     *mocks.BankKeeper
	MockFeeGrantKeeper *mocks.FeeGrantKeeper

	EncCfg TestEncodingConfig

	MsgServer feemarkettypes.MsgServer
}

// TestAccount represents an account used in the tests in x/auth/ante.
type TestAccount struct {
	Account authtypes.AccountI
	Priv    cryptotypes.PrivKey
}

func (s *TestSuite) CreateTestAccounts(numAccs int) []TestAccount {
	var accounts []TestAccount

	for i := 0; i < numAccs; i++ {
		priv, _, addr := testdata.KeyTestPubAddr()
		acc := s.AccountKeeper.NewAccountWithAddress(s.Ctx, addr)
		err := acc.SetAccountNumber(uint64(i + 1000))
		if err != nil {
			panic(err)
		}
		s.AccountKeeper.SetAccount(s.Ctx, acc)
		accounts = append(accounts, TestAccount{acc, priv})
	}

	return accounts
}

// SetupTestSuite setups a new test, with new app, context, and anteHandler.
func SetupTestSuite(t *testing.T, mock bool) *TestSuite {
	s := &TestSuite{}

	s.EncCfg = MakeTestEncodingConfig()
	ctx, testKeepers, _ := testkeeper.NewTestSetup(t)
	s.Ctx = ctx

	s.AccountKeeper = testKeepers.AccountKeeper
	s.FeeMarketKeeper = testKeepers.FeeMarketKeeper
	s.MockBankKeeper = mocks.NewBankKeeper(t)
	s.MockFeeGrantKeeper = mocks.NewFeeGrantKeeper(t)

	s.ClientCtx = client.Context{}.WithTxConfig(s.EncCfg.TxConfig)
	s.TxBuilder = s.ClientCtx.TxConfig.NewTxBuilder()

	s.FeeMarketKeeper.SetEnabledHeight(s.Ctx, -1)
	s.MsgServer = feemarketkeeper.NewMsgServer(s.FeeMarketKeeper)

	s.SetupHandlers(mock)
	s.SetT(t)
	return s
}

func (s *TestSuite) SetupHandlers(mock bool) {
	bankKeeper := s.BankKeeper
	feeGrantKeeper := s.FeeGrantKeeper

	if mock {
		bankKeeper = s.MockBankKeeper
		feeGrantKeeper = s.MockFeeGrantKeeper
	}

	// create basic antehandler with the feemarket decorator
	anteDecorators := []sdk.AnteDecorator{
		authante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		feemarketante.NewFeeMarketCheckDecorator( // fee market replaces fee deduct decorator
			s.AccountKeeper,
			bankKeeper,
			feeGrantKeeper,
			s.FeeMarketKeeper,
			authante.NewDeductFeeDecorator(
				s.AccountKeeper,
				bankKeeper,
				feeGrantKeeper,
				nil,
			),
		),
		authante.NewSigGasConsumeDecorator(s.AccountKeeper, authante.DefaultSigVerificationGasConsumer),
	}

	s.AnteHandler = sdk.ChainAnteDecorators(anteDecorators...)

	// create basic postHandler with the feemarket decorator
	postDecorators := []sdk.PostDecorator{
		feemarketpost.NewFeeMarketDeductDecorator(
			s.AccountKeeper,
			bankKeeper,
			s.FeeMarketKeeper,
		),
	}

	s.PostHandler = sdk.ChainPostDecorators(postDecorators...)
}

// TestCase represents a test case used in test tables.
type TestCase struct {
	Name              string
	Malleate          func(*TestSuite) TestCaseArgs
	StateUpdate       func(*TestSuite)
	RunAnte           bool
	RunPost           bool
	Simulate          bool
	ExpPass           bool
	ExpErr            error
	ExpectConsumedGas uint64
}

type TestCaseArgs struct {
	ChainID   string
	AccNums   []uint64
	AccSeqs   []uint64
	FeeAmount sdk.Coins
	GasLimit  uint64
	Msgs      []sdk.Msg
	Privs     []cryptotypes.PrivKey
}

// DeliverMsgs constructs a tx and runs it through the ante handler. This is used to set the context for a test case, for
// example to test for replay protection.
func (s *TestSuite) DeliverMsgs(t *testing.T, privs []cryptotypes.PrivKey, msgs []sdk.Msg, feeAmount sdk.Coins, gasLimit uint64, accNums, accSeqs []uint64, chainID string, simulate bool) (sdk.Context, error) {
	require.NoError(t, s.TxBuilder.SetMsgs(msgs...))
	s.TxBuilder.SetFeeAmount(feeAmount)
	s.TxBuilder.SetGasLimit(gasLimit)

	tx, txErr := s.CreateTestTx(privs, accNums, accSeqs, chainID)
	require.NoError(t, txErr)
	return s.AnteHandler(s.Ctx, tx, simulate)
}

func (s *TestSuite) RunTestCase(t *testing.T, tc TestCase, args TestCaseArgs) {
	require.NoError(t, s.TxBuilder.SetMsgs(args.Msgs...))
	s.TxBuilder.SetFeeAmount(args.FeeAmount)
	s.TxBuilder.SetGasLimit(args.GasLimit)

	// Theoretically speaking, ante handler unit tests should only test
	// ante handlers, but here we sometimes also test the tx creation
	// process.
	tx, txErr := s.CreateTestTx(args.Privs, args.AccNums, args.AccSeqs, args.ChainID)

	var (
		newCtx  sdk.Context
		anteErr error
		postErr error
	)

	if tc.RunAnte {
		newCtx, anteErr = s.AnteHandler(s.Ctx, tx, tc.Simulate)
	}

	// perform mid-tx state update if configured
	if tc.StateUpdate != nil {
		tc.StateUpdate(s)
	}

	if tc.RunPost && anteErr == nil {
		newCtx, postErr = s.PostHandler(s.Ctx, tx, tc.Simulate, true)
	}

	if tc.ExpPass {
		require.NoError(t, txErr)
		require.NoError(t, anteErr)
		require.NoError(t, postErr)
		require.NotNil(t, newCtx)

		s.Ctx = newCtx
		if tc.RunPost {
			consumedGas := newCtx.GasMeter().GasConsumed()
			require.Equal(t, tc.ExpectConsumedGas, consumedGas)
		}

	} else {
		switch {
		case txErr != nil:
			require.Error(t, txErr)
			require.ErrorIs(t, txErr, tc.ExpErr)

		case anteErr != nil:
			require.Error(t, anteErr)
			require.NoError(t, postErr)
			require.ErrorIs(t, anteErr, tc.ExpErr)

		case postErr != nil:
			require.NoError(t, anteErr)
			require.Error(t, postErr)
			require.ErrorIs(t, postErr, tc.ExpErr)

		default:
			t.Fatal("expected one of txErr, handleErr to be an error")
		}
	}
}

// CreateTestTx is a helper function to create a tx given multiple inputs.
func (s *TestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (authsigning.Tx, error) {
	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {
		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  s.ClientCtx.TxConfig.SignModeHandler().DefaultMode(),
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err := s.TxBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := authsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
		}
		sigV2, err := tx.SignWithPrivKey(
			s.ClientCtx.TxConfig.SignModeHandler().DefaultMode(), signerData,
			s.TxBuilder, priv, s.ClientCtx.TxConfig, accSeqs[i])
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = s.TxBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	return s.TxBuilder.GetTx(), nil
}

// NewTestFeeAmount is a test fee amount.
func NewTestFeeAmount() sdk.Coins {
	return sdk.NewCoins(sdk.NewInt64Coin("stake", 150))
}

// NewTestGasLimit is a test fee gas limit.
func NewTestGasLimit() uint64 {
	return 200000
}

// TestEncodingConfig specifies the concrete encoding types to use for a given app.
// This is provided for compatibility between protobuf and amino implementations.
type TestEncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// MakeTestEncodingConfig creates a test EncodingConfig for a test configuration.
func MakeTestEncodingConfig() TestEncodingConfig {
	amino := codec.NewLegacyAmino()

	interfaceRegistry := InterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	txCfg := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	std.RegisterLegacyAminoCodec(amino)
	std.RegisterInterfaces(interfaceRegistry)

	return TestEncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             cdc,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

func InterfaceRegistry() codectypes.InterfaceRegistry {
	interfaceRegistry := codectypes.NewInterfaceRegistry()

	// always register
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	authtypes.RegisterInterfaces(interfaceRegistry)

	// call extra registry functions
	feemarkettypes.RegisterInterfaces(interfaceRegistry)

	return interfaceRegistry
}
