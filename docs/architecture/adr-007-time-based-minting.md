<!--
order: false
-->

# ADR 007: Time-based minting

## Changelog

- 2026-11-04: initial draft
- 2026-11-05: `max_block_elapsed` parameter; security analysis for
  self-interested validators

## Status

PROPOSED (implementation: [#389](https://github.com/atomone-hub/atomone/pull/389))

## Abstract

`x/mint` mints a fixed share of the annual provisions every block, sized by
the `blocks_per_year` parameter. AtomOne produces blocks faster than that
parameter assumes, so the chain mints about 25% more ATONE per year than its
nominal inflation rate, above the 20% cap set by the constitution.

This ADR makes minting proportional to the time elapsed between blocks: each
block mints `annual_provisions * elapsed / year`, and the inflation-rate step
is scaled by the same elapsed time. Issuance then matches the nominal rate
regardless of the block time, and `blocks_per_year` is replaced by
`max_block_elapsed`, a cap on the time a single block accounts for.

## Context

`blocks_per_year` enters the `x/mint` math in two places:

1. **Block provision.** Each block mints `annual_provisions / blocks_per_year`.
   If the chain produces more blocks per year than the parameter assumes, it
   mints proportionally more per year.
2. **Inflation-rate step.** Each block moves the inflation rate toward
   `inflation_max` or `inflation_min` by
   `(1 - bonded_ratio / goal_bonded) * inflation_rate_change / blocks_per_year`.
   Faster blocks make the rate converge faster than `inflation_rate_change`
   per year.

`annual_provisions` (`inflation * total_supply`, recomputed every block) does
not depend on the parameter.

The parameter is set to 4,360,000, which corresponds to 7.24 s blocks over a
365.25-day year. Measured on atomone-1 at height 11,042,497 (2026-10-30), the
average block time since genesis (2024-10-18) is 5.81 s, and every measurement
window from the last 10,000 blocks to the whole chain lifetime falls between
5.77 s and 5.81 s. The chain therefore produces about 5.44 million blocks per
year and mints 1.25 times the nominal provisions.

With bonded ratio at 39.8%, below the 67% goal, the inflation rate is pinned at
`inflation_max` (20%). The effective issuance is about 25% per year: 40.1 M
ATONE per year instead of 32.0 M, and a staking reward rate of about 59.8%
per year instead of the nominal 47.8%. The constitution caps inflation at 20%
per year. The rate step converges at an effective 16.2% per year instead of
13%.

A block-count parameter cannot be kept accurate. Block time depends on the
consensus timeouts, the validator set, network latency and the block content;
any value set by governance starts drifting as soon as one of those changes,
and correcting it requires a measurement and a proposal every time. Minting
against elapsed time removes the dependency on block count altogether.

## Decision

### Provisions and rate step proportional to elapsed time

Let `elapsed` be the block time of the current block minus the block time of
the previous block, capped at the `max_block_elapsed` parameter (see below), and

```go
NanosecondsPerYear int64 = 31_557_600_000_000_000 // 365.25 days
```

Each block, in this order:

```text
inflation_rate_change_per_year = (1 - bonded_ratio / goal_bonded) * inflation_rate_change
inflation         = clamp(inflation + inflation_rate_change_per_year * elapsed / NanosecondsPerYear,
                          inflation_min, inflation_max)
annual_provisions = inflation * total_staking_supply
minted            = truncate(annual_provisions * elapsed / NanosecondsPerYear)
```

Both products are computed by multiplying by `elapsed` in nanoseconds before
dividing by `NanosecondsPerYear`, which keeps full `LegacyDec` precision. A
block with the same timestamp as the previous one mints nothing. A block with
a timestamp before the previous one is an error, as described below.

`InflationCalculationFn` takes the elapsed time as an extra argument:

```go
type InflationCalculationFn func(ctx context.Context, minter Minter, params Params, bondedRatio math.LegacyDec, elapsed time.Duration) math.LegacyDec
```

The mint event carries the elapsed time in nanoseconds in a new `elapsed_ns`
attribute, next to `bonded_ratio`, `inflation`, `annual_provisions` and
`amount`.

### Reference point: `previous_block_time`

The `Minter` state holds the timestamp of the previous block:

```proto
google.protobuf.Timestamp previous_block_time = 3 [(gogoproto.stdtime) = true, (gogoproto.nullable) = true];
```

Nil means there is no reference point yet. A block that finds it nil records
its own block time, recomputes `annual_provisions` so that queries stay
meaningful, mints nothing and emits no mint event. This happens in three
cases:

- **Height 1 of a new chain.** `InitGenesis` stores the minter without a
  `previous_block_time`, regardless of what the genesis file carries.
- **The first block after the upgrade that introduces this ADR.** A `Minter`
  stored by the previous binary has no field 3 and decodes with a nil
  `previous_block_time`, so no store migration is needed for it. The upgrade
  block forgoes one block of provisions (about 6 ATONE at current values).
- **The first block of a chain restarted from an exported genesis.**
  `ExportGenesis` writes the field as nil. The reference is bookkeeping that
  is only meaningful against the clock of the chain that recorded it, and a
  restarted chain starts a new clock, regardless of the `genesis_time` in its
  genesis file.

A block whose time is earlier than the recorded reference makes the mint
function return an error, which fails `FinalizeBlock` and halts the chain.
CometBFT stamps the initial-height block with `genesis_time` and requires
every later block time to be strictly greater than the previous one, and the
reference is never imported, so the case is unreachable unless state is
corrupted or consensus is broken. Halting is the SDK convention for a broken
invariant.

### Cap on a single block's elapsed time

`elapsed` is capped at the `max_block_elapsed` parameter, a duration with a
default of 8 s, set by the consensus version 3 migration on existing chains
and adjustable by governance. A block whose timestamp is further ahead of the
previous block's than the cap, after a halt or because its proposer set the
time ahead of the wall clock, mints the cap's worth of provisions and no more.
The parameter is meant to sit slightly above the block interval; Security
considerations gives the measurements behind the default and the effect of
other values.

At the measured annual provisions (32.0 M ATONE):

| Gap before the block | Uncapped mint | Mint with the 8 s cap |
| --- | --- | --- |
| One normal block, 5.8 s | 5.9 ATONE | 5.9 ATONE |
| 8 s | 8.1 ATONE | 8.1 ATONE |
| 1 hour | 3,650 ATONE | 8.1 ATONE |
| v4 upgrade, 66.5 minutes | 4,050 ATONE | 8.1 ATONE |
| 24 hours | 87,700 ATONE | 8.1 ATONE |
| 7 days | 614,000 ATONE | 8.1 ATONE |

A halted chain therefore mints nothing for the duration of the halt beyond
the first 8 s, there is no single-block reward spike after an upgrade or after
a restart from an exported genesis, and the largest single-block mint is
bounded. Blocks in normal operation stay below the cap, with rare exceptions:
over 4,000 recent blocks the 99.9th percentile interval is 6.93 s and the
maximum 10.26 s. The blocks that are capped are those after a missed proposer
(about 10.3 s) and those after an upgrade restart (11 to 25 s).

### Replacement of `blocks_per_year` by `max_block_elapsed`

`blocks_per_year` is removed from `Params`: the proto field is reserved
(`reserved 6; reserved "blocks_per_year";`), and it is absent from
`NewParams`, `DefaultParams`, parameter validation, the legacy param set and
the simulation randomizers. `max_block_elapsed` takes its place as proto field
7, a non-nullable `google.protobuf.Duration` that validation requires to be
positive. The `x/mint` consensus version is 3; the migration from version 2
sets `max_block_elapsed` to its default and re-stores `Params`, which also
drops the stored bytes of the removed field.

### Extension point

`x/mint` exposes a `MintFn` hook (`keeper.WithMintFn`), backported from
Cosmos SDK v0.53 ([cosmos/cosmos-sdk#24436](https://github.com/cosmos/cosmos-sdk/pull/24436)).
The keeper's default mint function,
`DefaultMintFn(types.DefaultInflationCalculationFn)`, implements the
time-based minting above, so AtomOne passes no option and wires nothing.
A future change to the inflation controller can replace the
`InflationCalculationFn` passed to `DefaultMintFn`, which already receives
the elapsed time, or replace the whole mint function, without changing the
time accounting in the minter state.

## Alternatives considered

### Keeping `blocks_per_year` and correcting it by governance

A parameter change to 5,440,000, the value implied by the measured 5.80 s
block time, fixes the current drift and needs no upgrade. It is proposed as an
interim step, but it leaves the mechanism in place: block time changes with
every consensus timeout change, validator set change and upgrade, and each
change reopens the gap until someone measures and proposes again. Removing the
dependency on block count is the only durable fix.

### Minting per epoch instead of per block

An epoch-based mint, once per N blocks or once per day of block time, does not
remove the dependency on block count unless the epoch itself is defined in
time, at which point it is the time-based design with a coarser granularity
and one large mint per epoch. The Cosmos SDK prototyped an epoch-based
`MintFn` for v0.52 and did not ship it; the related issues
([#5411](https://github.com/cosmos/cosmos-sdk/issues/5411),
[#22675](https://github.com/cosmos/cosmos-sdk/issues/22675)) were closed in
favour of a custom `MintFn`, which is the hook this ADR backports.

### Paying for halts: catch-up or cap

Celestia ([ADR-019](https://github.com/celestiaorg/celestia-app/blob/main/docs/architecture/adr-019-strict-inflation-schedule.md)),
Initia and Babylon mint the whole gap in the first block
after a halt. Crescent caps each block at 10 s. A catch-up spread over the
following blocks needs extra state to track the debt. The cap was chosen
because a catch-up pays the entire halt to the validators of one commit in one
block, and because there is no reason to issue provisions for a period in
which the chain produced no blocks. The cap value is discussed under Security
considerations.

### Cap as a constant or as a parameter

A constant in code needs an upgrade to follow a change of the block interval,
while the parameter follows it by a governance proposal. The parameter was
chosen because its value is a property of the network rather than of the
design, and because its tolerance is wide, unlike that of `blocks_per_year`:
set too low it under-mints on the slowest blocks, set too high it loosens the
bound on a majority coalition, and neither changes issuance in normal
operation by more than the under-mint on the slowest blocks, 0.01% at the
default. Validators cannot raise it with delegated stake, since AtomOne's
governance does not let them vote with delegations.

### Where the reference time lives

`BeginBlock` sees only the current header, so the previous block time must be
stored by the module. It is stored on `Minter` rather than under a separate
key so that it is exported, imported and migrated with the rest of the minter
state. Reading it from the `x/staking` historical info was rejected: entries
are pruned by `historical_entries`, and the mint hook would read another
module's store.

### Year length

`NanosecondsPerYear` uses a 365.25-day year, the convention behind the
inherited `blocks_per_year` (4,360,000 blocks of 7.24 s). Celestia uses the
Gregorian 365.2425 days. The difference is 0.002% of the annual provisions,
so the inherited convention is kept.

### `previous_block_time` across export and import

**Clear (implemented; Celestia does the same).** `ExportGenesis` writes the
field as nil and `InitGenesis` ignores it, so a chain always starts without a
reference and its first block mints nothing, exactly like a new chain and like
the upgrade block. The field is runtime bookkeeping that is only meaningful
against the clock of the chain that wrote it, and a restarted chain has a new
clock. The SDK simulation's import/export comparison (`TestAppImportExport`)
skips the minter key, as it already skips the staking queues and the slashing
bitmaps for state that is not meant to round-trip.

**Keep.** `ExportGenesis` writes the stored value and `InitGenesis` stores
what it reads, so module state round-trips unchanged. The first block of a
chain restarted from an export then measures its elapsed time from the
exported value, bounded by the cap, and a plain `atomoned export`, which
keeps the original `genesis_time` that CometBFT stamps on the first block,
makes the earlier-block-time case below reachable on every restart from an
export. Rejected: the reference has no meaning against another clock, and the
design would need a recovery path for a case that should be an invariant.

### A block time earlier than the reference

**Error (implemented).** Return an error from the mint hook, which fails
`FinalizeBlock` and halts the chain. With the reference cleared on import the
case is unreachable: an earlier block time means corrupted state or a
consensus bug, and halting is the SDK convention for a broken invariant.

**Reset.** Treat the block like the no-reference case: record its time and
mint nothing. Self-healing, and necessary with the "keep" design above, where
a plain `atomoned export` reproduces the case on every restart from an
exported state. Rejected together with that design.

Treating the block as zero elapsed while keeping the reference was rejected
in both designs: a reference in the future would stop minting until the chain
time caught up with it.

### Mint event on a block that mints nothing

The no-reference block can emit the mint event with `amount` and `elapsed_ns`
at 0, so that every block carries exactly one mint event as it does today, or
emit nothing, as Celestia does. The implemented choice is to emit nothing: the
event with two zero attributes is a sample that an indexer computing issuance
per second must special-case (0 / 0), while an absent event on one block per
chain start is handled by an indexer summing amounts without special-casing.

## Security considerations

Minting against block time makes issuance depend on BFT time, which
validators set. Validators are taken to be self-interested rather than
honest: a precommit timestamp set ahead of the wall clock is not punished, so
it has to be unprofitable rather than merely against the protocol. This
section states what a coalition of validators can do with block time, what
the chain already trusts it for, and what `max_block_elapsed` bounds.

**How block time is set.** In CometBFT v0.38 a block's time is the
voting-power-weighted median of the precommit timestamps in its last commit;
full nodes reject any other value, and require each block time to be strictly
greater than the previous one. The proposer selects which precommits form the
commit, any set above two thirds of the voting power. Nothing checks a block
time against local clocks. A validator running the stock software votes with
the later of its own clock and the proposed block time plus one millisecond.
On atomone-1 the seven largest validators hold a third of the voting power
and the twenty-five largest two thirds (2026-11-04).

**Three regimes.** Let `I` be the block interval (5.8 s), `C` the value of
`max_block_elapsed` and `p` the share of the voting power held by a coalition
that sets its precommit timestamps ahead of the wall clock; `p` is also its
share of the proposed blocks.

| Coalition share `p` | Control over block time | Sustained push |
| --- | --- | --- |
| up to 1/3 | none | unchanged |
| 1/3 to 1/2 | the blocks it proposes | `p * C / I`, a loss if `C < I / p` |
| above 1/2 | every block | `C / I` |

A coalition between a third and a half controls the time of the blocks it
proposes by including only as many other precommits as needed to reach two
thirds; the median then falls among its own timestamps. On the other blocks
the median falls among the other validators' timestamps. In this regime a
pushed block is followed by blocks proposed by the other validators, whose
timestamps are the pushed time plus one millisecond until the wall clock
catches up, so those blocks mint nothing. A single push brings issuance
forward without increasing it. A sustained push mints `C` once per cycle of
`1/p` blocks, which lasts `I / p` of wall-clock time, so issuance runs at `p *
C / I` times its normal rate and the coalition loses whenever `C < I / p`.
With `C` at or below twice the interval, no coalition up to half of the voting
power profits.

Above half, the coalition's precommits outweigh the rest in any commit, so
it sets every block's time; issuance runs at `C / I`
and chain time runs ahead of the wall clock by `C - I` per block. IBC light
clients on counterparty chains reject headers more than their
`max_clock_drift` (10 s by default) ahead of their local clock, so outbound
IBC stops within a few blocks, and the gap between block time and wall-clock
time is visible to every observer. A coalition above two thirds also sets the
block rate and every other rule.

**What the chain already trusts block time for.** The staking module
completes unbondings and redelegations by block time, the governance module
ends voting periods and runs quorum checks by block time, vesting accounts
release by block time, and IBC packet timeouts and light-client trusting
periods are measured in it. None of these is bounded. A coalition above a
third of the voting power that sets one block's time 21 days ahead completes
every unbonding delegation and closes every vote in that block. Time-based
minting adds no new trust assumption; it adds one more consumer of block
time, bounded by `max_block_elapsed`.

**Comparison with block-count minting.** Validators also set the block rate,
through their `timeout_commit` configuration and their latency, and under
block-count minting the block rate sets issuance: the 25% effective rate this
ADR corrects was set that way, without any decision being taken and without
any observable anomaly. A coalition above a third of the voting power can
slow the chain and reduce issuance arbitrarily. Under time-based minting the
block rate is irrelevant and the only lever is the timestamp push above.

**The value of `max_block_elapsed`.** The parameter should sit slightly
above the block interval. Measured on atomone-1 on 2026-11-04 over 4,000
consecutive blocks (mean 5.78 s, 99.9th percentile 6.93 s, maximum 10.26 s),
and over a 1,000-block window from three months earlier in which 1.5% of the
blocks took 10.3 to 10.6 s because a proposer was down, with the under-mint
as a share of the provisions of each window:

| `max_block_elapsed` | Majority bound | Quiet window | Degraded window |
| --- | --- | --- | --- |
| 6.5 s | 1.12x | 0.03% | 0.99% |
| 7 s | 1.21x | 0.02% | 0.86% |
| 8 s (default) | 1.38x | 0.01% | 0.60% |
| 10 s | 1.72x | 0.001% | 0.09% |
| 15 s | 2.6x | 0 | 0 |

The default of 8 s is 1.4 times the interval: no coalition up to half of the
voting power profits from a push, a majority is bounded to 1.38 times the
nominal issuance while visibly breaking IBC, and the cost in normal operation
is 0.01% of the provisions. On a slow block the cap never pays less than
block-count minting does: a 10.3 s block mints 5.8 s of provisions today and
8 s under the default.

**Proposer-based timestamps.** The structural fix is to have the proposer set
the block time and have every validator reject a proposal whose time is more
than `PRECISION` ahead of its own clock (505 ms in CometBFT's examples).
CometBFT's [proposer-based timestamps](https://github.com/cometbft/cometbft/blob/main/spec/consensus/proposer-based-timestamp/README.md)
(PBTS) do that, and a coalition then needs two thirds of the voting power to
set a block time at all. PBTS shipped in CometBFT v1.x, which has since been
[retracted](https://docs.cosmos.network/sdk/latest/release-family) and is not
part of any supported Cosmos SDK release family; the v0.38, v0.39 and v0.40
lines do not carry it. When AtomOne moves to a CometBFT line that does,
enabling PBTS leaves `max_block_elapsed` governing only halts.

Celestia, which has minted by elapsed block time since 2023 without a cap,
accepted the exposure as such: its design document notes that the weighted
median timestamp "has the same 1/3+ security model as the rest of
Tendermint".

## Consequences

### Positive

- Issuance per unit of time matches the nominal inflation rate regardless of
  block time, and the inflation-rate step converges at the configured
  `inflation_rate_change` per year. No parameter has to be re-measured and
  re-proposed when block time changes.
- `annual_provisions` becomes the exact basis for the staking reward rate: a
  rate computed from `annual_provisions`, the community tax and the bonded
  tokens is the realized one.

### Negative

- The `x/mint` `params` query response loses the `blocks_per_year` field.
  Explorers and wallets that read it, for example to correct the displayed
  reward rate by the observed block count, must be updated.
- Minting depends on BFT time, which validators set. A coalition above half
  of the voting power can raise issuance to at most `max_block_elapsed` over
  the block interval, 1.38 times the nominal rate at the default, by running
  chain time ahead of the wall clock and breaking outbound IBC; smaller
  coalitions cannot profit. See Security considerations.
- `max_block_elapsed` has to be kept slightly above the block interval. A
  value below the interval under-mints every block; a value far above it
  loosens the bound on a majority coalition.

### Neutral

- A chain halt is not paid for beyond the first `max_block_elapsed` (8 s by
  default): the time a chain spends halted is not minted.
- `previous_block_time` is cleared on export and on import. The first block
  of a chain restarted from an exported genesis records a new reference and
  mints nothing, regardless of the `genesis_time` in the file.
- Tokens minted before the upgrade are unchanged; the fix applies from the
  upgrade onward.

## References

- Cosmos SDK [PR #24436](https://github.com/cosmos/cosmos-sdk/pull/24436):
  `MintFn` hook in `x/mint` (v0.53)
- Cosmos SDK issues [#5411](https://github.com/cosmos/cosmos-sdk/issues/5411),
  [#19952](https://github.com/cosmos/cosmos-sdk/issues/19952) and
  [#22675](https://github.com/cosmos/cosmos-sdk/issues/22675)
- Celestia [ADR-019](https://github.com/celestiaorg/celestia-app/blob/main/docs/architecture/adr-019-strict-inflation-schedule.md):
  strict inflation schedule, and celestia-app
  [PR #1578](https://github.com/celestiaorg/celestia-app/pull/1578);
  celestia-app [`x/mint/keeper/genesis.go`](https://github.com/celestiaorg/celestia-app/blob/main/x/mint/keeper/genesis.go)
  clears `previous_block_time` on export and on import, and
  [`x/mint/types/minter.go`](https://github.com/celestiaorg/celestia-app/blob/main/x/mint/types/minter.go)
  mints the elapsed time without a cap
- Crescent [`x/mint/abci.go`](https://github.com/crescent-network/crescent/blob/main/x/mint/abci.go)
  caps the elapsed time at its `BlockTimeThreshold` parameter; Babylon
  [`x/mint/abci.go`](https://github.com/babylonlabs-io/babylon/blob/main/x/mint/abci.go)
  and Initia [`x/reward/abci.go`](https://github.com/initia-labs/initia/blob/main/x/reward/abci.go)
  mint the elapsed time without a cap
- CometBFT v0.38 [`state/validation.go`](https://github.com/cometbft/cometbft/blob/v0.38.x/state/validation.go)
  (block time rules) and
  [`consensus/state.go`](https://github.com/cometbft/cometbft/blob/v0.38.x/consensus/state.go)
  `voteTime` (precommit timestamp rule)
- CometBFT [`spec/consensus/time.md`](https://github.com/cometbft/cometbft/blob/main/spec/consensus/time.md)
  (BFT Time against PBTS, tolerated Byzantine fractions) and
  [`spec/consensus/proposer-based-timestamp/README.md`](https://github.com/cometbft/cometbft/blob/main/spec/consensus/proposer-based-timestamp/README.md)
  (the `timely` predicate)
- Cosmos SDK [release families](https://docs.cosmos.network/sdk/latest/release-family):
  CometBFT v1.x retracted
