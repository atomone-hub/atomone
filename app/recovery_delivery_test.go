//go:build recoverylock_test

package atomone_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	"cosmossdk.io/math"

	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	dynamicfeetypes "github.com/cosmos/cosmos-sdk/x/dynamicfee/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/app/recoverylock"
	"github.com/atomone-hub/atomone/app/recoverylock/core"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

type recoveryDeliveryFixture struct {
	recoveryFixture
	walletSigner, payerSigner, otherSigner *secp256k1.PrivKey
}

func newRecoveryDeliveryFixture(t *testing.T) recoveryDeliveryFixture {
	t.Helper()
	f := recoveryDeliveryFixture{
		recoveryFixture: newRecoveryFixture(),
		walletSigner:    secp256k1.GenPrivKey(),
		payerSigner:     secp256k1.GenPrivKey(),
		otherSigner:     secp256k1.GenPrivKey(),
	}
	f.listed = sdk.AccAddress(f.walletSigner.PubKey().Address())
	f.payer = sdk.AccAddress(f.payerSigner.PubKey().Address())
	f.other = sdk.AccAddress(f.otherSigner.PubKey().Address())
	cfg, err := core.NewConfig([][]byte{f.listed}, f.recovery, [][]byte{f.payer}, f.ctx.BlockHeight(), f.expiry.Unix())
	require.NoError(t, err)
	// Keep the same lock pointer captured by the installed ante and keeper guards.
	*f.app.RecoveryLock = *recoverylock.New(cfg, f.ctx.ChainID(), f.app.AppCodec(), &f.app.DistrKeeper, f.app.LastBlockHeight)
	f.ctx = f.ctx.WithExecMode(sdk.ExecModeFinalize)

	accounts := f.app.AccountKeeper
	require.NoError(t, accounts.Params.Set(f.ctx, authtypes.DefaultParams()))
	require.NoError(t, f.app.BankKeeper.SetParams(f.ctx, banktypes.DefaultParams()))
	params := dynamicfeetypes.DefaultParams()
	params.Enabled = false
	params.FeeDenom = photontypes.Denom
	require.NoError(t, f.app.DynamicfeeKeeper.SetParams(f.ctx, params))
	require.NoError(t, f.app.PhotonKeeper.SetParams(f.ctx, photontypes.DefaultParams()))
	for _, addr := range []sdk.AccAddress{f.listed, f.payer, f.other} {
		accounts.SetAccount(f.ctx, accounts.NewAccountWithAddress(f.ctx, addr))
	}
	return f
}

func (f recoveryDeliveryFixture) fund(t *testing.T, addr sdk.AccAddress, coins sdk.Coins) {
	t.Helper()
	require.NoError(t, f.app.BankKeeper.MintCoins(f.ctx, transfertypes.ModuleName, coins))
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, addr, coins))
}

func (f recoveryDeliveryFixture) signedTx(t *testing.T, fee sdk.Coins, payer *secp256k1.PrivKey, msgs ...sdk.Msg) []byte {
	t.Helper()
	return f.signTx(t, fee, f.walletSigner, payer, msgs...)
}

