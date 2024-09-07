package sims

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/atomone-hub/atomone/codec"
	sdk "github.com/atomone-hub/atomone/types"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
)

func TestGetSimulationLog(t *testing.T) {
	legacyAmino := codec.NewLegacyAmino()
	decoders := make(sdk.StoreDecoderRegistry)
	decoders[authtypes.StoreKey] = func(kvAs, kvBs kv.Pair) string { return "10" }

	tests := []struct {
		store       string
		kvPairs     []kv.Pair
		expectedLog string
	}{
		{
			"Empty",
			[]kv.Pair{{}},
			"",
		},
		{
			authtypes.StoreKey,
			[]kv.Pair{{Key: authtypes.GlobalAccountNumberKey, Value: legacyAmino.MustMarshal(uint64(10))}},
			"10",
		},
		{
			"OtherStore",
			[]kv.Pair{{Key: []byte("key"), Value: []byte("value")}},
			fmt.Sprintf("store A %X => %X\nstore B %X => %X\n", []byte("key"), []byte("value"), []byte("key"), []byte("value")),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.store, func(t *testing.T) {
			require.Equal(t, tt.expectedLog, GetSimulationLog(tt.store, decoders, tt.kvPairs, tt.kvPairs), tt.store)
		})
	}
}
