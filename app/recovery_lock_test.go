//go:build recoverylock_test

package atomone_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	icacontrollertypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	transferibcv2 "github.com/cosmos/ibc-go/v10/modules/apps/transfer/v2"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	channelv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	dynamicfeeante "github.com/cosmos/cosmos-sdk/x/dynamicfee/ante"
	dynamicfeetypes "github.com/cosmos/cosmos-sdk/x/dynamicfee/types"
	sdkgovmodule "github.com/cosmos/cosmos-sdk/x/gov"
	sdkgov "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	sdkgovbeta "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/atomone-hub/atomone/ante"
	atomone "github.com/atomone-hub/atomone/app"
	"github.com/atomone-hub/atomone/app/recoverylock"
	coretypes "github.com/atomone-hub/atomone/x/coredaos/types"
	govtypes "github.com/atomone-hub/atomone/x/gov/types/v1"
	govbeta "github.com/atomone-hub/atomone/x/gov/types/v1beta1"
	photonante "github.com/atomone-hub/atomone/x/photon/ante"
	photontypes "github.com/atomone-hub/atomone/x/photon/types"
)

type recoveryFixture struct {
	app                            *atomone.AtomOneApp
	ctx                            sdk.Context
	listed, recovery, payer, other sdk.AccAddress
	expiry                         time.Time
}

func newRecoveryFixture() recoveryFixture {
	app := atomone.NewAtomOneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, atomone.EmptyAppOptions{})
	cfg, chainID := recoverylock.MustLoad(nil, nil)
	height, expiry := cfg.Window()
	ctx := app.NewUncachedContext(false, tmproto.Header{ChainID: chainID, Height: height, Time: time.Unix(expiry-100, 0)})
	return recoveryFixture{
		app: app, ctx: ctx, listed: cfg.Listed()[0], recovery: cfg.Recovery(),
		payer: cfg.FeePayers()[0], other: sdk.AccAddress(make([]byte, 20)), expiry: time.Unix(expiry, 0),
	}
}

func (f recoveryFixture) ante(t *testing.T, ctx sdk.Context, payer, granter sdk.AccAddress, fee sdk.Coins, msgs ...sdk.Msg) error {
	t.Helper()
	builder := f.app.TxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	if len(payer) != 0 {
		builder.SetFeePayer(payer)
	}
	if len(granter) != 0 {
		builder.SetFeeGranter(granter)
	}
	builder.SetFeeAmount(fee)
	_, err := f.app.RecoveryLock.AnteHandle(ctx, builder.GetTx(), false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	return err
}

func photonFee() sdk.Coins {
	return sdk.NewCoins(sdk.NewInt64Coin(photontypes.Denom, 1000))
}

func TestRecoveryMessagePolicy(t *testing.T) {
	f := newRecoveryFixture()
	w, r, o := f.listed.String(), f.recovery.String(), f.other.String()
	v := sdk.ValAddress(f.listed).String()
	coins := sdk.NewCoins(sdk.NewInt64Coin("uatone", 1))
	for _, tc := range []struct {
		name  string
		msg   sdk.Msg
		allow bool
	}{
		{"SendATONE", &banktypes.MsgSend{FromAddress: w, ToAddress: r, Amount: coins}, true},
		{"SendPhoton", &banktypes.MsgSend{FromAddress: w, ToAddress: r, Amount: photonFee()}, true},
		{"SendVoucher", &banktypes.MsgSend{FromAddress: w, ToAddress: r, Amount: sdk.NewCoins(sdk.NewInt64Coin("ibc/0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF", 1))}, true},
		{"Incoming", &banktypes.MsgSend{FromAddress: o, ToAddress: w, Amount: coins}, true},
		{"AlternateRecipient", &banktypes.MsgSend{FromAddress: w, ToAddress: o, Amount: coins}, false},
		{"MultiSend", &banktypes.MsgMultiSend{Inputs: []banktypes.Input{{Address: w, Coins: coins}}, Outputs: []banktypes.Output{{Address: r, Coins: coins}}}, false},
		{"AuthzGrant", &authz.MsgGrant{Granter: w, Grantee: o}, false},
		{"AuthzRevoke", &authz.MsgRevoke{Granter: w, Grantee: o}, false},
		{"AuthzExec", &authz.MsgExec{Grantee: w}, false},
		{"FeeGrant", &feegrant.MsgGrantAllowance{Granter: w, Grantee: o}, false},
		{"FeeRevoke", &feegrant.MsgRevokeAllowance{Granter: w, Grantee: o}, false},
		{"IBCTransfer", &transfertypes.MsgTransfer{Sender: w, Receiver: o, Token: coins[0]}, false},
		{"IBC2SendPacket", &channelv2.MsgSendPacket{Signer: w}, false},
		{"ICARegister", &icacontrollertypes.MsgRegisterInterchainAccount{Owner: w}, false},
		{"ICASend", &icacontrollertypes.MsgSendTx{Owner: w}, false},
		{"GovSubmit", &govtypes.MsgSubmitProposal{Proposer: w, InitialDeposit: coins}, false},
		{"GovDeposit", &govtypes.MsgDeposit{Depositor: w, Amount: coins}, false},
		{"GovVote", &govtypes.MsgVote{Voter: w}, false},
		{"GovWeightedVote", &govtypes.MsgVoteWeighted{Voter: w}, false},
		{"GovLegacySubmit", &govbeta.MsgSubmitProposal{Proposer: w, InitialDeposit: coins}, false},
		{"GovLegacyDeposit", &govbeta.MsgDeposit{Depositor: w}, false},
		{"GovLegacyVote", &govbeta.MsgVote{Voter: w}, false},
		{"GovLegacyWeightedVote", &govbeta.MsgVoteWeighted{Voter: w}, false},
		{"SDKGovSubmit", &sdkgov.MsgSubmitProposal{Proposer: w}, false},
		{"SDKGovDeposit", &sdkgov.MsgDeposit{Depositor: w}, false},
		{"SDKGovVote", &sdkgov.MsgVote{Voter: w}, false},
		{"SDKGovLegacySubmit", &sdkgovbeta.MsgSubmitProposal{Proposer: w}, false},
		{"SDKGovLegacyDeposit", &sdkgovbeta.MsgDeposit{Depositor: w}, false},
		{"SDKGovLegacyVote", &sdkgovbeta.MsgVote{Voter: w}, false},
		{"LegacyContent", &govtypes.MsgExecLegacyContent{Authority: w}, false},
		{"Law", &govtypes.MsgProposeLaw{Authority: w}, false},
		{"Constitution", &govtypes.MsgProposeConstitutionAmendment{Authority: w}, false},
		{"CreateGovernor", &govtypes.MsgCreateGovernor{Address: w}, false},
		{"EditGovernor", &govtypes.MsgEditGovernor{Address: w}, false},
		{"GovernorStatus", &govtypes.MsgUpdateGovernorStatus{Address: w}, false},
		{"DelegateGovernor", &govtypes.MsgDelegateGovernor{DelegatorAddress: w}, false},
		{"UndelegateGovernor", &govtypes.MsgUndelegateGovernor{DelegatorAddress: w}, false},
		{"DAOAnnotate", &coretypes.MsgAnnotateProposal{Annotator: w}, false},
		{"DAOEndorse", &coretypes.MsgEndorseProposal{Endorser: w}, false},
		{"DAOExtend", &coretypes.MsgExtendVotingPeriod{Extender: w}, false},
		{"DAOVeto", &coretypes.MsgVetoProposal{Vetoer: w}, false},
		{"DAOParams", &coretypes.MsgUpdateParams{Authority: w}, false},
		{"MintPhoton", &photontypes.MsgMintPhoton{ToAddress: w}, false},
		{"UnlistedMintPhoton", &photontypes.MsgMintPhoton{ToAddress: o}, true},
		{"UnlistedIBCRelay", &channelv2.MsgRecvPacket{Signer: o}, true},
		{"DelegatorRewards", &distrtypes.MsgWithdrawDelegatorReward{DelegatorAddress: w, ValidatorAddress: v}, true},
		{"ValidatorCommission", &distrtypes.MsgWithdrawValidatorCommission{ValidatorAddress: v}, true},
		{"WithdrawAddress", &distrtypes.MsgSetWithdrawAddress{DelegatorAddress: w, WithdrawAddress: r}, true},
		{"CommunityPoolFunding", &distrtypes.MsgFundCommunityPool{Depositor: w}, false},
		{"CommunityPoolPayout", &distrtypes.MsgCommunityPoolSpend{Authority: o, Recipient: w}, true},
		{"Undelegate", &stakingtypes.MsgUndelegate{DelegatorAddress: w, ValidatorAddress: v}, true},
		{"CancelUnbonding", &stakingtypes.MsgCancelUnbondingDelegation{DelegatorAddress: w, ValidatorAddress: v}, true},
		{"Delegate", &stakingtypes.MsgDelegate{DelegatorAddress: w}, false},
		{"Redelegate", &stakingtypes.MsgBeginRedelegate{DelegatorAddress: w}, false},
		{"CreateValidator", &stakingtypes.MsgCreateValidator{ValidatorAddress: v}, false},
		{"EditValidator", &stakingtypes.MsgEditValidator{ValidatorAddress: v}, false},
		{"RotateValidatorKey", &stakingtypes.MsgRotateConsPubKey{ValidatorAddress: v}, false},
		{"Unjail", &slashingtypes.MsgUnjail{ValidatorAddr: v}, false},
		{"CreateVesting", &vestingtypes.MsgCreateVestingAccount{FromAddress: w}, false},
		{"CreatePeriodicVesting", &vestingtypes.MsgCreatePeriodicVestingAccount{FromAddress: w}, false},
		{"CreatePermanentLock", &vestingtypes.MsgCreatePermanentLockedAccount{FromAddress: w}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := f.ante(t, f.ctx, f.payer, nil, photonFee(), tc.msg)
			if tc.allow {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, recoverylock.ErrLocked)
			}
		})
	}
}

