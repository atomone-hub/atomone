package types

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

// NOTE(tb): can't replace the plain text proto path with
// `sdk.MsgTypeURL(types.MsgMintPhoton{})` because at this step it might not be
// registered and so it would return only "/".
var defaultTxFeeExceptions = []string{"/atomone.photon.v1.MsgMintPhoton"}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(defaultMintDisabled, defaultTxFeeExceptions)
}

// Validate validates the set of params
func (p Params) ValidateBasic() error {
	return nil
}
