package types

import (
	"slices"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// NewParams creates a new Params instance
func NewParams(mintDisabled bool, txFeeExceptions []string) Params {
	return Params{
		MintDisabled:    mintDisabled,
		TxFeeExceptions: txFeeExceptions,
	}
}

const (
	defaultMintDisabled = false
)

// NOTE(tb): Not possible to use `sdk.MsgTypeURL(types.MsgMintPhoton{})`
// instead of plain text because at this step the msg is not registered yet.
var defaultTxFeeExceptions = []string{"/atomone.photon.v1.MsgMintPhoton"}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(defaultMintDisabled, defaultTxFeeExceptions)
}

// ValidateBasic validates the set of params
func (p Params) ValidateBasic() error {
	// If used, the wildcard "*" in TxFeeExceptions must be the sole entry.
	// Mixing it with specific message type URLs is contradictory and rejected here.
	if slices.Contains(p.TxFeeExceptions, "*") && len(p.TxFeeExceptions) != 1 {
		return sdkerrors.ErrInvalidRequest.Wrapf(
			"tx_fee_exceptions: wildcard \"*\" must be the sole entry when used (got %d entries)",
			len(p.TxFeeExceptions),
		)
	}
	return nil
}
