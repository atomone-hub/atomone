# ADR 004: Nakamoto Bonus 

## Status

DRAFT

## Abstract

This ADR proposes a mechanism to increase the Nakamoto coefficient of a chain, by incentivizing delegators to delegate on validators having less stake. The ADR proposes to divide the staking reward in two components: a part proportional to the overall amount of stake of a validator - this coincides with the current staking reward model, and a part that is distributed uniformly to validators - the `Nakamoto bonus`.

The overall amount of the reward remains unchanged, however the criteria for assigning the reward changes.

## Context

At the time of writing, the AtomOne chain suffers from a concentration of delegations on the top staking validators. The current Nakamoto coefficient is 5, which is lower that the ideal (67, given that the current maximum number of validators at the time of writing is 100) and armful to the security of the chain.


### Reward Distribution

The chain reward is distributed across validators using an even distribution weighted by stake. This means that the reward a validator `j` receives after validating block `i` can be computed as follows:

$$
r_{ji} = \frac {x_{ji}}{S_i} * R_i.
$$

Where:

- $r_{ji}$ is the reward of validator $j$ for block $i$.
- $x_{ji}$ is the stake of validator $j$ at block $i$.
- $S_i$ is the total stake across all validators at block $i$.
- $R_i$ is the total reward obtained by validating block $i$.


Additionally, the reward obtained by a validator is further divided using a even distribution weighted by delegation across the delegators. Hence, the reward received by delegator $k$, that delegated his stake to validator $j$ for block $i$ can be computed as:

$$
rd_{ki} = \frac {d_{ki}}{x_{ji}} * r_{ji}.
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
R_i = PR_i + NB_i.
$$

Where:
- $R_i$ is the total reward obtained by validating block $i$.
- $PR_i$ is the proportional reward for block $i$.
- $NB_i$ is the Nakamoto Bonus for block $i$.

Moreover, we define a new parameter $\eta : 0 \le eta \le 1$ that specifies how NB is computed from the reward: $NB_i = R_i * \eta$.
Increasing $\eta$ increases NB, and decreasing it, decreases NB.

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

### Dynamic Change of the $\eta$ parameter

The $\eta$ coefficient will move between 0 and 100\% and its initial value of will be set to `5%`.
The coefficient will be updated every 120K blocks (~ one week) by performing increases or decreases of +- 5%.

The decision on whether $\eta$ needs to be increased or decreased is performed as follows:
1. Bonded validators are sorted by voting power and split into 3 groups (33 in high, 33 in medium, and 34 in low).
2. The average voting power of the high and low validator groups is computed.
3. If the average voting power of the high group is *3x* or more the average voting power of the low group, $\eta$ is increased, otherwise it is decreased.



## Sybil Attack

The rewarding mechanism as presented above can be taken advantage of by a sybil attack.

Specifically, a validator may profit by adding multiple validators to the chain and splitting its stake across such validators.

In fact, in this scenario assuming $y$ as the number of sybil instances:

$$
r_{ji} = y * (\frac {x_{ji}}{y * S_i} * PR_i + \frac{NB_i}{N_i}) = \frac {x_{ji}}{S_i} * PR_i + y * \frac{NB_i}{N_i}
$$

So the validator would keep intact its `Proportional Reward` and be rewarded y times the `Nakamoto Bonus`.

As a measure to mitigate this, we propose to adopt and ajust the mechanism of `proportional slashing` as presented in [adr-014.](https://github.com/cosmos/cosmos-sdk/blob/main/docs/architecture/adr-014-proportional-slashing.md)
Two (or more) validators are considered correlated if they fail within the same time period. The correlated validators are then slashed as follows:

$$
slash_percentage = k * ((power_1)^(1/r) + (power_2)^(1/r) + ... + (power_n)^(1/r))^r // where k and r are both on-chain constants
$$

Where
- $power_j$ refers to the voting power of validator $j$
- k is a chain specific constant

For example, assuming k=1 and r=2, if one validator of 10% faults, it gets a 10% slash, while if two validators of 5% each fault together, they both get a 20% slash ((sqrt(0.05)+sqrt(0.05))^2).



