package slashing_test

import (
	sdk "github.com/atomone-hub/atomone/types"
)

// The default power validators are initialized to have within tests
var InitTokens = sdk.TokensFromConsensusPower(200, sdk.DefaultPowerReduction)
