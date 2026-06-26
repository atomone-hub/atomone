package types

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// StakingKeeper defines the expected interface needed to interact with the
// staking module.
type StakingKeeper interface {
	// GetDelegatorBonded returns the total amount a delegator has bonded.
	GetDelegatorBonded(ctx context.Context, delegator sdk.AccAddress) (math.Int, error)
	// GetDelegatorUnbonding returns the total amount a delegator has unbonding.
	GetDelegatorUnbonding(ctx context.Context, delegator sdk.AccAddress) (math.Int, error)
}

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
}

// BankKeeper defines the expected account keeper used for simulations (noalias)
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}
