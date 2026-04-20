package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// FlattenAnyMsgs recursively unpacks Any-encoded messages, expanding
// authz.MsgExec wrappers so that all leaf messages are returned in a flat
// slice. Entries that cannot be unpacked are represented as nil.
func FlattenAnyMsgs(cdc codec.BinaryCodec, anyMsgs []*codectypes.Any) []sdk.Msg {
	var result []sdk.Msg
	for _, anyMsg := range anyMsgs {
		var msg sdk.Msg
		if err := cdc.UnpackAny(anyMsg, &msg); err != nil {
			result = append(result, nil)
			continue
		}
		if execMsg, ok := msg.(*authz.MsgExec); ok {
			result = append(result, FlattenAnyMsgs(cdc, execMsg.Msgs)...)
		} else {
			result = append(result, msg)
		}
	}
	return result
}
