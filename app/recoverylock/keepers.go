package recoverylock

import (
	"context"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// SendRestriction protects the concrete bank keeper used by the bank service.
func (l *Lock) SendRestriction(ctx context.Context, from, to sdk.AccAddress, _ sdk.Coins) (sdk.AccAddress, error) {
	if l.listedSource(ctx, from) && !l.config.IsRecovery(to) {
		return nil, ErrLocked.Wrap("listed funds may only go to the recovery address")
	}
	return to, nil
}

func (l *Lock) listedSource(ctx context.Context, from sdk.AccAddress) bool {
	return l.active(sdk.UnwrapSDKContext(ctx)) && l.config.IsListed(from)
}

// Bank wraps module callers so recovery refusals occur before balance writes.
type Bank struct {
	bankkeeper.Keeper
	Lock *Lock
}

func (b Bank) SendCoins(ctx context.Context, from, to sdk.AccAddress, amount sdk.Coins) error {
	if _, err := b.Lock.SendRestriction(ctx, from, to, amount); err != nil {
		return err
	}
	return b.Keeper.SendCoins(ctx, from, to, amount)
}

func (b Bank) InputOutputCoins(ctx context.Context, input banktypes.Input, outputs []banktypes.Output) error {
	if b.Lock.active(sdk.UnwrapSDKContext(ctx)) {
		from, err := b.Lock.cdc.InterfaceRegistry().SigningContext().AddressCodec().StringToBytes(input.Address)
		if err != nil {
			return err
		}
		if b.Lock.config.IsListed(from) {
			return ErrLocked.Wrap("listed multisends are disabled")
		}
	}
	return b.Keeper.InputOutputCoins(ctx, input, outputs)
}

func (b Bank) SendCoinsFromAccountToModule(ctx context.Context, from sdk.AccAddress, module string, amount sdk.Coins) error {
	if b.Lock.listedSource(ctx, from) {
		return ErrLocked.Wrap("listed accounts cannot fund modules")
	}
	return b.Keeper.SendCoinsFromAccountToModule(ctx, from, module, amount)
}

func (b Bank) DelegateCoinsFromAccountToModule(ctx context.Context, from sdk.AccAddress, module string, amount sdk.Coins) error {
	if b.Lock.listedSource(ctx, from) {
		return ErrLocked.Wrap("listed accounts cannot delegate funds")
	}
	return b.Keeper.DelegateCoinsFromAccountToModule(ctx, from, module, amount)
}

func (b Bank) DelegateCoins(ctx context.Context, from, to sdk.AccAddress, amount sdk.Coins) error {
	if b.Lock.listedSource(ctx, from) {
		return ErrLocked.Wrap("listed accounts cannot delegate funds")
	}
	return b.Keeper.DelegateCoins(ctx, from, to, amount)
}

// Router checks actual internal dispatch. Top-level messages use BaseApp's
// original router after passing the ante policy.
type Router struct {
	Inner baseapp.MessageRouter
	Lock  *Lock
}

func (r Router) Handler(msg sdk.Msg) baseapp.MsgServiceHandler {
	return r.wrap(r.Inner.Handler(msg))
}

func (r Router) HandlerByTypeURL(typeURL string) baseapp.MsgServiceHandler {
	return r.wrap(r.Inner.HandlerByTypeURL(typeURL))
}

func (r Router) wrap(handler baseapp.MsgServiceHandler) baseapp.MsgServiceHandler {
	if handler == nil {
		return nil
	}
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		if r.Lock.active(ctx) {
			listed, _, err := r.Lock.checkSigners(msg)
			if err != nil {
				return nil, err
			}
			if listed {
				return nil, ErrLocked.Wrap("listed accounts cannot act through internal dispatch")
			}
		}
		return handler(ctx, msg)
	}
}
