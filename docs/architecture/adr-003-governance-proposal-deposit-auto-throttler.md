# ADR 003: Governance Proposal Deposit Auto-Throttler

**Changelog**

- 26 Nov 2024: Initial draft
- 3 Dec 2024: Improve formulation
- 16 Dec 2024: Write remaining sections
- 17 Dec 2024: Finalize first revision

## **Status[](https://docs.cosmos.network/main/architecture/adr-template#status)**

DRAFT

## **Abstract[](https://docs.cosmos.network/main/architecture/adr-template#abstract)**

This ADR proposes a mechanism to dynamically adjust the value of `MinDeposit` in the `x/gov` module, and potentially the value `MinInitialDeposit` with a similar but independent mechanism. These parameters represent the minimum deposit required to submit a proposal (`MinInitialDeposit`) and to start its voting period (`MinDeposit`) - or *activate* a proposal. 

The `MinDeposit` value will dynamically adjust with time and upon proposals *activation* or *deactivation*. The goal is to prevent an excessive number of active proposals and give individuals enough time to study them and vote with complete information. 

While the ADR is focused on presenting the dynamic adjustment of `MinDeposit`, the same mechanism can be used to dynamically adjust `MinInitialDeposit` - with different values for parameters and focusing on proposals entering or exiting the deposit period instead of the voting period. The two dynamic systems would function in the same way, but independently of each other, allowing even `MinInitialDeposit` to grow bigger than `MinDeposit` at times.

The proposed mechanism can be vaguely compared to the auto-adjusting inflation rate that targets a 2/3 bonding ratio. However, in this case, it is used to automatically adjust the deposit based on the number of proposals.

## **Context[](https://docs.cosmos.network/main/architecture/adr-template#context)**

At the time of writing, the `x/gov` module on AtomOne uses a `MinDeposit` ****parameter set to 512 ATONEs, while the `MinInitialDepositRatio` is equal to 10% of the `MinDeposit`, which makes the `MinInitialDeposit` to be 51.2 ATONEs.

### Spam proposals

Many Cosmos chains have been suffering from governance spam, with several proposals being submitted that are aiming at spreading misinformation and scams. While ultimately for proposals that do not enter the voting period the impact for the chain is only an increase in the governance proposal index (as proposals are eventually deleted), the majority of these proposals contain harmful links that could potentially pose a risk to unsuspecting users.