func (f recoveryDeliveryFixture) signTx(t *testing.T, fee sdk.Coins, sender, payer *secp256k1.PrivKey, msgs ...sdk.Msg) []byte {
	t.Helper()
	cfg := f.app.TxConfig()
	builder := cfg.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	builder.SetGasLimit(500_000)
	builder.SetFeeAmount(fee)
	builder.SetFeePayer(sdk.AccAddress(payer.PubKey().Address()))

	signers := []*secp256k1.PrivKey{sender}
	if sender != payer {
		signers = append(signers, payer)
	}
	signerData := make([]authsigning.SignerData, len(signers))
	sigs := make([]signing.SignatureV2, len(signers))
	const mode = signing.SignMode_SIGN_MODE_DIRECT
	for i, signer := range signers {
		pub := signer.PubKey()
		addr := sdk.AccAddress(pub.Address())
		account := f.app.AccountKeeper.GetAccount(f.ctx, addr)
		signerData[i] = authsigning.SignerData{
			ChainID: f.ctx.ChainID(), AccountNumber: account.GetAccountNumber(),
			Sequence: account.GetSequence(), PubKey: pub, Address: addr.String(),
		}
		sigs[i] = signing.SignatureV2{
			PubKey: pub, Sequence: account.GetSequence(),
			Data: &signing.SingleSignatureData{SignMode: mode},
		}
	}
	require.NoError(t, builder.SetSignatures(sigs...))
	for i, signer := range signers {
		sig, err := clienttx.SignWithPrivKey(f.ctx, mode, signerData[i], builder, signer, cfg, signerData[i].Sequence)
		require.NoError(t, err)
		sigs[i] = sig
	}
	require.NoError(t, builder.SetSignatures(sigs...))
	txBytes, err := cfg.TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

func (f recoveryDeliveryFixture) deliver(t *testing.T, txBytes []byte) error {
	t.Helper()
	tx, err := f.app.TxConfig().TxDecoder()(txBytes)
	require.NoError(t, err)
	anteCtx, writeAnte := f.ctx.WithTxBytes(txBytes).CacheContext()
	anteCtx, err = f.app.GetBaseApp().AnteHandler()(anteCtx, tx, false)
	if err != nil {
		return err
	}
	writeAnte()

	// As in BaseApp, successful ante writes survive a later message failure.
	msgCtx, writeMsgs := anteCtx.WithMultiStore(f.ctx.MultiStore()).CacheContext()
	for _, msg := range tx.GetMsgs() {
		handler := f.app.MsgServiceRouter().Handler(msg)
		require.NotNil(t, handler, "route for %T", msg)
		_, err := handler(msgCtx, msg)
		require.NoError(t, err, "message execution for %T", msg)
	}
	writeMsgs()
	return nil
}

func TestRecoveryDeliverySend(t *testing.T) {
	for _, tc := range []struct {
		name, refusal                    string
		wrongPayer, atoneFee, thirdParty bool
	}{
		{name: "sponsored"},
		{name: "wrong payer", wrongPayer: true, refusal: "the configured fee payer must pay directly"},
		{name: "ATONE fee", atoneFee: true, refusal: "recovery requires a positive uphoton fee"},
		{name: "third party", thirdParty: true, refusal: "transaction must contain only direct recovery messages from listed accounts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRecoveryDeliveryFixture(t)
			bank := f.app.BankKeeper
			fee := photonFee()
			atoneFee := sdk.NewCoins(sdk.NewInt64Coin("uatone", 1000))
			sponsorBalance := fee.Add(atoneFee...)
			walletBalance := sdk.NewCoins(sdk.NewInt64Coin("uatone", 42000), sdk.NewInt64Coin(photontypes.Denom, 1234))
			f.fund(t, f.listed, walletBalance)
			f.fund(t, f.payer, sponsorBalance)
			f.fund(t, f.other, sponsorBalance)

			payer := f.payerSigner
			if tc.wrongPayer {
				payer = f.otherSigner
			}
			if tc.atoneFee {
				fee = atoneFee
			}
			to := f.recovery
			if tc.thirdParty {
				to = f.other
			}
			send := banktypes.NewMsgSend(f.listed, to, walletBalance)
			err := f.deliver(t, f.signedTx(t, fee, payer, send))

			feeCollector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
			sequence := func(addr sdk.AccAddress) uint64 {
				return f.app.AccountKeeper.GetAccount(f.ctx, addr).GetSequence()
			}
			require.Equal(t, sponsorBalance, bank.GetAllBalances(f.ctx, f.other))
			require.Zero(t, sequence(f.other))
			if tc.refusal != "" {
				require.ErrorIs(t, err, recoverylock.ErrLocked)
				require.ErrorContains(t, err, tc.refusal)
				require.Equal(t, walletBalance, bank.GetAllBalances(f.ctx, f.listed))
				require.Equal(t, sponsorBalance, bank.GetAllBalances(f.ctx, f.payer))
				require.Empty(t, bank.GetAllBalances(f.ctx, f.recovery))
				require.Empty(t, bank.GetAllBalances(f.ctx, feeCollector))
				require.Zero(t, sequence(f.listed))
				require.Zero(t, sequence(f.payer))
				return
			}

			require.NoError(t, err)
			require.Empty(t, bank.GetAllBalances(f.ctx, f.listed))
			require.Equal(t, walletBalance, bank.GetAllBalances(f.ctx, f.recovery))
			require.Equal(t, atoneFee, bank.GetAllBalances(f.ctx, f.payer))
			require.Equal(t, fee, bank.GetAllBalances(f.ctx, feeCollector))
			require.EqualValues(t, 1, sequence(f.listed))
			require.EqualValues(t, 1, sequence(f.payer))
		})
	}
}

