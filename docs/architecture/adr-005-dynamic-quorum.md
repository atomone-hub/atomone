# ADR 5: Dynamic Quorum

## Changelog

- 08 June 2025: Initial version

## Status

Implemented (https://github.com/atomone-hub/atomone/pull/135)

## Abstract

This ADR proposes a mechanism to dynamically adjust the `Quorum` parameter -
i.e. - in the `x/gov` module. This parameter represents the minimum
participation during a voting period required for the vote to be considered
valid.  

The quorum is dynamically adjusted after each vote based on the actual
participation. The new quorum is updated using an exponential moving average
[^1] of vote participation, meaning that the current vote’s participation is
weighted more heavily then previous vote participations. The exponential moving
average allows the quorum to react quickly to changes in participation.

The proposed mechanism is comparable to the `quorum adjustement mechanism` used
in Tezos [^2].

## Context

At the time of writing, the `x/gov` module on AtomOne uses a `Quorum` parameter
set to `0.25`. Since AtomOne has removed delegation-based voting [^3] in favor
of *direct voting* for most type of proposals, lower participation with
respect to the total voting power is to be expected. The mechanism proposed in
this ADR allows to find the proper quorum threshold based on actual
participation.

Dynamic quorum works in tandem with the deposit auto-throttler
([ADR-003](https://github.com/atomone-hub/atomone/blob/main/docs/architecture/adr-003-governance-proposal-deposit-auto-throttler.md))
to ensure that governance can proceed smoothly in a *direct voting* scenario.

## Alternatives

The main alternative to dynamic quorum is to reintroduce vote delegations. One
of the main reasons in favor of direct voting is that validators already have a
very specific role, which is foundational for AtomOne, their participation in
the consensus process. For this reason, the delegation of votes to validators
determines a conflation of roles: consensus and governance. An alternative
route, that could also be complementary to the dynamic quorum, is therefore to
separate vote delegations from consensus delegation and reintroduce
delegation-based voting in this scenario.

## Decision

The `Quorum` parameter will be replaced with a variable that is adjusted
dynamically based on previous voting participations. The dynamic quorum is
adjusted after every voting period.

The participation Exponential Moving Average - `pEMA` - is updated according to
the formula:

$$
pEMA_{t+1} = (0.8)pEMA_t + (0.2)p
$$

Where:

- $pEMA_{t+1}$  is the new participation exponential moving average value.
- $pEMA_t$ is the current participation exponential moving average value.
- $p$ is the participation observed during the previous voting period.

The quorum is then computed based on `pEMA` as follows:

$$
Q = Q_{min} + (Q_{max} - Q_{min}) \times pEMA
$$

Where:

- $Q$ is the resulting quorum value.
- $Q_{min}$ is the minimum quorum value allowed.
- $Q_{max}$ is the maximum quorum value allowed.
- $pEMA$ is the updated participation exponential moving average.

### Implementation

In the Implementation, the dynamic quorums of law and constitution
amendment proposals are separated from the other proposals. For each
of these quorum there is one `pEMA` state variable that is updated independently.

The following parameters are added to the `x/gov` module:

- `quorum_range` : struct containing `Min` and `Max` quorum values required to pass a proposal.
- `law_quorum_range` : struct containing `Min` and `Max` quorum values required to pass a law proposal.
- `constitution_amendment_quorum_range` : struct containing `Min` and `Max` quorum values required to pass a constitution amendment proposal.

### Querying the quorum value

Given that `Quorum` is no longer a fixed parameter, a new query endpoint is
also required. The endpoint will allow clients to fetch the current value of
the quorum.

## Consequences

If the participation during the voting period of a proposal is lower than
`pEMA`, and the current dynamic quorum is above `MinQuorum` , the quorum
required for the next proposal will decrease. On the other hand, if the
participation is higher then `pEMA`, and the current dynamic quorum is below
`MaxQuorum` , the quorum required for the next proposal will increase.

### Positive

- The quorum dynamically adjust to reflect voting participation.

### Negative

- Additional computation is required to compute the quorum.

### Neutral

- Increased number of governance parameters.
- Adds a new endpoint to query the `Quorum` value.

## References

- [^1]: [Exponential smoothing](https://en.wikipedia.org/wiki/Exponential_smoothing)
- [^2]: [Tezos, Quorum computation](https://opentezos.com/tezos-basics/governance-on-chain/#quorum-computation)
- [^3]: [AtomOne, Staking and Governance Separation: Introduction of Delegation-less](https://github.com/atomone-hub/genesis/blob/main/GOVERNANCE.md#3-staking-and-governance-separation-introduction-of-delegation-less)