func TestRecoveryFeeAndWindow(t *testing.T) {
	f := newRecoveryFixture()
	send := &banktypes.MsgSend{FromAddress: f.listed.String(), ToAddress: f.recovery.String(), Amount: photonFee()}
	incoming := &banktypes.MsgSend{FromAddress: f.other.String(), ToAddress: f.listed.String(), Amount: photonFee()}
	for _, tc := range []struct {
		name           string
		payer, granter sdk.AccAddress
		fee            sdk.Coins
		msgs           []sdk.Msg
		allow          bool
	}{
		{"Approved", f.payer, nil, photonFee(), []sdk.Msg{send, send}, true},
		{"FallbackListed", nil, nil, photonFee(), []sdk.Msg{send}, false},
		{"WrongPayer", f.other, nil, photonFee(), []sdk.Msg{send}, false},
		{"ListedPayerWithUnlistedSender", f.listed, nil, photonFee(), []sdk.Msg{incoming}, false},
		{"ListedGranterWithUnlistedSender", f.payer, f.listed, photonFee(), []sdk.Msg{incoming}, false},
		{"UnlistedGranterWithListedSender", f.payer, f.other, photonFee(), []sdk.Msg{send}, false},
		{"UnrelatedFeeGrant", f.payer, f.other, photonFee(), []sdk.Msg{incoming}, true},
		{"MissingFee", f.payer, nil, nil, []sdk.Msg{send}, false},
		{"WrongDenom", f.payer, nil, sdk.NewCoins(sdk.NewInt64Coin("uatone", 1)), []sdk.Msg{send}, false},
		{"MixedBatch", f.payer, nil, photonFee(), []sdk.Msg{incoming, send}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := f.ante(t, f.ctx, tc.payer, tc.granter, tc.fee, tc.msgs...)
			if tc.allow {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, recoverylock.ErrLocked)
			}
		})
	}
	for _, tc := range []struct {
		name  string
		ctx   sdk.Context
		allow bool
	}{
		{"Before", f.ctx.WithBlockHeight(f.ctx.BlockHeight() - 1), true},
		{"FirstActive", f.ctx, false},
		{"CheckTxNextHeight", f.ctx.WithBlockHeight(f.ctx.BlockHeight() - 1).WithIsCheckTx(true), false},
		{"AtExpiry", f.ctx.WithBlockTime(f.expiry), true},
		{"OtherChain", f.ctx.WithChainID("other"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := f.ante(t, tc.ctx, f.listed, nil, nil, send)
			if tc.allow {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, recoverylock.ErrLocked)
			}
		})
	}
}

func TestRecoveryAuthzAndGovernanceDispatch(t *testing.T) {
	f := newRecoveryFixture()
	send := &banktypes.MsgSend{FromAddress: f.listed.String(), ToAddress: f.recovery.String(), Amount: photonFee()}
	inner := authz.NewMsgExec(f.other, []sdk.Msg{send})
	outer := authz.NewMsgExec(f.payer, []sdk.Msg{&inner})
	require.ErrorIs(t, f.ante(t, f.ctx, f.payer, nil, photonFee(), &outer), recoverylock.ErrLocked)
	malformed := &authz.MsgExec{Grantee: f.other.String(), Msgs: []*codectypes.Any{{TypeUrl: "/invalid"}}}
	require.ErrorIs(t, f.ante(t, f.ctx, f.payer, nil, photonFee(), malformed), sdkerrors.ErrInvalidRequest)
	require.NoError(t, f.app.AuthzKeeper.SaveGrant(f.ctx, f.other, f.listed, authz.NewGenericAuthorization(sdk.MsgTypeURL(send)), nil))
	_, err := f.app.AuthzKeeper.DispatchActions(f.ctx, f.other, []sdk.Msg{send})
	require.ErrorIs(t, err, recoverylock.ErrLocked)

	router := f.app.GovKeeper.Router()
	for _, handler := range []func(sdk.Context, sdk.Msg) (*sdk.Result, error){
		router.Handler(send), router.HandlerByTypeURL(sdk.MsgTypeURL(send)),
	} {
		require.NotNil(t, handler)
		_, err := handler(f.ctx, send)
		require.ErrorIs(t, err, recoverylock.ErrLocked)
	}
}

