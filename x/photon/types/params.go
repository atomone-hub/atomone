package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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

var defaultTxFeeExceptions = []string{sdk.MsgTypeURL(&MsgMintPhoton{})}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(defaultMintDisabled, defaultTxFeeExceptions)
}

// Validate validates the set of params
func (p Params) ValidateBasic() error {
	return nil
}
