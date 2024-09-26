---
sidebar_position: 1
---

# `x/gov`

## Abstract

This paper specifies the Governance module for AtomOne, a fork
of the module of the Cosmos SDK, which was first described in the
[Cosmos Whitepaper](https://cosmos.network/about/whitepaper) in
June 2016.

The module enables Cosmos SDK based blockchain to support an on-chain governance
system. In this system, holders of the native staking token of the chain can vote
on proposals on a 1 token 1 vote basis. Next is a list of features the module
currently supports:

* **Proposal submission:** Users can submit proposals with a deposit. Once the
minimum deposit is reached, the proposal enters voting period. The minimum deposit
can be reached by collecting deposits from different users (including proposer) within deposit period.
* **Vote:** Participants can vote on proposals that reached `MinDeposit` and entered voting period.
* **Claiming deposit:** Users that deposited on proposals can recover their
deposits if the proposal was accepted or rejected. If the proposal never entered voting period
(minimum deposit not reached within deposit period), the deposit is burned.

This module is in use in [AtomOne](https://github.com/atomone-hub/atomone)).
Features that may be added in the future are described in [Future Improvements](#future-improvements).

## Contents

The following specification uses *ATONE* as the native staking token. The module
can be adapted to any Proof-Of-Stake blockchain by replacing *ATONE* with the native
staking token of the chain.


* [Concepts](#concepts)
    * [Proposal submission](#proposal-submission)
    * [Deposit](#deposit)
    * [Vote](#vote)
    * [Software Upgrade](#software-upgrade)
* [State](#state)
    * [Proposals](#proposals)
    * [Parameters and base types](#parameters-and-base-types)
    * [Deposit](#deposit-1)
    * [ValidatorGovInfo](#validatorgovinfo)
    * [Stores](#stores)
    * [Proposal Processing Queue](#proposal-processing-queue)
    * [Legacy Proposal](#legacy-proposal)
    * [Quorum Checks and Voting Period Extension](#quorum-checks-and-voting-period-extension)
    * [Constitution](#constitution)
    * [Law and Constitution Amendment Proposals](#law-and-constitution-amendment-proposals)
* [Messages](#messages)
    * [Proposal Submission](#proposal-submission-1)
    * [Deposit](#deposit-2)
    * [Vote](#vote-1)
* [Events](#events)
    * [EndBlocker](#endblocker)
    * [Handlers](#handlers)
* [Parameters](#parameters)
* [Client](#client)
    * [CLI](#cli)
    * [gRPC](#grpc)
    * [REST](#rest)
* [Metadata](#metadata)
    * [Proposal](#proposal-3)
    * [Vote](#vote-5)
* [Future Improvements](#future-improvements)

## Concepts

*Disclaimer: This is work in progress. Mechanisms are susceptible to change.*

The governance process is divided in a few steps that are outlined below:

* **Proposal submission:** Proposal is submitted to the blockchain with a
  deposit.
* **Vote:** Once deposit reaches a certain value (`MinDeposit`), proposal is
  confirmed and vote opens. Bonded Atone holders can then send `MsgVote`
  transactions to vote on the proposal.
* **Execution** After a period of time, the votes are tallied and depending
  on the result, the messages in the proposal will be executed.

### Proposal submission

#### Right to submit a proposal

Every account can submit proposals by sending a `MsgSubmitProposal` transaction.
Once a proposal is submitted, it is identified by its unique `proposalID`.

#### Proposal Messages

A proposal includes an array of `sdk.Msg`s which are executed automatically if the
proposal passes. The messages are executed by the governance `ModuleAccount` itself. Modules
such as `x/upgrade`, that want to allow certain messages to be executed by governance
only should add a whitelist within the respective msg server, granting the governance
module the right to execute the message once a quorum has been reached. The governance
module uses the `MsgServiceRouter` to check that these messages are correctly constructed
and have a respective path to execute on but do not perform a full validity check.

### Deposit

To prevent spam, proposals must be submitted with a deposit in the coins defined by
the `MinDeposit` param multiplied by the `MinInitialDepositRatio` param.

When a proposal is submitted, it has to be accompanied with a deposit that must be
greater than the `MinDeposit` multiplied by the `MinInitialDepositRatio` param,
but can be inferior to `MinDeposit` (since `MinDepositRatio` is a percentage).
The submitter doesn't need to pay for the entire deposit on their own. The newly
created proposal is stored in an *inactive proposal queue* and stays there until
its deposit passes the `MinDeposit`. Other token holders can increase the proposal's
deposit by sending a `Deposit` transaction. Deposits from token holders must always be
greater than `MinDeposit` multiplied by the `MinDepositRatio` param, or they will be
rejected. If a proposal doesn't pass the `MinDeposit` before the deposit end time
(the time when deposits are no longer accepted), the proposal will be destroyed: the
proposal will be removed from state and the deposit will be burned (see x/gov
`EndBlocker`). When a proposal deposit passes the `MinDeposit` threshold
(even during the proposal submission) before the deposit end time, the proposal will
be moved into the *active proposal queue* and the voting period will begin.

The deposit is kept in escrow and held by the governance `ModuleAccount` until the
proposal is finalized (passed or rejected).

#### Deposit refund

When a proposal is finalized, the coins from the deposit are refunded
regardless of wether the proposal is approved or rejected.
All refunded or burned deposits are removed from the state. Events are issued
when burning or refunding a deposit.

### Vote

#### Participants

*Participants* are users that have the right to vote on proposals. On the
AtomOne, participants are bonded Atone holders. Unbonded Atone holders and
other users do not get the right to participate in governance. However, they
can submit and deposit on proposals.

Note that when *participants* have bonded and unbonded Atones, their voting
power is calculated from their bonded Atone holdings only.

#### Voting period

Once a proposal reaches `MinDeposit`, it immediately enters `Voting period`. We
define `Voting period` as the interval between the moment the vote opens and
the moment the vote closes. `Voting period` should always be shorter than
`Unbonding period` to prevent double voting. The initial value of
`Voting period` is 3 weeks, which is also set as a hard lower bound.

#### Option set

The option set of a proposal refers to the set of choices a participant can
choose from when casting its vote.

The initial option set includes the following options:

* `Yes`
* `No`
* `Abstain`

`Abstain` option allows voters to signal that they do not intend to vote in
favor or against the proposal but accept the result of the vote.

#### Weighted Votes

[ADR-037](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-037-gov-split-vote.md)
introduces the weighted vote feature which allows a staker to split their votes
into several voting options. For example, it could use 70% of its voting power
to vote Yes and 30% of its voting power to vote No.

Often times the entity owning that address might not be a single individual.
For example, a company might have different stakeholders who want to vote
differently, and so it makes sense to allow them to split their
voting power. Currently, it is not possible for them to do "passthrough voting"
and giving their users voting rights over their tokens. However, with this system,
exchanges can poll their users for voting preferences, and then vote on-chain
proportionally to the results of the poll.

To represent weighted vote on chain, we use the following Protobuf message.

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/gov.proto#L27-L35
```

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/gov.proto#L134-L150
```

For a weighted vote to be valid, the `options` field must not contain duplicate
vote options, and the sum of weights of all options must be equal to 1.

### Quorum

Quorum is defined as the minimum percentage of voting power that needs to be
cast on a proposal for the result to be valid.

#### Threshold

Threshold is defined as the minimum proportion of `Yes` votes (excluding
`Abstain` votes) for the proposal to be accepted.

Initially, the threshold is set at 66.7% of `Yes` votes, excluding `Abstain`
votes. Note, the value is derived from the `TallyParams` on-chain parameter,
which is modifiable by governance. This means that proposals are accepted if:

* There exist bonded tokens.
* Quorum has been achieved.
* The proportion of `Abstain` votes is inferior to 1/1.
* The proportion of `Yes` votes, excluding `Abstain` votes, at the end of
  the voting period is superior to 2/3.

#### No inheritance

If a delegator does not vote, it won't inherit its validator vote.

Similarly, a validator's voting power is only equal to its own stake.

#### Validator’s punishment for non-voting

At present, validators are not punished for failing to vote.

#### Governance address

Later, we may add permissioned keys that could only sign txs from certain modules.
For the MVP, the `Governance address` will be the main validator address generated
at account creation. This address corresponds to a different PrivKey than the CometBFT
PrivKey which is responsible for signing consensus messages. Validators thus do not
have to sign governance transactions with the sensitive CometBFT PrivKey.

#### Burnable Params

There are three parameters that define if the deposit of a proposal should
be burned or returned to the depositors.

* `BurnVoteQuorum` burns the proposal deposit if the proposal deposit if the vote does not reach quorum.
* `BurnProposalDepositPrevote` burns the proposal deposit if it does not enter the voting phase.

> Note: These parameters are modifiable via governance.

## State

### Proposals

`Proposal` objects are used to tally votes and generally track the proposal's state.
They contain an array of arbitrary `sdk.Msg`'s which the governance module will attempt
to resolve and then execute if the proposal passes. `Proposal`'s are identified by a
unique id and contains a series of timestamps: `submit_time`, `deposit_end_time`,
`voting_start_time`, `voting_end_time` which track the lifecycle of a proposal

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/gov.proto#L51-L101
```

A proposal will generally require more than just a set of messages to explain its
purpose but need some greater justification and allow a means for interested participants
to discuss and debate the proposal.
In most cases, **it is encouraged to have an off-chain system that supports the on-chain governance process**.
To accommodate for this, a proposal contains a special **`metadata`** field, a string,
which can be used to add context to the proposal. The `metadata` field allows custom use for networks,
however, it is expected that the field contains a URL or some form of CID using a system such as
[IPFS](https://docs.ipfs.io/concepts/content-addressing/). To support the case of
interoperability across networks, the SDK recommends that the `metadata` represents
the following `JSON` template:

```json
{
  "title": "...",
  "description": "...",
  "forum": "...", // a link to the discussion platform (i.e. Discord)
  "other": "..." // any extra data that doesn't correspond to the other fields
}
```

This makes it far easier for clients to support multiple networks.

The metadata has a maximum length that is chosen by the app developer, and
passed into the gov keeper as a config. The default maximum length in the SDK is 255 characters.

#### Writing a module that uses governance

There are many aspects of a chain, or of the individual modules that you may want to
use governance to perform such as changing various parameters. This is very simple
to do. First, write out your message types and `MsgServer` implementation. Add an
`authority` field to the keeper which will be populated in the constructor with the
governance module account: `govKeeper.GetGovernanceAccount().GetAddress()`. Then for
the methods in the `msg_server.go`, perform a check on the message that the signer
matches `authority`. This will prevent any user from executing that message.

### Parameters and base types

`Parameters` define the rules according to which votes are run. There can only
be one active parameter set at any given time. If governance wants to change a
parameter set, either to modify a value or add/remove a parameter field, a new
parameter set has to be created and the previous one rendered inactive.

#### DepositParams

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/gov.proto#L167-L181
```

#### VotingParams

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/gov.proto#L183-L187
```

#### TallyParams

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/gov.proto#L189-L209
```

Parameters are stored in a global `GlobalParams` KVStore.

Additionally, we introduce some basic types:

```go
type Vote byte

const (
    VoteYes         = 0x1
    VoteAbstain     = 0x2
    VoteNo          = 0x3
)

type ProposalType  string

const (
    ProposalTypePlainText       = "Text"
    ProposalTypeSoftwareUpgrade = "SoftwareUpgrade"
)

type ProposalStatus byte


const (
    StatusNil           ProposalStatus = 0x00
    StatusDepositPeriod ProposalStatus = 0x01  // Proposal is submitted. Participants can deposit on it but not vote
    StatusVotingPeriod  ProposalStatus = 0x02  // MinDeposit is reached, participants can vote
    StatusPassed        ProposalStatus = 0x03  // Proposal passed and successfully executed
    StatusRejected      ProposalStatus = 0x04  // Proposal has been rejected
    StatusFailed        ProposalStatus = 0x05  // Proposal passed but failed execution
)
```

### Deposit

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/gov.proto#L37-L49
```

## Stores

:::note
Stores are KVStores in the multi-store. The key to find the store is the first parameter in the list
:::

We will use one KVStore `Governance` to store four mappings:

* A mapping from `proposalID|'proposal'` to `Proposal`.
* A mapping from `proposalID|'addresses'|address` to `Vote`. This mapping allows
  us to query all addresses that voted on the proposal along with their vote by
  doing a range query on `proposalID:addresses`.
* A mapping from `ParamsKey|'Params'` to `Params`. This map allows to query all
  x/gov params.
* A mapping from `VotingPeriodProposalKeyPrefix|proposalID` to a single byte. This allows
  us to know if a proposal is in the voting period or not with very low gas cost.

For pseudocode purposes, here are the two function we will use to read or write in stores:

* `load(StoreKey, Key)`: Retrieve item stored at key `Key` in store found at key `StoreKey` in the multistore
* `store(StoreKey, Key, value)`: Write value `Value` at key `Key` in store found at key `StoreKey` in the multistore

### Proposal Processing Queue

**Store:**

* `ProposalProcessingQueue`: A queue `queue[proposalID]` containing all the
  `ProposalIDs` of proposals that reached `MinDeposit`. During each `EndBlock`,
  all the proposals that have reached the end of their voting period are processed.
  To process a finished proposal, the application tallies the votes, computes the
  votes of each validator and checks if every validator in the validator set has
  voted. If the proposal is accepted, deposits are refunded. Finally, the proposal
  content `Handler` is executed.

And the pseudocode for the `ProposalProcessingQueue`:

```go
  in EndBlock do

    for finishedProposalID in GetAllFinishedProposalIDs(block.Time)
      proposal = load(Governance, <proposalID|'proposal'>) // proposal is a const key

      validators = Keeper.getAllValidators()
      tmpValMap := map(sdk.AccAddress)stakingtypes.ValidatorI

      // Tally
      voterIterator = rangeQuery(Governance, <proposalID|'addresses'>) //return all the addresses that voted on the proposal
      for each (voterAddress, vote) in voterIterator
        delegations = stakingKeeper.getDelegations(voterAddress) // get all delegations for current voter

        for each delegation in delegations
          proposal.updateTally(vote, delegation.Shares)

        _, isVal = stakingKeeper.getValidator(voterAddress)
        if (isVal)
          tmpValMap(voterAddress).Vote = vote

      tallyingParam = load(GlobalParams, 'TallyingParam')

      // Check if proposal is accepted or rejected
      totalNonAbstain := proposal.YesVotes + proposal.NoVotes
      if (proposal.Votes.YesVotes/totalNonAbstain > tallyingParam.Threshold)
        //  proposal was accepted at the end of the voting period
        //  refund deposits (non-voters already punished)
        for each (amount, depositor) in proposal.Deposits
          depositor.AtoneBalance += amount

        stateWriter, err := proposal.Handler()
        if err != nil
            // proposal passed but failed during state execution
            proposal.CurrentStatus = ProposalStatusFailed
         else
            // proposal pass and state is persisted
            proposal.CurrentStatus = ProposalStatusAccepted
            stateWriter.save()
      else
        // proposal was rejected
        proposal.CurrentStatus = ProposalStatusRejected

      store(Governance, <proposalID|'proposal'>, proposal)
```

### Legacy Proposal

A legacy proposal is the old implementation of governance proposal.
Contrary to proposal that can contain any messages, a legacy proposal allows
to submit a set of pre-defined proposals. These proposal are defined by their types.

While proposals should use the new implementation of the governance proposal, we need
still to use legacy proposal in order to submit a `software-upgrade` and a
`cancel-software-upgrade` proposal.

More information on how to submit proposals in the [client section](#client).

### Quorum Checks and Voting Period Extension

The module provides an extension mechanism for the voting period. By enforcing a delay
when quorum is reached too close to the end of the voting period, we ensure that the
community has enough time to understand all the proposal's implications and potentially
react accordingly without the worry of an imminent end to the voting period.

- `QuorumTimeout`: This parameter defines the time window after which, if the quorum
  is reached, the voting end time is extended. This value must be strictly less than
  `params.VotingPeriod`.
- `MaxVotingPeriodExtension`: This parameter defines the maximum amount of time by
  which a proposal's voting end time can be extended. This value must be greater or
  equal than `VotingPeriod - QuorumTimeout`.
- `QuorumCheckCount`: This parameter specifies the number of times a proposal
  should be checked for achieving quorum after the expiration of `QuorumTimeout`.
  It is used to determine the intervals at which these checks will take place. The
  intervals are calculated as `(VotingPeriod - QuorumTimeout) / QuorumCheckCount`.
  This avoids the need to check for quorum at the end of each block, which would have
  a significant impact on performance. Furthermore, if this value is set to 0, the
  quorum check and voting period extension system is considered *disabled*.

**Store:**

We also introduce a new `keeper.QuorumCheckQueue` similar to `keeper.ActiveProposalsQueue`
and `keeper.InactiveProposalsQueue`. This queue stores proposals that are due to be
checked for quorum. The key for each proposal in the queue is a pair containing the time
at which the proposal should be checked for quorum as the first part, and the `proposal.Id`
as the second. The value will instead be a `QuorumCheckQueueEntry` struct that will store:

- `QuorumTimeoutTime`, indicating the time at which this proposal will pass the
  `QuorumTimeout` and computed as `proposal.VotingStartTime + QuorumTimeout`
- `QuorumCheckCount`, a copy of the value of the module parameter with the same
  name at the time of first insertion of this proposal in the `QuorumCheckQueue`
- `QuorumChecksDone`, indicating the number of quorum checks that have been already
  performed, initially 0

When a proposal is added to the `keeper.ActiveProposalsQueue`, it is also added to the
`keeper.QuorumCheckQueue`. The time part of the key for the proposal in the
`QuorumCheckQueue` is initially calculated as `proposal.VotingStartTime + QuorumTimeout`
(i.e. the `QuorumTimeoutTime`), therefore scheduling the first quorum check to happen
right after `QuorumTimeout` has expired.

In the `EndBlocker()` function of the `x/gov` module, we add a new call to
`keeper.IterateQuorumCheckQueue()` between the calls to `keeper.IterateInactiveProposalsQueue()`
and `keeper.IterateActiveProposalsQueue(`, where we iterate over proposals
that are due to be checked for quorum, meaning that their time part of the key is
before the current block time.

If a proposal has reached quorum (approximately) before or right at the
`QuorumTimeout`- i.e. the `QuorumChecksDone` is 0, meaning more precisely
that no previous quorum checks were performed - remove it from the `QuorumCheckQueue`
and do nothing, the proposal should end as expected.

If a proposal has reached quorum after the `QuorumTimeout` - i.e.
`QuorumChecksDone` > 0 - update the `proposal.VotingEndTime` as
`ctx.BlockTime() + MaxVotingPeriodExtension` and remove it from the
`keeper.QuorumCheckQueue`.

If a proposal is still active and has not yet reached quorum, remove the corresponding
item from `keeper.QuorumCheckQueue`, modify the last `QuorumCheckQueueEntry` value by
incrementing `QuorumChecksDone` to record this latest unsuccessful quorum check, and add
the proposal back to `keeper.QuorumCheckQueue` with updated keys and value.

To compute the time part of the new key, add a quorum check interval - which is computed as
`(VotingPeriod - QuorumTimeout) / QuorumCheckCount` - to the time part of the last key used in
`keeper.QuorumCheckQueue` for this proposal. Specifically, use the formula
`lastKey.K1.Add((VotingPeriod - QuorumTimeout) / QuorumCheckCount)`. As previously stated,
the value will remain the same struct as before, with `QuorumChecksDone` incremented by 1 to reflect
the additional unsuccessful quorum check that was performed.

If a proposal has passed its `VoteEndTime` and has not reached quorum, it should be removed from
`keeper.QuorumCheckQueue` without any additional actions. The proposal's failure will be handled
in the subsequent `keeper.IterateActiveProposalsQueue`.

### Constitution

A `constitution` string can be set at genesis with arbitrary content and is intended to be used
to store the chain established constitution upon launch.
The `constitution` can be updated through Constitution Amendment Proposals which must include
a valid patch of the `constitution` string expressed in **unified diff** format.
Example (from [gnu.org](https://www.gnu.org/software/diffutils/manual/html_node/Example-Unified.html)):

```
--- lao	2002-02-21 23:30:39.942229878 -0800
+++ tzu	2002-02-21 23:30:50.442260588 -0800
@@ -1,7 +1,6 @@
-The Way that can be told of is not the eternal Way;
-The name that can be named is not the eternal name.
 The Nameless is the origin of Heaven and Earth;
-The Named is the mother of all things.
+The named is the mother of all things.
+
 Therefore let there always be non-being,
   so we may see their subtlety,
 And let there always be being,
@@ -9,3 +8,6 @@
 The two are the same,
 But after they are produced,
   they have different names.
+They both may be called deep and profound.
+Deeper and more profound,
+The door of all subtleties!
```

### Law and Constitution Amendment Proposals

If Law or Constitution Amendment Proposals are submitted - by providing either a 
`MsgProposeLaw` or a `MsgProposeConstitutionAmendment` in the `MsgSubmitProposal.messages`
field, the related proposal will be tallied using specific quorum and threshold values
instead of the default ones for regular proposals. More specifically, the following parameters
are added to enable this behavior:

- `constitution_amendment_quorum` which defines the quorum for constitution amendment proposals
- `constitution_amendment_threshold` which defines the minimum proportion of Yes votes for a
  Constitution Amendment proposal to pass.
- `law_quorum` which defines the quorum for law proposals
- `law_threshold` which defines the minimum proportion of Yes votes for a Law proposal to pass.

The `MsgProposeLaw` just contains for now an `authority` field indicating who will execute the
`sdk.Msg` (which should be the governance module account), and has no effects for now. The conent
of Laws is entirely defined in the proposal `summary`. Example: 

```
{
   "authority": "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn"
}
```

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/tx.proto#L195-L202
```

The `MsgProposeConstitutionAmendment` contains the `authority` field and also an `amendment` field
that needs to be a string representing a valid patch for the `constitution` expressed in 
unified diff format. Example:

```
{
   "authority": "cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn",
   "amendment": "--- src\\n+++ dst\\n@@ -1 +1 @@\\n-Old Constitution\\n+Modified Constitution\\n\"
}
```

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/tx.proto#L209-L219
```

Upon execution of the `MsgProposeConstitutionAmendment` (which will happen if the proposal passes)
The `constitution` string will be updated by applying the patch defined in the `amendment` string.
An error will be returned if the `amendment` string is malformed, so constitution amendment proposals
need to be crafted with care.

## Messages

### Proposal Submission

Proposals can be submitted by any account via a `MsgSubmitProposal` transaction.

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/tx.proto#L53-L82
```

All `sdk.Msgs` passed into the `messages` field of a `MsgSubmitProposal` message
must be registered in the app's `MsgServiceRouter`. Each of these messages must
have one signer, namely the gov module account. And finally, the metadata length
must not be larger than the `maxMetadataLen` config passed into the gov keeper.

**State modifications:**

* Generate new `proposalID`
* Create new `Proposal`
* Initialise `Proposal`'s attributes
* Decrease balance of sender by `InitialDeposit`
* If `MinDeposit` is reached:
    * Push `proposalID` in `ProposalProcessingQueue`
* Transfer `InitialDeposit` from the `Proposer` to the governance `ModuleAccount`

A `MsgSubmitProposal` transaction can be handled according to the following
pseudocode.

```go
// PSEUDOCODE //
// Check if MsgSubmitProposal is valid. If it is, create proposal //

upon receiving txGovSubmitProposal from sender do

  if !correctlyFormatted(txGovSubmitProposal)
    // check if proposal is correctly formatted and the messages have routes to other modules. Includes fee payment.
    // check if all messages' unique Signer is the gov acct.
    // check if the metadata is not too long.
    throw

  initialDeposit = txGovSubmitProposal.InitialDeposit
  if (initialDeposit.Atones <= 0) OR (sender.AtoneBalance < initialDeposit.Atones)
    // InitialDeposit is negative or null OR sender has insufficient funds
    throw

  if (txGovSubmitProposal.Type != ProposalTypePlainText) OR (txGovSubmitProposal.Type != ProposalTypeSoftwareUpgrade)

  sender.AtoneBalance -= initialDeposit.Atones

  depositParam = load(GlobalParams, 'DepositParam')

  proposalID = generate new proposalID
  proposal = NewProposal()

  proposal.Messages = txGovSubmitProposal.Messages
  proposal.Metadata = txGovSubmitProposal.Metadata
  proposal.TotalDeposit = initialDeposit
  proposal.SubmitTime = <CurrentTime>
  proposal.DepositEndTime = <CurrentTime>.Add(depositParam.MaxDepositPeriod)
  proposal.Deposits.append({initialDeposit, sender})
  proposal.Submitter = sender
  proposal.YesVotes = 0
  proposal.NoVotes = 0
  proposal.AbstainVotes = 0
  proposal.CurrentStatus = ProposalStatusOpen

  store(Proposals, <proposalID|'proposal'>, proposal) // Store proposal in Proposals mapping
  return proposalID
```

### Deposit

Once a proposal is submitted, if
`Proposal.TotalDeposit < ActiveParam.MinDeposit`, Atone holders can send
`MsgDeposit` transactions to increase the proposal's deposit.

A proposal can only be sumbitted if the proposer deposits at least
`ActiveParam.MinDeposit` * `ActiveParam.MinInitialDepositRatio`, where
`ActiveParam.MinInitialDepositRatio` must be a valid percentage between 0 and 1.

Any deposit from Atone holders (including the proposer) need to be of at least
`ActiveParam.MinDeposit` * `ActiveParam.MinDepositRatio`, where
`ActiveParam.MinDepositRatio` must be a valid percentage between 0 and 1.

Generally it is expected that
`ActiveParam.MinDepositRatio` <= `ActiveParam.MinInitialDepositRatio`



```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/tx.proto#L150-L165
```

**State modifications:**

* Decrease balance of sender by `deposit`
* Add `deposit` of sender in `proposal.Deposits`
* Increase `proposal.TotalDeposit` by sender's `deposit`
* If `MinDeposit` is reached:
    * Push `proposalID` in `ProposalProcessingQueueEnd`
* Transfer `Deposit` from the `proposer` to the governance `ModuleAccount`

A `MsgDeposit` transaction has to go through a number of checks to be valid.
These checks are outlined in the following pseudocode.

```go
// PSEUDOCODE //
// Check if MsgDeposit is valid. If it is, increase deposit and check if MinDeposit is reached

upon receiving txGovDeposit from sender do
  // check if proposal is correctly formatted. Includes fee payment.

  if !correctlyFormatted(txGovDeposit)
    throw

  proposal = load(Proposals, <txGovDeposit.ProposalID|'proposal'>) // proposal is a const key, proposalID is variable

  if (proposal == nil)
    // There is no proposal for this proposalID
    throw

  if (txGovDeposit.Deposit.Atones <= 0) OR (sender.AtoneBalance < txGovDeposit.Deposit.Atones) OR (proposal.CurrentStatus != ProposalStatusOpen)

    // deposit is negative or null
    // OR sender has insufficient funds
    // OR proposal is not open for deposit anymore

    throw

  depositParam = load(GlobalParams, 'DepositParam')

  if (CurrentBlock >= proposal.SubmitBlock + depositParam.MaxDepositPeriod)
    proposal.CurrentStatus = ProposalStatusClosed

  else
    // sender can deposit
    sender.AtoneBalance -= txGovDeposit.Deposit.Atones

    proposal.Deposits.append({txGovVote.Deposit, sender})
    proposal.TotalDeposit.Plus(txGovDeposit.Deposit)

    if (proposal.TotalDeposit >= depositParam.MinDeposit)
      // MinDeposit is reached, vote opens

      proposal.VotingStartBlock = CurrentBlock
      proposal.CurrentStatus = ProposalStatusActive
      ProposalProcessingQueue.push(txGovDeposit.ProposalID)

  store(Proposals, <txGovVote.ProposalID|'proposal'>, proposal)
```

### Vote

Once `ActiveParam.MinDeposit` is reached, voting period starts. From there,
bonded Atone holders are able to send `MsgVote` transactions to cast their
vote on the proposal.

```protobuf reference
https://github.com/atomone-hub/atomone/blob/b9631ed2e3b781cd82a14316f6086802d8cb4dcf/proto/atomone/gov/v1/tx.proto#L106-L123
```

**State modifications:**

* Record `Vote` of sender

:::note
Gas cost for this message has to take into account the future tallying of the vote in EndBlocker.
:::

Next is a pseudocode outline of the way `MsgVote` transactions are handled:

```go
  // PSEUDOCODE //
  // Check if MsgVote is valid. If it is, count vote//

  upon receiving txGovVote from sender do
    // check if proposal is correctly formatted. Includes fee payment.

    if !correctlyFormatted(txGovDeposit)
      throw

    proposal = load(Proposals, <txGovDeposit.ProposalID|'proposal'>)

    if (proposal == nil)
      // There is no proposal for this proposalID
      throw


    if  (proposal.CurrentStatus == ProposalStatusActive)


        // Sender can vote if
        // Proposal is active
        // Sender has some bonds

        store(Governance, <txGovVote.ProposalID|'addresses'|sender>, txGovVote.Vote)   // Voters can vote multiple times. Re-voting overrides previous vote. This is ok because tallying is done once at the end.
```

## Events

The governance module emits the following events:

### EndBlocker

| Type              | Attribute Key   | Attribute Value  |
|-------------------|-----------------|------------------|
| inactive_proposal | proposal_id     | {proposalID}     |
| inactive_proposal | proposal_result | {proposalResult} |
| active_proposal   | proposal_id     | {proposalID}     |
| active_proposal   | proposal_result | {proposalResult} |
| quorum_check      | proposal_id     | {proposalID}     |
| quorum_check      | proposal_result | {proposalResult} |

### Handlers

#### MsgSubmitProposal

| Type                | Attribute Key       | Attribute Value |
|---------------------|---------------------|-----------------|
| submit_proposal     | proposal_id         | {proposalID}    |
| submit_proposal [0] | voting_period_start | {proposalID}    |
| proposal_deposit    | amount              | {depositAmount} |
| proposal_deposit    | proposal_id         | {proposalID}    |
| message             | module              | governance      |
| message             | action              | submit_proposal |
| message             | sender              | {senderAddress} |

* [0] Event only emitted if the voting period starts during the submission.

#### MsgVote

| Type          | Attribute Key | Attribute Value |
|---------------|---------------|-----------------|
| proposal_vote | option        | {voteOption}    |
| proposal_vote | proposal_id   | {proposalID}    |
| message       | module        | governance      |
| message       | action        | vote            |
| message       | sender        | {senderAddress} |

#### MsgVoteWeighted

| Type          | Attribute Key | Attribute Value          |
| ------------- | ------------- | ------------------------ |
| proposal_vote | option        | {weightedVoteOptions}    |
| proposal_vote | proposal_id   | {proposalID}             |
| message       | module        | governance               |
| message       | action        | vote                     |
| message       | sender        | {senderAddress}          |

#### MsgDeposit

| Type                 | Attribute Key       | Attribute Value |
|----------------------|---------------------|-----------------|
| proposal_deposit     | amount              | {depositAmount} |
| proposal_deposit     | proposal_id         | {proposalID}    |
| proposal_deposit [0] | voting_period_start | {proposalID}    |
| message              | module              | governance      |
| message              | action              | deposit         |
| message              | sender              | {senderAddress} |

* [0] Event only emitted if the voting period starts during the submission.

## Parameters

The governance module contains the following parameters:

| Key                              | Type             | Example                                 |
|----------------------------------|------------------|-----------------------------------------|
| min_deposit                      | array (coins)    | [{"denom":"uatone","amount":"10000000"}] |
| max_deposit_period               | string (time ns) | "172800000000000" (17280s)              |
| voting_period                    | string (time ns) | "172800000000000" (17280s)              |
| quorum                           | string (dec)     | "0.334000000000000000"                  |
| threshold                        | string (dec)     | "0.500000000000000000"                  |
| burn_proposal_deposit_prevote    | bool             | false                                   |
| burn_vote_quorum                 | bool             | false                                   |
| min_initial_deposit_ratio        | string (dec)     | "0.100000000000000000"                  |
| min_deposit_ratio                | string (dec)     | "0.010000000000000000"                  |
| constitution_amendment_quorum    | string (dec)     | "0.334000000000000000"                  |
| constitution_amendment_threshold | string (dec)     | "0.900000000000000000"                  |
| law_quorum                       | string (dec)     | "0.334000000000000000"                  |
| law_threshold                    | string (dec)     | "0.900000000000000000"                  |
| quorum_timeout                   | string (time ns) | "172800000000000" (17280s)              |
| max_voting_period_extension      | string (time ns) | "172800000000000" (17280s)              |
| quorum_check_count               | uint64           | 2                                       |


**NOTE**: The governance module contains parameters that are objects unlike other
modules. If only a subset of parameters are desired to be changed, only they need
to be included and not the entire parameter object structure.

## Client

### CLI

A user can query and interact with the `gov` module using the CLI.

#### Query

The `query` commands allow users to query `gov` state.

```bash
atomoned query gov --help
```

##### deposit

The `deposit` command allows users to query a deposit for a given proposal from a given depositor.

```bash
atomoned query gov deposit [proposal-id] [depositer-addr] [flags]
```

Example:

```bash
atomoned query gov deposit 1 atone1..
```

Example Output:

```bash
amount:
- amount: "100"
  denom: atone
depositor: atone1..
proposal_id: "1"
```

##### deposits

The `deposits` command allows users to query all deposits for a given proposal.

```bash
atomoned query gov deposits [proposal-id] [flags]
```

Example:

```bash
atomoned query gov deposits 1
```

Example Output:

```bash
deposits:
- amount:
  - amount: "100"
    denom: atone
  depositor: atone1..
  proposal_id: "1"
pagination:
  next_key: null
  total: "0"
```

##### param

The `param` command allows users to query a given parameter for the `gov` module.

```bash
atomoned query gov param [param-type] [flags]
```

Example:

```bash
atomoned query gov param voting
```

Example Output:

```bash
voting_period: "172800000000000"
```

##### params

The `params` command allows users to query all parameters for the `gov` module.

```bash
atomoned query gov params [flags]
```

Example:

```bash
atomoned query gov params
```

Example Output:

```bash
deposit_params:
  max_deposit_period: "172800000000000"
  min_deposit:
  - amount: "10000000"
    denom: atone
tally_params:
  quorum: "0.334000000000000000"
  threshold: "0.500000000000000000"
voting_params:
  voting_period: "172800000000000"
```

##### proposal

The `proposal` command allows users to query a given proposal.

```bash
atomoned query gov proposal [proposal-id] [flags]
```

Example:

```bash
atomoned query gov proposal 1
```

Example Output:

```bash
deposit_end_time: "2022-03-30T11:50:20.819676256Z"
final_tally_result:
  abstain_count: "0"
  no_count: "0"
  yes_count: "0"
id: "1"
messages:
- '@type': /cosmos.bank.v1beta1.MsgSend
  amount:
  - amount: "10"
    denom: atone
  from_address: atone1..
  to_address: atone1..
metadata: AQ==
status: PROPOSAL_STATUS_DEPOSIT_PERIOD
submit_time: "2022-03-28T11:50:20.819676256Z"
total_deposit:
- amount: "10"
  denom: atone
voting_end_time: null
voting_start_time: null
```

##### proposals

The `proposals` command allows users to query all proposals with optional filters.

```bash
atomoned query gov proposals [flags]
```

Example:

```bash
atomoned query gov proposals
```

Example Output:

```bash
pagination:
  next_key: null
  total: "0"
proposals:
- deposit_end_time: "2022-03-30T11:50:20.819676256Z"
  final_tally_result:
    abstain_count: "0"
    no_count: "0"
    yes_count: "0"
  id: "1"
  messages:
  - '@type': /cosmos.bank.v1beta1.MsgSend
    amount:
    - amount: "10"
      denom: atone
    from_address: atone1..
    to_address: atone1..
  metadata: AQ==
  status: PROPOSAL_STATUS_DEPOSIT_PERIOD
  submit_time: "2022-03-28T11:50:20.819676256Z"
  total_deposit:
  - amount: "10"
    denom: atone
  voting_end_time: null
  voting_start_time: null
- deposit_end_time: "2022-03-30T14:02:41.165025015Z"
  final_tally_result:
    abstain_count: "0"
    no_count: "0"
    yes_count: "0"
  id: "2"
  messages:
  - '@type': /cosmos.bank.v1beta1.MsgSend
    amount:
    - amount: "10"
      denom: atone
    from_address: atone1..
    to_address: atone1..
  metadata: AQ==
  status: PROPOSAL_STATUS_DEPOSIT_PERIOD
  submit_time: "2022-03-28T14:02:41.165025015Z"
  total_deposit:
  - amount: "10"
    denom: atone
  voting_end_time: null
  voting_start_time: null
```

##### proposer

The `proposer` command allows users to query the proposer for a given proposal.

```bash
atomoned query gov proposer [proposal-id] [flags]
```

Example:

```bash
atomoned query gov proposer 1
```

Example Output:

```bash
proposal_id: "1"
proposer: atone1..
```

##### tally

The `tally` command allows users to query the tally of a given proposal vote.

```bash
atomoned query gov tally [proposal-id] [flags]
```

Example:

```bash
atomoned query gov tally 1
```

Example Output:

```bash
abstain: "0"
"no": "0"
"yes": "1"
```

##### vote

The `vote` command allows users to query a vote for a given proposal.

```bash
atomoned query gov vote [proposal-id] [voter-addr] [flags]
```

Example:

```bash
atomoned query gov vote 1 atone1..
```

Example Output:

```bash
option: VOTE_OPTION_YES
options:
- option: VOTE_OPTION_YES
  weight: "1.000000000000000000"
proposal_id: "1"
voter: atone1..
```

##### votes

The `votes` command allows users to query all votes for a given proposal.

```bash
atomoned query gov votes [proposal-id] [flags]
```

Example:

```bash
atomoned query gov votes 1
```

Example Output:

```bash
pagination:
  next_key: null
  total: "0"
votes:
- option: VOTE_OPTION_YES
  options:
  - option: VOTE_OPTION_YES
    weight: "1.000000000000000000"
  proposal_id: "1"
  voter: atone1..
```

#### Transactions

The `tx` commands allow users to interact with the `gov` module.

```bash
atomoned tx gov --help
```

##### deposit

The `deposit` command allows users to deposit tokens for a given proposal.

```bash
atomoned tx gov deposit [proposal-id] [deposit] [flags]
```

Example:

```bash
atomoned tx gov deposit 1 10000000atone --from atone1..
```

##### draft-proposal

The `draft-proposal` command allows users to draft any type of proposal.
The command returns a `draft_proposal.json`, to be used by `submit-proposal` after being completed.
The `draft_metadata.json` is meant to be uploaded to [IPFS](#metadata).

```bash
atomoned tx gov draft-proposal
```

##### generate-constitution-amendment

The `generate-constitution-amendment` command allows users to generate a constitution amendment
proposal message from the current constitution, either queried or provided as an `.md` file through
the flag `--current-constitution` and the provided updated constitution `.md` file.

```bash
atomoned tx gov generate-constitution-amendment path/to/updated/constitution.md
```

##### submit-proposal

The `submit-proposal` command allows users to submit a governance proposal along with some messages and metadata.
Messages, metadata and deposit are defined in a JSON file.

```bash
atomoned tx gov submit-proposal [path-to-proposal-json] [flags]
```

Example:

```bash
atomoned tx gov submit-proposal /path/to/proposal.json --from atone1..
```

where `proposal.json` contains:

```json
{
  "messages": [
    {
      "@type": "/cosmos.bank.v1beta1.MsgSend",
      "from_address": "atone1...", // The gov module module address
      "to_address": "atone1...",
      "amount":[{"denom": "atone","amount": "10"}]
    }
  ],
  "metadata": "AQ==",
  "deposit": "10atone",
  "title": "Proposal Title",
  "summary": "Proposal Summary"
}
```

:::note
By default the metadata, summary and title are both limited by 255 characters, this can be overridden by the application developer.
:::

##### submit-legacy-proposal

The `submit-legacy-proposal` command allows users to submit a governance legacy proposal along with an initial deposit.

```bash
atomoned tx gov submit-legacy-proposal [command] [flags]
```

Example:

```bash
atomoned tx gov submit-legacy-proposal --title="Test Proposal" --description="testing" --type="Text" --deposit="100000000atone" --from atone1..
```

Example (`cancel-software-upgrade`):

```bash
atomoned tx gov submit-legacy-proposal cancel-software-upgrade --title="Test Proposal" --description="testing" --deposit="100000000atone" --from atone1..
```

Example (`param-change`):

```bash
atomoned tx gov submit-legacy-proposal param-change proposal.json --from atone1..
```

```json
{
  "title": "Test Proposal",
  "description": "testing, testing, 1, 2, 3",
  "changes": [
    {
      "subspace": "staking",
      "key": "MaxValidators",
      "value": 100
    }
  ],
  "deposit": "10000000atone"
}
```

Example (`software-upgrade`):

```bash
atomoned tx gov submit-legacy-proposal software-upgrade v2 --title="Test Proposal" --description="testing, testing, 1, 2, 3" --upgrade-height 1000000 --from atone1..
```

##### vote

The `vote` command allows users to submit a vote for a given governance proposal.

```bash
atomoned tx gov vote [command] [flags]
```

Example:

```bash
atomoned tx gov vote 1 yes --from atone1..
```

##### weighted-vote

The `weighted-vote` command allows users to submit a weighted vote for a given governance proposal.

```bash
atomoned tx gov weighted-vote [proposal-id] [weighted-options] [flags]
```

Example:

```bash
atomoned tx gov weighted-vote 1 yes=0.5,no=0.5 --from atone1..
```

### gRPC

A user can query the `gov` module using gRPC endpoints.

#### Proposal

The `Proposal` endpoint allows users to query a given proposal.

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/Proposal
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/Proposal
```

Example Output:

```bash
{
  "proposal": {
    "proposalId": "1",
    "content": {"@type":"/cosmos.gov.v1beta1.TextProposal","description":"testing, testing, 1, 2, 3","title":"Test Proposal"},
    "status": "PROPOSAL_STATUS_VOTING_PERIOD",
    "finalTallyResult": {
      "yes": "0",
      "abstain": "0",
      "no": "0",
    },
    "submitTime": "2021-09-16T19:40:08.712440474Z",
    "depositEndTime": "2021-09-18T19:40:08.712440474Z",
    "totalDeposit": [
      {
        "denom": "atone",
        "amount": "10000000"
      }
    ],
    "votingStartTime": "2021-09-16T19:40:08.712440474Z",
    "votingEndTime": "2021-09-18T19:40:08.712440474Z",
    "title": "Test Proposal",
    "summary": "testing, testing, 1, 2, 3"
  }
}
```

Using v1:

```bash
cosmos.gov.v1.Query/Proposal
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1.Query/Proposal
```

Example Output:

```bash
{
  "proposal": {
    "id": "1",
    "messages": [
      {"@type":"/cosmos.bank.v1beta1.MsgSend","amount":[{"denom":"atone","amount":"10"}],"fromAddress":"atone1..","toAddress":"atone1.."}
    ],
    "status": "PROPOSAL_STATUS_VOTING_PERIOD",
    "finalTallyResult": {
      "yesCount": "0",
      "abstainCount": "0",
      "noCount": "0",
    },
    "submitTime": "2022-03-28T11:50:20.819676256Z",
    "depositEndTime": "2022-03-30T11:50:20.819676256Z",
    "totalDeposit": [
      {
        "denom": "atone",
        "amount": "10000000"
      }
    ],
    "votingStartTime": "2022-03-28T14:25:26.644857113Z",
    "votingEndTime": "2022-03-30T14:25:26.644857113Z",
    "metadata": "AQ==",
    "title": "Test Proposal",
    "summary": "testing, testing, 1, 2, 3"
  }
}
```


#### Proposals

The `Proposals` endpoint allows users to query all proposals with optional filters.

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/Proposals
```

Example:

```bash
grpcurl -plaintext \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/Proposals
```

Example Output:

```bash
{
  "proposals": [
    {
      "proposalId": "1",
      "status": "PROPOSAL_STATUS_VOTING_PERIOD",
      "finalTallyResult": {
        "yes": "0",
        "abstain": "0",
        "no": "0",
      },
      "submitTime": "2022-03-28T11:50:20.819676256Z",
      "depositEndTime": "2022-03-30T11:50:20.819676256Z",
      "totalDeposit": [
        {
          "denom": "atone",
          "amount": "10000000010"
        }
      ],
      "votingStartTime": "2022-03-28T14:25:26.644857113Z",
      "votingEndTime": "2022-03-30T14:25:26.644857113Z"
    },
    {
      "proposalId": "2",
      "status": "PROPOSAL_STATUS_DEPOSIT_PERIOD",
      "finalTallyResult": {
        "yes": "0",
        "abstain": "0",
        "no": "0",
      },
      "submitTime": "2022-03-28T14:02:41.165025015Z",
      "depositEndTime": "2022-03-30T14:02:41.165025015Z",
      "totalDeposit": [
        {
          "denom": "atone",
          "amount": "10"
        }
      ],
      "votingStartTime": "0001-01-01T00:00:00Z",
      "votingEndTime": "0001-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "total": "2"
  }
}

```

Using v1:

```bash
cosmos.gov.v1.Query/Proposals
```

Example:

```bash
grpcurl -plaintext \
    localhost:9090 \
    cosmos.gov.v1.Query/Proposals
```

Example Output:

```bash
{
  "proposals": [
    {
      "id": "1",
      "messages": [
        {"@type":"/cosmos.bank.v1beta1.MsgSend","amount":[{"denom":"atone","amount":"10"}],"fromAddress":"atone1..","toAddress":"atone1.."}
      ],
      "status": "PROPOSAL_STATUS_VOTING_PERIOD",
      "finalTallyResult": {
        "yesCount": "0",
        "abstainCount": "0",
        "noCount": "0",
      },
      "submitTime": "2022-03-28T11:50:20.819676256Z",
      "depositEndTime": "2022-03-30T11:50:20.819676256Z",
      "totalDeposit": [
        {
          "denom": "atone",
          "amount": "10000000010"
        }
      ],
      "votingStartTime": "2022-03-28T14:25:26.644857113Z",
      "votingEndTime": "2022-03-30T14:25:26.644857113Z",
      "metadata": "AQ==",
      "title": "Proposal Title",
      "summary": "Proposal Summary"
    },
    {
      "id": "2",
      "messages": [
        {"@type":"/cosmos.bank.v1beta1.MsgSend","amount":[{"denom":"atone","amount":"10"}],"fromAddress":"atone1..","toAddress":"atone1.."}
      ],
      "status": "PROPOSAL_STATUS_DEPOSIT_PERIOD",
      "finalTallyResult": {
        "yesCount": "0",
        "abstainCount": "0",
        "noCount": "0",
      },
      "submitTime": "2022-03-28T14:02:41.165025015Z",
      "depositEndTime": "2022-03-30T14:02:41.165025015Z",
      "totalDeposit": [
        {
          "denom": "atone",
          "amount": "10"
        }
      ],
      "metadata": "AQ==",
      "title": "Proposal Title",
      "summary": "Proposal Summary"
    }
  ],
  "pagination": {
    "total": "2"
  }
}
```

#### Vote

The `Vote` endpoint allows users to query a vote for a given proposal.

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/Vote
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1","voter":"atone1.."}' \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/Vote
```

Example Output:

```bash
{
  "vote": {
    "proposalId": "1",
    "voter": "atone1..",
    "option": "VOTE_OPTION_YES",
    "options": [
      {
        "option": "VOTE_OPTION_YES",
        "weight": "1000000000000000000"
      }
    ]
  }
}
```

Using v1:

```bash
cosmos.gov.v1.Query/Vote
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1","voter":"atone1.."}' \
    localhost:9090 \
    cosmos.gov.v1.Query/Vote
```

Example Output:

```bash
{
  "vote": {
    "proposalId": "1",
    "voter": "atone1..",
    "option": "VOTE_OPTION_YES",
    "options": [
      {
        "option": "VOTE_OPTION_YES",
        "weight": "1.000000000000000000"
      }
    ]
  }
}
```

#### Votes

The `Votes` endpoint allows users to query all votes for a given proposal.

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/Votes
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/Votes
```

Example Output:

```bash
{
  "votes": [
    {
      "proposalId": "1",
      "voter": "atone1..",
      "options": [
        {
          "option": "VOTE_OPTION_YES",
          "weight": "1000000000000000000"
        }
      ]
    }
  ],
  "pagination": {
    "total": "1"
  }
}
```

Using v1:

```bash
cosmos.gov.v1.Query/Votes
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1.Query/Votes
```

Example Output:

```bash
{
  "votes": [
    {
      "proposalId": "1",
      "voter": "atone1..",
      "options": [
        {
          "option": "VOTE_OPTION_YES",
          "weight": "1.000000000000000000"
        }
      ]
    }
  ],
  "pagination": {
    "total": "1"
  }
}
```

#### Params

The `Params` endpoint allows users to query all parameters for the `gov` module.

<!-- TODO: #10197 Querying governance params outputs nil values -->

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/Params
```

Example:

```bash
grpcurl -plaintext \
    -d '{"params_type":"voting"}' \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/Params
```

Example Output:

```bash
{
  "votingParams": {
    "votingPeriod": "172800s"
  },
  "depositParams": {
    "maxDepositPeriod": "0s"
  },
  "tallyParams": {
    "quorum": "MA==",
    "threshold": "MA==",
  }
}
```

Using v1:

```bash
cosmos.gov.v1.Query/Params
```

Example:

```bash
grpcurl -plaintext \
    -d '{"params_type":"voting"}' \
    localhost:9090 \
    cosmos.gov.v1.Query/Params
```

Example Output:

```bash
{
  "votingParams": {
    "votingPeriod": "172800s"
  }
}
```

#### Deposit

The `Deposit` endpoint allows users to query a deposit for a given proposal from a given depositor.

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/Deposit
```

Example:

```bash
grpcurl -plaintext \
    '{"proposal_id":"1","depositor":"atone1.."}' \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/Deposit
```

Example Output:

```bash
{
  "deposit": {
    "proposalId": "1",
    "depositor": "atone1..",
    "amount": [
      {
        "denom": "atone",
        "amount": "10000000"
      }
    ]
  }
}
```

Using v1:

```bash
cosmos.gov.v1.Query/Deposit
```

Example:

```bash
grpcurl -plaintext \
    '{"proposal_id":"1","depositor":"atone1.."}' \
    localhost:9090 \
    cosmos.gov.v1.Query/Deposit
```

Example Output:

```bash
{
  "deposit": {
    "proposalId": "1",
    "depositor": "atone1..",
    "amount": [
      {
        "denom": "atone",
        "amount": "10000000"
      }
    ]
  }
}
```

#### deposits

The `Deposits` endpoint allows users to query all deposits for a given proposal.

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/Deposits
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/Deposits
```

Example Output:

```bash
{
  "deposits": [
    {
      "proposalId": "1",
      "depositor": "atone1..",
      "amount": [
        {
          "denom": "atone",
          "amount": "10000000"
        }
      ]
    }
  ],
  "pagination": {
    "total": "1"
  }
}
```

Using v1:

```bash
cosmos.gov.v1.Query/Deposits
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1.Query/Deposits
```

Example Output:

```bash
{
  "deposits": [
    {
      "proposalId": "1",
      "depositor": "atone1..",
      "amount": [
        {
          "denom": "atone",
          "amount": "10000000"
        }
      ]
    }
  ],
  "pagination": {
    "total": "1"
  }
}
```

#### TallyResult

The `TallyResult` endpoint allows users to query the tally of a given proposal.

Using legacy v1beta1:

```bash
cosmos.gov.v1beta1.Query/TallyResult
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1beta1.Query/TallyResult
```

Example Output:

```bash
{
  "tally": {
    "yes": "1000000",
    "abstain": "0",
    "no": "0",
  }
}
```

Using v1:

```bash
cosmos.gov.v1.Query/TallyResult
```

Example:

```bash
grpcurl -plaintext \
    -d '{"proposal_id":"1"}' \
    localhost:9090 \
    cosmos.gov.v1.Query/TallyResult
```

Example Output:

```bash
{
  "tally": {
    "yes": "1000000",
    "abstain": "0",
    "no": "0",
  }
}
```

### REST

A user can query the `gov` module using REST endpoints.

#### proposal

The `proposals` endpoint allows users to query a given proposal.

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/proposals/{proposal_id}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/proposals/1
```

Example Output:

```bash
{
  "proposal": {
    "proposal_id": "1",
    "content": null,
    "status": "PROPOSAL_STATUS_VOTING_PERIOD",
    "final_tally_result": {
      "yes": "0",
      "abstain": "0",
      "no": "0",
    },
    "submit_time": "2022-03-28T11:50:20.819676256Z",
    "deposit_end_time": "2022-03-30T11:50:20.819676256Z",
    "total_deposit": [
      {
        "denom": "atone",
        "amount": "10000000010"
      }
    ],
    "voting_start_time": "2022-03-28T14:25:26.644857113Z",
    "voting_end_time": "2022-03-30T14:25:26.644857113Z"
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/proposals/{proposal_id}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/proposals/1
```

Example Output:

```bash
{
  "proposal": {
    "id": "1",
    "messages": [
      {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "atone1..",
        "to_address": "atone1..",
        "amount": [
          {
            "denom": "atone",
            "amount": "10"
          }
        ]
      }
    ],
    "status": "PROPOSAL_STATUS_VOTING_PERIOD",
    "final_tally_result": {
      "yes_count": "0",
      "abstain_count": "0",
      "no_count": "0",
    },
    "submit_time": "2022-03-28T11:50:20.819676256Z",
    "deposit_end_time": "2022-03-30T11:50:20.819676256Z",
    "total_deposit": [
      {
        "denom": "atone",
        "amount": "10000000"
      }
    ],
    "voting_start_time": "2022-03-28T14:25:26.644857113Z",
    "voting_end_time": "2022-03-30T14:25:26.644857113Z",
    "metadata": "AQ==",
    "title": "Proposal Title",
    "summary": "Proposal Summary"
  }
}
```

#### proposals

The `proposals` endpoint also allows users to query all proposals with optional filters.

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/proposals
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/proposals
```

Example Output:

```bash
{
  "proposals": [
    {
      "proposal_id": "1",
      "content": null,
      "status": "PROPOSAL_STATUS_VOTING_PERIOD",
      "final_tally_result": {
        "yes": "0",
        "abstain": "0",
        "no": "0",
      },
      "submit_time": "2022-03-28T11:50:20.819676256Z",
      "deposit_end_time": "2022-03-30T11:50:20.819676256Z",
      "total_deposit": [
        {
          "denom": "atone",
          "amount": "10000000"
        }
      ],
      "voting_start_time": "2022-03-28T14:25:26.644857113Z",
      "voting_end_time": "2022-03-30T14:25:26.644857113Z"
    },
    {
      "proposal_id": "2",
      "content": null,
      "status": "PROPOSAL_STATUS_DEPOSIT_PERIOD",
      "final_tally_result": {
        "yes": "0",
        "abstain": "0",
        "no": "0",
      },
      "submit_time": "2022-03-28T14:02:41.165025015Z",
      "deposit_end_time": "2022-03-30T14:02:41.165025015Z",
      "total_deposit": [
        {
          "denom": "atone",
          "amount": "10"
        }
      ],
      "voting_start_time": "0001-01-01T00:00:00Z",
      "voting_end_time": "0001-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "2"
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/proposals
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/proposals
```

Example Output:

```bash
{
  "proposals": [
    {
      "id": "1",
      "messages": [
        {
          "@type": "/cosmos.bank.v1beta1.MsgSend",
          "from_address": "atone1..",
          "to_address": "atone1..",
          "amount": [
            {
              "denom": "atone",
              "amount": "10"
            }
          ]
        }
      ],
      "status": "PROPOSAL_STATUS_VOTING_PERIOD",
      "final_tally_result": {
        "yes_count": "0",
        "abstain_count": "0",
        "no_count": "0",
      },
      "submit_time": "2022-03-28T11:50:20.819676256Z",
      "deposit_end_time": "2022-03-30T11:50:20.819676256Z",
      "total_deposit": [
        {
          "denom": "atone",
          "amount": "10000000010"
        }
      ],
      "voting_start_time": "2022-03-28T14:25:26.644857113Z",
      "voting_end_time": "2022-03-30T14:25:26.644857113Z",
      "metadata": "AQ==",
      "title": "Proposal Title",
      "summary": "Proposal Summary"
    },
    {
      "id": "2",
      "messages": [
        {
          "@type": "/cosmos.bank.v1beta1.MsgSend",
          "from_address": "atone1..",
          "to_address": "atone1..",
          "amount": [
            {
              "denom": "atone",
              "amount": "10"
            }
          ]
        }
      ],
      "status": "PROPOSAL_STATUS_DEPOSIT_PERIOD",
      "final_tally_result": {
        "yes_count": "0",
        "abstain_count": "0",
        "no_count": "0",
      },
      "submit_time": "2022-03-28T14:02:41.165025015Z",
      "deposit_end_time": "2022-03-30T14:02:41.165025015Z",
      "total_deposit": [
        {
          "denom": "atone",
          "amount": "10"
        }
      ],
      "voting_start_time": null,
      "voting_end_time": null,
      "metadata": "AQ==",
      "title": "Proposal Title",
      "summary": "Proposal Summary"
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "2"
  }
}
```

#### voter vote

The `votes` endpoint allows users to query a vote for a given proposal.

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/proposals/{proposal_id}/votes/{voter}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/proposals/1/votes/atone1..
```

Example Output:

```bash
{
  "vote": {
    "proposal_id": "1",
    "voter": "atone1..",
    "option": "VOTE_OPTION_YES",
    "options": [
      {
        "option": "VOTE_OPTION_YES",
        "weight": "1.000000000000000000"
      }
    ]
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/proposals/{proposal_id}/votes/{voter}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/proposals/1/votes/atone1..
```

Example Output:

```bash
{
  "vote": {
    "proposal_id": "1",
    "voter": "atone1..",
    "options": [
      {
        "option": "VOTE_OPTION_YES",
        "weight": "1.000000000000000000"
      }
    ],
    "metadata": ""
  }
}
```

#### votes

The `votes` endpoint allows users to query all votes for a given proposal.

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/proposals/{proposal_id}/votes
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/proposals/1/votes
```

Example Output:

```bash
{
  "votes": [
    {
      "proposal_id": "1",
      "voter": "atone1..",
      "option": "VOTE_OPTION_YES",
      "options": [
        {
          "option": "VOTE_OPTION_YES",
          "weight": "1.000000000000000000"
        }
      ]
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "1"
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/proposals/{proposal_id}/votes
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/proposals/1/votes
```

Example Output:

```bash
{
  "votes": [
    {
      "proposal_id": "1",
      "voter": "atone1..",
      "options": [
        {
          "option": "VOTE_OPTION_YES",
          "weight": "1.000000000000000000"
        }
      ],
      "metadata": ""
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "1"
  }
}
```

#### params

The `params` endpoint allows users to query all parameters for the `gov` module.

<!-- TODO: #10197 Querying governance params outputs nil values -->

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/params/{params_type}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/params/voting
```

Example Output:

```bash
{
  "voting_params": {
    "voting_period": "172800s"
  },
  "deposit_params": {
    "min_deposit": [
    ],
    "max_deposit_period": "0s"
  },
  "tally_params": {
    "quorum": "0.000000000000000000",
    "threshold": "0.000000000000000000",
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/params/{params_type}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/params/voting
```

Example Output:

```bash
{
  "voting_params": {
    "voting_period": "172800s"
  },
  "deposit_params": {
    "min_deposit": [
    ],
    "max_deposit_period": "0s"
  },
  "tally_params": {
    "quorum": "0.000000000000000000",
    "threshold": "0.000000000000000000",
  }
}
```

#### deposits

The `deposits` endpoint allows users to query a deposit for a given proposal from a given depositor.

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/proposals/{proposal_id}/deposits/{depositor}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/proposals/1/deposits/atone1..
```

Example Output:

```bash
{
  "deposit": {
    "proposal_id": "1",
    "depositor": "atone1..",
    "amount": [
      {
        "denom": "atone",
        "amount": "10000000"
      }
    ]
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/proposals/{proposal_id}/deposits/{depositor}
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/proposals/1/deposits/atone1..
```

Example Output:

```bash
{
  "deposit": {
    "proposal_id": "1",
    "depositor": "atone1..",
    "amount": [
      {
        "denom": "atone",
        "amount": "10000000"
      }
    ]
  }
}
```

#### proposal deposits

The `deposits` endpoint allows users to query all deposits for a given proposal.

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/proposals/{proposal_id}/deposits
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/proposals/1/deposits
```

Example Output:

```bash
{
  "deposits": [
    {
      "proposal_id": "1",
      "depositor": "atone1..",
      "amount": [
        {
          "denom": "atone",
          "amount": "10000000"
        }
      ]
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "1"
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/proposals/{proposal_id}/deposits
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/proposals/1/deposits
```

Example Output:

```bash
{
  "deposits": [
    {
      "proposal_id": "1",
      "depositor": "atone1..",
      "amount": [
        {
          "denom": "atone",
          "amount": "10000000"
        }
      ]
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "1"
  }
}
```

#### tally

The `tally` endpoint allows users to query the tally of a given proposal.

Using legacy v1beta1:

```bash
/cosmos/gov/v1beta1/proposals/{proposal_id}/tally
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1beta1/proposals/1/tally
```

Example Output:

```bash
{
  "tally": {
    "yes": "1000000",
    "abstain": "0",
    "no": "0",
  }
}
```

Using v1:

```bash
/cosmos/gov/v1/proposals/{proposal_id}/tally
```

Example:

```bash
curl localhost:1317/cosmos/gov/v1/proposals/1/tally
```

Example Output:

```bash
{
  "tally": {
    "yes": "1000000",
    "abstain": "0",
    "no": "0",
  }
}
```


## Metadata

The gov module has two locations for metadata where users can provide further context about the on-chain actions they are taking. By default all metadata fields have a 255 character length field where metadata can be stored in json format, either on-chain or off-chain depending on the amount of data required. Here we provide a recommendation for the json structure and where the data should be stored. There are two important factors in making these recommendations. First, that the gov and group modules are consistent with one another, note the number of proposals made by all groups may be quite large. Second, that client applications such as block explorers and governance interfaces have confidence in the consistency of metadata structure accross chains.

### Proposal

Location: off-chain as json object stored on IPFS (mirrors [group proposal](../group/README.md#metadata))

```json
{
  "title": "",
  "authors": [""],
  "summary": "",
  "details": "",
  "proposal_forum_url": "",
  "vote_option_context": "",
}
```

:::note
The `authors` field is an array of strings, this is to allow for multiple authors to be listed in the metadata.
In v0.46, the `authors` field is a comma-separated string. Frontends are encouraged to support both formats for backwards compatibility.
:::

### Vote

Location: on-chain as json within 255 character limit (mirrors [group vote](../group/README.md#metadata))

```json
{
  "justification": "",
}
```

## Future Improvements

The current documentation only describes the minimum viable product for the
governance module. Future improvements may include:

* **`BountyProposals`:** If accepted, a `BountyProposal` creates an open
  bounty. The `BountyProposal` specifies how many Atones will be given upon
  completion. These Atones will be taken from the `reserve pool`. After a
  `BountyProposal` is accepted by governance, anybody can submit a
  `SoftwareUpgradeProposal` with the code to claim the bounty. Note that once a
  `BountyProposal` is accepted, the corresponding funds in the `reserve pool`
  are locked so that payment can always be honored. In order to link a
  `SoftwareUpgradeProposal` to an open bounty, the submitter of the
  `SoftwareUpgradeProposal` will use the `Proposal.LinkedProposal` attribute.
  If a `SoftwareUpgradeProposal` linked to an open bounty is accepted by
  governance, the funds that were reserved are automatically transferred to the
  submitter.
* **Complex delegation:** Delegators could choose other representatives than
  their validators. Ultimately, the chain of representatives would always end
  up to a validator, but delegators could inherit the vote of their chosen
  representative before they inherit the vote of their validator. In other
  words, they would only inherit the vote of their validator if their other
  appointed representative did not vote.
* **Better process for proposal review:** There would be two parts to
  `proposal.Deposit`, one for anti-spam (same as in MVP) and an other one to
  reward third party auditors.