// nestedRecoveryExec builds a tree at the SDK depth limit and gives each
// executor the grants it needs to run with the lock inactive.
func (f recoveryFixture) nestedRecoveryExec(t *testing.T, signer, executor sdk.AccAddress) *authz.MsgExec {
	t.Helper()
	bank := f.app.BankKeeper
	require.NoError(t, bank.SetParams(f.ctx, banktypes.DefaultParams()))
	require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, photonFee()))
	require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, signer, photonFee()))
	send := &banktypes.MsgSend{FromAddress: signer.String(), ToAddress: f.recovery.String(), Amount: photonFee()}
	require.NoError(t, f.app.AuthzKeeper.SaveGrant(f.ctx, f.other, signer, authz.NewGenericAuthorization(sdk.MsgTypeURL(send)), nil))
	var msg sdk.Msg = send
	for range codectypes.MaxUnpackAnyRecursionDepth - 2 {
		exec := authz.NewMsgExec(f.other, []sdk.Msg{msg})
		msg = &exec
	}
	if !executor.Equals(f.other) {
		require.NoError(t, f.app.AuthzKeeper.SaveGrant(f.ctx, executor, f.other, authz.NewGenericAuthorization(sdk.MsgTypeURL(msg)), nil))
	}
	outer := authz.NewMsgExec(executor, []sdk.Msg{msg})
	return &outer
}

func TestRecoveryQueuedGovernanceNestedExec(t *testing.T) {
	for _, name := range []string{"listed", "unlisted"} {
		t.Run(name, func(t *testing.T) {
			f := newRecoveryFixture()
			signer := f.payer
			if name == "listed" {
				signer = f.listed
			}
			gov, staking, bank := f.app.GovKeeper, f.app.StakingKeeper, f.app.BankKeeper
			authority := gov.GetGovernanceAccount(f.ctx).GetAddress()
			exec := f.nestedRecoveryExec(t, signer, authority)
			votingEnd := f.ctx.BlockTime()
			votingStart := votingEnd.Add(-time.Second)
			before := f.ctx.WithBlockHeight(f.ctx.BlockHeight() - 1)
			queued := before.WithBlockTime(votingStart)

			// Give one voter all bonded stake so tallying reaches execution.
			stakingParams := stakingtypes.DefaultParams()
			require.NoError(t, staking.SetParams(queued, stakingParams))
			voter := f.payer
			operator := sdk.ValAddress(voter).String()
			val, err := stakingtypes.NewValidator(operator, &ed25519.PubKey{Key: make([]byte, ed25519.PubKeySize)}, stakingtypes.Description{})
			require.NoError(t, err)
			val.Status = stakingtypes.Bonded
			val.Tokens = sdk.DefaultPowerReduction
			val.DelegatorShares = math.LegacyNewDecFromInt(val.Tokens)
			require.NoError(t, staking.SetValidator(queued, val))
			require.NoError(t, staking.SetValidatorByPowerIndex(queued, val))
			require.NoError(t, staking.SetDelegation(queued, stakingtypes.NewDelegation(voter.String(), operator, val.DelegatorShares)))
			bond := sdk.NewCoins(sdk.NewCoin(stakingParams.BondDenom, val.Tokens))
			require.NoError(t, bank.MintCoins(queued, transfertypes.ModuleName, bond))
			require.NoError(t, bank.SendCoinsFromModuleToModule(queued, transfertypes.ModuleName, stakingtypes.BondedPoolName, bond))

			require.NoError(t, gov.Params.Set(queued, sdkgov.DefaultParams()))
			require.NoError(t, gov.ParticipationEMA.Set(queued, math.LegacyMustNewDecFromStr(sdkgov.DefaultParticipationEma)))
			const id uint64 = 1
			proposal, err := sdkgov.NewProposal([]sdk.Msg{exec}, id, votingStart, votingStart, "", "Nested execution", "Dispatch fixture", voter)
			require.NoError(t, err)
			proposal.Status = sdkgov.StatusVotingPeriod
			proposal.VotingStartTime = &votingStart
			proposal.VotingEndTime = &votingEnd
			require.NoError(t, gov.SetProposal(queued, proposal))
			queueKey := collections.Join(votingEnd, id)
			require.NoError(t, gov.ActiveProposalsQueue.Set(queued, queueKey, id))
			require.NoError(t, gov.AddVote(queued, id, voter, sdkgov.NewNonSplitVoteOption(sdkgov.OptionYes), ""))

			finishVoting := func(ctx sdk.Context) sdkgov.Proposal {
				t.Helper()
				require.NoError(t, sdkgovmodule.EndBlocker(ctx, gov))
				finished, err := gov.Proposals.Get(ctx, id)
				require.NoError(t, err)
				require.Equal(t, val.Tokens.String(), finished.FinalTallyResult.YesCount)
				queued, err := gov.ActiveProposalsQueue.Has(ctx, queueKey)
				require.NoError(t, err)
				require.False(t, queued)
				return finished
			}

			baseline, _ := before.CacheContext()
			want := finishVoting(baseline)
			require.Equal(t, sdkgov.StatusPassed, want.Status)
			require.Empty(t, bank.GetAllBalances(baseline, signer))
			require.Equal(t, photonFee(), bank.GetAllBalances(baseline, f.recovery))

			got := finishVoting(f.ctx)
			if signer.Equals(f.listed) {
				require.Equal(t, sdkgov.StatusFailed, got.Status)
				require.Contains(t, got.FailedReason, "listed accounts cannot act through internal dispatch")
				require.NotContains(t, got.FailedReason, "failed to execute message; message")
				require.Equal(t, photonFee(), bank.GetAllBalances(f.ctx, signer))
				require.Empty(t, bank.GetAllBalances(f.ctx, f.recovery))
			} else {
				require.Equal(t, want, got)
				require.Equal(t, bank.GetAllBalances(baseline, signer), bank.GetAllBalances(f.ctx, signer))
				require.Equal(t, bank.GetAllBalances(baseline, f.recovery), bank.GetAllBalances(f.ctx, f.recovery))
			}
		})
	}
}

func TestRecoveryICAHostNestedExec(t *testing.T) {
	for _, name := range []string{"listed", "unlisted"} {
		t.Run(name, func(t *testing.T) {
			f := newRecoveryFixture()
			signer := f.payer
			if name == "listed" {
				signer = f.listed
			}
			exec := f.nestedRecoveryExec(t, signer, f.other)
			const (
				port       = icatypes.HostPortID
				channel    = "channel-0"
				connection = "connection-0"
				controller = "icacontroller-fixture"
			)
			host, bank := f.app.ICAHostKeeper, f.app.BankKeeper
			host.SetParams(f.ctx, icahosttypes.Params{HostEnabled: true, AllowMessages: []string{sdk.MsgTypeURL(exec)}})
			host.SetInterchainAccountAddress(f.ctx, connection, controller, f.other.String())
			f.app.IBCKeeper.ChannelKeeper.SetChannel(f.ctx, port, channel, channeltypes.Channel{
				State:          channeltypes.OPEN,
				ConnectionHops: []string{connection},
				Version:        icatypes.NewDefaultMetadataString(connection, connection),
			})
			data, err := icatypes.SerializeCosmosTx(f.app.AppCodec(), []proto.Message{exec}, icatypes.EncodingProtobuf)
			require.NoError(t, err)
			packetData := icatypes.InterchainAccountPacketData{Type: icatypes.EXECUTE_TX, Data: data}
			packet := channeltypes.Packet{
				SourcePort: controller, DestinationPort: port, DestinationChannel: channel, Data: packetData.GetBytes(),
			}

			baseline, _ := f.ctx.WithBlockHeight(f.ctx.BlockHeight() - 1).CacheContext()
			want, err := host.OnRecvPacket(baseline, packet)
			require.NoError(t, err)
			require.NotEmpty(t, want)
			require.Empty(t, bank.GetAllBalances(baseline, signer))
			require.Equal(t, photonFee(), bank.GetAllBalances(baseline, f.recovery))

			got, err := host.OnRecvPacket(f.ctx, packet)
			if signer.Equals(f.listed) {
				require.ErrorIs(t, err, recoverylock.ErrLocked)
				require.ErrorContains(t, err, "listed accounts cannot act through internal dispatch")
				require.NotContains(t, err.Error(), "failed to execute message; message")
				require.Empty(t, got)
				require.Equal(t, photonFee(), bank.GetAllBalances(f.ctx, signer))
				require.Empty(t, bank.GetAllBalances(f.ctx, f.recovery))
			} else {
				require.NoError(t, err)
				require.Equal(t, want, got)
				require.Equal(t, bank.GetAllBalances(baseline, signer), bank.GetAllBalances(f.ctx, signer))
				require.Equal(t, bank.GetAllBalances(baseline, f.recovery), bank.GetAllBalances(f.ctx, f.recovery))
			}
		})
	}
}

