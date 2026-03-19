package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/atomone-hub/atomone/x/coredaos/testutil"
	"github.com/atomone-hub/atomone/x/coredaos/types"
	govtypesv1 "github.com/atomone-hub/atomone/x/gov/types/v1"
)

func TestMsgServerUpdateParams(t *testing.T) {
	timeDuration := time.Duration(1)
	testAcc := simtestutil.CreateRandomAccounts(3)
	bondedAcc := testAcc[0].String()
	unbondingAcc := testAcc[1].String()
	unbondedAcc := testAcc[2].String()

	tests := []struct {
		name        string
		msg         *types.MsgUpdateParams
		expectedErr string
		setupMocks  func(sdk.Context, *testutil.Mocks)
	}{
		{
			name: "empty authority field",
			msg: &types.MsgUpdateParams{
				Authority: "",
				Params:    types.Params{},
			},
			expectedErr: "invalid authority address: empty address string is not allowed: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "invalid authority field",
			msg: &types.MsgUpdateParams{
				Authority: "xxx",
				Params:    types.Params{},
			},
			expectedErr: "invalid authority address: decoding bech32 failed: invalid bech32 string length 3: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "voting period extension nil",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params:    types.Params{},
			},
			expectedErr: "voting period extension duration must not be nil",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "ok",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "steeringdao bonded",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            bondedAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "cannot update params while Steering DAO have bonded or unbonding tokens: core DAOs cannot stake",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(bondedAcc)).Return(math.NewInt(10), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(bondedAcc)).Return(math.NewInt(0), nil)
			},
		},
		{
			name: "oversightdao bonded",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					OversightDaoAddress:           bondedAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "cannot update params while Oversight DAO have bonded or unbonding tokens: core DAOs cannot stake",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(bondedAcc)).Return(math.NewInt(10), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(bondedAcc)).Return(math.NewInt(0), nil)
			},
		},
		{
			name: "oversightdao incorrect address",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					OversightDaoAddress:           "cosmosincorrectaddress",
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "invalid oversight DAO address: cosmosincorrectaddress: decoding bech32 failed: invalid separator index -1",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "steeringdao incorrect address",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            "cosmosincorrectaddress",
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "invalid steering DAO address: cosmosincorrectaddress: decoding bech32 failed: invalid separator index -1",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "steeringdao in unbonding",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            unbondingAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "cannot update params while Steering DAO have bonded or unbonding tokens: core DAOs cannot stake",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// Address is in unbonding
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(unbondingAcc)).Return(math.NewInt(0), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(unbondingAcc)).Return(math.NewInt(10), nil)
			},
		},
		{
			name: "oversightdao in unbonding",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					OversightDaoAddress:           unbondingAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "cannot update params while Oversight DAO have bonded or unbonding tokens: core DAOs cannot stake",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// Address is in unbonding
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(unbondingAcc)).Return(math.NewInt(0), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(unbondingAcc)).Return(math.NewInt(10), nil)
			},
		},
		{
			name: "ok oversight",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					OversightDaoAddress:           unbondedAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// Address is not bonded or in unbonding
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil)
			},
		},
		{
			name: "ok steeringdao",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            unbondedAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// Address is not bonded or in unbonding
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil)
			},
		},
		{
			name: "ok steeringdao and oversight",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            unbondedAcc,
					OversightDaoAddress:           unbondedAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// Address is not bonded or in unbonding
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil).Times(2)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil).Times(2)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, m, ctx := testutil.SetupMsgServer(t)
			tt.setupMocks(ctx, &m)
			params := types.DefaultParams()
			k.Params.Set(ctx, params)

			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.UpdateParams(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			got := k.GetParams(ctx)
			require.Equal(t, got, tt.msg.Params)
		})
	}
}

