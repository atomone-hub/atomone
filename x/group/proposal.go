package group

import (
	"github.com/atomone-hub/atomone/codec/types"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/types/tx"
)

// GetMsgs unpacks p.Messages Any's into sdk.Msg's
func (p *Proposal) GetMsgs() ([]sdk.Msg, error) {
	return tx.GetMsgs(p.Messages, "proposal")
}

// SetMsgs packs msgs into Any's
func (p *Proposal) SetMsgs(msgs []sdk.Msg) error {
	anys, err := tx.SetMsgs(msgs)
	if err != nil {
		return err
	}
	p.Messages = anys
	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (p Proposal) UnpackInterfaces(unpacker types.AnyUnpacker) error {
	return tx.UnpackInterfaces(unpacker, p.Messages)
}
