package v1beta1

import (
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	groupcodec "github.com/cosmos/cosmos-sdk/x/group/codec"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	govcodec "github.com/atomone-hub/atomone/x/gov/codec"
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the
// governance module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterInterface((*Content)(nil), nil)
	legacy.RegisterAminoMsg(cdc, &MsgSubmitProposal{}, "atomone/MsgSubmitProposal")
	legacy.RegisterAminoMsg(cdc, &MsgDeposit{}, "atomone/MsgDeposit")
	legacy.RegisterAminoMsg(cdc, &MsgVote{}, "atomone/MsgVote")
	legacy.RegisterAminoMsg(cdc, &MsgVoteWeighted{}, "atomone/MsgVoteWeighted")
	cdc.RegisterConcrete(&TextProposal{}, "atomone/TextProposal", nil)
}

// RegisterInterfaces registers the interfaces types with the Interface Registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitProposal{},
		&MsgVote{},
		&MsgVoteWeighted{},
		&MsgDeposit{},
	)
	registry.RegisterInterface(
		"atomone.gov.v1beta1.Content",
		(*Content)(nil),
		&TextProposal{},
	)

	// Register proposal types (this is actually done in related modules, but
	// since we are using an other gov module, we need to do it manually).
	registry.RegisterImplementations(
		(*Content)(nil),
		&paramsproposal.ParameterChangeProposal{},
		&upgradetypes.SoftwareUpgradeProposal{},       //nolint:staticcheck
		&upgradetypes.CancelSoftwareUpgradeProposal{}, //nolint:staticcheck
		&ibcclienttypes.ClientUpdateProposal{},        //nolint:staticcheck
		&ibcclienttypes.UpgradeProposal{},             //nolint:staticcheck
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func init() {
	// Register all Amino interfaces and concrete types on the authz  and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)
}