Some mitigations have been designed, like the [initial deposit requirement for proposals](https://github.com/cosmos/cosmos-sdk/pull/12771) and front-end level filtering. However, replacing the `MinInitialDepositRatio` with a dynamic `MinInitialDeposit` could allow the chain to deal more effectively with sudden influxes of large number of spam proposals.

### Excessive number of active proposals

Having too many active proposals at a time can be confusing, and dealing with them labor-intensive. The lower the number, the more attention stakers can pay to proposals without getting overwhelmed. This allows the chain governance to remain focused at all times, which will also increases its robustness.

## **Alternatives[](https://docs.cosmos.network/main/architecture/adr-template#alternatives)**

- Spam should also be filtered at the front-end level. This is already done by platforms like Mintscan or Keplr and a [set of filtering rules](https://docs.google.com/document/d/11FknyQr-hMsnfMkRfBUGHsLR18ZwttXZtPa7ZEQBXWg/edit) was already suggested.
- Query filtering: this is an other alternative suggested [in the same document](https://docs.google.com/document/d/11FknyQr-hMsnfMkRfBUGHsLR18ZwttXZtPa7ZEQBXWg/edit) as vote-based filtering rules are provided. The idea is to update the list of proposal endpoints, allowing the ability to toggle a filtering option on and off.
    
    > The endpoint that serves the proposals list is [Proposals](https://github.com/cosmos/cosmos-sdk/blob/v0.45.13-ics/x/gov/keeper/grpc_query.go#L38). This is the GRPC gov/proposals endpoint. This endpoint takes receives a [QueryProposalsRequest](https://github.com/cosmos/cosmos-sdk/blob/v0.45.13-ics/proto/cosmos/gov/v1beta1/query.proto#L68). To this request should be added an optional boolean field called “unfiltered”. If unfiltered is set to “true”, then the spam filtering will be disabled. Otherwise, by default, it is false, so filtering is active.
    > 
    
    > To do the filtering: Some other filtering happens in Proposals. This should be done first to narrow down the results. Then for each remaining proposal, [Tally](https://github.com/cosmos/cosmos-sdk/blob/v0.45.13-ics/x/gov/keeper/tally.go#L14) should be called to get the number of votes for each option. Then the same basic procedure as defined above can be followed
    > 

Vote-based filtering like the methods above is however harder to do on AtomOne, where the No with Veto option is removed.

Since spam proposals are also occasionally entering the voting period on some Cosmos chains, a possible mitigation to this issue could be raising the `MinDeposit` parameter. It is unclear however how high the deposit would need to be in order to disincentivize spam, and continuously adjusting the parameter value would in itself require constant attention and involvement of stakers, hence why we advocate for a dynamically adjusted threshold, that simply targets a low number of active proposals.

Moreover, there is currently no real alternative to throttle the number of *active* proposals for the purpose of mitigating voters fatigue.

## **Decision[](https://docs.cosmos.network/main/architecture/adr-template#decision)**

The `MinDeposit` parameter is replaced with a dynamically set variable that can be queried separately through a dedicated query. The `MinDeposit` parameter is therefore deprecated.

While the following description focuses on `MinDeposit`, the system can be replicated to dynamically adjust the `MinInitialDeposit` in place of a fixed `MinInitialDepositRatio`.

### Dynamically adjusting the `MinDeposit`

The `MinDeposit` should exponentially increase and decrease with time and upon proposal activation and deactivation to target having on average *N* proposals active at any time, with *N* being a low number, e.g. 1 or 2. The dynamic value will have a floor below which it will not be able to go. Conversely, it is allowed to grow with no bounds.

An update of the `MinDeposit` can be triggered **by the passage of a certain amount of time** - denoted as a *tick -* or **by the activation or deactivation of a proposal**, and the new value is calculated from the previous value as such:

$$
D_{t+1} = \max(D_{\min}, D_t \times (1+ sign(n_t - N) \times \alpha \times \sqrt[k]{| n_t - N |}))
\\
\alpha = \begin{cases} \alpha_{up} & n_t \gt N \\
\alpha_{down} & n_t \leq N
\end{cases}
\\
sign(n_t - N) = \begin{cases} 1 & n_t \geq N \\
-1 & n_t \lt N
\end{cases} \\ k \in \N \\
0 \lt \alpha_{down} \lt 1 \\ \\ 0 \lt \alpha_{up} \lt 1 \\ \alpha_{down} \lt \alpha_{up}\\ 
$$

Where:

- $D_{t+1}$ is the new deposit value, $D_{\min}$ is the floor deposit value and $D_t$ the deposit at time $t$
- $n_t$ is the number of active proposals at time $t$ and $N$ the target number of proposals
- $k$ expresses the *sensitivity* of the rate of increase or decrease to the distance of the number of active proposals $n_t$ with respect to the target $N$. Since the rate of change is proportional to this distance, $k$ can be used to tune how much getting further away from $N$ exacerbates things.
- $\alpha_{up}$ is the base rate of increase for each tick when the number of proposals exceeds the target by 1, and is a positive number between 0 and 1 (excluded)
- $\alpha_{down}$ is the base rate of decrease when the number of proposals is exactly at the target,  , and is a positive number between 0 and 1 (excluded)
- $\alpha_{down}  \lt \alpha_{up}$ implies that the rate of decrease will be slower than the rate of increase. A typical value might be $\alpha_{down} = \frac{\alpha_{up}}{2}$

### Regarding update frequency of $D_t$ and the possibility to update it lazily

In a naive implementation for time-based updates every $T$ time has elapsed (a *tick*) the current value of $n_t$ should be used to calculate $D_t$. Having the accurate $D_t$ available to query at all times is important to be able to correctly submit a proposal.
But in reality, all we need to know is when $n_t$ **actually** changed the last time (which happens when proposals enter or leave the active proposals queue, and in itself also trigger a `MinDeposit` update), and how much time has passed since then, anytime the $D_t$ value is requested.
Assume $n_{t_1}$ changed at a certain time $t_1$ (and $D_t$ was last updated at that time so it is equal to $D_{t_1}$), and has not changed up to a certain time $t_2$ when the deposit is queried or requested for a new proposal. At time $t_2$ all we need to know is $\tau = ticks = \lfloor \frac{\Delta t}{T} \rfloor = \lfloor \frac{t_2 - t_1}{T} \rfloor$ and  then a way to compute easily $D_{t_2}$ as a function of $D_{t_1}$ and the number of ticks elapsed $\tau$. Due to the fact that $n_t$ has not changed since $t_1$ and is still $n_{t_1}$, all we need is to do is apply the same rate of decrease/increase multiple times, i.e.

$$
D_{t+1} = \max(D_{\min}, D_t \times (1+ sign(n_t - N) \times \alpha \times \sqrt[k]{| n_t - N |})^{\tau})
$$

What does this mean in practice?

1. The deposit value need to only be actually updated in state when $n_t$ changes, i.e. when proposals actually enter or exit the voting period. Time-based updates can be avoided, and the current value calculated on-demand. To further simplify the computation, the value of $1+\alpha \times \sqrt[k]{| n_t - N |}$ could also be computed at this time (and maybe cached by nodes in memory instead of being stored in blockchain state) since it also won’t change until $n_t$ also changes again.
2. If the current deposit value at a certain time $t$ is requested - say $t_1$ was the last time the min deposit was actually updated in state due to a change of the total active proposals $n_t$, it can be computed lazily as
$D_t = \max( D_{\min}, D_{t_1} \times \gamma^\tau)$, where again $\tau = \lfloor \frac{t - t_1}{T} \rfloor$ - i.e. how many ticks have passed since $t_1$, and $\gamma = 1+ sign(n_t - N) \times \alpha \times \sqrt[k]{| n_t - N |}$ 

### Implementation

The following parameters are added to the `x/gov` module params:

- `min_deposit_floor`: floor value for the minimum deposit required for a proposal to enter the voting period
- `min_deposit_update_period`: duration that dictates after how long the dynamic minimum deposit should be recalculated for time-based updates.
- `target_active_proposals`:  the number of active proposals the dynamic minimum deposit should target.
- `min_deposit_increase_ratio`: the ratio of increase for the minimum deposit when the number of active proposals exceeds the target by 1.
- `min_deposit_decrease_ratio`: the ratio of increase for the minimum deposit when the number of active proposals is 1 less than the target.
- `min_deposit_sensitivity_target_distance`: A positive integer representing the sensitivity of the dynamic minimum deposit increase/decrease to the distance from the target number of active proposals. The higher the number, the lower the sensitivity. A value of 1 represents the highest sensitivity.

Proposals activation or deactivation - i.e. when a proposal is added or removed from the `keeper.ActiveProposalsQueue` - triggers a `MinDeposit` update which is also saved in state via setting the `LastMinDeposit`. Both the computed value and the time at which this was done (the `ctx.BlockTime()`) are stored. Whenever the `MinDeposit` is queried or requested the value is then calculated lazily using the value and timestamp of `LastMinDeposit` and computing the *ticks* passed since `LastMinDeposit.Time`, using the formula detailed above.

### Querying deposits value

Because the `MinDeposit` is no longer a fixed parameter, a new query endpoint is also required. The new endpoint will allow clients to fetch the expected price of the deposit.

### Replicating the same system for `MinInitialDeposit`

The same system described for `MinDeposit` could also be replicated for `MinInitialDeposit`. The only difference with the above formulation is that $n_t$ in this case is the number of proposals in deposit period, and updates of `MinInitialDeposit` would be triggered upon proposals entering or exiting the deposit period (the `keeper.InactiveProposalsQueue` in `x/gov`) and with time. Parameters would be an exact replica - just starting with `min_initial_deposit_*` instead - and the implementation would equally be very similar.

The two systems should be independent of each other and have no relation. The value of `MinInitialDeposit` should be allowed to even grow higher than `MinDeposit` in response to sudden spikes in inactive proposals. The dynamic system for `MinInitialDeposit` would have to be fine-tuned differently with respect to the one for `MinDeposit` to respond more rapidly to changes but also be able to decade quicker.

## **Consequences**

Simply put, if the total number of active proposals exceeds the target, the `MinDeposit` will exponentially increase over time. Also, the the higher the number of active proposals past the target threshold, the faster the increase will be. The `MinDeposit` can also decrease if the number of active proposals goes below the target, with a faster decrease the lower the number of active proposals. However, the value cannot go lower than a set `MinDepositFloor`.

### **Backwards Compatibility[](https://docs.cosmos.network/main/architecture/adr-template#backwards-compatibility)**

Existing non-active proposals will potentially see their `MinDeposit` requirement increase after the release of this feature. This will depend also on whether there are active proposals at the time of the upgrade.

### **Positive[](https://docs.cosmos.network/main/architecture/adr-template#positive)**

- Many active proposals are discouraged, but not prohibited.
- With a similar mechanism for the `MinInitialDeposit`, spamming governance proposals could become very costly especially if done in large numbers.

### **Negative[](https://docs.cosmos.network/main/architecture/adr-template#negative)**

- The cost of deposit can become prohibitive for regular users and price them out. However, this should be mitigated by the fact that total deposit can be crowdsourced.
- Increase the complexity of computing the deposit amount. Currently, the deposit is directly read from the governance parameters. With this change, it will come from a state variable that is re-evaluated every time new proposals enter or exit the voting period.

### **Neutral[](https://docs.cosmos.network/main/architecture/adr-template#neutral)**

- Increase the number of governance parameters
- Adds a new endpoint to query the `MinDeposit` value.

## **References[](https://docs.cosmos.network/main/architecture/adr-template#references)**

- [https://forum.cosmos.network/t/governance-proposal-deposit-auto-throttler/10121](https://forum.cosmos.network/t/governance-proposal-deposit-auto-throttler/10121)
