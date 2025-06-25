package e2e

import (
	"fmt"
	"os"

	"cosmossdk.io/log"
	atomone "github.com/atomone-hub/atomone/app"
	tmrand "github.com/cometbft/cometbft/libs/rand"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	paramsproptypes "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	atomoneparams "github.com/atomone-hub/atomone/app/params"
	distribtypes "github.com/atomone-hub/atomone/x/distribution/types"
	feemarkettypes "github.com/atomone-hub/atomone/x/feemarket/types"
	govv1types "github.com/atomone-hub/atomone/x/gov/types/v1"
	govv1beta1types "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

const (
	keyringPassphrase = "testpassphrase"
	keyringAppName    = "testnet"
)

type chain struct {
	dataDir    string
	id         string
	validators []*validator
	accounts   []*account //nolint:unused

	// initial accounts in genesis
	genesisAccounts        []*account
	genesisVestingAccounts map[string]sdk.AccAddress

	// codecs and chain config
	cdc      codec.Codec
	txConfig client.TxConfig
	bm       module.BasicManager
}

func newChain() (*chain, error) {
	tmpDir, err := os.MkdirTemp("", "atomone-e2e-testnet-")
	if err != nil {
		return nil, err
	}

	tempApp := atomone.NewAtomOneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, false, atomone.EmptyAppOptions{})

	return &chain{
		id:       "chain-" + tmrand.Str(6),
		dataDir:  tmpDir,
		cdc:      tempApp.AppCodec(),
		txConfig: tempApp.GetTxConfig(),
		bm:       tempApp.BasicModuleManager(),
	}, nil
}

func (c *chain) configDir() string {
	return fmt.Sprintf("%s/%s", c.dataDir, c.id)
}

func (c *chain) createAndInitValidators(count int) error {
	for i := 0; i < count; i++ {
		node := c.createValidator(i)

		// generate genesis files
		if err := node.init(); err != nil {
			return err
		}

		c.validators = append(c.validators, node)

		// create keys
		if err := node.createKey("val"); err != nil {
			return err
		}
		if err := node.createNodeKey(); err != nil {
			return err
		}
		if err := node.createConsensusKey(); err != nil {
			return err
		}
	}

	return nil
}

func (c *chain) createValidator(index int) *validator {
	return &validator{
		chain:   c,
		index:   index,
		moniker: fmt.Sprintf("%s-atomone-%d", c.id, index),
	}
}
