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
	legacy.RegisterAminoMsg(cdc, &MsgAnnotateProposal{}, "atomone/v1/MsgAnnotateProposal")
	legacy.RegisterAminoMsg(cdc, &MsgEndorseProposal{}, "atomone/v1/MsgEndorseProposal")
	legacy.RegisterAminoMsg(cdc, &MsgExtendVotingPeriod{}, "atomone/v1/MsgExtendVotingPeriod")
	legacy.RegisterAminoMsg(cdc, &MsgVetoProposal{}, "atomone/v1/MsgVetoProposal")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "atomone/x/coredaos/v1/MsgUpdateParams")
	cdc.RegisterConcrete(&Params{}, "atomone/coredaos/v1/Params", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAnnotateProposal{}, &MsgEndorseProposal{}, &MsgExtendVotingPeriod{}, &MsgVetoProposal{}, &MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)
}
