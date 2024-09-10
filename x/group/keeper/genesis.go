package keeper

import (
	"encoding/json"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/atomone-hub/atomone/codec"
	"github.com/atomone-hub/atomone/types"
	"github.com/atomone-hub/atomone/x/group"
	"github.com/cosmos/cosmos-sdk/types/errors"
)

// InitGenesis initializes the group module's genesis state.
func (k Keeper) InitGenesis(ctx types.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState group.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)

	if err := k.groupTable.Import(ctx.KVStore(k.key), genesisState.Groups, genesisState.GroupSeq); err != nil {
		panic(errors.Wrap(err, "groups")) //nolint: staticcheck
	}

	if err := k.groupMemberTable.Import(ctx.KVStore(k.key), genesisState.GroupMembers, 0); err != nil {
		panic(errors.Wrap(err, "group members")) //nolint: staticcheck
	}

	if err := k.groupPolicyTable.Import(ctx.KVStore(k.key), genesisState.GroupPolicies, 0); err != nil {
		panic(errors.Wrap(err, "group policies")) //nolint: staticcheck
	}

	if err := k.groupPolicySeq.InitVal(ctx.KVStore(k.key), genesisState.GroupPolicySeq); err != nil {
		panic(errors.Wrap(err, "group policy account seq")) //nolint: staticcheck
	}

	if err := k.proposalTable.Import(ctx.KVStore(k.key), genesisState.Proposals, genesisState.ProposalSeq); err != nil {
		panic(errors.Wrap(err, "proposals")) //nolint: staticcheck
	}

	if err := k.voteTable.Import(ctx.KVStore(k.key), genesisState.Votes, 0); err != nil {
		panic(errors.Wrap(err, "votes")) //nolint: staticcheck
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the group module's exported genesis.
func (k Keeper) ExportGenesis(ctx types.Context, _ codec.JSONCodec) *group.GenesisState {
	genesisState := group.NewGenesisState()

	var groups []*group.GroupInfo

	groupSeq, err := k.groupTable.Export(ctx.KVStore(k.key), &groups)
	if err != nil {
		panic(errors.Wrap(err, "groups")) //nolint: staticcheck
	}
	genesisState.Groups = groups
	genesisState.GroupSeq = groupSeq

	var groupMembers []*group.GroupMember
	_, err = k.groupMemberTable.Export(ctx.KVStore(k.key), &groupMembers)
	if err != nil {
		panic(errors.Wrap(err, "group members")) //nolint: staticcheck
	}
	genesisState.GroupMembers = groupMembers

	var groupPolicies []*group.GroupPolicyInfo
	_, err = k.groupPolicyTable.Export(ctx.KVStore(k.key), &groupPolicies)
	if err != nil {
		panic(errors.Wrap(err, "group policies")) //nolint: staticcheck
	}
	genesisState.GroupPolicies = groupPolicies
	genesisState.GroupPolicySeq = k.groupPolicySeq.CurVal(ctx.KVStore(k.key))

	var proposals []*group.Proposal
	proposalSeq, err := k.proposalTable.Export(ctx.KVStore(k.key), &proposals)
	if err != nil {
		panic(errors.Wrap(err, "proposals")) //nolint: staticcheck
	}
	genesisState.Proposals = proposals
	genesisState.ProposalSeq = proposalSeq

	var votes []*group.Vote
	_, err = k.voteTable.Export(ctx.KVStore(k.key), &votes)
	if err != nil {
		panic(errors.Wrap(err, "votes")) //nolint: staticcheck
	}
	genesisState.Votes = votes

	return genesisState
}
