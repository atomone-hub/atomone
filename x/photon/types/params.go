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

// NOTE(tb): Not possible to use `sdk.MsgTypeURL(types.MsgMintPhoton{})`
// instead of plain text because at this step the msg is not registered yet.
var defaultTxFeeExceptions = []string{"/atomone.photon.v1.MsgMintPhoton"}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(defaultMintDisabled, defaultTxFeeExceptions)
}

// Validate validates the set of params
func (p Params) ValidateBasic() error {
	return nil
}
