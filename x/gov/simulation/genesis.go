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
	DepositParamsMinDeposit    = "deposit_params_min_deposit"
	DepositParamsDepositPeriod = "deposit_params_deposit_period"
	// DepositMinInitialRatio                                  = "deposit_params_min_initial_ratio"
	VotingParamsVotingPeriod                                = "voting_params_voting_period"
	TallyParamsQuorum                                       = "tally_params_quorum"
	TallyParamsThreshold                                    = "tally_params_threshold"
	TallyParamsConstitutionAmendmentQuorum                  = "tally_params_constitution_amendment_quorum"
	TallyParamsConstitutionAmendmentThreshold               = "tally_params_constitution_amendment_threshold"
	TallyParamsLawQuorum                                    = "tally_params_law_quorum"
	TallyParamsLawThreshold                                 = "tally_params_law_threshold"
	DepositParamsMinDepositFloor                            = "deposit_params_min_deposit_floor"
	DepositParamsMinDepositUpdatePeriod                     = "deposit_params_min_deposit_update_period"
	DepositParamsMinDepositSensitivityTargetDistance        = "deposit_params_min_deposit_sensitivity_target_distance"
	DepositParamsMinDepositIncreaseRatio                    = "deposit_params_min_deposit_increase_ratio"
	DepositParamsMinDepositDecreaseRatio                    = "deposit_params_min_deposit_decrease_ratio"
	DepositParamsTargetActiveProposals                      = "deposit_params_target_active_proposals"
	DepositParamsMinInitialDepositFloor                     = "deposit_params_min_initial_deposit_floor"
	DepositParamsMinInitialDepositUpdatePeriod              = "deposit_params_min_initial_deposit_update_period"
	DepositParamsMinInitialDepositSensitivityTargetDistance = "deposit_params_min_initial_deposit_sensitivity_target_distance"
	DepositParamsMinInitialDepositIncreaseRatio             = "deposit_params_min_initial_deposit_increase_ratio"
	DepositParamsMinInitialDepositDecreaseRatio             = "deposit_params_min_initial_deposit_decrease_ratio"
	DepositParamsMinInitialDepositTargetProposals           = "deposit_params_min_initial_deposit_target_proposals"

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

// GenDepositParamsMinDepositUpdatePeriod returns randomized DepositParamsMinDepositUpdatePeriod
func GenDepositParamsMinDepositUpdatePeriod(r *rand.Rand, votingPeriod time.Duration) time.Duration {
	return time.Duration(simulation.RandIntBetween(r, 1, int(votingPeriod.Seconds()))) * time.Second
}

// GenDepositParamsMinDepositSensitivityTargetDistance returns randomized DepositParamsMinDepositSensitivityTargetDistance
func GenDepositParamsMinDepositSensitivityTargetDistance(r *rand.Rand) uint64 {
	return uint64(simulation.RandIntBetween(r, 1, 10))
}

// GenDepositParamsMinDepositChangeRatio returns randomized DepositParamsMinDepositChangeRatio
func GenDepositParamsMinDepositChangeRatio(r *rand.Rand, max, prec int) sdk.Dec {
	return sdk.NewDecWithPrec(int64(simulation.RandIntBetween(r, 0, max)), int64(prec))
}

// GenDepositParamsTargetActiveProposals returns randomized DepositParamsTargetActiveProposals
func GenDepositParamsTargetActiveProposals(r *rand.Rand) uint64 {
	return uint64(simulation.RandIntBetween(r, 1, 100))
}

// GenDepositParamsMinInitialDepositUpdatePeriod returns randomized DepositParamsMinInitialDepositUpdatePeriod
func GenDepositParamsMinInitialDepositUpdatePeriod(r *rand.Rand, depositPeriod time.Duration) time.Duration {
	return time.Duration(simulation.RandIntBetween(r, 1, int(depositPeriod.Seconds()))) * time.Second
}

// GenDepositParamsMinInitialDepositSensitivityTargetDistance returns randomized DepositParamsMinInitialDepositSensitivityTargetDistance
func GenDepositParamsMinInitialDepositSensitivityTargetDistance(r *rand.Rand) uint64 {
	return uint64(simulation.RandIntBetween(r, 1, 10))
}

// GenDepositParamsMinInitialDepositChangeRatio returns randomized DepositParamsMinInitialDepositChangeRatio
func GenDepositParamsMinInitialDepositChangeRatio(r *rand.Rand, max, prec int) sdk.Dec {
	return sdk.NewDecWithPrec(int64(simulation.RandIntBetween(r, 0, max)), int64(prec))
}

func GenDepositParamsMinInitialDepositTargetProposals(r *rand.Rand) uint64 {
	return uint64(simulation.RandIntBetween(r, 1, 100))
}

