package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/atomone-hub/atomone/x/gov/types"
)

var _, _ sdk.Msg = &MsgMintPhoton{}, &MsgUpdateParams{}

func NewMsgMintPhoton(toAddr sdk.AccAddress, amount sdk.Coin) *MsgMintPhoton {
	return &MsgMintPhoton{
		ToAddress: toAddr.String(),
		Amount:    amount,
	}
}

func (msg *MsgMintPhoton) Route() string {
	return RouterKey
}

func (msg *MsgMintPhoton) Type() string { return sdk.MsgTypeURL(msg) }

func (msg *MsgMintPhoton) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgMintPhoton) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgMintPhoton) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ToAddress); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid toAddress: %s", err) //nolint:staticcheck
	}
	if err := msg.Amount.Validate(); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "invalid coin to burn: %s", err) //nolint:staticcheck
	}
	if !msg.Amount.IsPositive() {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "coin to burn must be positive") //nolint:staticcheck
	}
	return nil
}

// Route implements the sdk.Msg interface.
func (msg MsgUpdateParams) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgUpdateParams) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.ValidateBasic()
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgUpdateParams) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgUpdateParams.
func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}
