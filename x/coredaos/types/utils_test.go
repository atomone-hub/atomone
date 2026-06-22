package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/atomone-hub/atomone/x/coredaos/types"
)

// maxAuthzNestingDepth mirrors the unexported constant in utils.go. FlattenAnyMsgs
// rejects nestings strictly deeper than this value.
const maxAuthzNestingDepth = 8

func flattenTestCodec() codec.BinaryCodec {
	registry := codectypes.NewInterfaceRegistry()
	authz.RegisterInterfaces(registry)
	types.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

// wrapMsgExec wraps inner in `depth` nested authz.MsgExec layers and returns the
// outermost message as an Any. depth == 0 returns inner unchanged.
func wrapMsgExec(t *testing.T, inner *codectypes.Any, depth int) *codectypes.Any {
	t.Helper()
	cur := inner
	for i := 0; i < depth; i++ {
		any, err := codectypes.NewAnyWithValue(&authz.MsgExec{
			Grantee: sdk.AccAddress("grantee").String(),
			Msgs:    []*codectypes.Any{cur},
		})
		require.NoError(t, err)
		cur = any
	}
	return cur
}

func mustPackMsg(t *testing.T, msg sdk.Msg) *codectypes.Any {
	t.Helper()
	any, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	return any
}

func TestFlattenAnyMsgs(t *testing.T) {
	cdc := flattenTestCodec()

	leaf := &types.MsgUpdateParams{
		Params: types.Params{OversightDaoAddress: "the-leaf-address"},
	}

	t.Run("bare message is returned as-is", func(t *testing.T) {
		out, err := types.FlattenAnyMsgs(cdc, []*codectypes.Any{mustPackMsg(t, leaf)})
		require.NoError(t, err)
		require.Len(t, out, 1)
		updateParams, ok := out[0].(*types.MsgUpdateParams)
		require.True(t, ok)
		require.Equal(t, "the-leaf-address", updateParams.Params.OversightDaoAddress)
	})

	t.Run("multiple wrappers and bare messages are all flattened", func(t *testing.T) {
		out, err := types.FlattenAnyMsgs(cdc, []*codectypes.Any{
			wrapMsgExec(t, mustPackMsg(t, leaf), 1),
			wrapMsgExec(t, mustPackMsg(t, leaf), 2),
			mustPackMsg(t, leaf),
		})
		require.NoError(t, err)
		require.Len(t, out, 3)
		for _, m := range out {
			updateParams, ok := m.(*types.MsgUpdateParams)
			require.True(t, ok)
			require.Equal(t, "the-leaf-address", updateParams.Params.OversightDaoAddress)
		}
	})

	t.Run("nesting at the limit is accepted", func(t *testing.T) {
		out, err := types.FlattenAnyMsgs(cdc, []*codectypes.Any{
			wrapMsgExec(t, mustPackMsg(t, leaf), maxAuthzNestingDepth),
		})
		require.NoError(t, err)
		require.Len(t, out, 1)
		_, ok := out[0].(*types.MsgUpdateParams)
		require.True(t, ok)
	})

	t.Run("nesting beyond the limit is rejected", func(t *testing.T) {
		out, err := types.FlattenAnyMsgs(cdc, []*codectypes.Any{
			wrapMsgExec(t, mustPackMsg(t, leaf), maxAuthzNestingDepth+1),
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "authz nesting depth exceeded")
		require.Nil(t, out)
	})
}