func TestRecoveryEncodedAuthzDepth(t *testing.T) {
	f := newRecoveryFixture()
	maxDepth := codectypes.MaxUnpackAnyRecursionDepth
	maxCalls := codectypes.MaxUnpackAnySubCalls
	bank := f.app.BankKeeper
	require.NoError(t, bank.SetParams(f.ctx, banktypes.DefaultParams()))
	require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, photonFee()))
	require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, f.other, photonFee()))
	send := &banktypes.MsgSend{FromAddress: f.other.String(), ToAddress: f.recovery.String(), Amount: photonFee()}
	listedSend := &banktypes.MsgSend{FromAddress: f.listed.String(), ToAddress: f.recovery.String(), Amount: photonFee()}
	wrap := func(msg sdk.Msg, depth int) sdk.Msg {
		for range depth {
			exec := authz.NewMsgExec(f.other, []sdk.Msg{msg})
			msg = &exec
		}
		return msg
	}
	encodeTx := func(t *testing.T, msg sdk.Msg) []byte {
		t.Helper()
		cfg := f.app.TxConfig()
		builder := cfg.NewTxBuilder()
		require.NoError(t, builder.SetMsgs(msg))
		builder.SetFeeAmount(photonFee())
		tx, err := cfg.TxEncoder()(builder.GetTx())
		require.NoError(t, err)
		return tx
	}
	decodeInternal := func(msg sdk.Msg) ([]sdk.Msg, error) {
		cdc := f.app.AppCodec()
		data, err := icatypes.SerializeCosmosTx(cdc, []proto.Message{msg}, icatypes.EncodingProtobuf)
		if err != nil {
			return nil, err
		}
		return icatypes.DeserializeCosmosTx(cdc, data, icatypes.EncodingProtobuf)
	}
	t.Run("transaction ante", func(t *testing.T) {
		decode := f.app.TxConfig().TxDecoder()
		tx, err := decode(encodeTx(t, wrap(send, maxDepth-2)))
		require.NoError(t, err)
		// Exercise the stock depth-sensitive decorators without signing a transaction.
		stock := sdk.ChainAnteDecorators(
			ante.NewGovVoteDecorator(f.app.AppCodec(), f.app.StakingKeeper),
			ante.NewGovSubmitProposalDecorator(f.app.AppCodec(), f.app.CoreDaosKeeper),
		)
		admitted := false
		next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			admitted = true
			return stock(ctx, tx, simulate)
		}
		_, err = f.app.RecoveryLock.AnteHandle(f.ctx.WithBlockHeight(f.ctx.BlockHeight()-1), tx, false, next)
		require.NoError(t, err)
		require.True(t, admitted)
		admitted = false
		_, err = f.app.RecoveryLock.AnteHandle(f.ctx, tx, false, next)
		require.NoError(t, err)
		require.True(t, admitted)

		tx, err = decode(encodeTx(t, wrap(listedSend, maxDepth-2)))
		require.NoError(t, err)
		admitted = false
		_, err = f.app.RecoveryLock.AnteHandle(f.ctx, tx, false, next)
		require.ErrorIs(t, err, recoverylock.ErrLocked)
		require.ErrorContains(t, err, "transaction must contain only direct recovery messages from listed accounts")
		require.False(t, admitted)
		// The stock governance decorators impose their own depth cap.
		tx, err = decode(encodeTx(t, wrap(send, maxDepth-1)))
		require.NoError(t, err)
		admitted = false
		_, err = f.app.RecoveryLock.AnteHandle(f.ctx, tx, false, next)
		require.True(t, admitted)
		require.ErrorContains(t, err, "authz nesting depth exceeded")
		_, err = decode(encodeTx(t, wrap(send, maxDepth)))
		require.ErrorContains(t, err, "max depth exceeded")
	})
	t.Run("internal dispatch", func(t *testing.T) {
		msgs, err := decodeInternal(wrap(send, maxDepth-1))
		require.NoError(t, err)
		baseline, _ := f.ctx.WithBlockHeight(f.ctx.BlockHeight() - 1).CacheContext()
		baseline = baseline.WithGasMeter(storetypes.NewInfiniteGasMeter())
		want, err := f.app.AuthzKeeper.DispatchActions(baseline, f.other, msgs)
		require.NoError(t, err)
		active, _ := f.ctx.CacheContext()
		active = active.WithGasMeter(storetypes.NewInfiniteGasMeter())
		got, err := f.app.AuthzKeeper.DispatchActions(active, f.other, msgs)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Equal(t, baseline.GasMeter().GasConsumed(), active.GasMeter().GasConsumed())
		require.Equal(t, photonFee(), bank.GetAllBalances(active, f.recovery))

		msgs, err = decodeInternal(wrap(listedSend, maxDepth-1))
		require.NoError(t, err)
		_, err = f.app.AuthzKeeper.DispatchActions(f.ctx, f.other, msgs)
		require.ErrorIs(t, err, recoverylock.ErrLocked)
		require.ErrorContains(t, err, "listed accounts cannot act through internal dispatch")
		_, err = decodeInternal(wrap(send, maxDepth))
		require.ErrorContains(t, err, "max depth exceeded")
	})
	t.Run("decoder subcall budget", func(t *testing.T) {
		leaves := make([]sdk.Msg, maxCalls)
		for i := range leaves {
			leaves[i] = send
		}
		exec := authz.NewMsgExec(f.other, leaves[:maxCalls-1])
		tx, err := f.app.TxConfig().TxDecoder()(encodeTx(t, &exec))
		require.NoError(t, err)
		_, err = f.app.RecoveryLock.AnteHandle(f.ctx, tx, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			return ctx, nil
		})
		require.NoError(t, err)
		_, err = decodeInternal(&exec)
		require.NoError(t, err)
		exec = authz.NewMsgExec(f.other, leaves)
		_, err = f.app.TxConfig().TxDecoder()(encodeTx(t, &exec))
		require.ErrorContains(t, err, "call limit exceeded")
		_, err = decodeInternal(&exec)
		require.ErrorContains(t, err, "call limit exceeded")
	})
	t.Run("cached in-memory bounds", func(t *testing.T) {
		require.NoError(t, f.ante(t, f.ctx, f.payer, nil, photonFee(), wrap(send, maxDepth-1)))
		err := f.ante(t, f.ctx, f.payer, nil, photonFee(), wrap(send, maxDepth))
		require.ErrorIs(t, err, recoverylock.ErrLocked)
		require.ErrorContains(t, err, "authz nesting exceeds the recovery limit")

		leaves := make([]sdk.Msg, maxCalls/2)
		for i := range leaves {
			leaves[i] = send
		}
		left := authz.NewMsgExec(f.other, leaves)
		// The root and both branch wrappers consume three calls.
		rightLeaves := maxCalls - len(leaves) - 3
		right := authz.NewMsgExec(f.other, leaves[:rightLeaves])
		exec := authz.NewMsgExec(f.other, []sdk.Msg{&left, &right})
		require.NoError(t, f.ante(t, f.ctx, f.payer, nil, photonFee(), &exec))

		right = authz.NewMsgExec(f.other, leaves[:rightLeaves+1])
		exec = authz.NewMsgExec(f.other, []sdk.Msg{&left, &right})
		err = f.ante(t, f.ctx, f.payer, nil, photonFee(), &exec)
		require.ErrorIs(t, err, recoverylock.ErrLocked)
		require.ErrorContains(t, err, "authz message count exceeds the recovery limit")
	})
}