func TestMsgServerAnnotateProposal(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(2)
	annotatorAcc := testAcc[0].String()
	steeringDAOAcc := testAcc[1].String()
	votingPeriodProposal := govtypesv1.Proposal{
		Title:   "Test Proposal",
		Summary: "A proposal",
		Id:      1,
		Status:  govtypesv1.StatusVotingPeriod,
	}
	votingPeriodProposalWithAnnotation := govtypesv1.Proposal{
		Title:      "Test Proposal",
		Summary:    "A proposal",
		Id:         1,
		Status:     govtypesv1.StatusVotingPeriod,
		Annotation: "Something",
	}
	depositPeriodProposal := govtypesv1.Proposal{
		Title:   "Test Proposal",
		Summary: "A proposal",
		Id:      2,
		Status:  govtypesv1.StatusDepositPeriod,
	}
	tests := []struct {
		name           string
		msg            *types.MsgAnnotateProposal
		expectedErr    string
		setupMocks     func(sdk.Context, *testutil.Mocks)
		setSteeringDAO bool
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgAnnotateProposal{},
			expectedErr: "invalid annotator address: empty address string is not allowed: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "wrong addr annotator",
			msg: &types.MsgAnnotateProposal{
				Annotator: "cosmosincorrectaddress",
			},
			expectedErr: "invalid annotator address: decoding bech32 failed: invalid separator index -1: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "empty annotation",
			msg: &types.MsgAnnotateProposal{
				Annotator: annotatorAcc,
			},
			expectedErr: "annotation cannot be empty: invalid request",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "annotator empty annotation",
			msg: &types.MsgAnnotateProposal{
				Annotator:  annotatorAcc,
				Annotation: "Something",
			},
			expectedErr: "Steering DAO address is not set: function is disabled",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "wrong annotator account",
			msg: &types.MsgAnnotateProposal{
				Annotator:  annotatorAcc,
				Annotation: "Something",
			},
			expectedErr:    "invalid authority; expected " + steeringDAOAcc + ", got " + annotatorAcc + ": expected core DAO account as only signer for this message",
			setupMocks:     func(ctx sdk.Context, m *testutil.Mocks) {},
			setSteeringDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
			},
			expectedErr: "proposal with ID 0 not found: unknown proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(0)).Return(govtypesv1.Proposal{}, false)
			},
			setSteeringDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
				ProposalId: 1,
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				call1 := m.GovKeeper.EXPECT().GetProposal(ctx, uint64(1)).Return(votingPeriodProposal, true)
				m.GovKeeper.EXPECT().SetProposal(ctx, votingPeriodProposalWithAnnotation).After(call1)
			},
			setSteeringDAO: true,
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
				ProposalId: 2,
			},
			expectedErr: "proposal with ID 2 is not in voting period: inactive proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(2)).Return(depositPeriodProposal, true)
			},
			setSteeringDAO: true,
		},
		{
			name: "already annotated proposal",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
				ProposalId: 3,
			},
			expectedErr: "proposal with ID 3 already has an annotation: annotation already present",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(3)).Return(votingPeriodProposalWithAnnotation, true)
			},
			setSteeringDAO: true,
		},
		{
			name: "already annotated proposal but overwrite",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
				ProposalId: 3,
				Overwrite:  true,
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(3)).Return(votingPeriodProposalWithAnnotation, true)
				m.GovKeeper.EXPECT().SetProposal(ctx, votingPeriodProposalWithAnnotation)
			},
			setSteeringDAO: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, m, ctx := testutil.SetupMsgServer(t)
			tt.setupMocks(ctx, &m)
			params := types.DefaultParams()
			if tt.setSteeringDAO {
				params.SteeringDaoAddress = steeringDAOAcc
			}
			k.Params.Set(ctx, params)
			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.AnnotateProposal(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgServerEndorseProposal(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(2)
	endorserAcc := testAcc[0].String()
	steeringDAOAcc := testAcc[1].String()
	votingPeriodProposal := govtypesv1.Proposal{
		Title:   "Test Proposal",
		Summary: "A proposal",
		Id:      1,
		Status:  govtypesv1.StatusVotingPeriod,
	}
	votingPeriodProposalWithEndorsement := govtypesv1.Proposal{
		Title:    "Test Proposal",
		Summary:  "A proposal",
		Id:       1,
		Status:   govtypesv1.StatusVotingPeriod,
		Endorsed: true,
	}
	depositPeriodProposal := govtypesv1.Proposal{
		Title:   "Test Proposal",
		Summary: "A proposal",
		Id:      2,
		Status:  govtypesv1.StatusDepositPeriod,
	}
	tests := []struct {
		name           string
		msg            *types.MsgEndorseProposal
		expectedErr    string
		setupMocks     func(sdk.Context, *testutil.Mocks)
		setSteeringDAO bool
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgEndorseProposal{},
			expectedErr: "invalid endorser address: empty address string is not allowed: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "wrong addr endorser",
			msg: &types.MsgEndorseProposal{
				Endorser: "cosmosincorrectaddress",
			},
			expectedErr: "invalid endorser address: decoding bech32 failed: invalid separator index -1: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "no steeringdao address",
			msg: &types.MsgEndorseProposal{
				Endorser: endorserAcc,
			},
			expectedErr: "Steering DAO address is not set: function is disabled",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "wrong endorser account",
			msg: &types.MsgEndorseProposal{
				Endorser: endorserAcc,
			},
			expectedErr:    "invalid authority; expected " + steeringDAOAcc + ", got " + endorserAcc + ": expected core DAO account as only signer for this message",
			setupMocks:     func(ctx sdk.Context, m *testutil.Mocks) {},
			setSteeringDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgEndorseProposal{
				Endorser: steeringDAOAcc,
			},
			expectedErr: "proposal with ID 0 not found: unknown proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(0)).Return(govtypesv1.Proposal{}, false)
			},
			setSteeringDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgEndorseProposal{
				Endorser:   steeringDAOAcc,
				ProposalId: 1,
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(1)).Return(votingPeriodProposal, true)
				m.GovKeeper.EXPECT().SetProposal(ctx, votingPeriodProposalWithEndorsement)
			},
			setSteeringDAO: true,
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgEndorseProposal{
				Endorser:   steeringDAOAcc,
				ProposalId: 2,
			},
			expectedErr: "proposal with ID 2 is not in voting period: inactive proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(2)).Return(depositPeriodProposal, true)
			},
			setSteeringDAO: true,
		},
		{
			name: "already endorsed proposal",
			msg: &types.MsgEndorseProposal{
				Endorser:   steeringDAOAcc,
				ProposalId: 3,
			},
			expectedErr: "proposal with ID 3 has already been endorsed: proposal already endorsed",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(3)).Return(votingPeriodProposalWithEndorsement, true)
			},
			setSteeringDAO: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, m, ctx := testutil.SetupMsgServer(t)
			tt.setupMocks(ctx, &m)
			params := types.DefaultParams()
			if tt.setSteeringDAO {
				params.SteeringDaoAddress = steeringDAOAcc
			}
			k.Params.Set(ctx, params)
			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.EndorseProposal(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgServerExtendVotingPeriod(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(2)
	extenderAcc := testAcc[0].String()
	steeringDAOAcc := testAcc[1].String()
	params := types.DefaultParams()
	votingEndTime := time.Now().Add(time.Hour * time.Duration(1))
	votingEndTimeExtended := votingEndTime.Add(*params.VotingPeriodExtensionDuration)

	votingPeriodProposal := govtypesv1.Proposal{
		Title:                     "Test Proposal",
		Summary:                   "A proposal",
		Id:                        1,
		Status:                    govtypesv1.StatusVotingPeriod,
		VotingEndTime:             &votingEndTime,
		TimesVotingPeriodExtended: 2,
	}
	votingPeriodProposalWithExtension := govtypesv1.Proposal{
		Title:                     "Test Proposal",
		Summary:                   "A proposal",
		Id:                        1,
		Status:                    govtypesv1.StatusVotingPeriod,
		VotingEndTime:             &votingEndTimeExtended,
		TimesVotingPeriodExtended: 3,
	}
	depositPeriodProposal := govtypesv1.Proposal{
		Title:         "Test Proposal",
		Summary:       "A proposal",
		Id:            2,
		Status:        govtypesv1.StatusDepositPeriod,
		VotingEndTime: &votingEndTime,
	}
	tests := []struct {
		name           string
		msg            *types.MsgExtendVotingPeriod
		expectedErr    string
		setupMocks     func(sdk.Context, *testutil.Mocks)
		setSteeringDAO bool
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgExtendVotingPeriod{},
			expectedErr: "invalid extender address: empty address string is not allowed: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "wrong addr extender",
			msg: &types.MsgExtendVotingPeriod{
				Extender: "cosmosincorrectaddress",
			},
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
			expectedErr: "invalid extender address: decoding bech32 failed: invalid separator index -1: invalid address",
		},
		{
			name: "function disabled",
			msg: &types.MsgExtendVotingPeriod{
				Extender: extenderAcc,
			},
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
			expectedErr: "Steering DAO address and Oversight DAO address are not set: function is disabled",
		},
		{
			name: "wrong extender account",
			msg: &types.MsgExtendVotingPeriod{
				Extender: extenderAcc,
			},
			expectedErr:    "invalid authority; expected " + steeringDAOAcc + ", got " + extenderAcc + ": expected core DAO account as only signer for this message",
			setupMocks:     func(ctx sdk.Context, m *testutil.Mocks) {},
			setSteeringDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgExtendVotingPeriod{
				Extender: steeringDAOAcc,
			},
			expectedErr: "proposal with ID 0 not found: unknown proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(0)).Return(govtypesv1.Proposal{}, false)
			},
			setSteeringDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgExtendVotingPeriod{
				Extender:   steeringDAOAcc,
				ProposalId: 1,
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(1)).Return(votingPeriodProposal, true)
				m.GovKeeper.EXPECT().RemoveFromActiveProposalQueue(ctx, uint64(1), votingEndTime)
				m.GovKeeper.EXPECT().InsertActiveProposalQueue(ctx, uint64(1), votingEndTimeExtended)
				m.GovKeeper.EXPECT().SetProposal(ctx, votingPeriodProposalWithExtension)
			},
			setSteeringDAO: true,
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgExtendVotingPeriod{
				Extender:   steeringDAOAcc,
				ProposalId: 2,
			},
			expectedErr: "proposal with ID 2 is not in voting period: inactive proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(2)).Return(depositPeriodProposal, true)
			},
			setSteeringDAO: true,
		},
		{
			name: "proposal cannot be extended",
			msg: &types.MsgExtendVotingPeriod{
				Extender:   steeringDAOAcc,
				ProposalId: 3,
			},
			expectedErr: "proposal with ID 3 has reached the maximum number of voting period extensions: invalid proposal content",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(3)).Return(votingPeriodProposalWithExtension, true)
			},
			setSteeringDAO: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, m, ctx := testutil.SetupMsgServer(t)
			tt.setupMocks(ctx, &m)
			params := types.DefaultParams()
			if tt.setSteeringDAO {
				params.SteeringDaoAddress = steeringDAOAcc
			}
			k.Params.Set(ctx, params)
			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.ExtendVotingPeriod(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgServerVetoProposal(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(3)
	vetoerAcc := testAcc[0].String()
	oversightDAOAcc := testAcc[1].String()
	emptyTally := govtypesv1.EmptyTallyResult()
	votingEndTime := time.Now().Add(time.Hour * time.Duration(1))
	votingPeriodProposal := govtypesv1.Proposal{
		Title:         "Test Proposal",
		Summary:       "A proposal",
		Id:            1,
		Status:        govtypesv1.StatusVotingPeriod,
		VotingEndTime: &votingEndTime,
	}
	proposalWithVeto := govtypesv1.Proposal{
		Title:            "Test Proposal",
		Summary:          "A proposal",
		Id:               1,
		Status:           govtypesv1.StatusVetoed,
		FinalTallyResult: &emptyTally,
		VotingEndTime:    &votingEndTime, // will be overwritten in the test
	}
	depositPeriodProposal := govtypesv1.Proposal{
		Title:   "Test Proposal",
		Summary: "A proposal",
		Id:      2,
		Status:  govtypesv1.StatusDepositPeriod,
	}
	proposalWithChangeOversightDAOMsgs, err := sdktx.SetMsgs([]sdk.Msg{&types.MsgUpdateParams{
		Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
		Params: types.Params{
			OversightDaoAddress: testAcc[2].String(),
		},
	}})
	require.NoError(t, err)
	proposalWithChangeOversightDAO := govtypesv1.Proposal{
		Title:    "Test Proposal",
		Summary:  "A proposal to change oversight DAO address",
		Id:       4,
		Status:   govtypesv1.StatusVotingPeriod,
		Messages: proposalWithChangeOversightDAOMsgs,
	}
	tests := []struct {
		name            string
		msg             *types.MsgVetoProposal
		expectedErr     string
		setupMocks      func(sdk.Context, *testutil.Mocks)
		setSteeringDAO  bool
		setOversightDAO bool
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgVetoProposal{},
			expectedErr: "invalid vetoer address: empty address string is not allowed: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "wrong addr vetoer",
			msg: &types.MsgVetoProposal{
				Vetoer: "cosmosincorrectaddress",
			},
			expectedErr: "invalid vetoer address: decoding bech32 failed: invalid separator index -1: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "function disabled",
			msg: &types.MsgVetoProposal{
				Vetoer: vetoerAcc,
			},
			expectedErr: "Oversight DAO address is not set: function is disabled",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "wrong vetoer account",
			msg: &types.MsgVetoProposal{
				Vetoer: vetoerAcc,
			},
			expectedErr:     "invalid authority; expected " + oversightDAOAcc + ", got " + vetoerAcc + ": expected core DAO account as only signer for this message",
			setupMocks:      func(ctx sdk.Context, m *testutil.Mocks) {},
			setOversightDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgVetoProposal{
				Vetoer: oversightDAOAcc,
			},
			expectedErr: "proposal with ID 0 not found: unknown proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(0)).Return(govtypesv1.Proposal{}, false)
			},
			setOversightDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgVetoProposal{
				Vetoer:     oversightDAOAcc,
				ProposalId: 1,
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// ensure proposalWithVeto has the correct VotingEndTime set
				// can only do it here because ctx is needed
				newVotingEndTime := ctx.BlockTime()
				proposalWithVeto.VotingEndTime = &newVotingEndTime

				call1 := m.GovKeeper.EXPECT().GetProposal(ctx, uint64(1)).Return(votingPeriodProposal, true)
				m.GovKeeper.EXPECT().RefundAndDeleteDeposits(ctx, uint64(1)).After(call1)
				m.GovKeeper.EXPECT().SetProposal(ctx, proposalWithVeto).After(call1)
				m.GovKeeper.EXPECT().DeleteVotes(ctx, uint64(1)).After(call1)
				call2 := m.GovKeeper.EXPECT().RemoveFromActiveProposalQueue(ctx, uint64(1), votingEndTime).After(call1)
				m.GovKeeper.EXPECT().UpdateMinInitialDeposit(ctx, true).After(call2)
				m.GovKeeper.EXPECT().UpdateMinDeposit(ctx, true).After(call2)
			},
			setOversightDAO: true,
		},
		{
			name: "ok burn deposit",
			msg: &types.MsgVetoProposal{
				Vetoer:      oversightDAOAcc,
				ProposalId:  1,
				BurnDeposit: true,
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// ensure proposalWithVeto has the correct VotingEndTime set
				// can only do it here because ctx is needed
				newVotingEndTime := ctx.BlockTime()
				proposalWithVeto.VotingEndTime = &newVotingEndTime

				call1 := m.GovKeeper.EXPECT().GetProposal(ctx, uint64(1)).Return(votingPeriodProposal, true)
				m.GovKeeper.EXPECT().DeleteAndBurnDeposits(ctx, uint64(1)).MaxTimes(1)
				m.GovKeeper.EXPECT().SetProposal(ctx, proposalWithVeto).After(call1)
				m.GovKeeper.EXPECT().DeleteVotes(ctx, uint64(1)).After(call1)
				call2 := m.GovKeeper.EXPECT().RemoveFromActiveProposalQueue(ctx, uint64(1), votingEndTime).After(call1)
				m.GovKeeper.EXPECT().UpdateMinInitialDeposit(ctx, true).After(call2)
				m.GovKeeper.EXPECT().UpdateMinDeposit(ctx, true).After(call2)
			},
			setOversightDAO: true,
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgVetoProposal{
				Vetoer:     oversightDAOAcc,
				ProposalId: 2,
			},
			expectedErr: "proposal with ID 2 is not in voting period: inactive proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(2)).Return(depositPeriodProposal, true)
			},
			setOversightDAO: true,
		},
		{
			name: "proposal already vetoed",
			msg: &types.MsgVetoProposal{
				Vetoer:     oversightDAOAcc,
				ProposalId: 3,
			},
			expectedErr: "proposal with ID 3 is not in voting period: inactive proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(3)).Return(proposalWithVeto, true)
			},
			setOversightDAO: true,
		},
		{
			name: "veto proposal with change to oversight DAO address",
			msg: &types.MsgVetoProposal{
				Vetoer:     oversightDAOAcc,
				ProposalId: 4,
			},
			expectedErr: "proposal with ID 4 contains a change of the oversight DAO address, vetoing it would prevent the replacement of the current oversight DAO: oversight DAO cannot veto this proposal",
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				m.GovKeeper.EXPECT().GetProposal(ctx, uint64(4)).Return(proposalWithChangeOversightDAO, true)
			},
			setOversightDAO: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, m, ctx := testutil.SetupMsgServer(t)
			tt.setupMocks(ctx, &m)
			params := types.DefaultParams()
			if tt.setOversightDAO {
				params.OversightDaoAddress = oversightDAOAcc
			}
			k.Params.Set(ctx, params)
			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.VetoProposal(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
