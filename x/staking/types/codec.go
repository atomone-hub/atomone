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

// RegisterLegacyAminoCodec registers the necessary x/staking interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgCreateValidator{}, "atomone/MsgCreateValidator")
	legacy.RegisterAminoMsg(cdc, &MsgEditValidator{}, "atomone/MsgEditValidator")
	legacy.RegisterAminoMsg(cdc, &MsgDelegate{}, "atomone/MsgDelegate")
	legacy.RegisterAminoMsg(cdc, &MsgUndelegate{}, "atomone/MsgUndelegate")
	legacy.RegisterAminoMsg(cdc, &MsgBeginRedelegate{}, "atomone/MsgBeginRedelegate")
	legacy.RegisterAminoMsg(cdc, &MsgCancelUnbondingDelegation{}, "atomone/MsgCancelUnbondingDelegation")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "atomone/x/staking/MsgUpdateParams")

	cdc.RegisterInterface((*isStakeAuthorization_Validators)(nil), nil)
	cdc.RegisterConcrete(&StakeAuthorization_AllowList{}, "atomone/StakeAuthorization/AllowList", nil)
	cdc.RegisterConcrete(&StakeAuthorization_DenyList{}, "atomone/StakeAuthorization/DenyList", nil)
	cdc.RegisterConcrete(&StakeAuthorization{}, "atomone/StakeAuthorization", nil)
	cdc.RegisterConcrete(Params{}, "atomone/x/staking/Params", nil)
}

// RegisterInterfaces registers the x/staking interfaces types with the interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateValidator{},
		&MsgEditValidator{},
		&MsgDelegate{},
		&MsgUndelegate{},
		&MsgBeginRedelegate{},
		&MsgCancelUnbondingDelegation{},
		&MsgUpdateParams{},
	)
	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&StakeAuthorization{},
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

	// Register all Amino interfaces and concrete types on the authz  and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)
}