package v1

type GovernorI interface {
	GetMoniker() string                  // moniker of the governor
	GetStatus() GovernorStatus           // status of the governor
	IsActive() bool                      // check if has a active status
	IsInactive() bool                    // check if has status inactive
	GetAddress() GovernorAddress         // governor address to receive/return governors delegations
	GetDescription() GovernorDescription // description of the governor
}
