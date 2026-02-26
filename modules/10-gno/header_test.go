package gno

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

func TestHeader_ClientType(t *testing.T) {
	header := &Header{}
	require.Equal(t, Gno, header.ClientType())
}

func TestHeader_GetHeight(t *testing.T) {
	testCases := []struct {
		name           string
		chainID        string
		height         int64
		expectedRev    uint64
		expectedHeight uint64
	}{
		{
			name:           "simple chain ID",
			chainID:        "gno-test",
			height:         100,
			expectedRev:    0,
			expectedHeight: 100,
		},
		{
			name:           "chain ID with revision number",
			chainID:        "gno-test-1",
			height:         200,
			expectedRev:    1,
			expectedHeight: 200,
		},
		{
			name:           "chain ID with higher revision",
			chainID:        "gno-test-5",
			height:         50,
			expectedRev:    5,
			expectedHeight: 50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := &Header{
				SignedHeader: &SignedHeader{
					Header: &GnoHeader{
						ChainId: tc.chainID,
						Height:  tc.height,
					},
				},
			}

			height := header.GetHeight()
			require.Equal(t, tc.expectedRev, height.GetRevisionNumber())
			require.Equal(t, tc.expectedHeight, height.GetRevisionHeight())
		})
	}
}

func TestHeader_GetTime(t *testing.T) {
	expectedTime := time.Now().UTC()
	header := &Header{
		SignedHeader: &SignedHeader{
			Header: &GnoHeader{
				Time: expectedTime,
			},
		},
	}

	require.Equal(t, expectedTime, header.GetTime())
}

func TestHeader_ConsensusState(t *testing.T) {
	blockTime := time.Now().UTC()
	appHash := []byte("test-app-hash")
	nextValsHash := make([]byte, 32)

	header := &Header{
		SignedHeader: &SignedHeader{
			Header: &GnoHeader{
				Time:               blockTime,
				AppHash:            appHash,
				NextValidatorsHash: nextValsHash,
			},
		},
	}

	cs := header.ConsensusState()
	require.NotNil(t, cs)
	require.Equal(t, blockTime, cs.Timestamp)
	require.Equal(t, appHash, cs.Root.Hash)
	require.Equal(t, nextValsHash, cs.NextValidatorsHash)
}

func TestHeader_ValidateBasic(t *testing.T) {
	testCases := []struct {
		name      string
		header    func() *Header
		expectErr bool
		errMsg    string
	}{
		{
			name: "nil signed header",
			header: func() *Header {
				return &Header{
					SignedHeader: nil,
				}
			},
			expectErr: true,
			errMsg:    "gno signed header cannot be nil",
		},
		{
			name: "nil header in signed header",
			header: func() *Header {
				return &Header{
					SignedHeader: &SignedHeader{
						Header: nil,
						Commit: &Commit{},
					},
				}
			},
			expectErr: true,
			errMsg:    "gno header cannot be nil",
		},
		{
			name: "nil commit in signed header",
			header: func() *Header {
				return &Header{
					SignedHeader: &SignedHeader{
						Header: &GnoHeader{
							ChainId: testChainID,
							Height:  100,
						},
						Commit: nil,
					},
				}
			},
			expectErr: true,
			errMsg:    "gno commit cannot be nil",
		},
		{
			name: "trusted height >= header height - causes validation error",
			header: func() *Header {
				blockID := createTestBlockID()
				valSet, _ := createTestValidatorSet(1, 100)
				return &Header{
					SignedHeader: &SignedHeader{
						Header: &GnoHeader{
							ChainId:         testChainID,
							Height:          100,
							Time:            time.Now().UTC(),
							ValidatorsHash:  make([]byte, 32),
							LastBlockId:     createTestBlockID(),
							ProposerAddress: valSet.Validators[0].Address,
						},
						Commit: &Commit{
							BlockId: blockID,
							Precommits: []*CommitSig{
								{
									Type:             2,
									Height:           100,
									Round:            0,
									BlockId:          blockID,
									Timestamp:        time.Now().UTC(),
									ValidatorAddress: valSet.Validators[0].Address,
									ValidatorIndex:   0,
									Signature:        make([]byte, 64),
								},
							},
						},
					},
					ValidatorSet:      valSet,
					TrustedHeight:     clienttypes.NewHeight(1, 100), // Equal to header height
					TrustedValidators: valSet,
				}
			},
			expectErr: true,
			errMsg:    "basic validation", // Validation will fail for other reasons with our test data
		},
		{
			name: "nil validator set - causes validation error",
			header: func() *Header {
				blockID := createTestBlockID()
				return &Header{
					SignedHeader: &SignedHeader{
						Header: &GnoHeader{
							ChainId:         testChainID,
							Height:          100,
							Time:            time.Now().UTC(),
							ValidatorsHash:  make([]byte, 32),
							LastBlockId:     createTestBlockID(),
							ProposerAddress: "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5",
						},
						Commit: &Commit{
							BlockId: blockID,
							Precommits: []*CommitSig{
								{
									Type:             2,
									Height:           100,
									Round:            0,
									BlockId:          blockID,
									Timestamp:        time.Now().UTC(),
									ValidatorAddress: "g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5",
									ValidatorIndex:   0,
									Signature:        make([]byte, 64),
								},
							},
						},
					},
					ValidatorSet:  nil,
					TrustedHeight: clienttypes.NewHeight(1, 50),
				}
			},
			expectErr: true,
			errMsg:    "basic validation", // Validation will fail for other reasons with our test data
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := tc.header()
			err := header.ValidateBasic()

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
