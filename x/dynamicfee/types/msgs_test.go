package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/dynamicfee/types"
)

func TestMsgUpdateParams(t *testing.T) {
	t.Run("should reject a message with an invalid authority address", func(t *testing.T) {
		msg := types.NewMsgUpdateParams("invalid", types.DefaultParams())
		err := msg.ValidateBasic()
		require.Error(t, err)
	})

	t.Run("should accept an empty message with a valid authority address", func(t *testing.T) {
		msg := types.NewMsgUpdateParams(sdk.AccAddress("test").String(), types.DefaultParams())
		err := msg.ValidateBasic()
		require.NoError(t, err)
	})
}