func TestRecoveryBankDebitsAndRefunds(t *testing.T) {
	f := newRecoveryFixture()
	bank := f.app.BankKeeper
	require.NoError(t, bank.SetParams(f.ctx, banktypes.DefaultParams()))
	module := transfertypes.ModuleName
	voucher := sdk.NewInt64Coin("ibc/0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF", 10)
	native := sdk.NewInt64Coin("uatone", 10)
	mint := sdk.NewCoins(voucher, native)
	require.NoError(t, bank.MintCoins(f.ctx, module, mint))
	require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, module, f.listed, mint))

	// The transfer module uses SendCoins for native escrow and
	// SendCoinsFromAccountToModule for vouchers before burning them.
	require.ErrorIs(t, bank.SendCoins(f.ctx, f.listed, f.other, sdk.NewCoins(native)), recoverylock.ErrLocked)
	require.ErrorIs(t, bank.SendCoinsFromAccountToModule(f.ctx, f.listed, module, sdk.NewCoins(voucher)), recoverylock.ErrLocked)
	require.ErrorIs(t, bank.DelegateCoinsFromAccountToModule(f.ctx, f.listed, stakingtypes.BondedPoolName, sdk.NewCoins(native)), recoverylock.ErrLocked)
	require.ErrorIs(t, bank.DelegateCoins(f.ctx, f.listed, f.other, sdk.NewCoins(native)), recoverylock.ErrLocked)
	require.ErrorIs(t, bank.InputOutputCoins(f.ctx, banktypes.Input{Address: f.listed.String(), Coins: mint}, []banktypes.Output{{Address: f.recovery.String(), Coins: mint}}), recoverylock.ErrLocked)
	require.Equal(t, mint, bank.GetAllBalances(f.ctx, f.listed))
	require.Empty(t, bank.GetAllBalances(f.ctx, f.other))
	require.Empty(t, bank.GetAllBalances(f.ctx, f.recovery))

	alternate := &banktypes.MsgSend{FromAddress: f.listed.String(), ToAddress: f.other.String(), Amount: mint}
	cache, _ := f.ctx.CacheContext()
	_, err := f.app.MsgServiceRouter().Handler(alternate)(cache, alternate)
	require.ErrorIs(t, err, recoverylock.ErrLocked)
	require.Equal(t, mint, bank.GetAllBalances(f.ctx, f.listed))

	send := &banktypes.MsgSend{FromAddress: f.listed.String(), ToAddress: f.recovery.String(), Amount: mint}
	_, err = f.app.MsgServiceRouter().Handler(send)(f.ctx, send)
	require.NoError(t, err)
	require.Equal(t, mint, bank.GetAllBalances(f.ctx, f.recovery))
	require.Empty(t, bank.GetAllBalances(f.ctx, f.listed))
	require.NoError(t, bank.SendCoins(f.ctx, f.recovery, f.listed, sdk.NewCoins(native)))
	require.Equal(t, math.NewInt(10), bank.GetBalance(f.ctx, f.listed, "uatone").Amount)
}

func TestRecoveryExpiryDoesNotHaltPreBlocker(t *testing.T) {
	for _, tc := range []struct {
		name   string
		offset time.Duration
	}{
		{"too little time for two unbonding periods", -time.Second},
		{"at expiry", 0},
		{"past expiry", time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRecoveryFixture()
			require.NoError(t, f.app.StakingKeeper.SetParams(f.ctx, stakingtypes.DefaultParams()))
			_, err := f.app.PreBlocker(f.ctx.WithBlockTime(f.expiry.Add(tc.offset)), nil)
			require.NoError(t, err)
		})
	}
}

func TestRecoveryNoBulkWithdrawResetAtActivation(t *testing.T) {
	f := newRecoveryFixture()
	require.NoError(t, f.app.DistrKeeper.SetDelegatorWithdrawAddr(f.ctx, f.listed, f.other))
	_, err := f.app.PreBlocker(f.ctx, nil)
	require.NoError(t, err)
	withdraw, err := f.app.DistrKeeper.GetDelegatorWithdrawAddr(f.ctx, f.listed)
	require.NoError(t, err)
	require.Equal(t, f.other, withdraw)
}

func TestRecoveryWithdrawAddressAfterActivation(t *testing.T) {
	for _, name := range []string{"delegator reward", "validator commission", "undelegate", "cancel unbonding"} {
		t.Run(name, func(t *testing.T) {
			db := dbm.NewMemDB()
			openApp := func() *atomone.AtomOneApp {
				return atomone.NewAtomOneApp(log.NewNopLogger(), db, nil, true, atomone.EmptyAppOptions{})
			}
			f := newRecoveryFixture()
			f.app = openApp()
			header := f.ctx.BlockHeader()
			ctx := f.app.NewUncachedContext(false, header)
			store := f.app.CommitMultiStore()
			require.NoError(t, store.SetInitialVersion(header.Height))
			require.NoError(t, f.app.DistrKeeper.SetDelegatorWithdrawAddr(ctx, f.listed, f.other))
			store.Commit()

			f.app = openApp()
			require.Equal(t, header.Height, f.app.LastBlockHeight())
			header.Height++
			ctx = f.app.NewUncachedContext(false, header)
			val := sdk.ValAddress(f.listed).String()
			var msg sdk.Msg
			switch name {
			case "delegator reward":
				msg = &distrtypes.MsgWithdrawDelegatorReward{DelegatorAddress: f.listed.String(), ValidatorAddress: val}
			case "validator commission":
				msg = &distrtypes.MsgWithdrawValidatorCommission{ValidatorAddress: val}
			case "undelegate":
				msg = &stakingtypes.MsgUndelegate{DelegatorAddress: f.listed.String(), ValidatorAddress: val}
			case "cancel unbonding":
				msg = &stakingtypes.MsgCancelUnbondingDelegation{DelegatorAddress: f.listed.String(), ValidatorAddress: val}
			}
			require.ErrorIs(t, f.ante(t, ctx, f.payer, nil, photonFee(), msg), recoverylock.ErrLocked)

			repair := distrtypes.NewMsgSetWithdrawAddress(f.listed, f.recovery)
			require.NoError(t, f.ante(t, ctx, f.payer, nil, photonFee(), repair))
			require.ErrorIs(t, f.ante(t, ctx, f.payer, nil, photonFee(), distrtypes.NewMsgSetWithdrawAddress(f.listed, f.other)), recoverylock.ErrLocked)
			require.NoError(t, f.app.DistrKeeper.Params.Set(ctx, distrtypes.DefaultParams()))
			_, err := f.app.MsgServiceRouter().Handler(repair)(ctx, repair)
			require.NoError(t, err)
			require.NoError(t, f.ante(t, ctx, f.payer, nil, photonFee(), msg))
		})
	}
}

