package ante

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
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

// flattenAnyMsgs recursively unpacks a list of Any-encoded messages, expanding
// authz.MsgExec wrappers so that all leaf messages are returned in a flat
// slice. Entries that cannot be unpacked are represented as nil — they still
// count towards the total length, which callers use to detect bundling.
func flattenAnyMsgs(cdc codec.BinaryCodec, anyMsgs []*codectypes.Any) []sdk.Msg {
	var result []sdk.Msg
	for _, anyMsg := range anyMsgs {
		var msg sdk.Msg
		if err := cdc.UnpackAny(anyMsg, &msg); err != nil {
			result = append(result, nil)
			continue
		}
		if execMsg, ok := msg.(*authz.MsgExec); ok {
			result = append(result, flattenAnyMsgs(cdc, execMsg.Msgs)...)
		} else {
			result = append(result, msg)
		}
	}
	return result
}
