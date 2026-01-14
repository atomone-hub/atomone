package gno_test

import (
	"time"

	ibcgno "github.com/atomone-hub/atomone/modules/10-gno"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
)

func (suite *GnoTestSuite) TestConsensusStateValidateBasic() {
	testCases := []struct {
		msg            string
		consensusState *ibcgno.ConsensusState
		expectPass     bool
	}{
		{
			"success",
			&ibcgno.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: suite.valsHash,
			},
			true,
		},
		{
			"success with sentinel",
			&ibcgno.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte(ibcgno.SentinelRoot)),
				NextValidatorsHash: suite.valsHash,
			},
			true,
		},
		{
			"root is nil",
			&ibcgno.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.MerkleRoot{},
				NextValidatorsHash: suite.valsHash,
			},
			false,
		},
		{
			"root is empty",
			&ibcgno.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.MerkleRoot{},
				NextValidatorsHash: suite.valsHash,
			},
			false,
		},
		{
			"nextvalshash is invalid",
			&ibcgno.ConsensusState{
				Timestamp:          suite.now,
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: []byte("hi"),
			},
			false,
		},

		{
			"timestamp is zero",
			&ibcgno.ConsensusState{
				Timestamp:          time.Time{},
				Root:               commitmenttypes.NewMerkleRoot([]byte("app_hash")),
				NextValidatorsHash: suite.valsHash,
			},
			false,
		},
	}

	for i, tc := range testCases {
		suite.Run(tc.msg, func() {
			// check just to increase coverage
			suite.Require().Equal(ibcgno.Gno, tc.consensusState.ClientType())
			suite.Require().Equal(tc.consensusState.GetRoot(), tc.consensusState.Root)

			err := tc.consensusState.ValidateBasic()
			if tc.expectPass {
				suite.Require().NoError(err, "valid test case %d failed: %s", i, tc.msg)
			} else {
				suite.Require().Error(err, "invalid test case %d passed: %s", i, tc.msg)
			}
		})
	}
}
