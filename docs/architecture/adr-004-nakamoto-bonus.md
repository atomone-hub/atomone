# ADR 004: Nakamoto Bonus 

## Changelog

- 19 May 2025: Initial version

## Status

DRAFT

## Abstract

The Nakamoto Coefficient is the main way used to measure the amount of decentralization of a blockchain. Many of the benefits of the blockchain technology can be directly attributed to its decentralization: its resilience to attacks, collusion and errors.

The Nakamoto Coefficient is defined as the minimum number of nodes that need to be compromised to harm the network [^1]. In the case of AtomOne, a voting power of 33.3% is enough to disrupt the consensus algorithm, therefore the Nakamoto Coefficient of the chain is the minimum number of validators having such voting power.

This ADR proposes a mechanism to increase the Nakamoto Coefficient of a chain, by incentivizing delegators to delegate on validators having less stake. The ADR proposes to divide the staking reward in two components: a part proportional to the overall amount of stake of a validator - this coincides with the current staking reward model, and a part that is distributed uniformly to validators. The part of the staking reward that is distributed uniformly is called `Nakamoto bonus` to underline its purpose: increasing the Nakamoto Coefficient.

The overall amount of the reward remains unchanged, however the criteria for assigning the reward changes.

## Context

At the time of writing, the AtomOne chain suffers from a concentration of delegations on the top staking validators. The current Nakamoto coefficient is **5**, which is lower that the ideal (34, given that the current maximum number of validators at the time of writing is 100) and armful to the security of the chain. There are multiple ways in which the issue of low decentralization could be addressed. One way is to incentivize delegators to increase the decentralization of AtomOne which is the aim of this ADR.


### Reward Distribution

The chain reward is distributed across validators using an even distribution weighted by stake. This means that the reward a validator `j` receives after validating block `i` can be computed as follows:

$$
r_{ji} = \frac {x_{ji}}{S_i} \times R_i
$$

Where:

- $r_{ji}$ is the reward of validator $j$ for block $i$.
- $x_{ji}$ is the stake of validator $j$ at block $i$.
- $S_i$ is the total stake across all validators at block $i$.
- $R_i$ is the total reward obtained by validating block $i$.


Additionally, the reward obtained by a validator is further divided using a even distribution weighted by delegation across the delegators. Hence, the reward received by delegator $k$, that delegated his stake to validator $j$ for block $i$ can be computed as:

$$
rd_{ki} = \frac {d_{ki}}{x_{ji}} \times r_{ji}
$$


Where:

- $rd_{ki}$ is the reward obtained by delegator $k$ for block $i$.
- $d_{ki}$ is the stake delegate by delegator $k$ at block $i$.
- $x_{ji}$ is the stake of validator $j$ at block $i$.
- $r_{ji}$ is the reward of validator $j$ for block $i$.

We can define a new metric, `Reward Per Stake (RPS)` as the amount a delegator would be rewarded for unit of stake, and we can compute it as:

$$
RPS_{ji} = \frac{r_{ji}}{x_{ji}}  
$$

Where:

- $RPS_{ji}$ is the reward per stake for validator $j$ for block $i$.
- $r_{ji}$ is the reward of validator $j$ for block $i$.
- $x_{ji}$ is the stake of validator $j$ at block $i$.

In essence, RPS tells us how much a delegator would receive by delegating to validator $j$. 

In the current model, it can easily be seen that RPS is equivalent across all validators and it can be derived from the first equation of this ADR that is equal to:

$$
RPS = \frac{R_{i}}{S{i}}
$$

This means, that from the point of view of a delegator, the decision on which validator to choose has no influence on their expected reward. Therefore, they might be as well delegating their stake to the most established validators.

## The Nakamoto Bonus

In this ADR, we propose to split the reward in two components, proportional reward (PR) and Nakamoto Bonus (NB) such that:

$$
R_i = PR_i + NB_i
$$

Where:
- $R_i$ is the total reward obtained by validating block $i$.
- $PR_i$ is the proportional reward for block $i$.
- $NB_i$ is the Nakamoto Bonus for block $i$.

The reward obtained by a validator $j$ at block $i$ is computed as:

$$
r_{ji} = \frac {x_{ji}}{S_i} \times PR_i + \frac{NB_i}{N_i}
$$

Where:
- $r_{ji}$ is the reward of validator $j$ for block $i$.
- $x_{ji}$ is the stake of validator $j$ at block $i$.
- $PR_i$ is the proportional reward for block $i$.
- $S_i$ is the total stake across all validators at block $i$.
- $NB_i$ is the Nakamoto Bonus for block $i$.
- $N_i$ is the total number of validators for block $i$.


The way the reward obtained by a validator is distributed across delegators remains unchanged.

If we compute the RPS, as defined before:

