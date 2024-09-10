package helpers

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	atomoneapp "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/baseapp"
	codectypes "github.com/atomone-hub/atomone/codec/types"
	cryptocodec "github.com/atomone-hub/atomone/crypto/codec"
	cryptotypes "github.com/atomone-hub/atomone/crypto/types"
	"github.com/atomone-hub/atomone/server"
	"github.com/atomone-hub/atomone/testutil/mock"
	simtestutil "github.com/atomone-hub/atomone/testutil/sims"
	sdk "github.com/atomone-hub/atomone/types"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	banktypes "github.com/atomone-hub/atomone/x/bank/types"
	stakingtypes "github.com/atomone-hub/atomone/x/staking/types"
)

var ParamStoreKey = []byte("paramstore")

// SimAppChainID hardcoded chainID for simulation
const (
	SimAppChainID = "atomone-app"
)

// DefaultConsensusParams defines the default Tendermint consensus params used
// in AtomOneApp testing.
var DefaultConsensusParams = &tmproto.ConsensusParams{
	Block: &tmproto.BlockParams{
		MaxBytes: 200000,
		MaxGas:   2000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

type PV struct {
	PrivKey cryptotypes.PrivKey
}

type EmptyAppOptions struct{}

func (EmptyAppOptions) Get(_ string) interface{} { return nil }

func Setup(t *testing.T) *atomoneapp.AtomOneApp {
	t.Helper()

	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	require.NoError(t, err)
	// create validator set with single validator
	validator := tmtypes.NewValidator(pubKey, 1)
	valSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := mock.NewPV()
	senderPubKey := senderPrivKey.PrivKey.PubKey()

	acc := authtypes.NewBaseAccount(senderPubKey.Address().Bytes(), senderPubKey, 0, 0)
	balance := banktypes.Balance{
		Address: acc.GetAddress().String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100000000000000))),
	}
	genesisAccounts := []authtypes.GenesisAccount{acc}
	app := SetupWithGenesisValSet(t, valSet, genesisAccounts, balance)

	return app
}

type paramStore struct {
	db *dbm.MemDB
}

func (ps *paramStore) Set(_ sdk.Context, value *tmproto.ConsensusParams) {
	bz, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	if err := ps.db.Set(ParamStoreKey, bz); err != nil {
		panic(err)
	}
}

func (ps *paramStore) Has(_ sdk.Context) bool {
	ok, err := ps.db.Has(ParamStoreKey)
	if err != nil {
		panic(err)
	}

	return ok
}

func (ps paramStore) Get(ctx sdk.Context) (*tmproto.ConsensusParams, error) {
	bz, err := ps.db.Get(ParamStoreKey)
	if err != nil {
		panic(err)
	}

	if len(bz) == 0 {
		return nil, errors.New("params not found")
	}

	var params tmproto.ConsensusParams
	if err := json.Unmarshal(bz, &params); err != nil {
		panic(err)
	}

	return &params, nil
}

// SetupWithGenesisValSet initializes a new AtomOneApp with a validator set and genesis accounts
// that also act as delegators. For simplicity, each validator is bonded with a delegation
// of one consensus engine unit in the default token of the AtomOneApp from first genesis
// account. A Nop logger is set in AtomOneApp.
func SetupWithGenesisValSet(t *testing.T, valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) *atomoneapp.AtomOneApp {
	t.Helper()

	atomoneApp, genesisState := setup()
	genesisState = genesisStateWithValSet(t, atomoneApp, genesisState, valSet, genAccs, balances...)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	// init chain will set the validator set and initialize the genesis accounts
	atomoneApp.InitChain(
		abci.RequestInitChain{
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	// commit genesis changes
	atomoneApp.Commit()
	atomoneApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		Height:             atomoneApp.LastBlockHeight() + 1,
		AppHash:            atomoneApp.LastCommitID().Hash,
		ValidatorsHash:     valSet.Hash(),
		NextValidatorsHash: valSet.Hash(),
	}})

	return atomoneApp
}

func setParamStore(pStore *paramStore) func(*baseapp.BaseApp) {
	return func(app *baseapp.BaseApp) { app.SetParamStore(pStore) }
}

func setup() (*atomoneapp.AtomOneApp, atomoneapp.GenesisState) {
	db := dbm.NewMemDB()
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[server.FlagInvCheckPeriod] = 5
	appOptions[server.FlagMinGasPrices] = "0uatone"

	encConfig := atomoneapp.RegisterEncodingConfig()

	baseappOptions := []func(*baseapp.BaseApp){
		setParamStore(&paramStore{db: dbm.NewMemDB()}),
	}
	atomoneApp := atomoneapp.NewAtomOneApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		atomoneapp.DefaultNodeHome,
		encConfig,
		appOptions,
		baseappOptions...,
	)
	return atomoneApp, atomoneapp.NewDefaultGenesisState(encConfig)
}

func genesisStateWithValSet(t *testing.T,
	app *atomoneapp.AtomOneApp, genesisState atomoneapp.GenesisState,
	valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount,
	balances ...banktypes.Balance,
) atomoneapp.GenesisState {
	t.Helper()
	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.DefaultPowerReduction

	for _, val := range valSet.Validators {
		pk, err := cryptocodec.FromTmPubKeyInterface(val.PubKey)
		require.NoError(t, err)
		pkAny, err := codectypes.NewAnyWithValue(pk)
		require.NoError(t, err)
		validator := stakingtypes.Validator{
			OperatorAddress: sdk.ValAddress(val.Address).String(),
			ConsensusPubkey: pkAny,
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          bondAmt,
			DelegatorShares: sdk.OneDec(),
			Description:     stakingtypes.Description{},
			UnbondingHeight: int64(0),
			UnbondingTime:   time.Unix(0, 0).UTC(),
			Commission:      stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), sdk.OneDec()))

	}
	// set validators and delegations
	stakingGenesis := stakingtypes.NewGenesisState(stakingtypes.DefaultParams(), validators, delegations)
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(stakingGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens to total supply
		totalSupply = totalSupply.Add(b.Coins...)
	}

	for range delegations {
		// add delegated tokens to total supply
		totalSupply = totalSupply.Add(sdk.NewCoin(sdk.DefaultBondDenom, bondAmt))
	}

	// add bonded amount to bonded pool module account
	balances = append(balances, banktypes.Balance{
		Address: authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String(),
		Coins:   sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, bondAmt)},
	})

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{}, []banktypes.SendEnabled{})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	return genesisState
}
