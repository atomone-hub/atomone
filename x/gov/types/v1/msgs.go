package v1

import (
	"fmt"

	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/atomone-hub/atomone/x/gov/codec"
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

// Route implements the sdk.Msg interface.
func (m MsgSubmitProposal) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (m MsgSubmitProposal) Type() string { return sdk.MsgTypeURL(&m) }

// ValidateBasic implements the sdk.Msg interface.
func (m MsgSubmitProposal) ValidateBasic() error {
	if m.Title == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "proposal title cannot be empty") //nolint:staticcheck
	}
	if m.Summary == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "proposal summary cannot be empty") //nolint:staticcheck
	}

	if _, err := sdk.AccAddressFromBech32(m.Proposer); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid proposer address: %s", err)
	}

	deposit := sdk.NewCoins(m.InitialDeposit...)
	if !deposit.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, deposit.String()) //nolint:staticcheck
	}

	if deposit.IsAnyNegative() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, deposit.String()) //nolint:staticcheck
	}

	// Check that either metadata or Msgs length is non nil.
	if len(m.Messages) == 0 && len(m.Metadata) == 0 {
		return sdkerrors.Wrap(types.ErrNoProposalMsgs, "either metadata or Msgs length must be non-nil") //nolint:staticcheck
	}

	msgs, err := m.GetMsgs()
	if err != nil {
		return err
	}

	for idx, msg := range msgs {
		if err := msg.ValidateBasic(); err != nil {
			return sdkerrors.Wrap(types.ErrInvalidProposalMsg, //nolint:staticcheck
				fmt.Sprintf("msg: %d, err: %s", idx, err.Error()))
		}
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (m MsgSubmitProposal) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&m)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgSubmitProposal.
func (m MsgSubmitProposal) GetSigners() []sdk.AccAddress {
	proposer, _ := sdk.AccAddressFromBech32(m.Proposer)
	return []sdk.AccAddress{proposer}
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

// Route implements the sdk.Msg interface.
func (msg MsgDeposit) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgDeposit) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgDeposit) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid depositor address: %s", err)
	}
	amount := sdk.NewCoins(msg.Amount...)
	if !amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, amount.String()) //nolint:staticcheck
	}
	if amount.IsAnyNegative() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, amount.String()) //nolint:staticcheck
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgDeposit) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgDeposit.
func (msg MsgDeposit) GetSigners() []sdk.AccAddress {
	depositor, _ := sdk.AccAddressFromBech32(msg.Depositor)
	return []sdk.AccAddress{depositor}
}

// NewMsgVote creates a message to cast a vote on an active proposal
//
//nolint:interfacer
func NewMsgVote(voter sdk.AccAddress, proposalID uint64, option VoteOption, metadata string) *MsgVote {
	return &MsgVote{proposalID, voter.String(), option, metadata}
}

// Route implements the sdk.Msg interface.
func (msg MsgVote) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgVote) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgVote) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Voter); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid voter address: %s", err)
	}
	if !ValidVoteOption(msg.Option) {
		return sdkerrors.Wrap(types.ErrInvalidVote, msg.Option.String()) //nolint:staticcheck
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgVote) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgVote.
func (msg MsgVote) GetSigners() []sdk.AccAddress {
	voter, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{voter}
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
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, WeightedVoteOptions(msg.Options).String()) //nolint:staticcheck
	}

	totalWeight := math.LegacyNewDec(0)
	usedOptions := make(map[VoteOption]bool)
	for _, option := range msg.Options {
		if !option.IsValid() {
			return sdkerrors.Wrap(types.ErrInvalidVote, option.String()) //nolint:staticcheck
		}
		weight, err := sdk.NewDecFromStr(option.Weight)
		if err != nil {
			return sdkerrors.Wrapf(types.ErrInvalidVote, "Invalid weight: %s", err) //nolint:staticcheck
		}
		totalWeight = totalWeight.Add(weight)
		if usedOptions[option.Option] {
			return sdkerrors.Wrap(types.ErrInvalidVote, "Duplicated vote option") //nolint:staticcheck
		}
		usedOptions[option.Option] = true
	}

	if totalWeight.GT(math.LegacyNewDec(1)) {
		return sdkerrors.Wrap(types.ErrInvalidVote, "Total weight overflow 1.00") //nolint:staticcheck
	}

	if totalWeight.LT(math.LegacyNewDec(1)) {
		return sdkerrors.Wrap(types.ErrInvalidVote, "Total weight lower than 1.00") //nolint:staticcheck
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgVoteWeighted) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgVoteWeighted.
func (msg MsgVoteWeighted) GetSigners() []sdk.AccAddress {
	voter, _ := sdk.AccAddressFromBech32(msg.Voter)
	return []sdk.AccAddress{voter}
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

// GetSigners returns the expected signers for a MsgExecLegacyContent.
func (c MsgExecLegacyContent) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(c.Authority)
	return []sdk.AccAddress{authority}
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

// Route implements the sdk.Msg interface.
func (msg MsgUpdateParams) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgUpdateParams) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.ValidateBasic()
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgUpdateParams) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgUpdateParams.
func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// Route implements the sdk.Msg interface.
func (msg MsgProposeConstitutionAmendment) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgProposeConstitutionAmendment) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgProposeConstitutionAmendment) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}
	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgProposeConstitutionAmendment) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgProposeConstitutionAmendment.
