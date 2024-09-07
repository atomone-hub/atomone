package types

import (
	"github.com/atomone-hub/atomone/codec"
	"github.com/atomone-hub/atomone/codec/legacy"
	"github.com/atomone-hub/atomone/codec/types"
	cryptocodec "github.com/atomone-hub/atomone/crypto/codec"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/types/msgservice"
	"github.com/atomone-hub/atomone/x/authz"
	authzcodec "github.com/atomone-hub/atomone/x/authz/codec"
	govcodec "github.com/atomone-hub/atomone/x/gov/codec"
	groupcodec "github.com/atomone-hub/atomone/x/group/codec"
)

// RegisterLegacyAminoCodec registers the necessary x/bank interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgSend{}, "atomone/MsgSend")
	legacy.RegisterAminoMsg(cdc, &MsgMultiSend{}, "atomone/MsgMultiSend")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "atomone/x/bank/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgSetSendEnabled{}, "atomone/MsgSetSendEnabled")

	cdc.RegisterConcrete(&SendAuthorization{}, "atomone/SendAuthorization", nil)
	cdc.RegisterConcrete(&Params{}, "atomone/x/bank/Params", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSend{},
		&MsgMultiSend{},
		&MsgUpdateParams{},
	)
	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&SendAuthorization{},
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

	// Register all Amino interfaces and concrete types on the authz and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)
}
