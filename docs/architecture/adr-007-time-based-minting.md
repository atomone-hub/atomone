<!--
order: false
-->

# ADR 007: Time-based minting

## Changelog

- 2026-11-04: initial draft

## Status

PROPOSED Not Implemented

## Abstract

`x/mint` mints a fixed share of the annual provisions every block, sized by
the `blocks_per_year` parameter. AtomOne produces blocks faster than that
parameter assumes, so the chain mints about 25% more ATONE per year than its
nominal inflation rate, above the 20% cap set by the constitution.

This ADR makes minting proportional to the time elapsed between blocks: each
block mints `annual_provisions * elapsed / year`, and the inflation-rate step
is scaled by the same elapsed time. Issuance then matches the nominal rate
whatever the block time, and `blocks_per_year` is removed.

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
ATONE per year instead of 32.0 M, and a staking APR of about 59.8% instead of
the nominal 47.8%. The constitution caps inflation at 20% per year. The rate
step converges at an effective 16.2% per year instead of 13%.

A block-count parameter cannot be kept accurate. Block time depends on the
consensus timeouts, the validator set, network latency and the block content;
any value set by governance starts drifting as soon as one of those changes,
and correcting it requires a measurement and a proposal every time. Minting
against elapsed time removes the dependency on block count altogether.

## Decision

### Provisions and rate step proportional to elapsed time

Let `elapsed` be the block time of the current block minus the block time of
the previous block, capped at `MaxBlockElapsed` (see below), and

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
a timestamp before the previous one resets the reference point, as described
below, and mints nothing.

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
meaningful, and mints nothing. It still emits the mint event, with `amount`
and `elapsed_ns` set to 0, so that every block has one. This happens in two
cases:

- **Height 1 of a new chain.** A genesis without a `previous_block_time`
  leaves the field nil.
- **The first block after the upgrade that introduces this ADR.** A `Minter`
  stored by the previous binary has no field 3 and decodes with a nil
  `previous_block_time`, so no store migration is needed for it. The upgrade
  block forgoes one block of provisions (about 6 ATONE at current values).

`ExportGenesis` exports `previous_block_time` as stored and `InitGenesis`
imports it as is, so module state round-trips unchanged through an export and
import. On a chain restarted from an exported genesis, the first block
measures its elapsed time from the exported `previous_block_time`; the cap
below bounds that block's mint to `MaxBlockElapsed` worth of provisions,
whatever the time between the export and the restart.

When a block's time is earlier than the recorded reference, which happens on a
restart from a genesis whose `genesis_time` predates the exported reference (a
plain `atomoned export` keeps the original `genesis_time`, which CometBFT
stamps on the first block), the reference is reset and nothing is minted for
that block: like the nil case, the block records its own time, recomputes
`annual_provisions` and emits the mint event with `amount` 0. The following
block mints at most the cap.

### Cap on a single block's elapsed time

`elapsed` is capped at `MaxBlockElapsed = 5 * time.Minute`, a package
constant. After a halt, the first block with a post-halt timestamp would
otherwise mint the whole gap at once, and the distribution module would
allocate it in that same block to the validators in its last commit.

At the measured annual provisions (32.0 M ATONE):

| Gap before the block | Uncapped mint | Mint with the 5-minute cap |
|---|---|---|
| One normal block, 5.8 s | 5.9 ATONE | 5.9 ATONE |
| 5 minutes | 304 ATONE | 304 ATONE |
| 1 hour | 3,650 ATONE | 304 ATONE |
| v4 upgrade, 66.5 minutes | 4,050 ATONE | 304 ATONE |
| 24 hours | 87,700 ATONE | 304 ATONE |
| 7 days | 614,000 ATONE | 304 ATONE |

The cap never binds in normal operation: over the last 1,000 sampled blocks
the 99th percentile interval is 6.2 s and the maximum 7.8 s, and the slow
blocks right after an upgrade restart take 11 to 25 s. Its only effect is the
halt policy: a halted chain mints nothing for the duration of the halt beyond
the first 5 minutes, there is no single-block reward spike after an upgrade or
after a restart from an exported genesis, and the largest single-block mint is
bounded.

### Removal of `blocks_per_year`

`blocks_per_year` is removed from `Params`. The proto field is reserved
(`reserved 6; reserved "blocks_per_year";`), and it is absent from
`NewParams`, `DefaultParams`, parameter validation, the legacy param set and
the simulation randomizers. The `x/mint` consensus version is 3; the migration
from version 2 re-stores `Params` so that the stored bytes of the removed
field are dropped.

### Extension point

`x/mint` exposes a `MintFn` hook (`keeper.WithMintFn`), backported from
upstream Cosmos SDK v0.53. The keeper's default mint function,
`DefaultMintFn(types.DefaultInflationCalculationFn)`, implements the
time-based minting above, so AtomOne passes no option and wires nothing.
A future change to the inflation controller can replace the
`InflationCalculationFn` passed to `DefaultMintFn`, which already receives
the elapsed time, or replace the whole mint function, without changing the
time accounting in the minter state.

### Upgrade

The `v5` upgrade handler runs `RunMigrations`, which applies the `x/mint`
migration from consensus version 2 to 3. It adds and deletes no store keys.

## Consequences

### Positive

- Issuance per unit of time matches the nominal inflation rate regardless of
  block time, and the inflation-rate step converges at the configured
  `inflation_rate_change` per year. No parameter has to be re-measured and
  re-proposed when block time changes.
- `annual_provisions` becomes the exact basis for APR: an APR computed from
  `annual_provisions`, community tax and bonded tokens is the realized one.

### Negative

- The `x/mint` `params` query response loses the `blocks_per_year` field.
  Explorers and wallets that read it, for example to correct APR by observed
  block count, must be updated.
- Minting depends on BFT time. BFT time is the voting-power-weighted median
  of the commit's vote timestamps, so moving it requires a majority of the
  voting power that signs the commit; light clients also reject headers too
  far ahead of their local clock.

### Neutral

- A chain halt is not paid for beyond the first 5 minutes: the time a chain
  spends halted is not minted.
- `previous_block_time` is part of the exported genesis and is re-imported as
  is. The first block of a chain restarted from an export mints at most
  `MaxBlockElapsed` worth of provisions. When the first block's time is
  earlier than the exported `previous_block_time`, as with a plain `atomoned
  export` that keeps the original `genesis_time`, that block resets the
  reference and mints nothing, and the following block mints at most the cap.
- Tokens minted before the upgrade are unchanged; the fix applies from the
  upgrade onward.

## References

- Cosmos SDK PR #24436: `MintFn` hook in `x/mint` (v0.53)
- Cosmos SDK issues #5411, #19952, #22675
- Celestia ADR-019: strict inflation schedule, and celestia-app PR #1578
