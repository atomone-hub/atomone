package atomone_test

import (
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"

	dbm "github.com/cosmos/cosmos-db"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	dynamicfeetypes "github.com/cosmos/cosmos-sdk/x/dynamicfee/types"
	sdkgovv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibccoretypes "github.com/cosmos/ibc-go/v10/modules/core/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"

	atomoneante "github.com/atomone-hub/atomone/ante"
	atomoneapp "github.com/atomone-hub/atomone/app"
)

// atomoneAppCreator builds an AtomOneApp usable as an ibctesting chain.
func atomoneAppCreator() ibctesting.AppCreator {
	return func() (ibctesting.TestingApp, map[string]json.RawMessage) {
		app := atomoneapp.NewAtomOneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, atomoneapp.EmptyAppOptions{})
		genesis := app.DefaultGenesis()

		// ibctesting drives every handshake step as a zero-fee tx
		// (simapp.SignAndDeliver builds a 0-amount fee coin). The dynamicfee
		// module is enabled in atomone's default genesis with a 0.01 min gas
		// price, which would reject those txs before any vote-related code
		// runs. Disable it for this fixture: fee economics are orthogonal to
		// the vote gate under test.
		if raw, ok := genesis[dynamicfeetypes.ModuleName]; ok {
			var modGen map[string]json.RawMessage
			if err := json.Unmarshal(raw, &modGen); err == nil {
				if praw, ok := modGen["params"]; ok {
					var params dynamicfeetypes.Params
					if err := app.AppCodec().UnmarshalJSON(praw, &params); err == nil {
						params.Enabled = false
						modGen["params"] = app.AppCodec().MustMarshalJSON(&params)
						if out, err := json.Marshal(modGen); err == nil {
							genesis[dynamicfeetypes.ModuleName] = out
						}
					}
				}
			}
		}
		return app, genesis
	}
}

