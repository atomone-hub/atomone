package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// Simulation parameter constants
const (
	DepositParamsMinDeposit                   = "deposit_params_min_deposit"
	DepositParamsDepositPeriod                = "deposit_params_deposit_period"
	DepositMinInitialRatio                    = "deposit_params_min_initial_ratio"
	VotingParamsVotingPeriod                  = "voting_params_voting_period"
	TallyParamsQuorum                         = "tally_params_quorum"
	TallyParamsThreshold                      = "tally_params_threshold"
	TallyParamsConstitutionAmendmentQuorum    = "tally_params_constitution_amendment_quorum"
	TallyParamsConstitutionAmendmentThreshold = "tally_params_constitution_amendment_threshold"
	TallyParamsLawQuorum                      = "tally_params_law_quorum"
	TallyParamsLawThreshold                   = "tally_params_law_threshold"
	GovernorStatusChangePeriod                = "governor_status_change_period"
	MinGovernorSelfDelegation                 = "min_governor_self_delegation"

	// NOTE: backport from v50
	MinDepositRatio          = "min_deposit_ratio"
	QuorumTimeout            = "quorum_timeout"
	MaxVotingPeriodExtension = "max_voting_period_extension"
	QuorumCheckCount         = "quorum_check_count"
)

// GenDepositParamsDepositPeriod returns randomized DepositParamsDepositPeriod
func GenDepositParamsDepositPeriod(r *rand.Rand) time.Duration {
	return time.Duration(simulation.RandIntBetween(r, 1, 2*60*60*24*2)) * time.Second
}

// GenDepositParamsMinDeposit returns randomized DepositParamsMinDeposit
func GenDepositParamsMinDeposit(r *rand.Rand) sdk.Coins {
	return sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, int64(simulation.RandIntBetween(r, 1, 1e3))))
}

// GenDepositMinInitialRatio returns randomized DepositMinInitialRatio
func GenDepositMinInitialDepositRatio(r *rand.Rand) sdk.Dec {
	return sdk.NewDec(int64(simulation.RandIntBetween(r, 0, 99))).Quo(sdk.NewDec(100))
}

// GenVotingParamsVotingPeriod returns randomized VotingParamsVotingPeriod
func GenVotingParamsVotingPeriod(r *rand.Rand) time.Duration {
	return time.Duration(simulation.RandIntBetween(r, 1, 2*60*60*24*2)) * time.Second
}

// GenTallyParamsQuorum returns randomized TallyParamsQuorum
func GenTallyParamsQuorum(r *rand.Rand) math.LegacyDec {
	return sdk.NewDecWithPrec(int64(simulation.RandIntBetween(r, 200, 400)), 3)
}

// GenTallyParamsThreshold returns randomized TallyParamsThreshold
func GenTallyParamsThreshold(r *rand.Rand) math.LegacyDec {
	return sdk.NewDecWithPrec(int64(simulation.RandIntBetween(r, 550, 700)), 3)
}

// GenMinDepositRatio returns randomized DepositMinRatio
func GenMinDepositRatio(r *rand.Rand) math.LegacyDec {
	return math.LegacyMustNewDecFromStr("0.01")
}

// GenTallyParamsQuorum returns randomized TallyParamsQuorum
func GenTallyParamsConstitutionalQuorum(r *rand.Rand, minDec sdk.Dec) math.LegacyDec {
	min := int(minDec.Mul(sdk.NewDec(1000)).RoundInt64())
	return sdk.NewDecWithPrec(int64(simulation.RandIntBetween(r, min, 600)), 3)
}

// GenTallyParamsThreshold returns randomized TallyParamsThreshold
func GenTallyParamsConstitutionalThreshold(r *rand.Rand, minDec sdk.Dec) math.LegacyDec {
	min := int(minDec.Mul(sdk.NewDec(1000)).RoundInt64())
	return sdk.NewDecWithPrec(int64(simulation.RandIntBetween(r, min, 950)), 3)
}

// GenQuorumTimeout returns a randomized QuorumTimeout between 0 and votingPeriod
func GenQuorumTimeout(r *rand.Rand, votingPeriod time.Duration) time.Duration {
	return time.Duration(simulation.RandIntBetween(r, 1, int(votingPeriod.Seconds()))) * time.Second
}

// GenMaxVotingPeriodExtension returns a randomized MaxVotingPeriodExtension
// greater than votingPeriod-quorumTimout.
func GenMaxVotingPeriodExtension(r *rand.Rand, votingPeriod, quorumTimout time.Duration) time.Duration {
	return time.Duration(simulation.RandIntBetween(r, 1, int(votingPeriod.Seconds())))*time.Second + (votingPeriod - quorumTimout)
}

// GenQuorumCheckCount returns a randomized QuorumCheckCount between 0 and 30
func GenQuorumCheckCount(r *rand.Rand) uint64 {
	return uint64(simulation.RandIntBetween(r, 0, 30))
}

