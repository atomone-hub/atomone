package keeper_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	atomoneapp "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/app/helpers"
	coredaoskeeper "github.com/atomone-hub/atomone/x/coredaos/keeper"
	"github.com/atomone-hub/atomone/x/coredaos/testutil"
	"github.com/atomone-hub/atomone/x/coredaos/types"
)

func TestMsgServerUpdateParams(t *testing.T) {
	timeDuration := time.Duration(1)
	testAcc := simtestutil.CreateRandomAccounts(4)
	bondedAcc := testAcc[0].String()
	unbondingAcc := testAcc[1].String()
	unbondedAcc := testAcc[2].String()
	unbondedAcc2 := testAcc[3].String()

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
			expectedErr: "cannot update params while Steering DAO has bonded or unbonding tokens: core DAOs cannot stake",
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
			expectedErr: "cannot update params while Oversight DAO has bonded or unbonding tokens: core DAOs cannot stake",
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
			expectedErr: "cannot update params while Steering DAO has bonded or unbonding tokens: core DAOs cannot stake",
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
			expectedErr: "cannot update params while Oversight DAO has bonded or unbonding tokens: core DAOs cannot stake",
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
					OversightDaoAddress:           unbondedAcc2,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			setupMocks: func(ctx sdk.Context, m *testutil.Mocks) {
				// Address is not bonded or in unbonding
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(unbondedAcc)).Return(math.NewInt(0), nil)
				m.StakingKeeper.EXPECT().GetDelegatorBonded(ctx, sdk.MustAccAddressFromBech32(unbondedAcc2)).Return(math.NewInt(0), nil)
				m.StakingKeeper.EXPECT().GetDelegatorUnbonding(ctx, sdk.MustAccAddressFromBech32(unbondedAcc2)).Return(math.NewInt(0), nil)
			},
		},
		{
			name: "steeringdao set to authority address",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "authority address cannot be the same as steering DAO address: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "oversightdao set to authority address",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					OversightDaoAddress:           "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "authority address cannot be the same as oversight DAO address: invalid address",
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "steeringdao and oversight same address",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            unbondedAcc,
					OversightDaoAddress:           unbondedAcc,
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "steering DAO address and oversight DAO address cannot be the same: " + unbondedAcc,
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
		{
			name: "steeringdao and oversight same address but different case",
			msg: &types.MsgUpdateParams{
				Authority: "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
				Params: types.Params{
					SteeringDaoAddress:            unbondedAcc,
					OversightDaoAddress:           strings.ToUpper(unbondedAcc),
					VotingPeriodExtensionDuration: &timeDuration,
				},
			},
			expectedErr: "steering DAO address and oversight DAO address cannot be the same: " + unbondedAcc,
			setupMocks:  func(ctx sdk.Context, m *testutil.Mocks) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms, k, m, ctx := testutil.SetupMsgServer(t)
			tt.setupMocks(ctx, &m)
			params := types.DefaultParams()
			require.NoError(t, k.Params.Set(ctx, params))

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

// govModuleAddr returns the gov module account address, which is the required
// signer for all messages contained in a governance proposal.
func govModuleAddr() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// collectionsJoin is a small alias to build the composite key used by the
// gov ActiveProposalsQueue (Pair[time.Time, uint64]).
func collectionsJoin(t time.Time, id uint64) collections.Pair[time.Time, uint64] {
	return collections.Join(t, id)
}

// submitProposalReal submits a proposal with the given messages into the real
// gov keeper. The proposal is left in the deposit period unless activate is
// true, in which case it is moved to the voting period.
func submitProposalReal(t *testing.T, app *atomoneapp.AtomOneApp, ctx sdk.Context, msgs []sdk.Msg, activate bool) govv1.Proposal {
	t.Helper()
	govAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	proposal, err := app.GovKeeper.SubmitProposal(ctx, msgs, "", "title", "summary", govAddr)
	require.NoError(t, err)
	if activate {
		require.NoError(t, app.GovKeeper.ActivateVotingPeriod(ctx, proposal))
		// re-fetch to get the populated VotingEndTime / StatusVotingPeriod
		proposal, err = app.GovKeeper.Proposals.Get(ctx, proposal.Id)
		require.NoError(t, err)
	}
	return proposal
}

// submitBankSendProposalReal submits a proposal whose only message is a bank
// MsgSend signed by the gov module account.
func submitBankSendProposalReal(t *testing.T, app *atomoneapp.AtomOneApp, ctx sdk.Context, activate bool) govv1.Proposal {
	t.Helper()
	govAddr := authtypes.NewModuleAddress(govtypes.ModuleName)
	recipient := sdk.AccAddress([]byte("recipient___________"))
	sendMsg := banktypes.NewMsgSend(govAddr, recipient, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1))))
	return submitProposalReal(t, app, ctx, []sdk.Msg{sendMsg}, activate)
}

