package keepers

import (
	icahost "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/host/keeper"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	atomoneante "github.com/atomone-hub/atomone/ante"
)

// govGateIBCModule wraps the ICA host IBC module to enforce governance rules
// on packet-delivered messages. The ante chain — where these rules are
// enforced for regular transactions — never runs for ICA host execution: the
// host module dispatches packet messages straight to the msg service router.
type govGateIBCModule struct {
	porttypes.IBCModule
	icaHostKeeper icahostkeeper.Keeper
	cdc           codec.Codec
	voteGate      atomoneante.GovVoteDecorator
}

// NewICAHostIBCModule returns the ICA host IBC module wrapped with the gov
// gate.
func NewICAHostIBCModule(
	icaHostKeeper icahostkeeper.Keeper,
	cdc codec.Codec,
	stakingKeeper *stakingkeeper.Keeper,
) porttypes.IBCModule {
	return govGateIBCModule{
		IBCModule:     icahost.NewIBCModule(icaHostKeeper),
		icaHostKeeper: icaHostKeeper,
		cdc:           cdc,
		voteGate:      atomoneante.NewGovVoteDecorator(cdc, stakingKeeper),
	}
}

// OnRecvPacket enforces the gov vote min-stake rule before handing the packet
// to the ICA host module. Without it, an unstaked interchain account could
// record governance votes end to end, bypassing the GovVoteDecorator.
func (m govGateIBCModule) OnRecvPacket(
	ctx sdk.Context,
	channelVersion string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	if err := m.validatePacketMsgs(ctx, packet); err != nil {
		ctx.Logger().Error("ICA host packet rejected by gov vote gate", "error", err.Error())
		ack := channeltypes.NewErrorAcknowledgement(err)
		icahostkeeper.EmitAcknowledgementEvent(ctx, packet, ack, err)
		return ack
	}
	return m.IBCModule.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

// validatePacketMsgs applies the gov vote min-stake rule to the messages
// carried by an EXECUTE_TX packet. Decoding mirrors the ICA host keeper, so
// if the packet cannot be decoded here it is passed through and the host
// keeper produces its own failure acknowledgement.
func (m govGateIBCModule) validatePacketMsgs(ctx sdk.Context, packet channeltypes.Packet) error {
	var data icatypes.InterchainAccountPacketData
	if err := data.UnmarshalJSON(packet.GetData()); err != nil {
		return nil
	}
	if data.Type != icatypes.EXECUTE_TX {
		return nil
	}

	appVersion, found := m.icaHostKeeper.GetAppVersion(ctx, packet.DestinationPort, packet.DestinationChannel)
	if !found {
		return nil
	}
	metadata, err := icatypes.MetadataFromVersion(appVersion)
	if err != nil {
		return nil
	}

	msgs, err := icatypes.DeserializeCosmosTx(m.cdc, data.Data, metadata.Encoding)
	if err != nil {
		return nil
	}

	return m.voteGate.ValidateVoteMsgs(ctx, msgs)
}
