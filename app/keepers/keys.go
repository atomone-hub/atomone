package keepers

import (
	sdk "github.com/atomone-hub/atomone/types"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	authzkeeper "github.com/atomone-hub/atomone/x/authz/keeper"
	banktypes "github.com/atomone-hub/atomone/x/bank/types"
	consensusparamtypes "github.com/atomone-hub/atomone/x/consensus/types"
	crisistypes "github.com/atomone-hub/atomone/x/crisis/types"
	distrtypes "github.com/atomone-hub/atomone/x/distribution/types"
	evidencetypes "github.com/atomone-hub/atomone/x/evidence/types"
	"github.com/atomone-hub/atomone/x/feegrant"
	minttypes "github.com/atomone-hub/atomone/x/mint/types"
	paramstypes "github.com/atomone-hub/atomone/x/params/types"
	slashingtypes "github.com/atomone-hub/atomone/x/slashing/types"
	stakingtypes "github.com/atomone-hub/atomone/x/staking/types"
	upgradetypes "github.com/atomone-hub/atomone/x/upgrade/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	govtypes "github.com/atomone-hub/atomone/x/gov/types"
)

func (appKeepers *AppKeepers) GenerateKeys() {
	// Define what keys will be used in the cosmos-sdk key/value store.
	// Cosmos-SDK modules each have a "key" that allows the application to reference what they've stored on the chain.
	appKeepers.keys = sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		crisistypes.StoreKey,
		minttypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		upgradetypes.StoreKey,
		evidencetypes.StoreKey,
		capabilitytypes.StoreKey,
		feegrant.StoreKey,
		authzkeeper.StoreKey,
		consensusparamtypes.StoreKey,
	)

	// Define transient store keys
	appKeepers.tkeys = sdk.NewTransientStoreKeys(paramstypes.TStoreKey)

	// MemKeys are for information that is stored only in RAM.
	appKeepers.memKeys = sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
}

func (appKeepers *AppKeepers) GetKVStoreKey() map[string]*storetypes.KVStoreKey {
	return appKeepers.keys
}

func (appKeepers *AppKeepers) GetTransientStoreKey() map[string]*storetypes.TransientStoreKey {
	return appKeepers.tkeys
}

func (appKeepers *AppKeepers) GetMemoryStoreKey() map[string]*storetypes.MemoryStoreKey {
	return appKeepers.memKeys
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (appKeepers *AppKeepers) GetKey(storeKey string) *storetypes.KVStoreKey {
	return appKeepers.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (appKeepers *AppKeepers) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return appKeepers.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (appKeepers *AppKeepers) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return appKeepers.memKeys[storeKey]
}
