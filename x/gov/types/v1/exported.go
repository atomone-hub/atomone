package v1

import (
	"github.com/atomone-hub/atomone/x/gov/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type GovernorI interface {
	GetMoniker() string                  // moniker of the governor
	GetStatus() GovernorStatus           // status of the governor
	IsActive() bool                      // check if has a active status
	IsInactive() bool                    // check if has status inactive
	GetAddress() types.GovernorAddress   // governor address to receive/return governors delegations
	GetDescription() GovernorDescription // description of the governor
	GetVotingPower() sdk.Dec             // voting power of the governor
}
