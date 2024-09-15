package keeper

// DONTCOVER

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// RegisterInvariants registers all governance invariants
func RegisterInvariants(ir sdk.InvariantRegistry, keeper *Keeper, bk types.BankKeeper) {
	ir.RegisterRoute(types.ModuleName, "module-account", ModuleAccountInvariant(keeper, bk))
	ir.RegisterRoute(types.ModuleName, "governors-voting-power", GovernorsVotingPowerInvariant(keeper, keeper.sk))
}

// AllInvariants runs all invariants of the governance module
func AllInvariants(keeper *Keeper, bk types.BankKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		return ModuleAccountInvariant(keeper, bk)(ctx)
	}
}

// ModuleAccountInvariant checks that the module account coins reflects the sum of
// deposit amounts held on store.
func ModuleAccountInvariant(keeper *Keeper, bk types.BankKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var expectedDeposits sdk.Coins

		keeper.IterateAllDeposits(ctx, func(deposit v1.Deposit) bool {
			expectedDeposits = expectedDeposits.Add(deposit.Amount...)
			return false
		})

		macc := keeper.GetGovernanceAccount(ctx)
		balances := bk.GetAllBalances(ctx, macc.GetAddress())

		// Require that the deposit balances are <= than the x/gov module's total
		// balances. We use the <= operator since external funds can be sent to x/gov
		// module's account and so the balance can be larger.
		broken := !balances.IsAllGTE(expectedDeposits)

		return sdk.FormatInvariant(types.ModuleName, "deposits",
			fmt.Sprintf("\tgov ModuleAccount coins: %s\n\tsum of deposit amounts:  %s\n",
				balances, expectedDeposits)), broken
	}
}

// GovernorsVotingPowerInvariant checks that the voting power of all governors
// is actually equal to the voting power resulting from the delegated validator shares
func GovernorsVotingPowerInvariant(keeper *Keeper, sk types.StakingKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			expectedVotingPower sdk.Dec
			actualVotingPower   sdk.Dec
			broken              bool
			brokenGovernorAddr  string
			fail                bool
			invariantStr        string
		)

		keeper.IterateGovernors(ctx, func(index int64, governor v1.GovernorI) bool {
			expectedVotingPower = governor.GetVotingPower()
			actualVotingPower = sdk.ZeroDec()
			fail = false
			keeper.IterateGovernorValShares(ctx, governor.GetAddress(), func(index int64, shares v1.GovernorValShares) bool {
				validatorAddr, err := sdk.ValAddressFromBech32(shares.ValidatorAddress)
				if err != nil {
					invariantStr = sdk.FormatInvariant(types.ModuleName, "governor %s voting power",
						fmt.Sprintf("failed to parse validator address %s: %v", shares.ValidatorAddress, err))
					fail = true
					return true
				}
				validator, found := sk.GetValidator(ctx, validatorAddr)
				if !found {
					invariantStr = sdk.FormatInvariant(types.ModuleName, "governor %s voting power",
						fmt.Sprintf("validator %s not found", validatorAddr.String()))
					fail = true
					return true
				}
				vp := shares.Shares.MulInt(validator.GetBondedTokens()).Quo(validator.GetDelegatorShares())
				actualVotingPower = actualVotingPower.Add(vp)
				return false
			})
			broken = !expectedVotingPower.Equal(actualVotingPower)
			if fail {
				broken = true
			}
			if broken {
				brokenGovernorAddr = governor.GetAddress().String()
			}
			return broken // break on first broken invariant
		})
		if !fail {
			invariantStr = sdk.FormatInvariant(types.ModuleName, "governor %s voting power",
				fmt.Sprintf("\texpected %s voting power: %s\n\tactual voting power: %s\n",
					brokenGovernorAddr, expectedVotingPower, actualVotingPower))
		}
		return invariantStr, broken
	}
}
