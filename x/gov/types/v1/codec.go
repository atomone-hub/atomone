package v1

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	groupcodec "github.com/cosmos/cosmos-sdk/x/group/codec"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	govcodec "github.com/atomone-hub/atomone/x/gov/codec"
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
	legacy.RegisterAminoMsg(cdc, &MsgCreateGovernor{}, "atomone/v1/MsgCreateGovernor")
	legacy.RegisterAminoMsg(cdc, &MsgEditGovernor{}, "atomone/v1/MsgEditGovernor")
	legacy.RegisterAminoMsg(cdc, &MsgDelegateGovernor{}, "atomone/v1/MsgDelegateGovernor")
	legacy.RegisterAminoMsg(cdc, &MsgUndelegateGovernor{}, "atomone/v1/MsgUndelegateGovernor")
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
		&MsgCreateGovernor{},
		&MsgEditGovernor{},
		&MsgDelegateGovernor{},
		&MsgUndelegateGovernor{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func init() {
	// Register all Amino interfaces and concrete types on the authz and gov Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
	RegisterLegacyAminoCodec(groupcodec.Amino)

	// Need to add registration in the atomone gov amino for all modules that
	// register their amino types in the legacy gov module.
	banktypes.RegisterLegacyAminoCodec(govcodec.Amino)
	consensustypes.RegisterLegacyAminoCodec(govcodec.Amino)
	crisistypes.RegisterLegacyAminoCodec(govcodec.Amino)
	distributiontypes.RegisterLegacyAminoCodec(govcodec.Amino)
	evidencetypes.RegisterLegacyAminoCodec(govcodec.Amino)
	minttypes.RegisterLegacyAminoCodec(govcodec.Amino)
	slashingtypes.RegisterLegacyAminoCodec(govcodec.Amino)
	stakingtypes.RegisterLegacyAminoCodec(govcodec.Amino)
	upgradetypes.RegisterLegacyAminoCodec(govcodec.Amino)
}
