package v4

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	collcodec "cosmossdk.io/collections/codec"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	sdkgov "github.com/cosmos/cosmos-sdk/x/gov/types"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/atomone-hub/atomone/app/keepers"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

// CreateUpgradeHandler returns a upgrade handler for AtomOne v4
// This versions contains the upgrade to Cosmos SDK v0.50 and IBC v10
func CreateUpgradeHandler(
	mm *module.Manager,
	cdc codec.Codec,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		storeService := runtime.NewKVStoreService(keepers.GetKey(sdkgov.StoreKey))
		sb := collections.NewSchemaBuilder(storeService)
		if err := migrateGovState(ctx, cdc, keepers.GovKeeper, sb); err != nil {
			return vm, err
		}

		return vm, nil
	}
}

// migrateGovState migrates all gov state from atomone.gov.v1 proto types to cosmos.gov.v1 proto types
// and initializes new params fields with default values.
func migrateGovState(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	var errs error

	if err := migrateParams(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov params: %w", err))
	}

	if err := migrateDeposits(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov deposits: %w", err))
	}

	if err := migrateVotes(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov votes: %w", err))
	}

	if err := migrateProposals(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov proposals: %w", err))
	}

	if err := migrateLastMinDeposit(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov last min deposit: %w", err))
	}

	if err := migrateLastMinInitialDeposit(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov last min initial deposit: %w", err))
	}

	if err := migrateQuorumCheckQueue(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov quorum check queue: %w", err))
	}

	if err := migrateGovernors(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov governors: %w", err))
	}

	if err := migrateGovernanceDelegations(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov governance delegations: %w", err))
	}

	if err := migrateGovernanceDelegationsByGovernor(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov governance delegations by governor: %w", err))
	}

	if err := migrateValidatorSharesByGovernor(ctx, cdc, govKeeper, sb); err != nil {
		errs = errors.Join(errs, fmt.Errorf("failed to migrate gov validator shares by governor: %w", err))
	}

	return errs
}

// migrateParams migrates Params from atomone.gov.v1 to cosmos.gov.v1 and sets new default values.
func migrateParams(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	paramsItem := collections.NewItem(sb, sdkgov.ParamsKey, "params", paramsValueCodec(cdc))

	params, err := paramsItem.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get params: %w", err)
	}

	// Set new params fields to default values
	defaultParams := sdkgovv1.DefaultParams()
	params.ProposalCancelRatio = defaultParams.ProposalCancelRatio
	params.ProposalCancelDest = authtypes.NewModuleAddress(sdkgov.ModuleName).String()
	params.MinDepositRatio = defaultParams.MinDepositRatio
	params.GovernorStatusChangePeriod = defaultParams.GovernorStatusChangePeriod
	params.MinGovernorSelfDelegation = math.NewInt(10000_000000).String() // to be eligible as governor must have 10K ATONE staked

	if err := govKeeper.Params.Set(ctx, params); err != nil {
		return fmt.Errorf("failed to set gov params: %w", err)
	}

	return nil
}

// migrateDeposits migrates Deposits from atomone.gov.v1 to cosmos.gov.v1.
func migrateDeposits(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	depositsMap := collections.NewMap(
		sb,
		sdkgov.DepositsKeyPrefix,
		"deposits",
		collections.PairKeyCodec(collections.Uint64Key, sdk.LengthPrefixedAddressKey(sdk.AccAddressKey)), //nolint: staticcheck
		depositValueCodec(cdc),
	)

	iter, err := depositsMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate deposits: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get deposit key: %w", err)
		}
		deposit, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get deposit value: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.Deposits.Set(ctx, key, deposit); err != nil {
			return fmt.Errorf("failed to set deposit: %w", err)
		}
	}

	return nil
}

// migrateVotes migrates Votes from atomone.gov.v1 to cosmos.gov.v1.
func migrateVotes(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	votesMap := collections.NewMap(
		sb,
		sdkgov.VotesKeyPrefix,
		"votes",
		collections.PairKeyCodec(collections.Uint64Key, sdk.LengthPrefixedAddressKey(sdk.AccAddressKey)), //nolint: staticcheck
		voteValueCodec(cdc),
	)

	iter, err := votesMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate votes: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get vote key: %w", err)
		}
		vote, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get vote value: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.Votes.Set(ctx, key, vote); err != nil {
			return fmt.Errorf("failed to set vote: %w", err)
		}
	}

	return nil
}

// migrateProposals migrates Proposals from atomone.gov.v1 to cosmos.gov.v1.
func migrateProposals(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	proposalsMap := collections.NewMap(
		sb,
		sdkgov.ProposalsKeyPrefix,
		"proposals",
		collections.Uint64Key,
		proposalValueCodec(cdc),
	)

	iter, err := proposalsMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate proposals: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get proposal key: %w", err)
		}
		proposal, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get proposal value: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.Proposals.Set(ctx, key, proposal); err != nil {
			return fmt.Errorf("failed to set proposal: %w", err)
		}
	}

	return nil
}

