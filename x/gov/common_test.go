package gov_test

import (
	"bytes"
	"log"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/atomone-hub/atomone/x/auth"
	_ "github.com/atomone-hub/atomone/x/bank"
	_ "github.com/atomone-hub/atomone/x/consensus"
	_ "github.com/atomone-hub/atomone/x/params"
	_ "github.com/atomone-hub/atomone/x/staking"

	"cosmossdk.io/math"

	"github.com/atomone-hub/atomone/crypto/keys/ed25519"
	cryptotypes "github.com/atomone-hub/atomone/crypto/types"
	"github.com/atomone-hub/atomone/runtime"
	"github.com/atomone-hub/atomone/testutil/configurator"
	simtestutil "github.com/atomone-hub/atomone/testutil/sims"
	sdk "github.com/atomone-hub/atomone/types"
	authkeeper "github.com/atomone-hub/atomone/x/auth/keeper"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	bankkeeper "github.com/atomone-hub/atomone/x/bank/keeper"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	stakingkeeper "github.com/atomone-hub/atomone/x/staking/keeper"
	stakingtypes "github.com/atomone-hub/atomone/x/staking/types"

	"github.com/atomone-hub/atomone/x/gov/keeper"
	"github.com/atomone-hub/atomone/x/gov/types"
	v1 "github.com/atomone-hub/atomone/x/gov/types/v1"
	"github.com/atomone-hub/atomone/x/gov/types/v1beta1"
)

var (
	valTokens           = sdk.TokensFromConsensusPower(1000000, sdk.DefaultPowerReduction)
	TestProposal        = v1beta1.NewTextProposal("Test", "description")
	TestDescription     = stakingtypes.NewDescription("T", "E", "S", "T", "Z")
	TestCommissionRates = stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())
)

// mkTestLegacyContent creates a MsgExecLegacyContent for testing purposes.
func mkTestLegacyContent(t *testing.T) *v1.MsgExecLegacyContent {
	msgContent, err := v1.NewLegacyContent(TestProposal, authtypes.NewModuleAddress(types.ModuleName).String())
	require.NoError(t, err)

	return msgContent
}

// SortAddresses - Sorts Addresses
func SortAddresses(addrs []sdk.AccAddress) {
	byteAddrs := make([][]byte, len(addrs))

	for i, addr := range addrs {
		byteAddrs[i] = addr.Bytes()
	}

	SortByteArrays(byteAddrs)

	for i, byteAddr := range byteAddrs {
		addrs[i] = byteAddr
	}
}

// implement `Interface` in sort package.
type sortByteArrays [][]byte

func (b sortByteArrays) Len() int {
	return len(b)
}

func (b sortByteArrays) Less(i, j int) bool {
	// bytes package already implements Comparable for []byte.
	switch bytes.Compare(b[i], b[j]) {
	case -1:
		return true
	case 0, 1:
		return false
	default:
		log.Panic("not fail-able with `bytes.Comparable` bounded [-1, 1].")
		return false
	}
}

func (b sortByteArrays) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}

// SortByteArrays - sorts the provided byte array
func SortByteArrays(src [][]byte) [][]byte {
	sorted := sortByteArrays(src)
	sort.Sort(sorted)
	return sorted
}

var pubkeys = []cryptotypes.PubKey{
	ed25519.GenPrivKey().PubKey(),
	ed25519.GenPrivKey().PubKey(),
	ed25519.GenPrivKey().PubKey(),
}

type suite struct {
	AccountKeeper authkeeper.AccountKeeper
	BankKeeper    bankkeeper.Keeper
	GovKeeper     *keeper.Keeper
	StakingKeeper *stakingkeeper.Keeper
	App           *runtime.App
}

func createTestSuite(t *testing.T) suite {
	res := suite{}

	app, err := simtestutil.SetupWithConfiguration(
		configurator.NewAppConfig(
			configurator.ParamsModule(),
			configurator.AuthModule(),
			configurator.StakingModule(),
			configurator.BankModule(),
			configurator.GovModule(),
			configurator.ConsensusModule(),
		),
		simtestutil.DefaultStartUpConfig(),
		&res.AccountKeeper, &res.BankKeeper, &res.GovKeeper, &res.StakingKeeper,
	)
	require.NoError(t, err)

	res.App = app
	return res
}