$$
RPS_{ji} = \frac{r_{ji}}{x_{ji}}  
$$

It can be easily seen that it is not possible anymore to remove the dependency on the validator choosen as it was without the Nakamoto Bonus.

More specifically, 


$$
RPS_{ji} = \frac{r_{ji}}{x_{ji}} = \frac{PR_i}{S_i} + \frac{NB_i}{N_i \times x_{ji}} 
$$

It can be seen that, the first member of the sum of $RPS_{ji}$ is independent from $j$, the specific validator, while the second is inversly proportional to $x_{ji}$ - the stake of validator $j$ at block $i$.

Therefore:
1. The choice of the validator changes the expected return of a delegator.
2. The effect of the Nakamoto Bonus is to reduce RPS in validators having higher stake.

In summary, a delegator is incentivized to delegate its stake to validators with the least stake as their RPS is higher.

### Dynamic change of the $\eta$ parameter

We define a new parameter $\eta \in [0,1]$ that specifies how the Nakamoto Bonus (NB) is computed from the reward:
$NB_i = R_i \times \eta$, where $R_i$ is the total reward obtained by validating block $i$.

Increasing $\eta$ increases the proportion of the reward distributed following the Nakamoto Bonus methodology. Conversely, decreasing $\eta$ increases the proportion of the block reward distributed proportionally to the validators.
The initial value of $\eta$ will be set to `3%`.
The coefficient will be updated every 120K blocks (~ one week) by performing increases or decreases of +- 3%.

The decision on whether $\eta$ needs to be increased or decreased is performed as follows:
1. Bonded validators are sorted by voting power and split into 3 groups (33 in high, 33 in medium, and 34 in low).
2. The average voting power of the high and low validator groups is computed.
3. If the average voting power of the high group is *3x* or more the average voting power of the low group, $\eta$ is increased, otherwise it is decreased.

### Validator Commissions

Unrestricted, validator-set commissions can be used to undermine the mechanisms proposed in this ADR. For example, a top validator could temporarily lower their commission to counteract the effects of the Nakamoto Bonus in the short term.
Moreover, unrestricted commissions may enable validators to exploit delegators. A validator could lower their commission to attract delegations, only to later increase it — potentially up to 100% — for personal gain.
To address this, this ADR proposes that the commission rate become a equal-for-all, network-wide parameter adjustable only through governance. This measure is also mandated by the AtomOne Constitution[^3].

## Consequences

The feature discussed in this ADR rewards delegators who behave in a way beneficial to the decentralization of the AtomOne chain. Delegators who will proactively redelegate their stake considering the state of distribution of voting power can expect a higher staking rewards.

### Positive

- The change in reward system incentivises delegators who increase the decentralization of the chain.

### Negative

- The rewarding mechanism as presented above can be taken advantage of by a sybil attack.

#### Sybil Attack


A validator may profit by adding multiple validators to the chain and splitting its stake across such validators.

In fact, in this scenario assuming $y$ as the number of sybil instances:

$$
r_{ji} = y \times (\frac {x_{ji}}{y \times S_i} \times PR_i + \frac{NB_i}{N_i}) = \frac {x_{ji}}{S_i} \times PR_i + y \times \frac{NB_i}{N_i}
$$

So the validator would keep intact its `Proportional Reward` and be rewarded y times the `Nakamoto Bonus`.

As a separate and additional feature to mitigate sybil attacks, we propose to adopt and adjust the mechanism of `proportional slashing` as presented in ADR-014 [^2] of the Cosmos Hub. 
Two (or more) validators are considered correlated if they fail within the same time period. The correlated validators are then slashed as follows:

$$
slash\_percentage = k \times ((power_1)^{(1/r)} + (power_2)^{(1/r)} + ... + (power_n)^{(1/r)})^r 
$$

Where
- $power_j$ refers to the voting power of validator $j$
- $k$ and $r$ are chain specific constants

For example, assuming $k=1$ and $r=2$, if one validator of 10% faults, it gets a 10% slash, while if two validators of 5% each fault together, they both get a 20% slash ($1 \times (0.05^{\frac{1}{2}}+0.05^{\frac{1}{2}})^2$).

`Proportional slashing` is not part of the Nakamoto Bonus feature, and it will be implemented as a separate feature.

### Neutral

- Increased number of distribution parameters.
- Adds a new endpoint to query the value of $\eta$.


[^1]: [Quantifying Decentralization](https://news.earn.com/quantifying-decentralization-e39db233c28e)
[^2]: [ADR-014 - Proportional Slashing](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-014-proportional-slashing.md)
[^3]: [AtomOne Constitution, Article 3, Section 9](https://github.com/atomone-hub/genesis/blob/50882cac6ea4e56b6703d7e3325f35073c75aa6b/CONSTITUTION.md#section-9-validators)

