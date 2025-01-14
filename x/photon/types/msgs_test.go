package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	appparams "github.com/atomone-hub/atomone/app/params"
)

func TestMsgMintPhoton_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgMintPhoton
		err  error
	}{
		{
			name: "fail: invalid toAddress",
			msg: MsgMintPhoton{
				ToAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "fail: negative amount",
			msg: MsgMintPhoton{
				ToAddress: sdk.AccAddress("test1").String(),
				Amount: sdk.Coin{
					Denom:  appparams.BondDenom,
					Amount: sdk.NewInt(-1),
				},
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "fail: not positive amount",
			msg: MsgMintPhoton{
				ToAddress: sdk.AccAddress("test1").String(),
				Amount: sdk.Coin{
					Denom:  appparams.BondDenom,
					Amount: sdk.NewInt(0),
				},
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "ok",
			msg: MsgMintPhoton{
				ToAddress: sdk.AccAddress("test1").String(),
				Amount:    sdk.NewInt64Coin(appparams.BondDenom, 1),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgMintPhoton_GetSignBytes(t *testing.T) {
	msg := MsgMintPhoton{
		ToAddress: "my_addr",
		Amount:    sdk.NewInt64Coin("uatone", 1),
	}

	bz := msg.GetSignBytes()

	expectedSignedBytes := `{
		"type": "atomone/photon/v1/MsgMintPhoton",
		"value": {
			"amount": {
				"amount":"1",
				"denom": "uatone"
			},
			"to_address": "my_addr"
		}
	}`
	require.JSONEq(t, expectedSignedBytes, string(bz))
}

func TestMsgUpdateParams_GetSignBytes(t *testing.T) {
	msg := MsgUpdateParams{
		Authority: "authority",
		Params: Params{
			MintDisabled:    true,
			TxFeeExceptions: []string{"tx1", "tx2"},
		},
	}

	bz := msg.GetSignBytes()

	expectedSignedBytes := `{
		"type": "atomone/x/photon/v1/MsgUpdateParams",
		"value": {
			"authority":"authority",
			"params": {
				"mint_disabled":true,
				"tx_fee_exceptions": ["tx1","tx2"]
			}
		}
	}`
	require.JSONEq(t, expectedSignedBytes, string(bz))
}
