package v1

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// NewGovernanceDelegation creates a new GovernanceDelegation instance
func NewGovernanceDelegation(delegatorAddr sdk.AccAddress, governorAddr sdkgovtypes.GovernorAddress) GovernanceDelegation {
	return GovernanceDelegation{
		DelegatorAddress: delegatorAddr.String(),
		GovernorAddress:  governorAddr.String(),
	}
}

// NewGovernorValShares creates a new GovernorValShares instance
func NewGovernorValShares(governorAddr sdkgovtypes.GovernorAddress, validatorAddress sdk.ValAddress, shares math.LegacyDec) GovernorValShares {
	if shares.IsNegative() {
		panic(fmt.Sprintf("invalid governor val shares: %s", shares))
	}

	return GovernorValShares{
		GovernorAddress:  governorAddr.String(),
		ValidatorAddress: validatorAddress.String(),
		Shares:           shares,
	}
}
