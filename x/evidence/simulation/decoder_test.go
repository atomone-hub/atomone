package simulation_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/depinject"
	"github.com/atomone-hub/atomone/crypto/keys/ed25519"
	sdk "github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/evidence/keeper"
	"github.com/atomone-hub/atomone/x/evidence/simulation"
	"github.com/atomone-hub/atomone/x/evidence/testutil"
	"github.com/atomone-hub/atomone/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/types/kv"
)

func TestDecodeStore(t *testing.T) {
	var evidenceKeeper keeper.Keeper

	err := depinject.Inject(testutil.AppConfig, &evidenceKeeper)
	require.NoError(t, err)

	dec := simulation.NewDecodeStore(evidenceKeeper)

	delPk1 := ed25519.GenPrivKey().PubKey()

	ev := &types.Equivocation{
		Height:           10,
		Time:             time.Now().UTC(),
		Power:            1000,
		ConsensusAddress: sdk.ConsAddress(delPk1.Address()).String(),
	}

	evBz, err := evidenceKeeper.MarshalEvidence(ev)
	require.NoError(t, err)

	kvPairs := kv.Pairs{
		Pairs: []kv.Pair{
			{
				Key:   types.KeyPrefixEvidence,
				Value: evBz,
			},
			{
				Key:   []byte{0x99},
				Value: []byte{0x99},
			},
		},
	}
	tests := []struct {
		name        string
		expectedLog string
	}{
		{"Evidence", fmt.Sprintf("%v\n%v", ev, ev)},
		{"other", ""},
	}

	for i, tt := range tests {
		i, tt := i, tt
		t.Run(tt.name, func(t *testing.T) {
			switch i {
			case len(tests) - 1:
				require.Panics(t, func() { dec(kvPairs.Pairs[i], kvPairs.Pairs[i]) }, tt.name)
			default:
				require.Equal(t, tt.expectedLog, dec(kvPairs.Pairs[i], kvPairs.Pairs[i]), tt.name)
			}
		})
	}
}