// migrateLastMinDeposit migrates LastMinDeposit from atomone.gov.v1 to cosmos.gov.v1.
func migrateLastMinDeposit(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	lastMinDepositItem := collections.NewItem(
		sb,
		sdkgov.LastMinDepositKey,
		"last_min_deposit",
		lastMinDepositValueCodec(cdc),
	)

	lastMinDeposit, err := lastMinDepositItem.Get(ctx)
	if err != nil {
		// If not set, skip migration
		if err.Error() == "collections: not found" {
			return nil
		}
		return fmt.Errorf("failed to get last min deposit: %w", err)
	}

	// Write back with new SDK type
	if err := govKeeper.LastMinDeposit.Set(ctx, lastMinDeposit); err != nil {
		return fmt.Errorf("failed to set last min deposit: %w", err)
	}

	return nil
}

// migrateLastMinInitialDeposit migrates LastMinInitialDeposit from atomone.gov.v1 to cosmos.gov.v1.
func migrateLastMinInitialDeposit(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	lastMinInitialDepositItem := collections.NewItem(
		sb,
		sdkgov.LastMinInitialDepositKey,
		"last_min_initial_deposit",
		lastMinDepositValueCodec(cdc),
	)

	lastMinInitialDeposit, err := lastMinInitialDepositItem.Get(ctx)
	if err != nil {
		// If not set, skip migration
		if err.Error() == "collections: not found" {
			return nil
		}
		return fmt.Errorf("failed to get last min initial deposit: %w", err)
	}

	// Write back with new SDK type
	if err := govKeeper.LastMinInitialDeposit.Set(ctx, lastMinInitialDeposit); err != nil {
		return fmt.Errorf("failed to set last min initial deposit: %w", err)
	}

	return nil
}

// migrateQuorumCheckQueue migrates QuorumCheckQueue from atomone.gov.v1 to cosmos.gov.v1.
func migrateQuorumCheckQueue(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	quorumCheckQueueMap := collections.NewMap(
		sb,
		sdkgov.QuorumCheckQueuePrefix,
		"quorum_check_queue",
		collections.PairKeyCodec(sdk.TimeKey, collections.Uint64Key),
		quorumCheckQueueEntryValueCodec(cdc),
	)

	iter, err := quorumCheckQueueMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate quorum check queue: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get quorum check queue key: %w", err)
		}
		entry, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get quorum check queue entry: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.QuorumCheckQueue.Set(ctx, key, entry); err != nil {
			return fmt.Errorf("failed to set quorum check queue entry: %w", err)
		}
	}

	return nil
}

// migrateGovernors migrates Governors from atomone.gov.v1 to cosmos.gov.v1.
func migrateGovernors(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	governorsMap := collections.NewMap(
		sb,
		sdkgov.GovernorsKeyPrefix,
		"governors",
		sdkgov.GovernorAddressKey,
		governorValueCodec(cdc),
	)

	iter, err := governorsMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate governors: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get governor key: %w", err)
		}
		governor, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get governor value: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.Governors.Set(ctx, key, governor); err != nil {
			return fmt.Errorf("failed to set governor: %w", err)
		}
	}

	return nil
}

// migrateGovernanceDelegations migrates GovernanceDelegations from atomone.gov.v1 to cosmos.gov.v1.
func migrateGovernanceDelegations(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	governanceDelegationsMap := collections.NewMap(
		sb,
		sdkgov.GovernanceDelegationKeyPrefix,
		"governance_delegations",
		sdk.AccAddressKey,
		governanceDelegationValueCodec(cdc),
	)

	iter, err := governanceDelegationsMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate governance delegations: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get governance delegation key: %w", err)
		}
		delegation, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get governance delegation value: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.GovernanceDelegations.Set(ctx, key, delegation); err != nil {
			return fmt.Errorf("failed to set governance delegation: %w", err)
		}
	}

	return nil
}

// migrateGovernanceDelegationsByGovernor migrates GovernanceDelegationsByGovernor from atomone.gov.v1 to cosmos.gov.v1.
func migrateGovernanceDelegationsByGovernor(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	governanceDelegationsByGovernorMap := collections.NewMap(
		sb,
		sdkgov.GovernanceDelegationsByGovernorKeyPrefix,
		"governance_delegations_by_governor",
		collections.PairKeyCodec(sdkgov.GovernorAddressKey, sdk.AccAddressKey),
		governanceDelegationValueCodec(cdc),
	)

	iter, err := governanceDelegationsByGovernorMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate governance delegations by governor: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get governance delegation by governor key: %w", err)
		}
		delegation, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get governance delegation by governor value: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.GovernanceDelegationsByGovernor.Set(ctx, key, delegation); err != nil {
			return fmt.Errorf("failed to set governance delegation by governor: %w", err)
		}
	}

	return nil
}