// GenMinGovernorSelfDelegation returns a randomized MinGovernorSelfDelegation
func GenMinGovernorSelfDelegation(r *rand.Rand) math.Int {
	return math.NewInt(int64(simulation.RandIntBetween(r, 1000, 10000000)))
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	startingProposalID := uint64(simState.Rand.Intn(100))

	var minDeposit sdk.Coins
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinDeposit, &minDeposit, simState.Rand,
		func(r *rand.Rand) { minDeposit = GenDepositParamsMinDeposit(r) },
	)

	var depositPeriod time.Duration
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsDepositPeriod, &depositPeriod, simState.Rand,
		func(r *rand.Rand) { depositPeriod = GenDepositParamsDepositPeriod(r) },
	)

	var minInitialDepositRatio sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositMinInitialRatio, &minInitialDepositRatio, simState.Rand,
		func(r *rand.Rand) { minInitialDepositRatio = GenDepositMinInitialDepositRatio(r) },
	)

	var votingPeriod time.Duration
	simState.AppParams.GetOrGenerate(
		simState.Cdc, VotingParamsVotingPeriod, &votingPeriod, simState.Rand,
		func(r *rand.Rand) { votingPeriod = GenVotingParamsVotingPeriod(r) },
	)

	var quorum sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, TallyParamsQuorum, &quorum, simState.Rand,
		func(r *rand.Rand) { quorum = GenTallyParamsQuorum(r) },
	)

	var threshold sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, TallyParamsThreshold, &threshold, simState.Rand,
		func(r *rand.Rand) { threshold = GenTallyParamsThreshold(r) },
	)

	var minDepositRatio math.LegacyDec
	simState.AppParams.GetOrGenerate(simState.Cdc, MinDepositRatio, &minDepositRatio, simState.Rand, func(r *rand.Rand) { minDepositRatio = GenMinDepositRatio(r) })

	var lawQuorum sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, TallyParamsLawQuorum, &lawQuorum, simState.Rand,
		func(r *rand.Rand) { lawQuorum = GenTallyParamsConstitutionalQuorum(r, quorum) },
	)

	var lawThreshold sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, TallyParamsLawThreshold, &lawThreshold, simState.Rand,
		func(r *rand.Rand) { lawThreshold = GenTallyParamsConstitutionalThreshold(r, threshold) },
	)

	var amendmentsQuorum sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, TallyParamsConstitutionAmendmentQuorum, &amendmentsQuorum, simState.Rand,
		func(r *rand.Rand) { amendmentsQuorum = GenTallyParamsConstitutionalQuorum(r, lawQuorum) },
	)

	var amendmentsThreshold sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, TallyParamsConstitutionAmendmentThreshold, &amendmentsThreshold, simState.Rand,
		func(r *rand.Rand) { amendmentsThreshold = GenTallyParamsConstitutionalThreshold(r, lawThreshold) },
	)

	var quorumTimout time.Duration
	simState.AppParams.GetOrGenerate(simState.Cdc, QuorumTimeout, &quorumTimout, simState.Rand, func(r *rand.Rand) { quorumTimout = GenQuorumTimeout(r, votingPeriod) })

	var maxVotingPeriodExtension time.Duration
	simState.AppParams.GetOrGenerate(simState.Cdc, MaxVotingPeriodExtension, &maxVotingPeriodExtension, simState.Rand, func(r *rand.Rand) {
		maxVotingPeriodExtension = GenMaxVotingPeriodExtension(r, votingPeriod, quorumTimout)
	})

	var quorumCheckCount uint64
	simState.AppParams.GetOrGenerate(simState.Cdc, QuorumCheckCount, &quorumCheckCount, simState.Rand, func(r *rand.Rand) { quorumCheckCount = GenQuorumCheckCount(r) })

	var governorStatusChangePeriod time.Duration
	simState.AppParams.GetOrGenerate(
		simState.Cdc, GovernorStatusChangePeriod, &governorStatusChangePeriod, simState.Rand,
		func(r *rand.Rand) { governorStatusChangePeriod = GenDepositParamsDepositPeriod(r) },
	)

	var minGovernorSelfDelegation math.Int
	simState.AppParams.GetOrGenerate(
		simState.Cdc, MinGovernorSelfDelegation, &minGovernorSelfDelegation, simState.Rand,
		func(r *rand.Rand) { minGovernorSelfDelegation = GenMinGovernorSelfDelegation(r) },
	)

	govGenesis := v1.NewGenesisState(
		startingProposalID,
		v1.NewParams(minDeposit, depositPeriod, votingPeriod, quorum.String(), threshold.String(), amendmentsQuorum.String(), amendmentsThreshold.String(), lawQuorum.String(), lawThreshold.String(), minInitialDepositRatio.String(), simState.Rand.Intn(2) == 0, simState.Rand.Intn(2) == 0, minDepositRatio.String(), quorumTimout, maxVotingPeriodExtension, quorumCheckCount, governorStatusChangePeriod, minGovernorSelfDelegation.String()),
	)

	bz, err := json.MarshalIndent(&govGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated governance parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(govGenesis)
}
