package group

import (
	codectypes "github.com/atomone-hub/atomone/codec"
	"github.com/atomone-hub/atomone/codec/legacy"
	cdctypes "github.com/atomone-hub/atomone/codec/types"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/types/msgservice"
	authzcodec "github.com/atomone-hub/atomone/x/authz/codec"
	govcodec "github.com/atomone-hub/atomone/x/gov/codec"
	groupcodec "github.com/atomone-hub/atomone/x/group/codec"
)

// RegisterLegacyAminoCodec registers all the necessary group module concrete
// types and interfaces with the provided codec reference.
// These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codectypes.LegacyAmino) {
	cdc.RegisterInterface((*DecisionPolicy)(nil), nil)
	cdc.RegisterConcrete(&ThresholdDecisionPolicy{}, "atomone/ThresholdDecisionPolicy", nil)
	cdc.RegisterConcrete(&PercentageDecisionPolicy{}, "atomone/PercentageDecisionPolicy", nil)

	legacy.RegisterAminoMsg(cdc, &MsgCreateGroup{}, "atomone/MsgCreateGroup")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateGroupMembers{}, "atomone/MsgUpdateGroupMembers")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateGroupAdmin{}, "atomone/MsgUpdateGroupAdmin")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateGroupMetadata{}, "atomone/MsgUpdateGroupMetadata")
	legacy.RegisterAminoMsg(cdc, &MsgCreateGroupWithPolicy{}, "atomone/MsgCreateGroupWithPolicy")
	legacy.RegisterAminoMsg(cdc, &MsgCreateGroupPolicy{}, "atomone/MsgCreateGroupPolicy")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateGroupPolicyAdmin{}, "atomone/MsgUpdateGroupPolicyAdmin")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateGroupPolicyDecisionPolicy{}, "atomone/MsgUpdateGroupDecisionPolicy")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateGroupPolicyMetadata{}, "atomone/MsgUpdateGroupPolicyMetadata")
	legacy.RegisterAminoMsg(cdc, &MsgSubmitProposal{}, "atomone/group/MsgSubmitProposal")
	legacy.RegisterAminoMsg(cdc, &MsgWithdrawProposal{}, "atomone/group/MsgWithdrawProposal")
	legacy.RegisterAminoMsg(cdc, &MsgVote{}, "atomone/group/MsgVote")
	legacy.RegisterAminoMsg(cdc, &MsgExec{}, "atomone/group/MsgExec")
	legacy.RegisterAminoMsg(cdc, &MsgLeaveGroup{}, "atomone/group/MsgLeaveGroup")
}

// RegisterInterfaces registers the interfaces types with the interface registry.
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateGroup{},
		&MsgUpdateGroupMembers{},
		&MsgUpdateGroupAdmin{},
		&MsgUpdateGroupMetadata{},
		&MsgCreateGroupWithPolicy{},
		&MsgCreateGroupPolicy{},
		&MsgUpdateGroupPolicyAdmin{},
		&MsgUpdateGroupPolicyDecisionPolicy{},
		&MsgUpdateGroupPolicyMetadata{},
		&MsgSubmitProposal{},
		&MsgWithdrawProposal{},
		&MsgVote{},
		&MsgExec{},
		&MsgLeaveGroup{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)

	registry.RegisterInterface(
		"atomone.group.v1.DecisionPolicy",
		(*DecisionPolicy)(nil),
		&ThresholdDecisionPolicy{},
		&PercentageDecisionPolicy{},
	)
}

func init() {
	// Register all Amino interfaces and concrete types on the authz  and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)
}
