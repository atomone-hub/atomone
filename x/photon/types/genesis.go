package types

// NewGenesisState creates a new genesis state for the governance module
func NewGenesisState(params Params) *GenesisState {
	return &GenesisState{
		Params: params,
	}
}

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return NewGenesisState(DefaultParams())
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	return gs.Params.ValidateBasic()
}
