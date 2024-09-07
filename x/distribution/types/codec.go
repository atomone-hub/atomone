package types

import (
	"github.com/atomone-hub/atomone/codec"
	"github.com/atomone-hub/atomone/codec/legacy"
	"github.com/atomone-hub/atomone/codec/types"
	cryptocodec "github.com/atomone-hub/atomone/crypto/codec"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/types/msgservice"
	authzcodec "github.com/atomone-hub/atomone/x/authz/codec"
	govcodec "github.com/atomone-hub/atomone/x/gov/codec"
	govtypes "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	groupcodec "github.com/atomone-hub/atomone/x/group/codec"
)

// RegisterLegacyAminoCodec registers the necessary x/distribution interfaces
// and concrete types on the provided LegacyAmino codec. These types are used
// for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgWithdrawDelegatorReward{}, "atomone/MsgWithdrawDelegationReward")
	legacy.RegisterAminoMsg(cdc, &MsgWithdrawValidatorCommission{}, "atomone/MsgWithdrawValCommission")
	legacy.RegisterAminoMsg(cdc, &MsgSetWithdrawAddress{}, "atomone/MsgModifyWithdrawAddress")
	legacy.RegisterAminoMsg(cdc, &MsgFundCommunityPool{}, "atomone/MsgFundCommunityPool")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "atomone/distribution/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgCommunityPoolSpend{}, "atomone/distr/MsgCommunityPoolSpend")

	cdc.RegisterConcrete(Params{}, "atomone/x/distribution/Params", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgWithdrawDelegatorReward{},
		&MsgWithdrawValidatorCommission{},
		&MsgSetWithdrawAddress{},
		&MsgFundCommunityPool{},
		&MsgUpdateParams{},
		&MsgCommunityPoolSpend{},
	)

	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&CommunityPoolSpendProposal{})

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

	// Register all Amino interfaces and concrete types on the authz  and gov Amino codec
	// so that this can later be used to properly serialize MsgGrant and MsgExec
	// instances.
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)
}
