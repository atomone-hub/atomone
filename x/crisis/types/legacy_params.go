package types

import (
	"fmt"

	sdk "github.com/atomone-hub/atomone/types"
	paramtypes "github.com/atomone-hub/atomone/x/params/types"
)

// ParamStoreKeyConstantFee is the constant fee parameter
var ParamStoreKeyConstantFee = []byte("ConstantFee")

// Deprecated: Type declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable(
		paramtypes.NewParamSetPair(ParamStoreKeyConstantFee, sdk.Coin{}, validateConstantFee),
	)
}

func validateConstantFee(i interface{}) error {
	v, ok := i.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if !v.IsValid() {
		return fmt.Errorf("invalid constant fee: %s", v)
	}

	return nil
}