func TestRecoveryAutomaticRewardPayouts(t *testing.T) {
	for _, path := range []string{"validator removal", "redelegation slash"} {
		for _, destination := range []string{"unsafe", "recovery", "unlisted", "before activation", "expired"} {
			t.Run(path+"/"+destination, func(t *testing.T) {
				f := newRecoveryFixture()
				owner, withdraw, want := f.listed, f.other, f.listed
				ctx := f.ctx.WithBlockHeight(f.ctx.BlockHeight() + 1)
				switch destination {
				case "recovery":
					withdraw, want = f.recovery, f.recovery
				case "unlisted":
					owner, withdraw, want = f.other, f.payer, f.payer
				case "before activation":
					ctx, want = f.ctx.WithBlockHeight(f.ctx.BlockHeight()-1), withdraw
				case "expired":
					ctx, want = ctx.WithBlockTime(f.expiry), withdraw
				}
				staking, distr, bank := f.app.StakingKeeper, f.app.DistrKeeper, f.app.BankKeeper
				valAddr := sdk.ValAddress(owner)
				if path == "redelegation slash" {
					valAddr = sdk.ValAddress(f.recovery)
				}
				val, err := stakingtypes.NewValidator(valAddr.String(), &ed25519.PubKey{Key: make([]byte, ed25519.PubKeySize)}, stakingtypes.Description{})
				require.NoError(t, err)
				require.NoError(t, staking.SetParams(ctx, stakingtypes.DefaultParams()))
				require.NoError(t, staking.SetValidator(ctx, val))
				require.NoError(t, distr.FeePool.Set(ctx, distrtypes.InitialFeePool()))
				require.NoError(t, distr.Hooks().AfterValidatorCreated(ctx, valAddr))
				require.NoError(t, distr.SetDelegatorWithdrawAddr(ctx, owner, withdraw))

				reward := sdk.NewCoins(sdk.NewInt64Coin("stake", 20))
				require.NoError(t, bank.MintCoins(ctx, transfertypes.ModuleName, reward))
				require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, transfertypes.ModuleName, distrtypes.ModuleName, reward))
				decReward := sdk.NewDecCoinsFromCoins(reward...)
				if path == "validator removal" {
					require.NoError(t, distr.SetValidatorOutstandingRewards(ctx, valAddr, distrtypes.ValidatorOutstandingRewards{Rewards: decReward}))
					require.NoError(t, distr.SetValidatorAccumulatedCommission(ctx, valAddr, distrtypes.ValidatorAccumulatedCommission{Commission: decReward}))
					require.NoError(t, staking.RemoveValidator(ctx, valAddr))
				} else {
					stake := sdk.NewCoins(sdk.NewInt64Coin("stake", 100))
					require.NoError(t, bank.MintCoins(ctx, transfertypes.ModuleName, stake))
					require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, transfertypes.ModuleName, stakingtypes.NotBondedPoolName, stake))
					// Seed the existing delegation without going through an active lock.
					val.Tokens, val.DelegatorShares = stake[0].Amount, math.LegacyNewDec(100)
					require.NoError(t, staking.SetValidator(ctx, val))
					require.NoError(t, distr.Hooks().BeforeDelegationCreated(ctx, owner, valAddr))
					require.NoError(t, staking.SetDelegation(ctx, stakingtypes.NewDelegation(owner.String(), valAddr.String(), val.DelegatorShares)))
					require.NoError(t, distr.Hooks().AfterDelegationModified(ctx.WithBlockHeight(ctx.BlockHeight()-1), owner, valAddr))
					require.NoError(t, distr.AllocateTokensToValidator(ctx, val, decReward))
					redelegation := stakingtypes.NewRedelegation(owner, sdk.ValAddress(f.payer), valAddr, ctx.BlockHeight(), ctx.BlockTime().Add(time.Hour),
						val.Tokens, val.DelegatorShares, 1, staking.ValidatorAddressCodec(), f.app.AccountKeeper.AddressCodec())
					slashed, err := staking.SlashRedelegation(ctx, val, redelegation, ctx.BlockHeight(), math.LegacyNewDecWithPrec(1, 1))
					require.NoError(t, err)
					require.Equal(t, math.NewInt(10), slashed)
				}

				require.Equal(t, reward[0].String(), bank.GetBalance(ctx, want, reward[0].Denom).String())
				if !want.Equals(withdraw) {
					require.True(t, bank.GetBalance(ctx, withdraw, reward[0].Denom).IsZero())
				}
				stored, err := distr.GetDelegatorWithdrawAddr(ctx.WithBlockTime(f.expiry), owner)
				require.NoError(t, err)
				require.Equal(t, want, stored)
			})
		}
	}
}

