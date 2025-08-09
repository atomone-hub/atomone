package v1

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers all the necessary types and interfaces for the
// governance module.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgSubmitProposal{}, "atomone/v1/MsgSubmitProposal")
	legacy.RegisterAminoMsg(cdc, &MsgDeposit{}, "atomone/v1/MsgDeposit")
	legacy.RegisterAminoMsg(cdc, &MsgVote{}, "atomone/v1/MsgVote")
	legacy.RegisterAminoMsg(cdc, &MsgVoteWeighted{}, "atomone/v1/MsgVoteWeighted")
	legacy.RegisterAminoMsg(cdc, &MsgExecLegacyContent{}, "atomone/v1/MsgExecLegacyContent")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateParams{}, "atomone/x/gov/v1/MsgUpdateParams")
	legacy.RegisterAminoMsg(cdc, &MsgProposeConstitutionAmendment{}, "atomone/x/gov/v1/MsgProposeAmendment")
	legacy.RegisterAminoMsg(cdc, &MsgProposeLaw{}, "atomone/x/gov/v1/MsgProposeLaw")
}

// RegisterInterfaces registers the interfaces types with the Interface Registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitProposal{},
		&MsgVote{},
		&MsgVoteWeighted{},
		&MsgDeposit{},
		&MsgExecLegacyContent{},
		&MsgUpdateParams{},
		&MsgProposeConstitutionAmendment{},
		&MsgProposeLaw{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
