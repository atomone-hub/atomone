package v1beta1

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	sdkgovtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// Content defines an interface that a proposal must implement. It contains
// information such as the title and description along with the type and routing
// information for the appropriate handler to process the proposal. Content can
// have additional fields, which will handled by a proposal's Handler.
type Content interface {
	GetTitle() string
	GetDescription() string
	ProposalRoute() string
	ProposalType() string
	ValidateBasic() error
	String() string
}

// Handler defines a function that handles a proposal after it has passed the
// governance process.
type Handler func(ctx sdk.Context, content Content) error

// WrapHandler converts a Cosmos SDK gov Handler to GovGen gov Handler
func WrapSDKHandler(sdkHandler sdkgovtypes.Handler) Handler {
	return func(ctx sdk.Context, content Content) error {
		return sdkHandler(ctx, content)
	}
}

type HandlerRoute struct {
	Handler  Handler
	RouteKey string
}

// IsManyPerContainerType implements the depinject.ManyPerContainerType interface.
func (HandlerRoute) IsManyPerContainerType() {}
