package gno_test

import (
	"errors"
	"time"

	errorsmod "cosmossdk.io/errors"

	"github.com/cometbft/cometbft/crypto/tmhash"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"

	ibcgno "github.com/atomone-hub/atomone/modules/10-gno"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
)

func (suite *GnoTestSuite) TestMisbehaviour() {
	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	misbehaviour := &ibcgno.Misbehaviour{
		Header1:  suite.header,
		Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, suite.valSet, suite.valSet, suite.valSet, suite.signers),
		ClientId: clientID,
	}

	suite.Require().Equal(exported.Tendermint, misbehaviour.ClientType())
}

func (suite *GnoTestSuite) TestMisbehaviourValidateBasic() {
	altPrivVal := cmttypes.NewMockPV()
	altPubKey, err := altPrivVal.GetPubKey()
	suite.Require().NoError(err)

	revisionHeight := int64(height.RevisionHeight)

	altVal := cmttypes.NewValidator(altPubKey, revisionHeight)

	// Create alternative validator set with only altVal
	altValSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{altVal})

	// Create signer array and ensure it is in same order as bothValSet
	bothValSet, bothSigners := getBothSigners(suite, altVal, altPrivVal)

	altSignerArr := []cmttypes.PrivValidator{altPrivVal}

	heightMinus1 := clienttypes.NewHeight(0, height.RevisionHeight-1)

	testCases := []struct {
		name                 string
		misbehaviour         *ibcgno.Misbehaviour
		malleateMisbehaviour func(misbehaviour *ibcgno.Misbehaviour) error
		expErr               error
	}{
		{
			"valid fork misbehaviour, two headers at same height have different time",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now.Add(time.Minute), suite.valSet, suite.valSet, suite.valSet, suite.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			nil,
		},
		{
			"valid time misbehaviour, both headers at different heights are at same time",
			&ibcgno.Misbehaviour{
				Header1:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight+5), heightMinus1, suite.now, suite.valSet, suite.valSet, suite.valSet, suite.signers),
				Header2:  suite.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			nil,
		},
		{
			"misbehaviour Header1 is nil",
			ibcgno.NewMisbehaviour(clientID, nil, suite.header),
			func(m *ibcgno.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibcgno.ErrInvalidHeader, "misbehaviour Header1 cannot be nil"),
		},
		{
			"misbehaviour Header2 is nil",
			ibcgno.NewMisbehaviour(clientID, suite.header, nil),
			func(m *ibcgno.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibcgno.ErrInvalidHeader, "misbehaviour Header2 cannot be nil"),
		},
		{
			"valid misbehaviour with different trusted headers",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), clienttypes.NewHeight(0, height.RevisionHeight-3), suite.now.Add(time.Minute), suite.valSet, suite.valSet, bothValSet, suite.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			nil,
		},
		{
			"trusted height is 0 in Header1",
			&ibcgno.Misbehaviour{
				Header1:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), clienttypes.ZeroHeight(), suite.now.Add(time.Minute), suite.valSet, suite.valSet, suite.valSet, suite.signers),
				Header2:  suite.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibcgno.ErrInvalidHeaderHeight, "misbehaviour Header1 cannot have zero revision height"),
		},
		{
			"trusted height is 0 in Header2",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), clienttypes.ZeroHeight(), suite.now.Add(time.Minute), suite.valSet, suite.valSet, suite.valSet, suite.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibcgno.ErrInvalidHeaderHeight, "misbehaviour Header2 cannot have zero revision height"),
		},
		{
			"trusted valset is nil in Header1",
			&ibcgno.Misbehaviour{
				Header1:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now.Add(time.Minute), suite.valSet, suite.valSet, nil, suite.signers),
				Header2:  suite.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibcgno.ErrInvalidValidatorSet, "trusted validator set in Header1 cannot be empty"),
		},
		{
			"trusted valset is nil in Header2",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now.Add(time.Minute), suite.valSet, suite.valSet, nil, suite.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			errorsmod.Wrap(ibcgno.ErrInvalidValidatorSet, "trusted validator set in Header2 cannot be empty"),
		},
		{
			"invalid client ID ",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, suite.valSet, suite.valSet, suite.valSet, suite.signers),
				ClientId: "GAI",
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			errors.New("identifier GAI has invalid length"),
		},
		{
			"chainIDs do not match",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader("ethermint", int64(height.RevisionHeight), heightMinus1, suite.now, suite.valSet, suite.valSet, suite.valSet, suite.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			errorsmod.Wrap(clienttypes.ErrInvalidMisbehaviour, "headers must have identical chainIDs"),
		},
		{
			"header2 height is greater",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, 6, clienttypes.NewHeight(0, height.RevisionHeight+1), suite.now, suite.valSet, suite.valSet, suite.valSet, suite.signers),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error { return nil },
			errors.New("Header1 height is less than Header2 height"),
		},
		{
			"header 1 doesn't have 2/3 majority",
			&ibcgno.Misbehaviour{
				Header1:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, bothValSet, bothValSet, suite.valSet, bothSigners),
				Header2:  suite.header,
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error {
				// voteSet contains only altVal which is less than 2/3 of total power (height/1height)
				wrongVoteSet := cmttypes.NewVoteSet(chainID, int64(misbehaviour.Header1.GetHeight().GetRevisionHeight()), 1, cmtproto.PrecommitType, altValSet)
				blockID, err := cmttypes.BlockIDFromProto(&misbehaviour.Header1.Commit.BlockID)
				if err != nil {
					return err
				}

				extCommit, err := cmttypes.MakeExtCommit(*blockID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), misbehaviour.Header1.Commit.Round, wrongVoteSet, altSignerArr, suite.now, false)
				misbehaviour.Header1.Commit = extCommit.ToCommit().ToProto()
				return err
			},
			errors.New("validator set did not commit to header"),
		},
		{
			"header 2 doesn't have 2/3 majority",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, bothValSet, bothValSet, suite.valSet, bothSigners),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error {
				// voteSet contains only altVal which is less than 2/3 of total power (height/1height)
				wrongVoteSet := cmttypes.NewVoteSet(chainID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), 1, cmtproto.PrecommitType, altValSet)
				blockID, err := cmttypes.BlockIDFromProto(&misbehaviour.Header2.Commit.BlockID)
				if err != nil {
					return err
				}

				extCommit, err := cmttypes.MakeExtCommit(*blockID, int64(misbehaviour.Header2.GetHeight().GetRevisionHeight()), misbehaviour.Header2.Commit.Round, wrongVoteSet, altSignerArr, suite.now, false)
				misbehaviour.Header2.Commit = extCommit.ToCommit().ToProto()
				return err
			},
			errors.New("validator set did not commit to header"),
		},
		{
			"validators sign off on wrong commit",
			&ibcgno.Misbehaviour{
				Header1:  suite.header,
				Header2:  suite.chainA.CreateTMClientHeader(chainID, int64(height.RevisionHeight), heightMinus1, suite.now, bothValSet, bothValSet, suite.valSet, bothSigners),
				ClientId: clientID,
			},
			func(misbehaviour *ibcgno.Misbehaviour) error {
				tmBlockID := ibctesting.MakeBlockID(tmhash.Sum([]byte("other_hash")), 3, tmhash.Sum([]byte("other_partset")))
				misbehaviour.Header2.Commit.BlockID = tmBlockID.ToProto()
				return nil
			},
			errors.New("header 2 failed validation"),
		},
	}

	for i, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.malleateMisbehaviour(tc.misbehaviour)
			suite.Require().NoError(err)
			err = tc.misbehaviour.ValidateBasic()

			if tc.expErr == nil {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.name)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.name)
				suite.Require().ErrorContains(err, tc.expErr.Error())
			}
		})
	}
}