func TestMsgServerAnnotateProposal(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(2)
	annotatorAcc := testAcc[0].String()
	steeringDAOAcc := testAcc[1].String()

	tests := []struct {
		name        string
		msg         *types.MsgAnnotateProposal
		expectedErr string
		// proposalState describes how to set up the proposal in the real gov
		// keeper before calling the method: "none", "voting", "deposit",
		// "voting-annotated".
		proposalState  string
		setSteeringDAO bool
		// assertAnnotation, if non-empty, is the expected annotation after a
		// successful call.
		assertAnnotation string
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgAnnotateProposal{},
			expectedErr: "invalid annotator address: empty address string is not allowed: invalid address",
		},
		{
			name: "wrong addr annotator",
			msg: &types.MsgAnnotateProposal{
				Annotator: "cosmosincorrectaddress",
			},
			expectedErr: "invalid annotator address: decoding bech32 failed: invalid separator index -1: invalid address",
		},
		{
			name: "empty annotation",
			msg: &types.MsgAnnotateProposal{
				Annotator: annotatorAcc,
			},
			expectedErr: "annotation cannot be empty: invalid request",
		},
		{
			name: "annotator empty annotation",
			msg: &types.MsgAnnotateProposal{
				Annotator:  annotatorAcc,
				Annotation: "Something",
			},
			expectedErr: "Steering DAO address is not set: function is disabled",
		},
		{
			name: "wrong annotator account",
			msg: &types.MsgAnnotateProposal{
				Annotator:  annotatorAcc,
				Annotation: "Something",
			},
			expectedErr:    "invalid authority; expected " + steeringDAOAcc + ", got " + annotatorAcc + ": expected core DAO account as only signer for this message",
			setSteeringDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
				ProposalId: 9999,
			},
			expectedErr:    "proposal with ID 9999 not found: unknown proposal",
			proposalState:  "none",
			setSteeringDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
			},
			proposalState:    "voting",
			setSteeringDAO:   true,
			assertAnnotation: "Something",
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
			},
			expectedErr:    "is not in voting period: inactive proposal",
			proposalState:  "deposit",
			setSteeringDAO: true,
		},
		{
			name: "already annotated proposal",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "Something",
			},
			expectedErr:    "already has an annotation: annotation already present",
			proposalState:  "voting-annotated",
			setSteeringDAO: true,
		},
		{
			name: "already annotated proposal but overwrite",
			msg: &types.MsgAnnotateProposal{
				Annotator:  steeringDAOAcc,
				Annotation: "New annotation",
				Overwrite:  true,
			},
			proposalState:    "voting-annotated",
			setSteeringDAO:   true,
			assertAnnotation: "New annotation",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := helpers.Setup(t)
			ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})
			ms := coredaoskeeper.NewMsgServer(app.CoreDaosKeeper)

			params := types.DefaultParams()
			if tt.setSteeringDAO {
				params.SteeringDaoAddress = steeringDAOAcc
			}
			require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, params))

			// Set up the proposal in the real gov keeper as required.
			switch tt.proposalState {
			case "voting":
				p := submitBankSendProposalReal(t, app, ctx, true)
				tt.msg.ProposalId = p.Id
			case "deposit":
				p := submitBankSendProposalReal(t, app, ctx, false)
				tt.msg.ProposalId = p.Id
			case "voting-annotated":
				p := submitBankSendProposalReal(t, app, ctx, true)
				p.Annotation = "Existing"
				require.NoError(t, app.GovKeeper.SetProposal(ctx, p))
				tt.msg.ProposalId = p.Id
			}

			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.AnnotateProposal(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
				return
			}
			require.NoError(t, err)
			if tt.assertAnnotation != "" {
				got, err := app.GovKeeper.Proposals.Get(ctx, tt.msg.ProposalId)
				require.NoError(t, err)
				require.Equal(t, tt.assertAnnotation, got.Annotation)
			}
		})
	}
}