// TestICAGovVoteGate verifies that the min-stake voting rule, normally
// enforced by the ante GovVoteDecorator, also applies to messages delivered
// through the ICA host. The host module dispatches packet messages straight
// to the msg service router without running the ante chain, so without a
// gate on that path an unstaked interchain account can record governance
// votes end to end.
func TestICAGovVoteGate(t *testing.T) {
	coord := ibctesting.NewCustomAppCoordinator(t, 2, atomoneAppCreator())
	chainA := coord.GetChain(ibctesting.GetChainID(1)) // controller chain
	chainB := coord.GetChain(ibctesting.GetChainID(2)) // AtomOne host chain
	appA := chainA.App.(*atomoneapp.AtomOneApp)
	appB := chainB.App.(*atomoneapp.AtomOneApp)

	// 1. permissionless IBC setup: clients + connection. Host params are
	// left at the ibc-go defaults (HostEnabled=true, AllowMessages=["*"]).
	path := ibctesting.NewPath(chainA, chainB)
	path.SetupConnections()

	// 2. ICA channel handshake; registers an unstaked interchain account on the host
	version := string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: path.EndpointA.ConnectionID,
		HostConnectionId:       path.EndpointB.ConnectionID,
		Address:                "",
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))
	path.EndpointA.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointB.ChannelConfig.PortID = icatypes.HostPortID
	path.EndpointA.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointB.ChannelConfig.Order = channeltypes.ORDERED
	path.EndpointA.ChannelConfig.Version = version
	path.EndpointB.ChannelConfig.Version = version

	owner := chainA.SenderAccount.GetAddress().String()
	controllerPortID, err := icatypes.NewControllerPortID(owner)
	require.NoError(t, err)

	channelSequence := appA.GetIBCKeeper().ChannelKeeper.GetNextChannelSequence(chainA.GetContext())
	require.NoError(t, appA.ICAControllerKeeper.RegisterInterchainAccount(
		chainA.GetContext(), path.EndpointA.ConnectionID, owner, version, channeltypes.ORDERED))
	chainA.NextBlock()
	path.EndpointA.ChannelID = channeltypes.FormatChannelIdentifier(channelSequence)
	path.EndpointA.ChannelConfig.PortID = controllerPortID

	require.NoError(t, path.EndpointB.ChanOpenTry())
	require.NoError(t, path.EndpointA.ChanOpenAck())
	require.NoError(t, path.EndpointB.ChanOpenConfirm())

	icaAddrStr, found := appB.ICAHostKeeper.GetInterchainAccountAddress(
		chainB.GetContext(), path.EndpointB.ConnectionID, controllerPortID)
	require.True(t, found, "interchain account must be registered on the host during the handshake")
	icaAddr, err := sdk.AccAddressFromBech32(icaAddrStr)
	require.NoError(t, err)

	// the ICA account holds no delegations — unstaked by construction
	delegationCount := 0
	require.NoError(t, appB.StakingKeeper.IterateDelegatorDelegations(chainB.GetContext(), icaAddr, func(_ stakingtypes.Delegation) bool {
		delegationCount++
		return false
	}))
	require.Zero(t, delegationCount, "ICA voter must be unstaked for the test to be meaningful")

	// 3. active proposal in voting period on the host chain
	propMsg := banktypes.NewMsgSend(
		authtypes.NewModuleAddress("gov"),
		chainB.SenderAccount.GetAddress(),
		sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1000))),
	)
	proposal, err := sdkgovv1.NewProposal(
		[]sdk.Msg{propMsg}, 1,
		chainB.ProposedHeader.Time, chainB.ProposedHeader.Time.Add(time.Hour),
		"metadata", "gate test title", "gate test summary", icaAddr,
	)
	require.NoError(t, err)
	require.NoError(t, appB.GovKeeper.SetProposal(chainB.GetContext(), proposal))
	require.NoError(t, appB.GovKeeper.ActivateVotingPeriod(chainB.GetContext(), proposal))

	// sendAndRecv sends an EXECUTE_TX packet from the controller and delivers
	// it on the host as a real MsgRecvPacket transaction, through the IBC
	// router — i.e. through the wrapped ICA host stack.
	sendAndRecv := func(t *testing.T, msgs []proto.Message) *abci.ExecTxResult {
		data, err := icatypes.SerializeCosmosTx(appA.AppCodec(), msgs, icatypes.EncodingProtobuf)
		require.NoError(t, err)
		packetData := icatypes.InterchainAccountPacketData{Type: icatypes.EXECUTE_TX, Data: data}
		timeoutTimestamp := uint64(chainA.GetContext().BlockTime().Add(time.Hour).UnixNano())
		sequence, err := appA.ICAControllerKeeper.SendTx(
			chainA.GetContext(), path.EndpointA.ConnectionID, controllerPortID, packetData, timeoutTimestamp)
		require.NoError(t, err)
		coord.CommitBlock(chainA)
		require.NoError(t, path.EndpointB.UpdateClient())
		packet := channeltypes.NewPacket(
			packetData.GetBytes(), sequence,
			controllerPortID, path.EndpointA.ChannelID,
			icatypes.HostPortID, path.EndpointB.ChannelID,
			clienttypes.ZeroHeight(), timeoutTimestamp,
		)
		res, err := path.EndpointB.RecvPacketWithResult(packet)
		require.NoError(t, err)
		return res
	}

	ackFromEvents := func(t *testing.T, events []abci.Event) string {
		for _, e := range events {
			if e.Type != channeltypes.EventTypeWriteAck {
				continue
			}
			for _, attr := range e.Attributes {
				if attr.Key == channeltypes.AttributeKeyAckHex {
					ackBz, err := hex.DecodeString(attr.Value)
					require.NoError(t, err)
					return string(ackBz)
				}
			}
		}
		t.Fatal("write_acknowledgement event not found")
		return ""
	}

	// gateErrorFromEvents extracts the rejection reason the gate emitted as an
	// event. Rejected packets get a generic error acknowledgement (ibc-go
	// redacts error details from acknowledgements because they are written to
	// state), and the events of an unsuccessful packet carry the
	// ibccallbackerror- prefix on their types and attribute keys.
	gateErrorFromEvents := func(t *testing.T, events []abci.Event) string {
		for _, e := range events {
			if e.Type != ibccoretypes.ErrorAttributeKeyPrefix+icatypes.EventTypePacket {
				continue
			}
			for _, attr := range e.Attributes {
				if attr.Key == ibccoretypes.ErrorAttributeKeyPrefix+icatypes.AttributeKeyAckError {
					return attr.Value
				}
			}
		}
		t.Fatal("gate rejection event not found")
		return ""
	}

	hasVote := func(t *testing.T) bool {
		has, err := appB.GovKeeper.Votes.Has(chainB.GetContext(), collections.Join(uint64(1), icaAddr))
		require.NoError(t, err)
		return has
	}

	t.Run("control: unstaked vote via tx path is rejected by the ante", func(t *testing.T) {
		unstaked := chainB.SenderAccounts[7]
		unstakedSender := ibctesting.SenderAccount{SenderPrivKey: unstaked.SenderPrivKey, SenderAccount: unstaked.SenderAccount}
		_, err := chainB.SendMsgsWithSender(unstakedSender, &sdkgovv1.MsgVote{
			ProposalId: 1,
			Voter:      unstaked.SenderAccount.GetAddress().String(),
			Option:     sdkgovv1.OptionYes,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "insufficient stake for voting")
	})

	t.Run("unstaked ICA vote packet is rejected", func(t *testing.T) {
		res := sendAndRecv(t, []proto.Message{&sdkgovv1.MsgVote{
			ProposalId: 1,
			Voter:      icaAddrStr,
			Option:     sdkgovv1.OptionYes,
		}})
		require.Contains(t, ackFromEvents(t, res.Events), "error", "the packet must be rejected")
		require.Contains(t, gateErrorFromEvents(t, res.Events), "insufficient stake for voting")
		require.False(t, hasVote(t), "no vote may be recorded for the unstaked ICA")
	})

	t.Run("unstaked ICA weighted vote packet is rejected", func(t *testing.T) {
		res := sendAndRecv(t, []proto.Message{&sdkgovv1.MsgVoteWeighted{
			ProposalId: 1,
			Voter:      icaAddrStr,
			Options: []*sdkgovv1.WeightedVoteOption{
				{Option: sdkgovv1.OptionYes, Weight: "1.0"},
			},
		}})
		require.Contains(t, gateErrorFromEvents(t, res.Events), "insufficient stake for voting")
		require.False(t, hasVote(t), "no vote may be recorded for the unstaked ICA")
	})

	t.Run("non-vote ICA packet still executes", func(t *testing.T) {
		fund := banktypes.NewMsgSend(
			chainB.SenderAccount.GetAddress(),
			icaAddr,
			sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(5_000_000))),
		)
		_, err := chainB.SendMsgs(fund)
		require.NoError(t, err)

		send := banktypes.NewMsgSend(
			icaAddr,
			chainB.SenderAccount.GetAddress(),
			sdk.NewCoins(sdk.NewCoin("stake", math.NewInt(1_000))),
		)
		res := sendAndRecv(t, []proto.Message{send})
		ack := ackFromEvents(t, res.Events)
		require.NotContains(t, ack, "insufficient stake for voting", "non-vote packets must not be blocked by the gate")

		balance := appB.BankKeeper.GetBalance(chainB.GetContext(), icaAddr, "stake")
		require.Equal(t, math.NewInt(4_999_000), balance.Amount, "the packet must have executed")
	})

	t.Run("gate disabled: ICA vote is recorded (pass-through works)", func(t *testing.T) {
		atomoneante.SetMinStakedTokens(math.LegacyZeroDec())
		defer atomoneante.SetMinStakedTokens(math.LegacyNewDec(1000000))

		res := sendAndRecv(t, []proto.Message{&sdkgovv1.MsgVote{
			ProposalId: 1,
			Voter:      icaAddrStr,
			Option:     sdkgovv1.OptionYes,
		}})
		ack := ackFromEvents(t, res.Events)
		require.NotContains(t, ack, "insufficient stake for voting")
		require.True(t, hasVote(t), "the vote must be recorded once the min-stake rule is satisfied (disabled here)")
	})
}