// RandomizedGenState generates a random GenesisState for gov
func RandomizedGenState(simState *module.SimulationState) {
	startingProposalID := uint64(simState.Rand.Intn(100))

	// var minDeposit sdk.Coins
	// simState.AppParams.GetOrGenerate(
	//	simState.Cdc, DepositParamsMinDeposit, &minDeposit, simState.Rand,
	//	func(r *rand.Rand) { minDeposit = GenDepositParamsMinDeposit(r) },
	//)

	var depositPeriod time.Duration
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsDepositPeriod, &depositPeriod, simState.Rand,
		func(r *rand.Rand) { depositPeriod = GenDepositParamsDepositPeriod(r) },
	)

	// var minInitialDepositRatio sdk.Dec
	// simState.AppParams.GetOrGenerate(
	// 	simState.Cdc, DepositMinInitialRatio, &minInitialDepositRatio, simState.Rand,
	// 	func(r *rand.Rand) { minInitialDepositRatio = GenDepositMinInitialDepositRatio(r) },
	// )

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

	var minDepositFloor sdk.Coins
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinDepositFloor, &minDepositFloor, simState.Rand,
		func(r *rand.Rand) { minDepositFloor = GenDepositParamsMinDeposit(r) },
	)

	var minDepositUpdatePeriod time.Duration
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsDepositPeriod, &minDepositUpdatePeriod, simState.Rand,
		func(r *rand.Rand) { minDepositUpdatePeriod = GenDepositParamsMinDepositUpdatePeriod(r, votingPeriod) },
	)

	var minDepositSensitivityTargetDistance uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinDepositSensitivityTargetDistance, &minDepositSensitivityTargetDistance, simState.Rand,
		func(r *rand.Rand) {
			minDepositSensitivityTargetDistance = GenDepositParamsMinDepositSensitivityTargetDistance(r)
		},
	)

	var minDepositIncreaseRatio sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinDepositIncreaseRatio, &minDepositIncreaseRatio, simState.Rand,
		func(r *rand.Rand) { minDepositIncreaseRatio = GenDepositParamsMinDepositChangeRatio(r, 300, 3) },
	)

	var minDepositDecreaseRatio sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinDepositDecreaseRatio, &minDepositDecreaseRatio, simState.Rand,
		func(r *rand.Rand) {
			minDepositDecreaseRatio = GenDepositParamsMinDepositChangeRatio(r,
				int(minDepositIncreaseRatio.MulInt64(1000).QuoInt64(2).TruncateInt64()), 3)
		},
	)

	var targetActiveProposals uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsTargetActiveProposals, &targetActiveProposals, simState.Rand,
		func(r *rand.Rand) { targetActiveProposals = GenDepositParamsTargetActiveProposals(r) },
	)

	var minInitialDepositFloor sdk.Coins
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinInitialDepositFloor, &minInitialDepositFloor, simState.Rand,
		func(r *rand.Rand) {
			ratio := GenDepositMinInitialDepositRatio(r)
			minInitialDepositFloor = sdk.NewCoins()
			for _, coin := range minDepositFloor {
				minInitialDepositFloor = append(minInitialDepositFloor, sdk.NewCoin(coin.Denom, ratio.MulInt(coin.Amount).TruncateInt()))
			}
		},
	)

	var minInitialDepositUpdatePeriod time.Duration
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinInitialDepositUpdatePeriod, &minInitialDepositUpdatePeriod, simState.Rand,
		func(r *rand.Rand) {
			minInitialDepositUpdatePeriod = GenDepositParamsMinInitialDepositUpdatePeriod(r, depositPeriod)
		},
	)

	var minInitialDepositSensitivityTargetDistance uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinInitialDepositSensitivityTargetDistance, &minInitialDepositSensitivityTargetDistance, simState.Rand,
		func(r *rand.Rand) {
			minInitialDepositSensitivityTargetDistance = GenDepositParamsMinInitialDepositSensitivityTargetDistance(r)
		},
	)

	var minInitialDepositIncreaseRatio sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinInitialDepositIncreaseRatio, &minInitialDepositIncreaseRatio, simState.Rand,
		func(r *rand.Rand) {
			minInitialDepositIncreaseRatio = GenDepositParamsMinInitialDepositChangeRatio(r, 300, 3)
		},
	)

	var minInitialDepositDecreaseRatio sdk.Dec
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinInitialDepositDecreaseRatio, &minInitialDepositDecreaseRatio, simState.Rand,
		func(r *rand.Rand) {
			minInitialDepositDecreaseRatio = GenDepositParamsMinInitialDepositChangeRatio(r,
				int(minInitialDepositIncreaseRatio.MulInt64(1000).QuoInt64(2).TruncateInt64()), 3)
		},
	)

	var minInitialDepositTargetProposals uint64
	simState.AppParams.GetOrGenerate(
		simState.Cdc, DepositParamsMinInitialDepositTargetProposals, &minInitialDepositTargetProposals, simState.Rand,
		func(r *rand.Rand) {
			minInitialDepositTargetProposals = GenDepositParamsMinInitialDepositTargetProposals(r)
		},
	)

	govGenesis := v1.NewGenesisState(
		startingProposalID,
		v1.NewParams(depositPeriod, votingPeriod, quorum.String(), threshold.String(), amendmentsQuorum.String(),
			amendmentsThreshold.String(), lawQuorum.String(), lawThreshold.String(), // minInitialDepositRatio.String(),
			simState.Rand.Intn(2) == 0, simState.Rand.Intn(2) == 0, minDepositRatio.String(), quorumTimout,
			maxVotingPeriodExtension, quorumCheckCount, minDepositFloor, minDepositUpdatePeriod,
			minDepositSensitivityTargetDistance, minDepositIncreaseRatio.String(), minDepositDecreaseRatio.String(),
			targetActiveProposals, minInitialDepositFloor, minInitialDepositUpdatePeriod,
			minInitialDepositSensitivityTargetDistance, minInitialDepositIncreaseRatio.String(),
			minInitialDepositDecreaseRatio.String(), minInitialDepositTargetProposals),
	)

	bz, err := json.MarshalIndent(&govGenesis, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated governance parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(govGenesis)
}
