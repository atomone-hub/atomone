package keeper

// DONTCOVER

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// RegisterInvariants registers all governance invariants
func RegisterInvariants(ir sdk.InvariantRegistry, keeper *Keeper, bk types.BankKeeper) {
	ir.RegisterRoute(types.ModuleName, "module-account", ModuleAccountInvariant(keeper, bk))
	ir.RegisterRoute(types.ModuleName, "governors-voting-power", GovernorsVotingPowerInvariant(keeper, keeper.sk))
	ir.RegisterRoute(types.ModuleName, "governors-delegations", GovernorsDelegationsInvariant(keeper, keeper.sk))
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
					invariantStr = sdk.FormatInvariant(types.ModuleName, fmt.Sprintf("governor %s voting power", governor.GetAddress().String()),
						fmt.Sprintf("failed to parse validator address %s: %v", shares.ValidatorAddress, err))
					fail = true
					return true
				}
				validator, found := sk.GetValidator(ctx, validatorAddr)
				if !found {
					invariantStr = sdk.FormatInvariant(types.ModuleName, fmt.Sprintf("governor %s voting power", governor.GetAddress().String()),
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
			invariantStr = sdk.FormatInvariant(types.ModuleName, fmt.Sprintf("governor %s voting power", brokenGovernorAddr),
				fmt.Sprintf("\texpected voting power: %s\n\tactual voting power: %s\n", expectedVotingPower, actualVotingPower))
		}
		return invariantStr, broken
	}
}

// GovernorsDelegationsInvariant checks that the validator shares resulting from all
// governor delegations actually correspond to the stored governor validator shares
func GovernorsDelegationsInvariant(keeper *Keeper, sk types.StakingKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken       = false
			invariantStr string
		)

		keeper.IterateGovernors(ctx, func(index int64, governor v1.GovernorI) bool {
			// check that if governor is active, it has a valid governance self-delegation
			if governor.IsActive() {
				if del, ok := keeper.GetGovernanceDelegation(ctx, sdk.AccAddress(governor.GetAddress())); !ok || !governor.GetAddress().Equals(types.MustGovernorAddressFromBech32(del.GovernorAddress)) {
					invariantStr = sdk.FormatInvariant(types.ModuleName, fmt.Sprintf("governor %s delegations", governor.GetAddress().String()),
						"active governor without governance self-delegation")
					broken = true
					return true
				}
			}

			valShares := make(map[string]sdk.Dec)
			keeper.IterateGovernorDelegations(ctx, governor.GetAddress(), func(index int64, delegation v1.GovernanceDelegation) bool {
				delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)
				keeper.sk.IterateDelegations(ctx, delAddr, func(_ int64, delegation stakingtypes.DelegationI) (stop bool) {
					validatorAddr := delegation.GetValidatorAddr()
					shares := delegation.GetShares()
					if _, ok := valShares[validatorAddr.String()]; !ok {
						valShares[validatorAddr.String()] = sdk.ZeroDec()
					}
					valShares[validatorAddr.String()] = valShares[validatorAddr.String()].Add(shares)
					return false
				})
				return false
			})

			for valAddrStr, shares := range valShares {
				validatorAddr, _ := sdk.ValAddressFromBech32(valAddrStr)
				valShares, ok := keeper.GetGovernorValShares(ctx, governor.GetAddress(), validatorAddr)
				if !ok {
					invariantStr = sdk.FormatInvariant(types.ModuleName, fmt.Sprintf("governor %s delegations", governor.GetAddress().String()),
						fmt.Sprintf("validator %s shares not found", valAddrStr))
					broken = true
					return true
				}
				if !valShares.Shares.Equal(shares) {
					invariantStr = sdk.FormatInvariant(types.ModuleName, fmt.Sprintf("governor %s delegations", governor.GetAddress().String()),
						fmt.Sprintf("stored shares %s for validator %s do not match actual shares %s", valShares.Shares, valAddrStr, shares))
					broken = true
					return true
				}
			}

			keeper.IterateGovernorValShares(ctx, governor.GetAddress(), func(index int64, shares v1.GovernorValShares) bool {
				if _, ok := valShares[shares.ValidatorAddress]; !ok && shares.Shares.GT(sdk.ZeroDec()) {
					invariantStr = sdk.FormatInvariant(types.ModuleName, fmt.Sprintf("governor %s delegations", governor.GetAddress().String()),
						fmt.Sprintf("non-zero (%s) shares stored for validator %s where there should be none", shares.Shares, shares.ValidatorAddress))
					broken = true
					return true
				}
				return false
			})

			return broken
		})
		return invariantStr, broken
	}
}
