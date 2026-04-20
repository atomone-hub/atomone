package ante

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	atomoneerrors "github.com/atomone-hub/atomone/types/errors"
)

// iterateMsg calls fn for each message in msgs. For authz.MsgExec messages,
// fn is called for each inner message instead of the exec message itself.
// Nested MsgExec wrappers are expanded recursively.
// Returns ErrUnauthorized if an inner message cannot be unpacked.
func iterateMsg(cdc codec.BinaryCodec, msgs []sdk.Msg, fn func(sdk.Msg) error) error {
	for _, m := range msgs {
		if execMsg, ok := m.(*authz.MsgExec); ok {
			for _, anyInner := range execMsg.Msgs {
				var inner sdk.Msg
				if err := cdc.UnpackAny(anyInner, &inner); err != nil {
					return errorsmod.Wrap(atomoneerrors.ErrUnauthorized, "cannot unmarshal authz exec msgs")
				}
				// Recurse to handle nested MsgExec wrappers.
				if err := iterateMsg(cdc, []sdk.Msg{inner}, fn); err != nil {
					return err
				}
			}
			continue
		}
		if err := fn(m); err != nil {
			return err
		}
	}
	return nil
}