func TestRecoveryDeliveryRequiresSponsorSignature(t *testing.T) {
	f := newRecoveryDeliveryFixture(t)
	f.fund(t, f.payer, photonFee())
	balance := sdk.NewCoins(sdk.NewInt64Coin("uatone", 42_000))
	f.fund(t, f.listed, balance)
	send := banktypes.NewMsgSend(f.listed, f.recovery, balance)
	cfg := f.app.TxConfig()
	tx, err := cfg.TxDecoder()(f.signedTx(t, photonFee(), f.payerSigner, send))
	require.NoError(t, err)
	builder, err := cfg.WrapTxBuilder(tx)
	require.NoError(t, err)
	sigs, err := tx.(authsigning.SigVerifiableTx).GetSignaturesV2()
	require.NoError(t, err)
	payerSig := sigs[1].Data.(*signing.SingleSignatureData)
	payerSig.Signature[0] ^= 1
	require.NoError(t, builder.SetSignatures(sigs...))
	txBytes, err := cfg.TxEncoder()(builder.GetTx())
	require.NoError(t, err)

	require.ErrorIs(t, f.deliver(t, txBytes), sdkerrors.ErrUnauthorized)
	require.Equal(t, balance, f.app.BankKeeper.GetAllBalances(f.ctx, f.listed))
	require.Equal(t, photonFee(), f.app.BankKeeper.GetAllBalances(f.ctx, f.payer))
	require.Empty(t, f.app.BankKeeper.GetAllBalances(f.ctx, f.recovery))
}

func (f recoveryDeliveryFixture) seedDelegation(t *testing.T) (stakingtypes.Validator, sdk.Coin) {
	t.Helper()
	staking, distr := f.app.StakingKeeper, f.app.DistrKeeper
	current := f.ctx
	before := current.WithBlockHeight(current.BlockHeight() - 1)
	params := stakingtypes.DefaultParams()
	params.BondDenom = "uatone"
	require.NoError(t, staking.SetParams(before, params))
	require.NoError(t, distr.Params.Set(before, distrtypes.DefaultParams()))
	require.NoError(t, distr.FeePool.Set(before, distrtypes.InitialFeePool()))

	valAddr := sdk.ValAddress(f.listed)
	val, err := stakingtypes.NewValidator(valAddr.String(), &ed25519.PubKey{Key: make([]byte, ed25519.PubKeySize)}, stakingtypes.Description{})
	require.NoError(t, err)
	rate := math.LegacyNewDecWithPrec(2, 1)
	val.Commission = stakingtypes.NewCommission(rate, rate, rate)
	require.NoError(t, staking.SetValidator(before, val))
	require.NoError(t, distr.Hooks().AfterValidatorCreated(before, valAddr))

	stake := sdk.NewInt64Coin(params.BondDenom, 100)
	f.fund(t, f.listed, sdk.NewCoins(stake))
	_, err = staking.Delegate(before, f.listed, stake.Amount, stakingtypes.Unbonded, val, true)
	require.NoError(t, err)
	val, err = staking.GetValidator(current, valAddr)
	require.NoError(t, err)
	return val, stake
}

