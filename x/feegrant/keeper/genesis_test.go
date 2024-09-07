package keeper_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	codectypes "github.com/atomone-hub/atomone/codec/types"
	"github.com/atomone-hub/atomone/crypto/keys/secp256k1"
	"github.com/atomone-hub/atomone/testutil"
	"github.com/atomone-hub/atomone/testutil/testdata"
	sdk "github.com/atomone-hub/atomone/types"
	moduletestutil "github.com/atomone-hub/atomone/types/module/testutil"
	authtypes "github.com/atomone-hub/atomone/x/auth/types"
	"github.com/atomone-hub/atomone/x/feegrant"
	"github.com/atomone-hub/atomone/x/feegrant/keeper"
	"github.com/atomone-hub/atomone/x/feegrant/module"
	feegranttestutil "github.com/atomone-hub/atomone/x/feegrant/testutil"
)

type GenesisTestSuite struct {
	suite.Suite
	ctx            sdk.Context
	feegrantKeeper keeper.Keeper
}

func (suite *GenesisTestSuite) SetupTest() {
	key := sdk.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, sdk.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModuleBasic{})

	ctrl := gomock.NewController(suite.T())
	accountKeeper := feegranttestutil.NewMockAccountKeeper(ctrl)
	accountKeeper.EXPECT().GetAccount(gomock.Any(), granteeAddr).Return(authtypes.NewBaseAccountWithAddress(granteeAddr)).AnyTimes()

	suite.feegrantKeeper = keeper.NewKeeper(encCfg.Codec, key, accountKeeper)

	suite.ctx = testCtx.Ctx
}

var (
	granteePub  = secp256k1.GenPrivKey().PubKey()
	granterPub  = secp256k1.GenPrivKey().PubKey()
	granteeAddr = sdk.AccAddress(granteePub.Address())
	granterAddr = sdk.AccAddress(granterPub.Address())
)

func (suite *GenesisTestSuite) TestImportExportGenesis() {
	coins := sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1_000)))
	now := suite.ctx.BlockHeader().Time
	oneYear := now.AddDate(1, 0, 0)
	msgSrvr := keeper.NewMsgServerImpl(suite.feegrantKeeper)

	allowance := &feegrant.BasicAllowance{SpendLimit: coins, Expiration: &oneYear}
	err := suite.feegrantKeeper.GrantAllowance(suite.ctx, granterAddr, granteeAddr, allowance)
	suite.Require().NoError(err)

	genesis, err := suite.feegrantKeeper.ExportGenesis(suite.ctx)
	suite.Require().NoError(err)
	// revoke fee allowance
	_, err = msgSrvr.RevokeAllowance(sdk.WrapSDKContext(suite.ctx), &feegrant.MsgRevokeAllowance{
		Granter: granterAddr.String(),
		Grantee: granteeAddr.String(),
	})
	suite.Require().NoError(err)
	err = suite.feegrantKeeper.InitGenesis(suite.ctx, genesis)
	suite.Require().NoError(err)

	newGenesis, err := suite.feegrantKeeper.ExportGenesis(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(genesis, newGenesis)
}

func (suite *GenesisTestSuite) TestInitGenesis() {
	any, err := codectypes.NewAnyWithValue(&testdata.Dog{})
	suite.Require().NoError(err)

	testCases := []struct {
		name          string
		feeAllowances []feegrant.Grant
	}{
		{
			"invalid granter",
			[]feegrant.Grant{
				{
					Granter: "invalid granter",
					Grantee: granteeAddr.String(),
				},
			},
		},
		{
			"invalid grantee",
			[]feegrant.Grant{
				{
					Granter: granterAddr.String(),
					Grantee: "invalid grantee",
				},
			},
		},
		{
			"invalid allowance",
			[]feegrant.Grant{
				{
					Granter:   granterAddr.String(),
					Grantee:   granteeAddr.String(),
					Allowance: any,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			err := suite.feegrantKeeper.InitGenesis(suite.ctx, &feegrant.GenesisState{Allowances: tc.feeAllowances})
			suite.Require().Error(err)
		})
	}
}

func TestGenesisTestSuite(t *testing.T) {
	suite.Run(t, new(GenesisTestSuite))
}
