package keepers

import (
	storetypes "github.com/atomone-hub/atomone/store/types"
	sdk "github.com/atomone-hub/atomone/types"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	authzkeeper "github.com/atomone-hub/atomone/x/authz/keeper"
	banktypes "github.com/atomone-hub/atomone/x/bank/types"
	capabilitytypes "github.com/atomone-hub/atomone/x/capability/types"
	consensusparamtypes "github.com/atomone-hub/atomone/x/consensus/types"
	crisistypes "github.com/atomone-hub/atomone/x/crisis/types"
	distrtypes "github.com/atomone-hub/atomone/x/distribution/types"
	evidencetypes "github.com/atomone-hub/atomone/x/evidence/types"
	"github.com/atomone-hub/atomone/x/feegrant"
	govtypes "github.com/atomone-hub/atomone/x/gov/types"
	minttypes "github.com/atomone-hub/atomone/x/mint/types"
	paramstypes "github.com/atomone-hub/atomone/x/params/types"
	slashingtypes "github.com/atomone-hub/atomone/x/slashing/types"
	stakingtypes "github.com/atomone-hub/atomone/x/staking/types"
	upgradetypes "github.com/atomone-hub/atomone/x/upgrade/types"
)

func (appKeepers *AppKeepers) GenerateKeys() {
	// Define what keys will be used in the cosmos-sdk key/value store.
	// Cosmos-SDK modules each have a "key" that allows the application to reference what they've stored on the chain.
	appKeepers.keys = sdk.NewKVStoreKeys(
		authtypes.StoreKey,
		authzkeeper.StoreKey,
		banktypes.StoreKey,
		capabilitytypes.StoreKey,
		consensusparamtypes.StoreKey,
		crisistypes.StoreKey,
		distrtypes.StoreKey,
		evidencetypes.StoreKey,
		feegrant.StoreKey,
		govtypes.StoreKey,
		minttypes.StoreKey,
		paramstypes.StoreKey,
		stakingtypes.StoreKey,
		slashingtypes.StoreKey,
		upgradetypes.StoreKey,
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
