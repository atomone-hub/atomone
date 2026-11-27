//go:build recoverylock_test

package atomone_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	dynamicfeetypes "github.com/cosmos/cosmos-sdk/x/dynamicfee/types"

	atomone "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/app/helpers"
	"github.com/atomone-hub/atomone/app/recoverylock"
	"github.com/atomone-hub/atomone/app/recoverylock/core"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

func TestRecoveryBlockLifecycle(t *testing.T) {
	for _, mode := range []struct {
		name    string
		dynamic bool
	}{{"dynamic", true}, {"legacy", false}} {
		t.Run(mode.name, func(t *testing.T) {
			testRecoveryBlockLifecycle(t, mode.dynamic)
		})
	}
}

func testRecoveryBlockLifecycle(t *testing.T, dynamic bool) {
	t.Helper()
	const activation int64 = 2
	_, chainID := recoverylock.MustLoad(nil, nil)
	expiry := time.Unix(2_000_000_000, 0).UTC()
	start := expiry.Add(-time.Minute)
	wallet, sponsor, other := secp256k1.GenPrivKey(), secp256k1.GenPrivKey(), secp256k1.GenPrivKey()
	listed := sdk.AccAddress(wallet.PubKey().Address())
	payer := sdk.AccAddress(sponsor.PubKey().Address())
	unlisted := sdk.AccAddress(other.PubKey().Address())
	recovery := sdk.AccAddress([]byte("recovery destination"))
	pay := sdk.NewCoins(sdk.NewInt64Coin(photontypes.Denom, 50_000))
	atoneFee := sdk.NewCoins(sdk.NewInt64Coin("uatone", 50_000))
	walletFunds := sdk.NewCoins(sdk.NewInt64Coin("uatone", 10))
	sponsorFunds := pay.Add(pay...).Add(pay...).Add(atoneFee...)
	otherFunds := sponsorFunds
	one := sdk.NewCoins(sdk.NewInt64Coin("uatone", 1))
	validator := tmtypes.NewValidator(make(cmted25519.PubKey, cmted25519.PubKeySize), 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	newChain := func(lockChainID string) recoveryDeliveryFixture {
		app := atomone.NewAtomOneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, atomone.EmptyAppOptions{}, baseapp.SetChainID(chainID))
		cfg, err := core.NewConfig([][]byte{listed}, recovery, [][]byte{payer}, activation, expiry.Unix())
		require.NoError(t, err)
		*app.RecoveryLock = *recoverylock.New(cfg, lockChainID, app.AppCodec(), &app.DistrKeeper, app.LastBlockHeight)
		cdc := app.AppCodec()
		genesis, err := simtestutil.GenesisStateWithValSet(cdc, app.DefaultGenesis(), valSet,
			[]authtypes.GenesisAccount{
				authtypes.NewBaseAccount(listed, wallet.PubKey(), 0, 0),
				authtypes.NewBaseAccount(payer, sponsor.PubKey(), 0, 0),
				authtypes.NewBaseAccount(unlisted, other.PubKey(), 0, 0),
			},
			banktypes.Balance{Address: listed.String(), Coins: walletFunds},
			banktypes.Balance{Address: payer.String(), Coins: sponsorFunds},
			banktypes.Balance{Address: unlisted.String(), Coins: otherFunds},
		)
		require.NoError(t, err)
		fees := dynamicfeetypes.GetGenesisStateFromAppState(cdc, genesis)
		fees.Params.Enabled = dynamic
		fees.Params.FeeDenom = photontypes.Denom
		genesis[dynamicfeetypes.ModuleName] = cdc.MustMarshalJSON(&fees)
		state, err := json.Marshal(genesis)
		require.NoError(t, err)
		params := *helpers.DefaultConsensusParams
		_, err = app.InitChain(&abci.RequestInitChain{ChainId: chainID, Time: start, InitialHeight: 1, ConsensusParams: &params, AppStateBytes: state})
		require.NoError(t, err)
		return recoveryDeliveryFixture{
			recoveryFixture: recoveryFixture{app: app, listed: listed, payer: payer, other: unlisted, recovery: recovery, expiry: expiry},
			walletSigner:    wallet, payerSigner: sponsor, otherSigner: other,
		}
	}
	locked, baseline := newChain(chainID), newChain(chainID+"-inactive")
	commit := func(f recoveryDeliveryFixture, height int64, blockTime time.Time, txs ...[]byte) []*abci.ExecTxResult {
		res, err := f.app.FinalizeBlock(&abci.RequestFinalizeBlock{
			Height: height, Time: blockTime, Txs: txs,
			ProposerAddress: validator.Address, NextValidatorsHash: valSet.Hash(),
		})
		require.NoError(t, err)
		require.Len(t, res.TxResults, len(txs))
		_, err = f.app.Commit()
		require.NoError(t, err)
		require.Equal(t, height, f.app.LastBlockHeight())
		return res.TxResults
	}
	check := func(f recoveryDeliveryFixture, tx []byte) *abci.ResponseCheckTx {
		res, err := f.app.CheckTx(&abci.RequestCheckTx{Tx: tx, Type: abci.CheckTxType_New})
		require.NoError(t, err)
		return res
	}
	checkUnlisted := func(tx []byte) {
		got := check(locked, tx)
		require.Zero(t, got.Code, got.Log)
		require.Positive(t, got.GasUsed)
		require.Equal(t, check(baseline, tx), got)
	}
	committed := func(height int64, blockTime time.Time) sdk.Context {
		return locked.app.NewUncachedContext(false, tmproto.Header{ChainID: chainID, Height: height, Time: blockTime})
	}

	commit(locked, activation-1, start)
	commit(baseline, activation-1, start)
	locked.ctx = committed(activation-1, start)
	send := banktypes.NewMsgSend(listed, recovery, one)
	wrong := locked.signedTx(t, atoneFee, sponsor, send)
	refused := check(locked, wrong)
	require.Equal(t, recoverylock.ErrLocked.ABCICode(), refused.Code)
	require.Equal(t, recoverylock.ErrLocked.Codespace(), refused.Codespace)
	require.Contains(t, refused.Log, "recovery requires a positive uphoton fee")
	require.NotZero(t, check(baseline, wrong).Code)

	if dynamic {
		underpriced := locked.signedTx(t, photonFee(), sponsor, send)
		got := check(locked, underpriced)
		require.Equal(t, sdkerrors.ErrInsufficientFee.ABCICode(), got.Code, got.Log)
		require.Contains(t, got.Log, "minGasPrice")
	}

	success := locked.signedTx(t, pay, sponsor, send)
	got := check(locked, success)
	require.Zero(t, got.Code, got.Log)
	got = check(baseline, success)
	require.Zero(t, got.Code, got.Log)

	// Sign with the sequences advanced by the successful CheckTx. The first
	// message of this transaction must roll back when the second one fails.
	checked := locked
	checked.ctx = locked.app.GetContextForCheckTx(nil)
	tooMuch := banktypes.NewMsgSend(listed, recovery, walletFunds)
	failure := checked.signedTx(t, pay, sponsor, send, tooMuch)
	got = check(locked, failure)
	require.Zero(t, got.Code, got.Log)
	got = check(baseline, failure)
	require.Zero(t, got.Code, got.Log)

	unlistedSend := banktypes.NewMsgSend(unlisted, recovery, one)
	unlistedTx := locked.signTx(t, pay, other, other, unlistedSend)
	checkUnlisted(unlistedTx)
	activatedAt := start.Add(time.Second)
	results := commit(locked, activation, activatedAt, wrong, success, failure, unlistedTx)
	control := commit(baseline, activation, activatedAt, wrong, success, failure, unlistedTx)
	require.Equal(t, recoverylock.ErrLocked.ABCICode(), results[0].Code)
	require.Equal(t, recoverylock.ErrLocked.Codespace(), results[0].Codespace)
	require.Contains(t, results[0].Log, "recovery requires a positive uphoton fee")
	require.Zero(t, results[1].Code, results[1].Log)
	require.Equal(t, sdkerrors.ErrInsufficientFunds.ABCICode(), results[2].Code)
	require.Equal(t, sdkerrors.ErrInsufficientFunds.Codespace(), results[2].Codespace)
	require.Contains(t, results[2].Log, "message index: 1")
	require.Zero(t, results[3].Code, results[3].Log)
	require.NotEmpty(t, results[3].Events)
	require.Equal(t, control[3], results[3])

	locked.ctx = committed(activation, activatedAt)
	bank, accounts := locked.app.BankKeeper, locked.app.AccountKeeper
	require.Equal(t, walletFunds.Sub(one...), bank.GetAllBalances(locked.ctx, listed))
	require.Equal(t, one.Add(one...), bank.GetAllBalances(locked.ctx, recovery))
	require.Equal(t, sponsorFunds.Sub(pay...).Sub(pay...), bank.GetAllBalances(locked.ctx, payer))
	require.Equal(t, pay.Add(pay...).Add(pay...), bank.GetAllBalances(locked.ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName)))
	require.EqualValues(t, 2, accounts.GetAccount(locked.ctx, listed).GetSequence())
	require.EqualValues(t, 2, accounts.GetAccount(locked.ctx, payer).GetSequence())

	// Before the expiry block commits, CheckTx still uses the active header.
	alternate := banktypes.NewMsgSend(listed, unlisted, one)
	expiring := locked.signedTx(t, pay, sponsor, alternate)
	refused = check(locked, expiring)
	require.Equal(t, recoverylock.ErrLocked.ABCICode(), refused.Code)
	require.Equal(t, recoverylock.ErrLocked.Codespace(), refused.Codespace)
	got = check(baseline, expiring)
	require.Zero(t, got.Code, got.Log)
	unlistedTx = locked.signTx(t, pay, other, other, unlistedSend)
	checkUnlisted(unlistedTx)
	results = commit(locked, activation+1, expiry, expiring, unlistedTx)
	control = commit(baseline, activation+1, expiry, expiring, unlistedTx)
	require.Zero(t, results[0].Code, results[0].Log)
	require.Zero(t, results[1].Code, results[1].Log)
	require.Equal(t, control, results)

	locked.ctx = committed(activation+1, expiry)
	require.Equal(t, walletFunds.Sub(one...).Sub(one...), bank.GetAllBalances(locked.ctx, listed))
	require.EqualValues(t, 3, accounts.GetAccount(locked.ctx, listed).GetSequence())
	require.EqualValues(t, 3, accounts.GetAccount(locked.ctx, payer).GetSequence())
	// Refill is unnecessary: the unlisted account can sponsor the next send.
	afterExpiry := locked.signTx(t, pay, wallet, other, alternate)
	got = check(locked, afterExpiry)
	require.Zero(t, got.Code, got.Log)
	require.Equal(t, check(baseline, afterExpiry), got)
}