func (msg MsgProposeConstitutionAmendment) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// Route implements the sdk.Msg interface.
func (msg MsgProposeLaw) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgProposeLaw) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgProposeLaw) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgProposeLaw) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgProposeLaw.
func (msg MsgProposeLaw) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// NewMsgCreateGovernor creates a new MsgCreateGovernor instance
func NewMsgCreateGovernor(address sdk.AccAddress, description GovernorDescription) *MsgCreateGovernor {
	return &MsgCreateGovernor{Address: address.String(), Description: description}
}

// Route implements the sdk.Msg interface.
func (msg MsgCreateGovernor) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgCreateGovernor) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgCreateGovernor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrap(err.Error())
	}

	if _, err := msg.Description.EnsureLength(); err != nil {
		return types.ErrInvalidGovernanceDescription.Wrap(err.Error())
	}
	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgCreateGovernor) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgCreateGovernor.
func (msg MsgCreateGovernor) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

// NewMsgEditGovernor creates a new MsgEditGovernor instance
func NewMsgEditGovernor(addr sdk.AccAddress, description GovernorDescription) *MsgEditGovernor {
	return &MsgEditGovernor{Address: addr.String(), Description: description}
}

// Route implements the sdk.Msg interface.
func (msg MsgEditGovernor) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgEditGovernor) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgEditGovernor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrap(err.Error())
	}
	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgEditGovernor) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgEditGovernor.
func (msg MsgEditGovernor) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}

// NewMsgDelegateGovernor creates a new MsgDelegateGovernor instance
func NewMsgDelegateGovernor(delegator sdk.AccAddress, governor types.GovernorAddress) *MsgDelegateGovernor {
	return &MsgDelegateGovernor{DelegatorAddress: delegator.String(), GovernorAddress: governor.String()}
}

// Route implements the sdk.Msg interface.
func (msg MsgDelegateGovernor) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgDelegateGovernor) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgDelegateGovernor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.DelegatorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrap(err.Error())
	}
	if _, err := types.GovernorAddressFromBech32(msg.GovernorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid governor address: %s", err)
	}
	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgDelegateGovernor) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgDelegateGovernor.
func (msg MsgDelegateGovernor) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	return []sdk.AccAddress{delegator}
}

// NewMsgUndelegateGovernor creates a new MsgUndelegateGovernor instance
func NewMsgUndelegateGovernor(delegator sdk.AccAddress) *MsgUndelegateGovernor {
	return &MsgUndelegateGovernor{DelegatorAddress: delegator.String()}
}

// Route implements the sdk.Msg interface.
func (msg MsgUndelegateGovernor) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgUndelegateGovernor) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUndelegateGovernor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.DelegatorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrap(err.Error())
	}
	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgUndelegateGovernor) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgUndelegateGovernor.
func (msg MsgUndelegateGovernor) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	return []sdk.AccAddress{delegator}
}

// NewMsgUpdateGovernorStatus creates a new MsgUpdateGovernorStatus instance
func NewMsgUpdateGovernorStatus(address sdk.AccAddress, status GovernorStatus) *MsgUpdateGovernorStatus {
	return &MsgUpdateGovernorStatus{Address: address.String(), Status: status}
}

// Route implements the sdk.Msg interface.
func (msg MsgUpdateGovernorStatus) Route() string { return types.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgUpdateGovernorStatus) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateGovernorStatus) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrap(err.Error())
	}
	if !msg.Status.IsValid() {
		return types.ErrInvalidGovernorStatus.Wrap(msg.Status.String())
	}
	return nil
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgUpdateGovernorStatus) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgUpdateGovernorStatus.
func (msg MsgUpdateGovernorStatus) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Address)
	return []sdk.AccAddress{addr}
}
