package ante

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/feemarket/types"
)

type FeeMarketKeeper interface {
	GetMinGasPrice(sdk.Context, string) (sdk.DecCoin, error)
	GetParams(sdk.Context) (types.Params, error)
}

// AccountKeeper defines the contract needed for AccountKeeper related APIs.
// Interface provides support to use non-sdk AccountKeeper for AnteHandler's decorators.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// FeeGrantKeeper defines the expected feegrant keeper.
type FeeGrantKeeper interface {
	UseGrantedFees(ctx context.Context, granter, grantee sdk.AccAddress, fee sdk.Coins, msgs []sdk.Msg) error
}

// BankKeeper defines the contract needed for supply related APIs.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}
