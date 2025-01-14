package simulation_test

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/atomone-hub/atomone/x/gov/simulation"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// TestRandomizedGenState tests the normal scenario of applying RandomizedGenState.
// Abnormal scenarios are not tested here.
func TestRandomizedGenState(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)

	simState := module.SimulationState{
		AppParams:    make(simtypes.AppParams),
		Cdc:          cdc,
		Rand:         r,
		NumBonded:    3,
		Accounts:     simtypes.RandomAccounts(r, 3),
		InitialStake: sdkmath.NewInt(1000),
		GenState:     make(map[string]json.RawMessage),
	}

	simulation.RandomizedGenState(&simState)

	var govGenesis v1.GenesisState
	simState.Cdc.MustUnmarshalJSON(simState.GenState[types.ModuleName], &govGenesis)

	const (
		tallyQuorum        = "0.294000000000000000"
		tallyThreshold     = "0.611000000000000000"
		amendmentQuorum    = "0.568000000000000000"
		amendmentThreshold = "0.933000000000000000"
		lawQuorum          = "0.540000000000000000"
		lawThreshold       = "0.931000000000000000"
	)

	var (
		minDepositUpdatePeriod        = time.Duration(67011000000000)
		minInitialDepositUpdatePeriod = time.Duration(66992000000000)
	)

	require.Equal(t, []sdk.Coin{}, govGenesis.Params.MinDeposit)
	require.Equal(t, "52h44m19s", govGenesis.Params.MaxDepositPeriod.String())
	require.Equal(t, float64(278770), govGenesis.Params.VotingPeriod.Seconds())
	require.Equal(t, tallyQuorum, govGenesis.Params.Quorum)
	require.Equal(t, tallyThreshold, govGenesis.Params.Threshold)
	require.Equal(t, amendmentQuorum, govGenesis.Params.ConstitutionAmendmentQuorum)
	require.Equal(t, amendmentThreshold, govGenesis.Params.ConstitutionAmendmentThreshold)
	require.Equal(t, lawQuorum, govGenesis.Params.LawQuorum)
	require.Equal(t, lawThreshold, govGenesis.Params.LawThreshold)
	require.Equal(t, "", govGenesis.Params.MinInitialDepositRatio)
	require.Equal(t, "26h19m52s", govGenesis.Params.QuorumTimeout.String())
	require.Equal(t, "120h29m51s", govGenesis.Params.MaxVotingPeriodExtension.String())
	require.Equal(t, uint64(17), govGenesis.Params.QuorumCheckCount)
	require.Equal(t, uint64(0x28), govGenesis.StartingProposalId)
	require.Equal(t, []*v1.Deposit{}, govGenesis.Deposits)
	require.Equal(t, []*v1.Vote{}, govGenesis.Votes)
	require.Equal(t, []*v1.Proposal{}, govGenesis.Proposals)
	require.Equal(t, "", govGenesis.Constitution)
	require.Equal(t, v1.MinDepositThrottler{
		FloorValue:                sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(915))),
		UpdatePeriod:              &minDepositUpdatePeriod,
		TargetActiveProposals:     10,
		SensitivityTargetDistance: 1,
		IncreaseRatio:             "0.128000000000000000",
		DecreaseRatio:             "0.018000000000000000",
	}, *govGenesis.Params.MinDepositThrottler)
	require.Equal(t, v1.MinInitialDepositThrottler{
		FloorValue:                sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(805))),
		UpdatePeriod:              &minInitialDepositUpdatePeriod,
		TargetProposals:           23,
		SensitivityTargetDistance: 2,
		IncreaseRatio:             "0.090000000000000000",
		DecreaseRatio:             "0.030000000000000000",
	}, *govGenesis.Params.MinInitialDepositThrottler)
}

// TestRandomizedGenState tests abnormal scenarios of applying RandomizedGenState.
func TestRandomizedGenState1(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	s := rand.NewSource(1)
	r := rand.New(s)
	// all these tests will panic
	tests := []struct {
		simState module.SimulationState
		panicMsg string
	}{
		{ // panic => reason: incomplete initialization of the simState
			module.SimulationState{}, "invalid memory address or nil pointer dereference"},
		{ // panic => reason: incomplete initialization of the simState
			module.SimulationState{
				AppParams: make(simtypes.AppParams),
				Cdc:       cdc,
				Rand:      r,
			}, "assignment to entry in nil map"},
	}

	for _, tt := range tests {
		require.Panicsf(t, func() { simulation.RandomizedGenState(&tt.simState) }, tt.panicMsg)
	}
}