func TestMsgServerEndorseProposal(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(2)
	endorserAcc := testAcc[0].String()
	steeringDAOAcc := testAcc[1].String()

	tests := []struct {
		name           string
		msg            *types.MsgEndorseProposal
		expectedErr    string
		proposalState  string
		setSteeringDAO bool
		assertEndorsed bool
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgEndorseProposal{},
			expectedErr: "invalid endorser address: empty address string is not allowed: invalid address",
		},
		{
			name: "wrong addr endorser",
			msg: &types.MsgEndorseProposal{
				Endorser: "cosmosincorrectaddress",
			},
			expectedErr: "invalid endorser address: decoding bech32 failed: invalid separator index -1: invalid address",
		},
		{
			name: "no steeringdao address",
			msg: &types.MsgEndorseProposal{
				Endorser: endorserAcc,
			},
			expectedErr: "Steering DAO address is not set: function is disabled",
		},
		{
			name: "wrong endorser account",
			msg: &types.MsgEndorseProposal{
				Endorser: endorserAcc,
			},
			expectedErr:    "invalid authority; expected " + steeringDAOAcc + ", got " + endorserAcc + ": expected core DAO account as only signer for this message",
			setSteeringDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgEndorseProposal{
				Endorser:   steeringDAOAcc,
				ProposalId: 9999,
			},
			expectedErr:    "proposal with ID 9999 not found: unknown proposal",
			proposalState:  "none",
			setSteeringDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgEndorseProposal{
				Endorser: steeringDAOAcc,
			},
			proposalState:  "voting",
			setSteeringDAO: true,
			assertEndorsed: true,
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgEndorseProposal{
				Endorser: steeringDAOAcc,
			},
			expectedErr:    "is not in voting period: inactive proposal",
			proposalState:  "deposit",
			setSteeringDAO: true,
		},
		{
			name: "already endorsed proposal",
			msg: &types.MsgEndorseProposal{
				Endorser: steeringDAOAcc,
			},
			expectedErr:    "has already been endorsed: proposal already endorsed",
			proposalState:  "voting-endorsed",
			setSteeringDAO: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := helpers.Setup(t)
			ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})
			ms := coredaoskeeper.NewMsgServer(app.CoreDaosKeeper)

			params := types.DefaultParams()
			if tt.setSteeringDAO {
				params.SteeringDaoAddress = steeringDAOAcc
			}
			require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, params))

			switch tt.proposalState {
			case "voting":
				p := submitBankSendProposalReal(t, app, ctx, true)
				tt.msg.ProposalId = p.Id
			case "deposit":
				p := submitBankSendProposalReal(t, app, ctx, false)
				tt.msg.ProposalId = p.Id
			case "voting-endorsed":
				p := submitBankSendProposalReal(t, app, ctx, true)
				p.Endorsed = true
				require.NoError(t, app.GovKeeper.SetProposal(ctx, p))
				tt.msg.ProposalId = p.Id
			}

			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.EndorseProposal(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
				return
			}
			require.NoError(t, err)
			if tt.assertEndorsed {
				got, err := app.GovKeeper.Proposals.Get(ctx, tt.msg.ProposalId)
				require.NoError(t, err)
				require.True(t, got.Endorsed)
			}
		})
	}
}

