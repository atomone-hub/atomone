package testutil

import (
	"github.com/atomone-hub/atomone/codec/types"
	"github.com/atomone-hub/atomone/crypto/codec"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/types/msgservice"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgCounter{},
		&MsgCounter2{},
		&MsgKeyValue{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Counter_serviceDesc)
	msgservice.RegisterMsgServiceDesc(registry, &_Counter2_serviceDesc)
	msgservice.RegisterMsgServiceDesc(registry, &_KeyValue_serviceDesc)

	codec.RegisterInterfaces(registry)
}

var _ sdk.Msg = &MsgCounter{}

func (msg *MsgCounter) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{} }
func (msg *MsgCounter) ValidateBasic() error {
	if msg.Counter >= 0 {
		return nil
	}
	return sdkerrors.Wrap(sdkerrors.ErrInvalidSequence, "counter should be a non-negative integer")
}

var _ sdk.Msg = &MsgCounter2{}

func (msg *MsgCounter2) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{} }
func (msg *MsgCounter2) ValidateBasic() error {
	if msg.Counter >= 0 {
		return nil
	}
	return sdkerrors.Wrap(sdkerrors.ErrInvalidSequence, "counter should be a non-negative integer")
}

var _ sdk.Msg = &MsgKeyValue{}

func (msg *MsgKeyValue) GetSigners() []sdk.AccAddress {
	if len(msg.Signer) == 0 {
		return []sdk.AccAddress{}
	}

	return []sdk.AccAddress{sdk.MustAccAddressFromBech32(msg.Signer)}
}

func (msg *MsgKeyValue) ValidateBasic() error {
	if msg.Key == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "key cannot be nil")
	}
	if msg.Value == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "value cannot be nil")
	}
	return nil
}