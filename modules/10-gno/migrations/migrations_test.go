package migrations_test

import (
	"testing"
	"time"

	testifysuite "github.com/stretchr/testify/suite"

	ibcgno "github.com/atomone-hub/atomone/modules/10-gno"
	ibcgnomigrations "github.com/atomone-hub/atomone/modules/10-gno/migrations"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

type MigrationsTestSuite struct {
	testifysuite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

func (suite *MigrationsTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func TestGnoTestSuite(t *testing.T) {
	testifysuite.Run(t, new(MigrationsTestSuite))
}

// test pruning of multiple expired gno consensus states
func (suite *MigrationsTestSuite) TestPruneExpiredConsensusStates() {
	// create multiple gno clients and a solo machine client
	// the solo machine is used to verify this pruning function only modifies
	// the gno store.

	numTMClients := 3
	paths := make([]*ibctesting.Path, numTMClients)

	for i := range numTMClients {
		path := ibctesting.NewPath(suite.chainA, suite.chainB)
		path.SetupClients()

		paths[i] = path
	}

	solomachine := ibctesting.NewSolomachine(suite.T(), suite.chainA.Codec, ibctesting.DefaultSolomachineClientID, "testing", 1)
	smClientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), solomachine.ClientID)

	// set client state
	bz, err := suite.chainA.App.AppCodec().MarshalInterface(solomachine.ClientState())
	suite.Require().NoError(err)
	smClientStore.Set(host.ClientStateKey(), bz)

	bz, err = suite.chainA.App.AppCodec().MarshalInterface(solomachine.ConsensusState())
	suite.Require().NoError(err)
	smHeight := clienttypes.NewHeight(0, 1)
	smClientStore.Set(host.ConsensusStateKey(smHeight), bz)

	pruneHeightMap := make(map[*ibctesting.Path][]exported.Height)
	unexpiredHeightMap := make(map[*ibctesting.Path][]exported.Height)

	for _, path := range paths {
		// collect all heights expected to be pruned
		var pruneHeights []exported.Height
		pruneHeights = append(pruneHeights, path.EndpointA.GetClientLatestHeight())

		// these heights will be expired and also pruned
		for range 3 {
			err := path.EndpointA.UpdateClient()
			suite.Require().NoError(err)

			pruneHeights = append(pruneHeights, path.EndpointA.GetClientLatestHeight())
		}

		// double chedck all information is currently stored
		for _, pruneHeight := range pruneHeights {
			consState, ok := suite.chainA.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
			suite.Require().True(ok)
			suite.Require().NotNil(consState)

			ctx := suite.chainA.GetContext()
			clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

			processedTime, ok := ibcgno.GetProcessedTime(clientStore, pruneHeight)
			suite.Require().True(ok)
			suite.Require().NotNil(processedTime)

			processedHeight, ok := ibcgno.GetProcessedHeight(clientStore, pruneHeight)
			suite.Require().True(ok)
			suite.Require().NotNil(processedHeight)

			expectedConsKey := ibcgno.GetIterationKey(clientStore, pruneHeight)
			suite.Require().NotNil(expectedConsKey)
		}
		pruneHeightMap[path] = pruneHeights
	}

	// Increment the time by a week
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)

	for _, path := range paths {
		// create the consensus state that can be used as trusted height for next update
		var unexpiredHeights []exported.Height
		err := path.EndpointA.UpdateClient()
		suite.Require().NoError(err)
		unexpiredHeights = append(unexpiredHeights, path.EndpointA.GetClientLatestHeight())

		err = path.EndpointA.UpdateClient()
		suite.Require().NoError(err)
		unexpiredHeights = append(unexpiredHeights, path.EndpointA.GetClientLatestHeight())

		unexpiredHeightMap[path] = unexpiredHeights
	}

	// Increment the time by another week, then update the client.
	// This will cause the consensus states created before the first time increment
	// to be expired
	suite.coordinator.IncrementTimeBy(7 * 24 * time.Hour)
	totalPruned, err := ibcgnomigrations.PruneExpiredConsensusStates(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), suite.chainA.GetSimApp().IBCKeeper.ClientKeeper)
	suite.Require().NoError(err)
	suite.Require().NotZero(totalPruned)

	for _, path := range paths {
		ctx := suite.chainA.GetContext()
		clientStore := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(ctx, path.EndpointA.ClientID)

		// ensure everything has been pruned
		for i, pruneHeight := range pruneHeightMap[path] {
			consState, ok := suite.chainA.GetConsensusState(path.EndpointA.ClientID, pruneHeight)
			suite.Require().False(ok, i)
			suite.Require().Nil(consState, i)

			processedTime, ok := ibcgno.GetProcessedTime(clientStore, pruneHeight)
			suite.Require().False(ok, i)
			suite.Require().Equal(uint64(0), processedTime, i)

			processedHeight, ok := ibcgno.GetProcessedHeight(clientStore, pruneHeight)
			suite.Require().False(ok, i)
			suite.Require().Nil(processedHeight, i)

			expectedConsKey := ibcgno.GetIterationKey(clientStore, pruneHeight)
			suite.Require().Nil(expectedConsKey, i)
		}

		// ensure metadata is set for unexpired consensus state
		for _, height := range unexpiredHeightMap[path] {
			consState, ok := suite.chainA.GetConsensusState(path.EndpointA.ClientID, height)
			suite.Require().True(ok)
			suite.Require().NotNil(consState)

			processedTime, ok := ibcgno.GetProcessedTime(clientStore, height)
			suite.Require().True(ok)
			suite.Require().NotEqual(uint64(0), processedTime)

			processedHeight, ok := ibcgno.GetProcessedHeight(clientStore, height)
			suite.Require().True(ok)
			suite.Require().NotEqual(clienttypes.ZeroHeight(), processedHeight)

			consKey := ibcgno.GetIterationKey(clientStore, height)
			suite.Require().Equal(host.ConsensusStateKey(height), consKey)
		}
	}

	// verify that solomachine client and consensus state were not removed
	smClientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), solomachine.ClientID)
	bz = smClientStore.Get(host.ClientStateKey())
	suite.Require().NotEmpty(bz)

	bz = smClientStore.Get(host.ConsensusStateKey(smHeight))
	suite.Require().NotEmpty(bz)
}
