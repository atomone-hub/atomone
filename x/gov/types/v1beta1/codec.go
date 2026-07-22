package v1beta1

import (
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
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
	)
	registry.RegisterImplementations(
		(*Content)(nil),
		&upgradetypes.SoftwareUpgradeProposal{}, //nolint:staticcheck
	)
	registry.RegisterImplementations(
		(*Content)(nil),
		&upgradetypes.CancelSoftwareUpgradeProposal{}, //nolint:staticcheck
	)
	// Legacy IBC 02-client proposal content types. These are registered by
	// ibc-go against the SDK's gov Content interface, but not against ours, so
	// we must register them manually. Historical mainnet proposals wrapping a
	// ClientUpdateProposal or UpgradeProposal in a MsgExecLegacyContent would
	// otherwise fail to unpack.
	registry.RegisterImplementations(
		(*Content)(nil),
		&ibcclienttypes.ClientUpdateProposal{}, //nolint:staticcheck
		&ibcclienttypes.UpgradeProposal{},      //nolint:staticcheck
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
