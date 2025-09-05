package e2e

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/unknownproto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	txsigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

func decodeTx(cdc codec.Codec, txBytes []byte) (*sdktx.Tx, error) {
	var raw sdktx.TxRaw

	// reject all unknown proto fields in the root TxRaw
	err := unknownproto.RejectUnknownFieldsStrict(txBytes, &raw, cdc.InterfaceRegistry())
	if err != nil {
		return nil, fmt.Errorf("failed to reject unknown fields: %w", err)
	}

	if err := cdc.Unmarshal(txBytes, &raw); err != nil {
		return nil, err
	}

	var body sdktx.TxBody
	if err := cdc.Unmarshal(raw.BodyBytes, &body); err != nil {
		return nil, fmt.Errorf("failed to decode tx: %w", err)
	}

	var authInfo sdktx.AuthInfo

	// reject all unknown proto fields in AuthInfo
	err = unknownproto.RejectUnknownFieldsStrict(raw.AuthInfoBytes, &authInfo, cdc.InterfaceRegistry())
	if err != nil {
		return nil, fmt.Errorf("failed to reject unknown fields: %w", err)
	}

	if err := cdc.Unmarshal(raw.AuthInfoBytes, &authInfo); err != nil {
		return nil, fmt.Errorf("failed to decode auth info: %w", err)
	}

	return &sdktx.Tx{
		Body:       &body,
		AuthInfo:   &authInfo,
		Signatures: raw.Signatures,
	}, nil
}

func concatFlags(originalCollection []string, commandFlags []string, generalFlags []string) []string {
	originalCollection = append(originalCollection, commandFlags...)
	originalCollection = append(originalCollection, generalFlags...)

	return originalCollection
}

func (s *IntegrationTestSuite) signAndBroadcastMsg(c *chain, key keyring.Record, msgs ...sdk.Msg) {
	// Fetch account
	endpoint := fmt.Sprintf("http://%s", s.valResources[c.id][0].GetHostPort("1317/tcp"))
	addr, err := key.GetAddress()
	s.Require().NoError(err)
	acc := s.queryAccount(endpoint, addr.String())

	// Sign tx
	tx := s.signMsg(c, key, acc.GetAccountNumber(), acc.GetSequence(), "", msgs...)

	// Broadcast tx
	bz, err := tx.Marshal()
	s.Require().NoError(err)
	res, err := s.rpcClient(c, 0).BroadcastTxSync(context.Background(), bz)
	s.Require().NoError(err, "broadcast TX error")
	s.Require().Zero(res.Code, "check TX error: %s", res.Log)

	// Ensure tx success
	err = s.waitAtomOneTx(endpoint, res.Hash.String(), nil)
	s.Require().NoError(err, "run TX error")
}

func (s *IntegrationTestSuite) signMsg(c *chain, key keyring.Record,
	accountNum, sequence uint64, memo string, msgs ...sdk.Msg,
) *sdktx.Tx {
	txBuilder := c.txConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msgs...)
	s.Require().NoError(err)
	txBuilder.SetMemo(memo)
	txBuilder.SetFeeAmount(sdk.NewCoins(standardFees))
	txBuilder.SetGasLimit(200000)

	// For SIGN_MODE_DIRECT, calling SetSignatures calls setSignerInfos on
	// TxBuilder under the hood, and SignerInfos is needed to generate the sign
	// bytes. This is the reason for setting SetSignatures here, with a nil
	// signature.
	//
	// Note: This line is not needed for SIGN_MODE_LEGACY_AMINO, but putting it
	// also doesn't affect its generated sign bytes, so for code's simplicity
	// sake, we put it here.
	pk, err := key.GetPubKey()
	s.Require().NoError(err)

	sig := txsigning.SignatureV2{
		PubKey: pk,
		Data: &txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: sequence,
	}

	err = txBuilder.SetSignatures(sig)
	s.Require().NoError(err)

	bytesToSign, err := authsigning.GetSignBytesAdapter(
		context.TODO(),
		c.txConfig.SignModeHandler(),
		txsigning.SignMode_SIGN_MODE_DIRECT,
		authsigning.SignerData{
			ChainID:       c.id,
			AccountNumber: accountNum,
			Sequence:      sequence,
		},
		txBuilder.GetTx(),
	)
	s.Require().NoError(err, "error getting sign bytes")

	privKey := key.GetLocal().PrivKey.GetCachedValue().(cryptotypes.PrivKey)
	sigBytes, err := privKey.Sign(bytesToSign)
	s.Require().NoError(err, "error signing sign bytes")

	sig = txsigning.SignatureV2{
		PubKey: pk,
		Data: &txsigning.SingleSignatureData{
			SignMode:  txsigning.SignMode_SIGN_MODE_DIRECT,
			Signature: sigBytes,
		},
		Sequence: sequence,
	}
	err = txBuilder.SetSignatures(sig)
	s.Require().NoError(err, "error setting signature")

	signedTx := txBuilder.GetTx()
	bz, err := c.txConfig.TxEncoder()(signedTx)
	s.Require().NoError(err)
	tx, err := decodeTx(c.cdc, bz)
	s.Require().NoError(err)
	return tx
}