func TestMsgServerExtendVotingPeriod(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(2)
	extenderAcc := testAcc[0].String()
	steeringDAOAcc := testAcc[1].String()

	tests := []struct {
		name           string
		msg            *types.MsgExtendVotingPeriod
		expectedErr    string
		proposalState  string
		setSteeringDAO bool
		assertExtended bool
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgExtendVotingPeriod{},
			expectedErr: "invalid extender address: empty address string is not allowed: invalid address",
		},
		{
			name: "wrong addr extender",
			msg: &types.MsgExtendVotingPeriod{
				Extender: "cosmosincorrectaddress",
			},
			expectedErr: "invalid extender address: decoding bech32 failed: invalid separator index -1: invalid address",
		},
		{
			name: "function disabled",
			msg: &types.MsgExtendVotingPeriod{
				Extender: extenderAcc,
			},
			expectedErr: "Steering DAO address and Oversight DAO address are not set: function is disabled",
		},
		{
			name: "wrong extender account",
			msg: &types.MsgExtendVotingPeriod{
				Extender: extenderAcc,
			},
			expectedErr:    "invalid authority; expected " + steeringDAOAcc + ", got " + extenderAcc + ": expected core DAO account as only signer for this message",
			setSteeringDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgExtendVotingPeriod{
				Extender:   steeringDAOAcc,
				ProposalId: 9999,
			},
			expectedErr:    "proposal with ID 9999 not found: unknown proposal",
			proposalState:  "none",
			setSteeringDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgExtendVotingPeriod{
				Extender: steeringDAOAcc,
			},
			proposalState:  "voting",
			setSteeringDAO: true,
			assertExtended: true,
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgExtendVotingPeriod{
				Extender: steeringDAOAcc,
			},
			expectedErr:    "is not in voting period: inactive proposal",
			proposalState:  "deposit",
			setSteeringDAO: true,
		},
		{
			name: "proposal cannot be extended",
			msg: &types.MsgExtendVotingPeriod{
				Extender: steeringDAOAcc,
			},
			expectedErr:    "has reached the maximum number of voting period extensions: invalid proposal content",
			proposalState:  "voting-maxed",
			setSteeringDAO: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := helpers.Setup(t)
			ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})
			ms := coredaoskeeper.NewMsgServer(app.CoreDaosKeeper)

			params := types.DefaultParams()
			if tt.setSteeringDAO {
				params.SteeringDaoAddress = steeringDAOAcc
			}
			require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, params))

			var origEndTime time.Time
			switch tt.proposalState {
			case "voting":
				p := submitBankSendProposalReal(t, app, ctx, true)
				origEndTime = *p.VotingEndTime
				tt.msg.ProposalId = p.Id
			case "deposit":
				p := submitBankSendProposalReal(t, app, ctx, false)
				tt.msg.ProposalId = p.Id
			case "voting-maxed":
				p := submitBankSendProposalReal(t, app, ctx, true)
				p.TimesVotingPeriodExtended = params.VotingPeriodExtensionsLimit
				require.NoError(t, app.GovKeeper.SetProposal(ctx, p))
				tt.msg.ProposalId = p.Id
			}

			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.ExtendVotingPeriod(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
				return
			}
			require.NoError(t, err)
			if tt.assertExtended {
				got, err := app.GovKeeper.Proposals.Get(ctx, tt.msg.ProposalId)
				require.NoError(t, err)
				expectedEndTime := origEndTime.Add(*params.VotingPeriodExtensionDuration)
				require.WithinDuration(t, expectedEndTime, *got.VotingEndTime, time.Second)
				require.Equal(t, uint32(1), got.TimesVotingPeriodExtended)
				// the proposal must have been re-queued under the new end time
				has, err := app.GovKeeper.ActiveProposalsQueue.Has(ctx, collectionsJoin(*got.VotingEndTime, got.Id))
				require.NoError(t, err)
				require.True(t, has)
			}
		})
	}
}

