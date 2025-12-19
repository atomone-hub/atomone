# ADR 004: Governors System

## Changelog

- 2025-01-17: Initial version

## Status

Implemented (https://github.com/atomone-hub/atomone/pull/73), (https://github.com/atomone-hub/cosmos-sdk/pull/36), (https://github.com/atomone-hub/atomone/pull/258)

## Abstract

This ADR proposes an enhanced governance model introducing “*Governors*” as a
special class of participants within the governance module. Unlike traditional
Cosmos-SDK governance, users can delegate their governance power to active
governors who meet specific eligibility criteria (e.g., minimum governance
self-delegation) while staking delegations to validators are not counted towards
determining their governance voting power (no validator inheritance).
The proposed changes alters vote tallying adding a separate and governance-specific
delegation mechanism to the one in `x/staking`, while trying to minimize impact on
tally and overall chain performance, and also define rules for transitioning
between governor statuses. The aim is to give the governance system a flexible
structure, enabling representative governance while maintaining the separation
of powers between validators and governors.

## Context

While the standard Cosmos-SDK governance module allows validators to vote on
proposals with all their delegated tokens unless delegators override their votes,
AtomOne removes this feature to prevent validators from having undue influence and
segregate the role of validator to securing the network. However, this approach
leads to the necessity for all users to be actively involved in governance, which
is not necessarily ideal. As of now, the governance module primarily relies on
on direct token voting (where each delegator votes with their own staked tokens).
However, as on-chain governance grows more complex, certain community members
might still prefer to delegate their voting power to specialized actors (governors)
that represent their political views. This shift requires formalizing participant
roles and creating processes for delegating power, ensuring alignment with the
network’s interests and enabling accurate tally calculations.

## Decision

1. Introduction of `Governor` Entities:
   - A `Governor` is created through a `MsgCreateGovernor` transaction.  
   - The system enforces a minimum governance self-delegation requirement
   before an account can achieve active governor status, which in practice
   translates to a minimum stake requirement.
   - The system tracks each governor’s address, description, status (e.g., active),
   and last status change time (to limit frequent status toggles).

2. Delegation and Undelegation:
   - Users (delegators) can delegate their governance power to a governor via
   `MsgDelegateGovernor`. Only a single governor can be chosen at a time, and
   the delegation is always for the full voting power of the delegator.
   - If a delegator wishes to delegate to a different governor, they may do so
   directly, automatically redelegating from any existing governor, and this
   can be done at any time with no restrictions, except for active governors
   themselves which are force to delegate their governance power to themselves.
   - A user can also choose to undelegate from a governor with
   `MsgUndelegateGovernor`, reverting to direct voting only.

3. Tally Logic:
   - Each delegator’s staked tokens contribute to the total voting power of their
   chosen governor since governance delegations are for the full voting power.
   - During tallying, the system aggregates all delegated voting power plus any
   governor’s own staked tokens (minus any deducted shares for direct votes by
   delegators).  
   - Only votes from active governors or direct votes made by delegators are
   counted. If a delegator votes directly, the corresponding shares are deducted
   from the governor’s aggregated shares to avoid double counting (similarly
   to how it's done in canonical Cosmos-SDK governance for validators).

4. Transitioning Governor Status:
   - A governor can switch their status (e.g., from inactive to active) by meeting
   the min self-delegation threshold and updating their status with a
   `MsgUpdateGovernorStatus`. Governors that set themselves to inactive are allowed
   to delegate their governance voting power to another governor. A status change
   can however only occur after a waiting period (`GovernorStatusChangePeriod`).
   - Attempts to rapidly change status are disallowed; changes can only occur
    after the specified waiting period (`GovernorStatusChangePeriod`).

5. Parameters:
   - `MinGovernorSelfDelegation`: The minimum number of tokens a governor must  
   stake to achieve or maintain active status.  
   - `GovernorStatusChangePeriod`: The time a governor must wait since the last
   status change before being allowed to change status again.  

## Consequences

### Positive

- Allows specialized participants (governors) to manage governance responsibilities,
potentially improving governance participation.  
- Reduces complexity for casual stakers by letting them delegate governance
authority without having to participate actively every proposal, while always retaining
the option to vote directly.
- Retain segregation from the staking delegations system allowing governance and
staking delegations to be managed independently. This however does not prevent a
validator from also being a governor, but the two roles are kept separate.

### Negative

- Introduces more complexity in the governance codebase and state, with potential
performance implications for tallying and querying.
- Requires users to learn how to delegate to or become a governor, which in itself
may be a hurdle for some users.
- Requires clients and frontends to adapt to the new governance model, potentially
requiring changes to existing interfaces.

### Neutral

- The fundamental governance mechanisms (e.g., proposals, deposit structures)
remain largely the same.  
- Governor-based delegation is optional; delegators can still vote independently
if they prefer.

## References

- [GIST following discussions during AtomOne Constitution Working Group](https://gist.github.com/giunatale/95e9b43f6e265ba32b29e2769f7b8a37)
- [Governors System PR](https://github.com/atomone-hub/atomone/pull/73)
- [Governors Initial Implementation Draft PR](https://github.com/atomone-hub/atomone/pull/16)
