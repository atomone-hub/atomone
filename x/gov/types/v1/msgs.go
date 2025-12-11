package v1

import (
	"fmt"

	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	sdkgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

var (
	_, _, _, _, _, _, _, _ sdk.Msg                            = &MsgSubmitProposal{}, &MsgDeposit{}, &MsgVote{}, &MsgVoteWeighted{}, &MsgExecLegacyContent{}, &MsgUpdateParams{}, &MsgProposeConstitutionAmendment{}, &MsgProposeLaw{}
	_, _                   codectypes.UnpackInterfacesMessage = &MsgSubmitProposal{}, &MsgExecLegacyContent{}
)

// NewMsgSubmitProposal creates a new MsgSubmitProposal.
//
//nolint:interfacer
func NewMsgSubmitProposal(messages []sdk.Msg, initialDeposit sdk.Coins, proposer, metadata, title, summary string) (*MsgSubmitProposal, error) {
	m := &MsgSubmitProposal{
		InitialDeposit: initialDeposit,
		Proposer:       proposer,
		Metadata:       metadata,
		Title:          title,
		Summary:        summary,
	}

	anys, err := sdktx.SetMsgs(messages)
	if err != nil {
		return nil, err
	}

	m.Messages = anys

	return m, nil
}

// GetMsgs unpacks m.Messages Any's into sdk.Msg's
func (m *MsgSubmitProposal) GetMsgs() ([]sdk.Msg, error) {
	return sdktx.GetMsgs(m.Messages, "sdk.MsgProposal")
}

// SetMsgs packs sdk.Msg's into m.Messages Any's
// NOTE: this will overwrite any existing messages
func (m *MsgSubmitProposal) SetMsgs(msgs []sdk.Msg) error {
	anys, err := sdktx.SetMsgs(msgs)
	if err != nil {
		return err
	}

	m.Messages = anys
	return nil
}

// ValidateBasic implements the sdk.Msg interface.
func (m MsgSubmitProposal) ValidateBasic() error {
	if m.Title == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("proposal title cannot be empty")
	}
	if m.Summary == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("proposal summary cannot be empty")
	}

	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid proposer address: %s", err)
	}

	deposit := sdk.NewCoins(m.InitialDeposit...)
	if !deposit.IsValid() {
		return sdkerrors.ErrInvalidCoins.Wrap(deposit.String())
	}

	if deposit.IsAnyNegative() {
		return sdkerrors.ErrInvalidCoins.Wrap(deposit.String())
	}

	// Check that either metadata or Msgs length is non nil.
	if len(m.Messages) == 0 && len(m.Metadata) == 0 {
		return sdkgovtypes.ErrNoProposalMsgs.Wrap("either metadata or Msgs length must be non-nil")
	}

	msgs, err := m.GetMsgs()
	if err != nil {
		return err
	}

	for idx, msg := range msgs {
		if msg, ok := msg.(sdk.HasValidateBasic); ok {
			if err := msg.ValidateBasic(); err != nil {
				return sdkgovtypes.ErrInvalidProposalMsg.Wrap(fmt.Sprintf("msg: %d, err: %s", idx, err.Error()))
			}
		}
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (m MsgSubmitProposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return sdktx.UnpackInterfaces(unpacker, m.Messages)
}

// NewMsgDeposit creates a new MsgDeposit instance
//
//nolint:interfacer
func NewMsgDeposit(depositor sdk.AccAddress, proposalID uint64, amount sdk.Coins) *MsgDeposit {
	return &MsgDeposit{proposalID, depositor.String(), amount}
}

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid depositor address: %s", err)
	}
	amount := sdk.NewCoins(msg.Amount...)
	if !amount.IsValid() {
		return sdkerrors.ErrInvalidCoins.Wrap(amount.String())
	}
	if amount.IsAnyNegative() {
		return sdkerrors.ErrInvalidCoins.Wrap(amount.String())
	}

	return nil
}

// NewMsgVote creates a message to cast a vote on an active proposal
//
//nolint:interfacer
func NewMsgVote(voter sdk.AccAddress, proposalID uint64, option VoteOption, metadata string) *MsgVote {
	return &MsgVote{proposalID, voter.String(), option, metadata}
}

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgVote) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid voter address: %s", err)
	}
	if !ValidVoteOption(msg.Option) {
		return sdkgovtypes.ErrInvalidVote.Wrap(msg.Option.String())
	}

	return nil
}

// NewMsgVoteWeighted creates a message to cast a vote on an active proposal
//
//nolint:interfacer
func NewMsgVoteWeighted(voter sdk.AccAddress, proposalID uint64, options WeightedVoteOptions, metadata string) *MsgVoteWeighted {
	return &MsgVoteWeighted{proposalID, voter.String(), options, metadata}
}

// Route implements the sdk.Msg interface.
func (msg MsgVoteWeighted) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgVoteWeighted) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgVoteWeighted) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid voter address: %s", err)
	}
	if len(msg.Options) == 0 {
		return sdkerrors.ErrInvalidRequest.Wrap(WeightedVoteOptions(msg.Options).String())
	}

	totalWeight := math.LegacyNewDec(0)
	usedOptions := make(map[VoteOption]bool)
	for _, option := range msg.Options {
		if !option.IsValid() {
			return sdkgovtypes.ErrInvalidVote.Wrap(option.String())
		}
		weight, err := math.LegacyNewDecFromStr(option.Weight)
		if err != nil {
			return sdkgovtypes.ErrInvalidVote.Wrapf("Invalid weight: %s", err)
		}
		totalWeight = totalWeight.Add(weight)
		if usedOptions[option.Option] {
			return sdkgovtypes.ErrInvalidVote.Wrap("Duplicated vote option")
		}
		usedOptions[option.Option] = true
	}

	if totalWeight.GT(math.LegacyNewDec(1)) {
		return sdkgovtypes.ErrInvalidVote.Wrap("Total weight overflow 1.00")
	}

	if totalWeight.LT(math.LegacyNewDec(1)) {
		return sdkgovtypes.ErrInvalidVote.Wrap("Total weight lower than 1.00")
	}

	return nil
}

// NewMsgExecLegacyContent creates a new MsgExecLegacyContent instance
//
//nolint:interfacer
func NewMsgExecLegacyContent(content *codectypes.Any, authority string) *MsgExecLegacyContent {
	return &MsgExecLegacyContent{
		Content:   content,
		Authority: authority,
	}
}

// ValidateBasic implements the sdk.Msg interface.
func (c MsgExecLegacyContent) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(c.Authority)
	if err != nil {
		return err
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (c MsgExecLegacyContent) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var content v1beta1.Content
	return unpacker.UnpackAny(c.Content, &content)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.ValidateBasic()
}

func NewMsgProposeConstitutionAmendment(authority sdk.AccAddress, amendment string) *MsgProposeConstitutionAmendment {
	return &MsgProposeConstitutionAmendment{
		Authority: authority.String(),
		Amendment: amendment,
	}
}

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgProposeConstitutionAmendment) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if msg.Amendment == "" {
		return sdkgovtypes.ErrInvalidProposalContent.Wrap("amendment cannot be empty")
	}

	_, err := types.ParseUnifiedDiff(msg.Amendment)
	if err != nil {
		return sdkgovtypes.ErrInvalidProposalContent.Wrap(err.Error())
	}

	return nil
}

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgProposeLaw) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return nil
}

func (msg MsgProposeConstitutionAmendment) IsProposalKindConstitutionAmendment() {}

func (msg MsgProposeLaw) IsProposalKindLaw() {}