func TestMsgServerVetoProposal(t *testing.T) {
	testAcc := simtestutil.CreateRandomAccounts(3)
	vetoerAcc := testAcc[0].String()
	oversightDAOAcc := testAcc[1].String()
	newOversightAddr := testAcc[2].String()
	extDuration := time.Hour

	tests := []struct {
		name            string
		msg             *types.MsgVetoProposal
		expectedErr     string
		proposalState   string
		setOversightDAO bool
		assertVetoed    bool
	}{
		{
			name:        "empty msg",
			msg:         &types.MsgVetoProposal{},
			expectedErr: "invalid vetoer address: empty address string is not allowed: invalid address",
		},
		{
			name: "wrong addr vetoer",
			msg: &types.MsgVetoProposal{
				Vetoer: "cosmosincorrectaddress",
			},
			expectedErr: "invalid vetoer address: decoding bech32 failed: invalid separator index -1: invalid address",
		},
		{
			name: "function disabled",
			msg: &types.MsgVetoProposal{
				Vetoer: vetoerAcc,
			},
			expectedErr: "Oversight DAO address is not set: function is disabled",
		},
		{
			name: "wrong vetoer account",
			msg: &types.MsgVetoProposal{
				Vetoer: vetoerAcc,
			},
			expectedErr:     "invalid authority; expected " + oversightDAOAcc + ", got " + vetoerAcc + ": expected core DAO account as only signer for this message",
			setOversightDAO: true,
		},
		{
			name: "non existing proposal",
			msg: &types.MsgVetoProposal{
				Vetoer:     oversightDAOAcc,
				ProposalId: 9999,
			},
			expectedErr:     "proposal with ID 9999 not found: unknown proposal",
			proposalState:   "none",
			setOversightDAO: true,
		},
		{
			name: "ok",
			msg: &types.MsgVetoProposal{
				Vetoer: oversightDAOAcc,
			},
			proposalState:   "voting",
			setOversightDAO: true,
			assertVetoed:    true,
		},
		{
			name: "ok burn deposit",
			msg: &types.MsgVetoProposal{
				Vetoer:      oversightDAOAcc,
				BurnDeposit: true,
			},
			proposalState:   "voting",
			setOversightDAO: true,
			assertVetoed:    true,
		},
		{
			name: "proposal not in voting period",
			msg: &types.MsgVetoProposal{
				Vetoer: oversightDAOAcc,
			},
			expectedErr:     "is not in voting period: inactive proposal",
			proposalState:   "deposit",
			setOversightDAO: true,
		},
		{
			name: "veto proposal with change to oversight DAO address",
			msg: &types.MsgVetoProposal{
				Vetoer: oversightDAOAcc,
			},
			expectedErr:     "contains a change of the oversight DAO address, vetoing it would prevent the replacement of the current oversight DAO: oversight DAO cannot veto this proposal",
			proposalState:   "voting-change-oversight",
			setOversightDAO: true,
		},
		{
			name: "veto proposal with disablement of oversight DAO",
			msg: &types.MsgVetoProposal{
				Vetoer: oversightDAOAcc,
			},
			expectedErr:     "contains a change of the oversight DAO address, vetoing it would prevent the replacement of the current oversight DAO: oversight DAO cannot veto this proposal",
			proposalState:   "voting-disable-oversight",
			setOversightDAO: true,
		},
		{
			name: "veto proposal with change to oversight DAO address wrapped in authz.MsgExec",
			msg: &types.MsgVetoProposal{
				Vetoer: oversightDAOAcc,
			},
			expectedErr:     "contains a change of the oversight DAO address, vetoing it would prevent the replacement of the current oversight DAO: oversight DAO cannot veto this proposal",
			proposalState:   "voting-change-oversight-wrapped",
			setOversightDAO: true,
		},
		{
			name: "veto proposal with change to oversight DAO address double-wrapped in authz.MsgExec",
			msg: &types.MsgVetoProposal{
				Vetoer: oversightDAOAcc,
			},
			expectedErr:     "contains a change of the oversight DAO address, vetoing it would prevent the replacement of the current oversight DAO: oversight DAO cannot veto this proposal",
			proposalState:   "voting-change-oversight-double-wrapped",
			setOversightDAO: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := helpers.Setup(t)
			ctx := app.NewUncachedContext(true, tmproto.Header{Time: time.Now()})
			ms := coredaoskeeper.NewMsgServer(app.CoreDaosKeeper)

			params := types.DefaultParams()
			if tt.setOversightDAO {
				params.OversightDaoAddress = oversightDAOAcc
			}
			require.NoError(t, app.CoreDaosKeeper.Params.Set(ctx, params))

			govAddr := govModuleAddr()
			// a MsgUpdateParams that changes the oversight DAO address.
			changeOversightMsg := &types.MsgUpdateParams{
				Authority: govAddr,
				Params: types.Params{
					OversightDaoAddress:           newOversightAddr,
					VotingPeriodExtensionDuration: &extDuration,
				},
			}
			// a MsgUpdateParams that disables (empties) the oversight DAO address.
			disableOversightMsg := &types.MsgUpdateParams{
				Authority: govAddr,
				Params: types.Params{
					OversightDaoAddress:           "",
					VotingPeriodExtensionDuration: &extDuration,
				},
			}

			switch tt.proposalState {
			case "voting":
				p := submitBankSendProposalReal(t, app, ctx, true)
				tt.msg.ProposalId = p.Id
			case "deposit":
				p := submitBankSendProposalReal(t, app, ctx, false)
				tt.msg.ProposalId = p.Id
			case "voting-change-oversight":
				p := submitProposalReal(t, app, ctx, []sdk.Msg{changeOversightMsg}, true)
				tt.msg.ProposalId = p.Id
			case "voting-disable-oversight":
				p := submitProposalReal(t, app, ctx, []sdk.Msg{disableOversightMsg}, true)
				tt.msg.ProposalId = p.Id
			case "voting-change-oversight-wrapped":
				inner, err := codectypes.NewAnyWithValue(changeOversightMsg)
				require.NoError(t, err)
				exec := &authz.MsgExec{Grantee: govAddr, Msgs: []*codectypes.Any{inner}}
				p := submitProposalReal(t, app, ctx, []sdk.Msg{exec}, true)
				tt.msg.ProposalId = p.Id
			case "voting-change-oversight-double-wrapped":
				inner, err := codectypes.NewAnyWithValue(changeOversightMsg)
				require.NoError(t, err)
				execAny, err := codectypes.NewAnyWithValue(&authz.MsgExec{Grantee: govAddr, Msgs: []*codectypes.Any{inner}})
				require.NoError(t, err)
				outer := &authz.MsgExec{Grantee: govAddr, Msgs: []*codectypes.Any{execAny}}
				p := submitProposalReal(t, app, ctx, []sdk.Msg{outer}, true)
				tt.msg.ProposalId = p.Id
			}

			if err := tt.msg.ValidateBasic(); err != nil {
				if tt.expectedErr != "" {
					require.EqualError(t, err, tt.expectedErr)
					return
				}
				require.NoError(t, err)
			}
			_, err := ms.VetoProposal(ctx, tt.msg)
			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
				return
			}
			require.NoError(t, err)
			if tt.assertVetoed {
				got, err := app.GovKeeper.Proposals.Get(ctx, tt.msg.ProposalId)
				require.NoError(t, err)
				require.Equal(t, govv1.StatusVetoed, got.Status)
				// final tally must be reset to empty
				emptyTally := govv1.EmptyTallyResult()
				require.Equal(t, &emptyTally, got.FinalTallyResult)
				// voting ends immediately (set to block time)
				require.WithinDuration(t, ctx.BlockTime(), *got.VotingEndTime, time.Second)
				// proposal removed from active queue under its original end time
				has, err := app.GovKeeper.ActiveProposalsQueue.Has(ctx, collectionsJoin(*got.VotingEndTime, got.Id))
				require.NoError(t, err)
				require.False(t, has)
			}
		})
	}
}
