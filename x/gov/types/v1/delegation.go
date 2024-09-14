package v1

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
)

// NewGovernanceDelegation creates a new GovernanceDelegation instance
func NewGovernanceDelegation(delegatorAddr sdk.AccAddress, governorAddr types.GovernorAddress) GovernanceDelegation {
	return GovernanceDelegation{
		DelegatorAddress: delegatorAddr.String(),
		GovernorAddress:  governorAddr.String(),
	}
}

// RegisterCodec registers the necessary types and interfaces for the module
func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(GovernanceDelegation{}, "gov/Delegation", nil)
}

// String implements the Stringer interface for GovernanceDelegation
func (gd GovernanceDelegation) String() string {
	return fmt.Sprintf("Delegator: %s, Governor: %s, Percentage: %s", gd.DelegatorAddress, gd.GovernorAddress)
}

// NewGovernorValShares creates a new GovernorValShares instance
func NewGovernorValShares(governorAddr types.GovernorAddress, validatorAddress sdk.ValAddress, shares sdk.Dec) GovernorValShares {
	if shares.IsNegative() {
		panic(fmt.Sprintf("invalid governor val shares: %s", shares))
	}

	return GovernorValShares{
		GovernorAddress:  governorAddr.String(),
		ValidatorAddress: validatorAddress.String(),
		Shares:           shares,
	}
}

// String implements the Stringer interface for GovernorValShares
func (gvs GovernorValShares) String() string {
	return fmt.Sprintf("Governor: %s, Validator: %s, Shares: %s", gvs.GovernorAddress, gvs.ValidatorAddress, gvs.Shares)
}