// migrateValidatorSharesByGovernor migrates ValidatorSharesByGovernor from atomone.gov.v1 to cosmos.gov.v1.
func migrateValidatorSharesByGovernor(ctx context.Context, cdc codec.Codec, govKeeper *govkeeper.Keeper, sb *collections.SchemaBuilder) error {
	validatorSharesByGovernorMap := collections.NewMap(
		sb,
		sdkgov.ValidatorSharesByGovernorKeyPrefix,
		"validator_shares_by_governor",
		collections.PairKeyCodec(sdkgov.GovernorAddressKey, sdk.ValAddressKey),
		governorValSharesValueCodec(cdc),
	)

	iter, err := validatorSharesByGovernorMap.Iterate(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to iterate validator shares by governor: %w", err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, err := iter.Key()
		if err != nil {
			return fmt.Errorf("failed to get validator shares by governor key: %w", err)
		}
		valShares, err := iter.Value()
		if err != nil {
			return fmt.Errorf("failed to get validator shares by governor value: %w", err)
		}

		// Write back with new SDK type
		if err := govKeeper.ValidatorSharesByGovernor.Set(ctx, key, valShares); err != nil {
			return fmt.Errorf("failed to set validator shares by governor: %w", err)
		}
	}

	return nil
}

// Value codecs that decode atomone.gov.v1 types and convert to cosmos.gov.v1 types

// paramsValueCodec is a codec for encoding params in a backward compatible way
func paramsValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.Params] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.Params](cdc), func(bytes []byte) (sdkgovv1.Params, error) {
		c := new(v1.Params)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.Params{}, err
		}

		return *v1.ConvertAtomOneParamsToSDK(c), nil
	})
}

// depositValueCodec is a codec for encoding deposits in a backward compatible way
func depositValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.Deposit] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.Deposit](cdc), func(bytes []byte) (sdkgovv1.Deposit, error) {
		c := new(v1.Deposit)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.Deposit{}, err
		}

		return *v1.ConvertAtomOneDepositToSDK(c), nil
	})
}

// voteValueCodec is a codec for encoding votes in a backward compatible way
func voteValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.Vote] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.Vote](cdc), func(bytes []byte) (sdkgovv1.Vote, error) {
		c := new(v1.Vote)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.Vote{}, err
		}

		return *v1.ConvertAtomOneVoteToSDK(c), nil
	})
}

// proposalValueCodec is a codec for encoding proposals in a backward compatible way
func proposalValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.Proposal] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.Proposal](cdc), func(bytes []byte) (sdkgovv1.Proposal, error) {
		c := new(v1.Proposal)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.Proposal{}, err
		}

		return *v1.ConvertAtomOneProposalToSDK(c), nil
	})
}

// lastMinDepositValueCodec is a codec for encoding last min deposits in a backward compatible way
func lastMinDepositValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.LastMinDeposit] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.LastMinDeposit](cdc), func(bytes []byte) (sdkgovv1.LastMinDeposit, error) {
		c := new(v1.LastMinDeposit)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.LastMinDeposit{}, err
		}

		return *v1.ConvertAtomOneLastMinDepositToSDK(c), nil
	})
}

// quorumCheckQueueEntryValueCodec is a codec for encoding quorum check queue entries in a backward compatible way
func quorumCheckQueueEntryValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.QuorumCheckQueueEntry] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.QuorumCheckQueueEntry](cdc), func(bytes []byte) (sdkgovv1.QuorumCheckQueueEntry, error) {
		c := new(v1.QuorumCheckQueueEntry)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.QuorumCheckQueueEntry{}, err
		}

		return *v1.ConvertAtomOneQuorumCheckQueueEntryToSDK(c), nil
	})
}

// governorValueCodec is a codec for encoding governors in a backward compatible way
func governorValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.Governor] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.Governor](cdc), func(bytes []byte) (sdkgovv1.Governor, error) {
		c := new(v1.Governor)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.Governor{}, err
		}

		return *v1.ConvertAtomOneGovernorToSDK(c), nil
	})
}

// governanceDelegationValueCodec is a codec for encoding governance delegations in a backward compatible way
func governanceDelegationValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.GovernanceDelegation] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.GovernanceDelegation](cdc), func(bytes []byte) (sdkgovv1.GovernanceDelegation, error) {
		c := new(v1.GovernanceDelegation)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.GovernanceDelegation{}, err
		}

		return *v1.ConvertAtomOneGovernanceDelegationToSDK(c), nil
	})
}

// governorValSharesValueCodec is a codec for encoding governor val shares in a backward compatible way
func governorValSharesValueCodec(cdc codec.Codec) collcodec.ValueCodec[sdkgovv1.GovernorValShares] {
	return collcodec.NewAltValueCodec(codec.CollValue[sdkgovv1.GovernorValShares](cdc), func(bytes []byte) (sdkgovv1.GovernorValShares, error) {
		c := new(v1.GovernorValShares)
		err := cdc.Unmarshal(bytes, c)
		if err != nil {
			return sdkgovv1.GovernorValShares{}, err
		}

		return *v1.ConvertAtomOneGovernorValSharesToSDK(c), nil
	})
}
