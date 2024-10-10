package types

// NewParams creates a new Params instance
func NewParams(mintDisabled bool) Params {
	return Params{
		MintDisabled: mintDisabled,
	}
}

const (
	defaultMintDisabled = false
)

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(defaultMintDisabled)
}

// Validate validates the set of params
func (p Params) ValidateBasic() error {
	return nil
}
