package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgMintPhoton{}, "atomone/photon/v1/MsgMintPhoton")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "atomone/x/photon/v1/MsgUpdateParams")
	cdc.RegisterConcrete(&Params{}, "atomone/photo/v1/Params", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgMintPhoton{}, &MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)
}
