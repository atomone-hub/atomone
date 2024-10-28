package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestMsgBurn_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgBurn
		err  error
	}{
		{
			name: "fail: invalid toAddress",
			msg: MsgBurn{
				ToAddress: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "fail: negative amount",
			msg: MsgBurn{
				ToAddress: sdk.AccAddress("test1").String(),
				Amount: sdk.Coin{
					Denom:  "uatone",
					Amount: sdk.NewInt(-1),
				},
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "fail: not positive amount",
			msg: MsgBurn{
				ToAddress: sdk.AccAddress("test1").String(),
				Amount: sdk.Coin{
					Denom:  "uatone",
					Amount: sdk.NewInt(0),
				},
			},
			err: sdkerrors.ErrInvalidCoins,
		},
		{
			name: "ok",
			msg: MsgBurn{
				ToAddress: sdk.AccAddress("test1").String(),
				Amount:    sdk.NewInt64Coin("uatone", 1),
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
