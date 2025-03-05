package testutils

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type Account struct {
	PrivKey cryptotypes.PrivKey
	PubKey  cryptotypes.PubKey
	Address sdk.AccAddress
	ConsKey cryptotypes.PrivKey
}

func (acc Account) Equals(acc2 Account) bool {
	return acc.Address.Equals(acc2.Address)
}

func RandomAccounts(r *rand.Rand, n int) []Account {
	accs := make([]Account, n)

	for i := 0; i < n; i++ {
		pkSeed := make([]byte, 15)
		r.Read(pkSeed)

		accs[i].PrivKey = secp256k1.GenPrivKeyFromSecret(pkSeed)
		accs[i].PubKey = accs[i].PrivKey.PubKey()
		accs[i].Address = sdk.AccAddress(accs[i].PubKey.Address())

		accs[i].ConsKey = ed25519.GenPrivKeyFromSecret(pkSeed)
	}

	return accs
}

func CreateRandomTx(txCfg client.TxConfig, account Account, nonce, numberMsgs, timeout uint64, gasLimit uint64, fees ...sdk.Coin) (authsigning.Tx, error) {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < int(numberMsgs); i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: account.Address.String(),
			ToAddress:   account.Address.String(),
		}
	}

	txBuilder := txCfg.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	sigV2 := signing.SignatureV2{
		PubKey: account.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: nonce,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBuilder.SetTimeoutHeight(timeout)

	txBuilder.SetFeeAmount(fees)

	txBuilder.SetGasLimit(gasLimit)

	return txBuilder.GetTx(), nil
}

func CreateRandomTxBz(txCfg client.TxConfig, account Account, nonce, numberMsgs, timeout, gasLimit uint64) ([]byte, error) {
	tx, err := CreateRandomTx(txCfg, account, nonce, numberMsgs, timeout, gasLimit)
	if err != nil {
		return nil, err
	}

	return txCfg.TxEncoder()(tx)
}

func CreateRandomMsgs(acc sdk.AccAddress, numberMsgs int) []sdk.Msg {
	msgs := make([]sdk.Msg, numberMsgs)
	for i := 0; i < numberMsgs; i++ {
		msgs[i] = &banktypes.MsgSend{
			FromAddress: acc.String(),
			ToAddress:   acc.String(),
		}
	}

	return msgs
}
