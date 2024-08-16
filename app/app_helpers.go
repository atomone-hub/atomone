package atomone

import (
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctestingtypes "github.com/cosmos/ibc-go/v7/testing/types"
	icstest "github.com/cosmos/interchain-security/v3/testutil/integration"
	ibcproviderkeeper "github.com/cosmos/interchain-security/v3/x/ccv/provider/keeper"

	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
)

// ProviderApp interface implementations for icstest tests

// GetProviderKeeper implements the ProviderApp interface.
func (app *AtomOneApp) GetProviderKeeper() ibcproviderkeeper.Keeper { //nolint:nolintlint
	return app.ProviderKeeper
}

// GetStakingKeeper implements the TestingApp interface. Needed for ICS.
func (app *AtomOneApp) GetStakingKeeper() ibctestingtypes.StakingKeeper { //nolint:nolintlint
	return app.StakingKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (app *AtomOneApp) GetIBCKeeper() *ibckeeper.Keeper { //nolint:nolintlint
	return app.IBCKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (app *AtomOneApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper { //nolint:nolintlint
	return app.ScopedIBCKeeper
}

// GetTestStakingKeeper implements the ProviderApp interface.
func (app *AtomOneApp) GetTestStakingKeeper() icstest.TestStakingKeeper { //nolint:nolintlint
	return app.StakingKeeper
}

// GetTestBankKeeper implements the ProviderApp interface.
func (app *AtomOneApp) GetTestBankKeeper() icstest.TestBankKeeper { //nolint:nolintlint
	return app.BankKeeper
}

// GetTestSlashingKeeper implements the ProviderApp interface.
func (app *AtomOneApp) GetTestSlashingKeeper() icstest.TestSlashingKeeper { //nolint:nolintlint
	return app.SlashingKeeper
}

// GetTestDistributionKeeper implements the ProviderApp interface.
func (app *AtomOneApp) GetTestDistributionKeeper() icstest.TestDistributionKeeper { //nolint:nolintlint
	return app.DistrKeeper
}

func (app *AtomOneApp) GetTestAccountKeeper() icstest.TestAccountKeeper { //nolint:nolintlint
	return app.AccountKeeper
}
