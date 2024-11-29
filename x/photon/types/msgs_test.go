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
