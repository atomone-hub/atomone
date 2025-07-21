package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _, _, _, _, _ sdk.Msg = &MsgAnnotateProposal{}, &MsgEndorseProposal{}, &MsgExtendVotingPeriod{}, &MsgVetoProposal{}, &MsgUpdateParams{}

// NewMsgAnnotateProposal creates a new MsgAnnotateProposal instance
func NewMsgAnnotateProposal(signer sdk.AccAddress, proposalID uint64, annotation string) *MsgAnnotateProposal {
	return &MsgAnnotateProposal{
		Annotator:  signer.String(),
		ProposalId: proposalID,
		Annotation: annotation,
	}
}

// Route implements the sdk.Msg interface.
func (msg *MsgAnnotateProposal) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg *MsgAnnotateProposal) Type() string {
	return sdk.MsgTypeURL(msg)
}

// GetSigners implements the sdk.Msg interface.
func (msg *MsgAnnotateProposal) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Annotator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// GetSignBytes implements the sdk.Msg interface.
func (msg *MsgAnnotateProposal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg *MsgAnnotateProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Annotator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid annotator address: %s", err)
	}
	if len(msg.Annotation) == 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "annotation cannot be empty")
	}
	return nil
}

// NewMsgEndorseProposal creates a new MsgEndorseProposal instance
func NewMsgEndorseProposal(signer sdk.AccAddress, proposalID uint64) *MsgEndorseProposal {
	return &MsgEndorseProposal{
		Endorser:   signer.String(),
		ProposalId: proposalID,
	}
}

// Route implements the sdk.Msg interface.
func (msg *MsgEndorseProposal) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg *MsgEndorseProposal) Type() string {
	return sdk.MsgTypeURL(msg)
}

// GetSigners implements the sdk.Msg interface.
func (msg *MsgEndorseProposal) GetSigners() []sdk.AccAddress {
	endorser, err := sdk.AccAddressFromBech32(msg.Endorser)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{endorser}
}

// GetSignBytes implements the sdk.Msg interface.
func (msg *MsgEndorseProposal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg *MsgEndorseProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Endorser); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid endorser address: %s", err)
	}
	return nil
}

// NewMsgExtendVotingPeriod creates a new MsgExtendVotingPeriod instance
func NewMsgExtendVotingPeriod(signer sdk.AccAddress, proposalID uint64) *MsgExtendVotingPeriod {
	return &MsgExtendVotingPeriod{
		Extender:   signer.String(),
		ProposalId: proposalID,
	}
}

// Route implements the sdk.Msg interface.
func (msg *MsgExtendVotingPeriod) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg *MsgExtendVotingPeriod) Type() string {
	return sdk.MsgTypeURL(msg)
}

// GetSigners implements the sdk.Msg interface.
func (msg *MsgExtendVotingPeriod) GetSigners() []sdk.AccAddress {
	extender, err := sdk.AccAddressFromBech32(msg.Extender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{extender}
}

// GetSignBytes implements the sdk.Msg interface.
func (msg *MsgExtendVotingPeriod) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg *MsgExtendVotingPeriod) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Extender); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid extender address: %s", err)
	}
	return nil
}

// NewMsgVetoProposal creates a new MsgVetoProposal instance
func NewMsgVetoProposal(signer sdk.AccAddress, proposalID uint64) *MsgVetoProposal {
	return &MsgVetoProposal{
		Vetoer:     signer.String(),
		ProposalId: proposalID,
	}
}

// Route implements the sdk.Msg interface.
func (msg *MsgVetoProposal) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg *MsgVetoProposal) Type() string {
	return sdk.MsgTypeURL(msg)
}

// GetSigners implements the sdk.Msg interface.
func (msg *MsgVetoProposal) GetSigners() []sdk.AccAddress {
	vetoer, err := sdk.AccAddressFromBech32(msg.Vetoer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{vetoer}
}

// GetSignBytes implements the sdk.Msg interface.
func (msg *MsgVetoProposal) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg *MsgVetoProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Vetoer); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid vetoer address: %s", err)
	}
	return nil
}

// NewMsgUpdateParams creates a new MsgUpdateParams instance
func NewMsgUpdateParams(authority string, params Params) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}
}

// Route implements the sdk.Msg interface.
func (msg *MsgUpdateParams) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface.
func (msg *MsgUpdateParams) Type() string {
	return sdk.MsgTypeURL(msg)
}

// ValidateBasic implements the sdk.Msg interface.
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.ValidateBasic()
}

// GetSigners implements the sdk.Msg interface.
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{authority}
}

// GetSignBytes implements the sdk.Msg interface.
func (msg *MsgUpdateParams) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}