func TestRecoveryIBCRefunds(t *testing.T) {
	for _, channel := range []struct {
		name, source string
		v2           bool
	}{
		{"ICS20", "channel-0", false},
		{"IBCv2", "07-tendermint-0", true},
	} {
		for _, voucher := range []bool{false, true} {
			label := "/Native"
			if voucher {
				label = "/Voucher"
			}
			t.Run(channel.name+label, func(t *testing.T) {
				f := newRecoveryFixture()
				bank := f.app.BankKeeper
				transfer := f.app.TransferKeeper
				require.NoError(t, bank.SetParams(f.ctx, banktypes.DefaultParams()))
				transfer.SetParams(f.ctx, transfertypes.DefaultParams())
				port := transfertypes.PortID
				denom := transfertypes.NewDenom("uatone")
				if voucher {
					denom = transfertypes.NewDenom("uatone", transfertypes.NewHop(port, channel.source))
				}
				token := transfertypes.Token{Denom: denom, Amount: "10"}
				coin, err := token.ToCoin()
				require.NoError(t, err)
				coins := sdk.NewCoins(coin.Add(coin))
				require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, coins))
				require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, f.listed, coins))

				data := transfertypes.NewFungibleTokenPacketData(denom.Path(), token.Amount, f.listed.String(), f.other.String(), "")
				payload := channelv2.Payload{
					SourcePort: port, DestinationPort: port, Version: transfertypes.V1,
					Encoding: transfertypes.EncodingJSON, Value: data.GetBytes(),
				}
				v2 := transferibcv2.NewIBCModule(transfer)
				send := func(ctx sdk.Context, seq uint64) error {
					if channel.v2 {
						return v2.OnSendPacket(ctx, channel.source, "07-tendermint-1", seq, payload, f.listed)
					}
					return transfer.SendTransfer(ctx, port, channel.source, token, f.listed)
				}
				require.ErrorIs(t, send(f.ctx, 3), recoverylock.ErrLocked)
				require.Equal(t, coins, bank.GetAllBalances(f.ctx, f.listed))
				before := f.ctx.WithBlockHeight(f.ctx.BlockHeight() - 1)
				require.NoError(t, send(before, 1))
				require.NoError(t, send(before, 2))
				require.Empty(t, bank.GetAllBalances(f.ctx, f.listed))
				if channel.v2 {
					require.NoError(t, v2.OnTimeoutPacket(f.ctx, channel.source, "07-tendermint-1", 1, payload, f.other))
					require.NoError(t, v2.OnAcknowledgementPacket(f.ctx, channel.source, "07-tendermint-1", 2, channelv2.ErrorAcknowledgement[:], payload, f.other))
				} else {
					packet, err := transfertypes.PacketDataV1ToV2(data)
					require.NoError(t, err)
					require.NoError(t, transfer.OnTimeoutPacket(f.ctx, port, channel.source, packet))
					ack := channeltypes.NewErrorAcknowledgement(transfertypes.ErrReceiveFailed)
					require.NoError(t, transfer.OnAcknowledgementPacket(f.ctx, port, channel.source, packet, ack))
				}
				require.Equal(t, coins, bank.GetAllBalances(f.ctx, f.listed))
			})
		}
	}
}

func TestRecoveryAbsentRoutes(t *testing.T) {
	router := newRecoveryFixture().app.MsgServiceRouter()
	for _, typeURL := range []string{
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"/cosmos.group.v1.MsgExec",
		"/cosmos.staking.v1beta1.MsgTokenizeShares",
		"/cosmos.staking.v1beta1.MsgRedeemTokensForShares",
		"/atomone.photon.v1.MsgBurnPhoton",
	} {
		require.Nil(t, router.HandlerByTypeURL(typeURL), typeURL)
	}
}

func TestRecoveryICAHostDispatch(t *testing.T) {
	f := newRecoveryFixture()
	host := f.app.ICAHostKeeper
	host.SetParams(f.ctx, icahosttypes.Params{HostEnabled: true, AllowMessages: []string{sdk.MsgTypeURL(&banktypes.MsgSend{})}})
	const (
		port       = "icahost"
		channel    = "channel-0"
		connection = "connection-0"
		controller = "icacontroller-fixture"
	)
	f.app.IBCKeeper.ChannelKeeper.SetChannel(f.ctx, port, channel, channeltypes.Channel{
		State:          channeltypes.OPEN,
		ConnectionHops: []string{connection},
		Version:        icatypes.NewDefaultMetadataString(connection, connection),
	})
	bank := f.app.BankKeeper
	require.NoError(t, bank.SetParams(f.ctx, banktypes.DefaultParams()))
	for _, addr := range []sdk.AccAddress{f.listed, f.other} {
		host.SetInterchainAccountAddress(f.ctx, connection, controller, addr.String())
		coins := photonFee()
		require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, coins))
		require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, addr, coins))
		send := &banktypes.MsgSend{FromAddress: addr.String(), ToAddress: f.recovery.String(), Amount: coins}
		data, err := icatypes.SerializeCosmosTx(f.app.AppCodec(), []proto.Message{send}, icatypes.EncodingProtobuf)
		require.NoError(t, err)
		packet := icatypes.InterchainAccountPacketData{Type: icatypes.EXECUTE_TX, Data: data}
		_, err = host.OnRecvPacket(f.ctx, channeltypes.Packet{
			SourcePort: controller, DestinationPort: port, DestinationChannel: channel, Data: packet.GetBytes(),
		})
		if addr.Equals(f.listed) {
			require.ErrorIs(t, err, recoverylock.ErrLocked)
			require.Equal(t, coins, bank.GetAllBalances(f.ctx, addr))
		} else {
			require.NoError(t, err)
			require.Empty(t, bank.GetAllBalances(f.ctx, addr))
		}
	}
	require.Equal(t, photonFee(), bank.GetAllBalances(f.ctx, f.recovery))
}

func TestRecoveryVestingBalance(t *testing.T) {
	f := newRecoveryFixture()
	bank := f.app.BankKeeper
	minted := sdk.NewCoins(sdk.NewInt64Coin("uatone", 10))
	locked := sdk.NewCoins(sdk.NewInt64Coin("uatone", 9))
	unlock := f.ctx.BlockTime().Add(time.Minute)
	base, ok := f.app.AccountKeeper.NewAccountWithAddress(f.ctx, f.listed).(*authtypes.BaseAccount)
	require.True(t, ok)
	account, err := vestingtypes.NewDelayedVestingAccount(base, locked, unlock.Unix())
	require.NoError(t, err)
	f.app.AccountKeeper.SetAccount(f.ctx, account)
	require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, minted))
	require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, f.listed, minted))

	require.ErrorIs(t, bank.SendCoins(f.ctx, f.listed, f.recovery, minted), sdkerrors.ErrInsufficientFunds)
	require.Equal(t, minted, bank.GetAllBalances(f.ctx, f.listed))
	require.NoError(t, bank.SendCoins(f.ctx, f.listed, f.recovery, minted.Sub(locked...)))
	require.Equal(t, locked, bank.GetAllBalances(f.ctx, f.listed))
	require.NoError(t, bank.SendCoins(f.ctx.WithBlockTime(unlock), f.listed, f.recovery, locked))
	require.Equal(t, minted, bank.GetAllBalances(f.ctx, f.recovery))
}

func TestRecoveryPhotonModuleDebit(t *testing.T) {
	f := newRecoveryFixture()
	bank := f.app.BankKeeper
	params := stakingtypes.DefaultParams()
	params.BondDenom = "uatone"
	require.NoError(t, f.app.StakingKeeper.SetParams(f.ctx, params))
	require.NoError(t, f.app.PhotonKeeper.SetParams(f.ctx, photontypes.DefaultParams()))

	stake := sdk.NewInt64Coin(params.BondDenom, 100)
	amount := sdk.NewInt64Coin(params.BondDenom, 1)
	for _, address := range []sdk.AccAddress{f.listed, f.other} {
		coins := sdk.NewCoins(stake)
		require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, coins))
		require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, address, coins))
	}
	listed := &photontypes.MsgMintPhoton{ToAddress: f.listed.String(), Amount: amount}
	require.ErrorIs(t, f.ante(t, f.ctx, f.payer, nil, nil, listed), recoverylock.ErrLocked)
	handler := f.app.MsgServiceRouter().Handler(listed)
	_, err := handler(f.ctx, listed)
	require.ErrorIs(t, err, recoverylock.ErrLocked)
	require.Equal(t, sdk.NewCoins(stake), bank.GetAllBalances(f.ctx, f.listed))
	require.True(t, bank.GetSupply(f.ctx, photontypes.Denom).IsZero())

	unlisted := &photontypes.MsgMintPhoton{ToAddress: f.other.String(), Amount: amount}
	require.NoError(t, f.ante(t, f.ctx, nil, nil, nil, unlisted))
	_, err = handler(f.ctx, unlisted)
	require.NoError(t, err)
	require.Equal(t, stake.Sub(amount), bank.GetBalance(f.ctx, f.other, params.BondDenom))
	require.True(t, bank.GetBalance(f.ctx, f.other, photontypes.Denom).IsPositive())
}

