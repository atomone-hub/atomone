package spectre

import (
	"bytes"
	"context"
	"fmt"

	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"
	ibcerrors "github.com/cosmos/ibc-go/v10/modules/core/errors"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Keeper holds what the spectre module's message service needs: the codec,
// the IBC store provider to reach client stores, and the authority allowed to
// submit the governance message.
type Keeper struct {
	cdc           codec.BinaryCodec
	storeProvider clienttypes.StoreProvider
	authority     string
}

// NewKeeper creates the spectre module keeper.
func NewKeeper(cdc codec.BinaryCodec, storeProvider clienttypes.StoreProvider, authority string) *Keeper {
	return &Keeper{cdc: cdc, storeProvider: storeProvider, authority: authority}
}

// GetAuthority returns the address allowed to sign the module's messages.
func (k *Keeper) GetAuthority() string {
	return k.authority
}

// MigrateForkParameters validates and applies msg. It is only reachable
// through a governance proposal: the message's signer must be the authority.
func (k Keeper) MigrateForkParameters(goCtx context.Context, msg *MsgMigrateForkParameters) (*MsgMigrateForkParametersResponse, error) {
	if msg.Authority != k.authority {
		return nil, errorsmod.Wrapf(ibcerrors.ErrUnauthorized, "expected %s got %s", k.authority, msg.Authority)
	}
	if err := host.ClientIdentifierValidator(msg.ClientId); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	clientStore := k.storeProvider.ClientStore(ctx, msg.ClientId)
	cs, found := getClientState(ctx, clientStore, k.cdc)
	if !found {
		return nil, errorsmod.Wrap(clienttypes.ErrClientNotFound, msg.ClientId)
	}

	if err := cs.migrateForkParameters(ctx, clientStore, k.cdc, &msg.ForkParameters); err != nil {
		return nil, err
	}
	return &MsgMigrateForkParametersResponse{}, nil
}

// migrateForkParameters replaces the client's fork schedule in place, for an
// active client, before or after the fork's activation. A schedule a client
// is created with is otherwise fixed for its lifetime: no update message
// carries one, client recovery requires an identical schedule, and a stalled
// client stays active, so nothing else can ever change it. When a fork is
// scheduled after the client's creation (Gloas on mainnet, today), the client
// would stop verifying at its activation, permanently.
//
// The update is governance-trusted, and deliberately bound to nothing else:
// a wrong schedule can stop verification but cannot make an update verify
// false data, because updates still need a 2/3 sync committee signature over
// the chain's real fork domain and Merkle proofs into signed roots. The
// checks are the same as the Gno ethereum light client's MigrateForkParameters:
//
//   - the client must be active (frozen clients recover, not migrate);
//   - the schedule must actually change;
//   - the genesis fork version must stay the same;
//   - the client state must validate under the new schedule: fork epochs in
//     order, and the client's latest slot still in a supported fork (Electra
//     or later).
func (cs *ClientState) migrateForkParameters(
	ctx context.Context,
	clientStore storetypes.KVStore,
	cdc codec.BinaryCodec,
	newForkParameters *ForkParameters,
) error {
	if cs.Frozen {
		return errorsmod.Wrap(ErrFrozen, "a frozen client recovers, it does not migrate")
	}
	if forksEqual(&cs.ForkParameters, newForkParameters) {
		return fmt.Errorf("fork parameters do not change")
	}
	if !bytes.Equal(cs.ForkParameters.GenesisForkVersion, newForkParameters.GenesisForkVersion) {
		return errorsmod.Wrap(ErrInvalidClientState, "genesis fork version must stay the same")
	}

	updated := cs.Clone()
	updated.ForkParameters = *newForkParameters.Clone()
	if err := updated.Validate(); err != nil {
		return errorsmod.Wrap(err, "client state does not validate under the new fork schedule")
	}

	setClientState(ctx, clientStore, cdc, updated)
	return nil
}

// forksEqual reports whether two fork schedules are identical field by field.
func forksEqual(a, b *ForkParameters) bool {
	if !bytes.Equal(a.GenesisForkVersion, b.GenesisForkVersion) {
		return false
	}
	forkEqual := func(x, y Fork) bool {
		return x.Epoch == y.Epoch && bytes.Equal(x.Version, y.Version)
	}
	return forkEqual(a.Altair, b.Altair) &&
		forkEqual(a.Bellatrix, b.Bellatrix) &&
		forkEqual(a.Capella, b.Capella) &&
		forkEqual(a.Deneb, b.Deneb) &&
		forkEqual(a.Electra, b.Electra) &&
		forkEqual(a.Fulu, b.Fulu) &&
		forkEqual(a.Gloas, b.Gloas)
}
