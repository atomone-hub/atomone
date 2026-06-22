package types

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	atomoneerrors "github.com/atomone-hub/atomone/types/errors"
)

const maxAuthzNestingDepth = 8

// FlattenAnyMsgs recursively unpacks Any-encoded messages, expanding
// authz.MsgExec wrappers so that all leaf messages are returned in a flat
// slice. Entries that cannot be unpacked are represented as nil.
func FlattenAnyMsgs(cdc codec.BinaryCodec, anyMsgs []*codectypes.Any) ([]sdk.Msg, error) {
	return flattenAnyMsgsWithDepth(cdc, anyMsgs, 0)
}

func flattenAnyMsgsWithDepth(cdc codec.BinaryCodec, anyMsgs []*codectypes.Any, depth int) ([]sdk.Msg, error) {
	if depth > maxAuthzNestingDepth {
		return nil, errorsmod.Wrap(atomoneerrors.ErrUnauthorized, "authz nesting depth exceeded")
	}
	var result []sdk.Msg
	for _, anyMsg := range anyMsgs {
		var msg sdk.Msg
		if err := cdc.UnpackAny(anyMsg, &msg); err != nil {
			result = append(result, nil)
			continue
		}
		if execMsg, ok := msg.(*authz.MsgExec); ok {
			subMsgs, err := flattenAnyMsgsWithDepth(cdc, execMsg.Msgs, depth+1)
			if err != nil {
				return nil, err
			}
			result = append(result, subMsgs...)
		} else {
			result = append(result, msg)
		}
	}
	return result, nil
}