// Exercise the ante handler installed by NewAtomOneApp without signing or dispatch.
func TestRecoveryAppAnteWiring(t *testing.T) {
	f := newRecoveryFixture()
	bank, accounts := f.app.BankKeeper, f.app.AccountKeeper
	require.NoError(t, accounts.Params.Set(f.ctx, authtypes.DefaultParams()))
	params := dynamicfeetypes.DefaultParams()
	params.Enabled = false
	params.FeeDenom = photontypes.Denom
	require.NoError(t, f.app.DynamicfeeKeeper.SetParams(f.ctx, params))
	require.NoError(t, f.app.PhotonKeeper.SetParams(f.ctx, photontypes.DefaultParams()))

	for _, addr := range []sdk.AccAddress{f.listed, f.payer, f.other} {
		accounts.SetAccount(f.ctx, accounts.NewAccountWithAddress(f.ctx, addr))
		require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, photonFee()))
		require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, addr, photonFee()))
	}

	for _, tc := range []struct {
		name    string
		ctx     sdk.Context
		payer   sdk.AccAddress
		wantErr error
	}{
		{"BeforeActivation", f.ctx.WithBlockHeight(f.ctx.BlockHeight() - 1), f.other, nil},
		{"AtActivationWithSponsor", f.ctx, f.payer, nil},
		{"AtActivationWithUnapprovedSponsor", f.ctx, f.other, recoverylock.ErrLocked},
		{"AtExpiry", f.ctx.WithBlockTime(f.expiry), f.other, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := tc.ctx.CacheContext()
			tx := f.app.TxConfig().NewTxBuilder()
			tx.SetGasLimit(100_000)
			tx.SetFeeAmount(photonFee())
			tx.SetFeePayer(tc.payer)
			require.NoError(t, tx.SetMsgs(&banktypes.MsgSend{
				FromAddress: f.listed.String(), ToAddress: f.recovery.String(), Amount: photonFee(),
			}))
			placeholder := signing.SignatureV2{Data: &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT}}
			require.NoError(t, tx.SetSignatures(placeholder, placeholder))
			txBytes, err := f.app.TxConfig().TxEncoder()(tx.GetTx())
			require.NoError(t, err)
			decoded, err := f.app.TxConfig().TxDecoder()(txBytes)
			require.NoError(t, err)

			_, err = f.app.AnteHandler()(ctx.WithTxBytes(txBytes), decoded, true)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Equal(t, photonFee(), bank.GetAllBalances(ctx, tc.payer))
			} else {
				require.NoError(t, err)
				require.Empty(t, bank.GetAllBalances(ctx, tc.payer))
			}
		})
	}
}

func TestRecoveryFeeDeduction(t *testing.T) {
	for _, mode := range []struct {
		name    string
		dynamic bool
	}{
		{"Dynamic", true},
		{"Fallback", false},
	} {
		t.Run(mode.name, func(t *testing.T) {
			f := newRecoveryFixture()
			bank, accounts, grants := f.app.BankKeeper, f.app.AccountKeeper, f.app.FeeGrantKeeper
			params := dynamicfeetypes.DefaultParams()
			params.Enabled = mode.dynamic
			params.FeeDenom = photontypes.Denom
			require.NoError(t, f.app.DynamicfeeKeeper.SetParams(f.ctx, params))
			require.NoError(t, f.app.DynamicfeeKeeper.SetState(f.ctx, dynamicfeetypes.DefaultState()))
			require.NoError(t, f.app.PhotonKeeper.SetParams(f.ctx, photontypes.DefaultParams()))
			ante := sdk.ChainAnteDecorators(
				f.app.RecoveryLock,
				photonante.NewValidateFeeDecorator(f.app.PhotonKeeper),
				dynamicfeeante.NewDynamicfeeCheckDecorator(accounts, bank, grants, f.app.DynamicfeeKeeper,
					authante.NewDeductFeeDecorator(accounts, bank, grants, nil)),
			)

			fee := photonFee()
			balance := fee.Add(fee...)
			for _, addr := range []sdk.AccAddress{f.listed, f.payer, f.other} {
				require.NoError(t, bank.MintCoins(f.ctx, transfertypes.ModuleName, balance))
				require.NoError(t, bank.SendCoinsFromModuleToAccount(f.ctx, transfertypes.ModuleName, addr, balance))
			}
			require.NoError(t, grants.GrantAllowance(f.ctx, f.listed, f.other, &feegrant.BasicAllowance{SpendLimit: balance}))
			tx := f.app.TxConfig().NewTxBuilder()
			tx.SetGasLimit(50_000)
			tx.SetFeeAmount(fee)
			tx.SetFeePayer(f.other)
			tx.SetFeeGranter(f.listed)
			require.NoError(t, tx.SetMsgs(&banktypes.MsgSend{
				FromAddress: f.other.String(), ToAddress: f.recovery.String(), Amount: fee,
			}))
			_, err := ante(f.ctx, tx.GetTx(), false)
			require.ErrorIs(t, err, recoverylock.ErrLocked)
			grant, err := grants.GetAllowance(f.ctx, f.listed, f.other)
			require.NoError(t, err)
			require.Equal(t, &feegrant.BasicAllowance{SpendLimit: balance}, grant)
			require.Equal(t, balance, bank.GetAllBalances(f.ctx, f.listed))

			tx = f.app.TxConfig().NewTxBuilder()
			tx.SetGasLimit(50_000)
			tx.SetFeeAmount(fee)
			tx.SetFeePayer(f.payer)
			require.NoError(t, tx.SetMsgs(&banktypes.MsgSend{
				FromAddress: f.listed.String(), ToAddress: f.recovery.String(), Amount: fee,
			}))
			signable, ok := tx.GetTx().(authsigning.SigVerifiableTx)
			require.True(t, ok)
			signers, err := signable.GetSigners()
			require.NoError(t, err)
			require.Equal(t, [][]byte{f.listed, f.payer}, signers)

			if !mode.dynamic {
				t.Run("ZeroFee", func(t *testing.T) {
					// NewCoins would drop the explicit zero coin.
					tx.SetFeeAmount(sdk.Coins{sdk.NewInt64Coin(photontypes.Denom, 0)})
					_, err := ante(f.ctx, tx.GetTx(), false)
					require.ErrorIs(t, err, recoverylock.ErrLocked)
				})
				tx.SetFeeAmount(fee)
			}

			_, err = ante(f.ctx, tx.GetTx(), false)
			require.NoError(t, err)
			require.Equal(t, balance, bank.GetAllBalances(f.ctx, f.listed))
			require.Equal(t, fee, bank.GetAllBalances(f.ctx, f.payer))
			require.Equal(t, fee, bank.GetAllBalances(f.ctx, accounts.GetModuleAddress(authtypes.FeeCollectorName)))
		})
	}
}