func TestRecoveryDeliveryRewardAndCommission(t *testing.T) {
	for _, destination := range []string{"self", "recovery"} {
		t.Run(destination, func(t *testing.T) {
			f := newRecoveryDeliveryFixture(t)
			val, stake := f.seedDelegation(t)
			bank, distr := f.app.BankKeeper, f.app.DistrKeeper
			f.fund(t, f.payer, photonFee().Add(photonFee()...))
			rewards := sdk.NewCoins(sdk.NewInt64Coin(stake.Denom, 50))
			require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, rewards))
			require.NoError(t, bank.SendCoinsFromModuleToModule(f.ctx, transfertypes.ModuleName, distrtypes.ModuleName, rewards))
			require.NoError(t, distr.AllocateTokensToValidator(f.ctx, val, sdk.NewDecCoinsFromCoins(rewards...)))

			recipient, other := f.listed, f.recovery
			withdraw := &distrtypes.MsgWithdrawDelegatorReward{
				DelegatorAddress: f.listed.String(), ValidatorAddress: val.OperatorAddress,
			}
			msgs := []sdk.Msg{withdraw}
			if destination == "recovery" {
				recipient, other = f.recovery, f.listed
				msgs = []sdk.Msg{distrtypes.NewMsgSetWithdrawAddress(f.listed, f.recovery), withdraw}
			}
			require.NoError(t, f.deliver(t, f.signedTx(t, photonFee(), f.payerSigner, msgs...)))
			require.Equal(t, sdk.NewInt64Coin(stake.Denom, 40), bank.GetBalance(f.ctx, recipient, stake.Denom))
			require.Empty(t, bank.GetAllBalances(f.ctx, other))

			commission := &distrtypes.MsgWithdrawValidatorCommission{ValidatorAddress: val.OperatorAddress}
			require.NoError(t, f.deliver(t, f.signedTx(t, photonFee(), f.payerSigner, commission)))
			require.Equal(t, rewards, bank.GetAllBalances(f.ctx, recipient))
			require.Empty(t, bank.GetAllBalances(f.ctx, other))
			require.Empty(t, bank.GetAllBalances(f.ctx, f.payer))
		})
	}
}

func TestRecoveryDeliveryUndelegateAndCancel(t *testing.T) {
	f := newRecoveryDeliveryFixture(t)
	val, stake := f.seedDelegation(t)
	staking := f.app.StakingKeeper
	valAddr := sdk.ValAddress(f.listed)
	f.fund(t, f.payer, photonFee().Add(photonFee()...))

	amount := sdk.NewInt64Coin(stake.Denom, 40)
	undelegate := &stakingtypes.MsgUndelegate{
		DelegatorAddress: f.listed.String(), ValidatorAddress: val.OperatorAddress, Amount: amount,
	}
	require.NoError(t, f.deliver(t, f.signedTx(t, photonFee(), f.payerSigner, undelegate)))
	ubd, err := staking.GetUnbondingDelegation(f.ctx, f.listed, valAddr)
	require.NoError(t, err)
	require.Len(t, ubd.Entries, 1)
	require.Equal(t, amount.Amount, ubd.Entries[0].Balance)
	require.Equal(t, f.ctx.BlockHeight(), ubd.Entries[0].CreationHeight)
	delegation, err := staking.GetDelegation(f.ctx, f.listed, valAddr)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(60), delegation.Shares)

	cancel := &stakingtypes.MsgCancelUnbondingDelegation{
		DelegatorAddress: f.listed.String(), ValidatorAddress: val.OperatorAddress,
		Amount: amount, CreationHeight: f.ctx.BlockHeight(),
	}
	f.ctx = f.ctx.WithBlockHeight(f.ctx.BlockHeight() + 1)
	require.NoError(t, f.deliver(t, f.signedTx(t, photonFee(), f.payerSigner, cancel)))
	_, err = staking.GetUnbondingDelegation(f.ctx, f.listed, valAddr)
	require.ErrorIs(t, err, stakingtypes.ErrNoUnbondingDelegation)
	delegation, err = staking.GetDelegation(f.ctx, f.listed, valAddr)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDec(100), delegation.Shares)
	require.Empty(t, f.app.BankKeeper.GetAllBalances(f.ctx, f.listed))
	require.Empty(t, f.app.BankKeeper.GetAllBalances(f.ctx, f.payer))
}
