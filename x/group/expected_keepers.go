package group

import (
	sdk "github.com/atomone-hub/atomone/types"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
)

type AccountKeeper interface {
	// NewAccount returns a new account with the next account number. Does not save the new account to the store.
	NewAccount(sdk.Context, authtypes.AccountI) authtypes.AccountI

	// GetAccount retrieves an account from the store.
	GetAccount(sdk.Context, sdk.AccAddress) authtypes.AccountI

	// SetAccount sets an account in the store.
	SetAccount(sdk.Context, authtypes.AccountI)
	// RemoveAccount removes an account in the store.
	RemoveAccount(ctx sdk.Context, acc authtypes.AccountI)
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}