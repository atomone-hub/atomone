<!--
order: false
-->

# ADR 007: Time-based minting

## Changelog

- 2026-11-04: initial draft

## Status

PROPOSED Not Implemented

Open points, to be settled before this ADR is accepted (see Alternatives
considered and Security considerations):

1. `previous_block_time` across export and import: keep (implemented) or clear.
2. A block time earlier than the reference: reset (implemented) or error.
3. Mint event on a block that mints nothing: emit with zero attributes
   (implemented) or none.
4. The value of `MaxBlockElapsed`: 5 minutes (implemented) or tighter.

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
| --- | --- | --- |
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
bounded. The cap also bounds what a coalition that controls block time can
mint in one block; see Security considerations.

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
and one large mint per epoch. Upstream prototyped an epoch-based `MintFn` for
v0.52 and did not ship it; the related issues (#5411, #22675) were closed in
favour of a custom `MintFn`, which is the hook this ADR backports.

### Paying for halts: catch-up or cap

Celestia (ADR-019), Initia and Babylon mint the whole gap in the first block
after a halt. Crescent caps each block at 10 s. A catch-up spread over the
following blocks needs extra state to track the debt. The cap was chosen
because a catch-up pays the entire halt to the validators of one commit in one
block, and because a chain that is not producing blocks is not providing the
service that inflation pays for. The cap value is discussed under Security
considerations.

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

**Keep (implemented).** `ExportGenesis` writes the stored value and
`InitGenesis` stores what it reads. Module state round-trips unchanged through
export and import, which the SDK simulation (`TestAppImportExport`) checks for
every store. The first block of a chain restarted from an export measures its
elapsed time from the exported value, bounded by the cap, and the
earlier-block-time case below becomes reachable.

**Clear (Celestia).** `ExportGenesis` writes the field as nil and
`InitGenesis` ignores it, so a chain always starts without a reference and its
first block mints nothing, exactly like a new chain and like the upgrade block.
The field is runtime bookkeeping that is only meaningful against the clock of
the chain that wrote it, and a restarted chain has a new clock. The
simulation's import/export comparison must then skip the minter key, as it
already skips the staking queues and the slashing bitmaps for state that is
not meant to round-trip; this is one entry in the test's skip list. With this
design the earlier-block-time case is unreachable in practice: CometBFT stamps
the initial-height block with `genesis_time` and requires every later block
time to be strictly greater than the previous one.

### A block time earlier than the reference

**Reset (implemented).** Treat the block like the no-reference case: record
its time and mint nothing. Self-healing; chosen because, with the "keep"
design above, a plain `atomoned export` reproduces this case on every restart
from an exported state, and an error would halt that restart.

**Error.** Return an error from the mint hook, which fails `FinalizeBlock` and
halts the chain. Appropriate only if the case is unreachable, which the
"clear" design makes true: an earlier block time then means corrupted state or
a consensus bug, and halting is the SDK convention for a broken invariant.

Treating the block as zero elapsed while keeping the reference was rejected
in both designs: a reference in the future would stop minting until the chain
time caught up with it.

### Mint event on a block that mints nothing

The no-reference block, and the reset block if kept, can emit the mint event
with `amount` and `elapsed_ns` at 0, so that every block carries exactly one
mint event as it does today, or emit nothing, as Celestia does. The
implemented choice is to emit. The event with two zero attributes is a sample
that an indexer computing issuance per second must special-case (0 / 0); an
absent event is one block without the event, which an indexer summing amounts
handles without special-casing.

## Security considerations

Minting against block time replaces a quantity validators cannot influence,
the block count, with one they can: BFT time.

**How block time is set.** In CometBFT v0.38 a block's time is the
voting-power-weighted median of the precommit timestamps in its last commit;
full nodes reject any other value, and require each block time to be strictly
greater than the previous one. The proposer selects which precommits form the
commit (any set above two thirds of the voting power), so a coalition that
proposes a block controls its time if it holds more than half of the voting
power in the commit it assembles: more than a third of the total, when it
includes just enough honest precommits to reach two thirds. On blocks proposed
by honest validators, a coalition below half of the total voting power cannot
move the median beyond the honest timestamps. Nothing checks the proposed time
against local clocks; that is proposer-based timestamps, which ship in
CometBFT v1.

**What a coalition can do.** It can set the time of the blocks it proposes
ahead of wall-clock time. Each such block mints `min(elapsed, MaxBlockElapsed)`
worth of provisions, so a block pushed far ahead mints at most the cap: 304
ATONE for a 5-minute cap at the current annual provisions, 52 times a normal
5.8 s block. The distribution module pays it to the validators and delegators
of the previous commit pro rata to voting power, so the coalition's own gain
is its stake share of the extra amount.

**Why it is a time shift rather than extra issuance.** Honest validators sign
precommits with timestamp `max(now, proposed_block_time + 1 ms)`. Once the
chain time has been pushed ahead of wall-clock time, honest-proposed blocks
advance it by 1 ms and mint nothing measurable, until wall-clock time catches
up with the chain time. Over any period that starts and ends with chain time
in step with wall-clock time, total issuance equals what the elapsed
wall-clock time mints. A push brings issuance forward, paid to the stake
distribution at the time of the push; it does not increase it unless the
coalition pushes without end, in which case chain time runs ahead of
wall-clock time without bound. IBC light clients on counterparty chains reject
headers more than their `max_clock_drift` (10 s by default) ahead of their
local clock, so a sustained push disables outbound IBC within minutes, and the
gap between block time and wall-clock time is visible to every observer.

**What the cap bounds.** The cap bounds the single-block spike and the rate at
which issuance can be pulled forward. With the cap at `C` and a coalition that
proposes a share `p` of the blocks, chain time advances by at most `p * C` per
block, and issuance runs at most `p * C / 5.8 s` times its normal rate while
the push lasts. For a coalition just above a third of the voting power:

| Cap `C` | Mint of one pushed block | Relative to a normal block | Issuance rate with p = 1/3 |
| --- | --- | --- | --- |
| 5 min (implemented) | 304 ATONE | 52x | 17x |
| 60 s | 61 ATONE | 10x | 3.4x |
| 30 s | 30 ATONE | 5.2x | 1.7x |

Normal operation never approaches these values: the slowest block in the
sampled 1,000 blocks is 7.8 s, and the restart blocks after an upgrade take
11 to 25 s. A tighter cap costs nothing in normal operation and only changes
how much of a halt is paid. A budget over a window of blocks (at most `N`
seconds of elapsed time per `M` blocks) was considered and rejected: it needs
a nominal block time to define `N / M`, which is the parameter this ADR
removes.

**Comparison with the current design.** Under block-count minting a coalition
cannot change issuance at all. The new attack surface is the time shift
above, available to the same coalition (above a third of the voting power)
that can already halt the chain, and bounded by the cap and by the catch-up.

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
- Minting depends on BFT time, which validators set. A coalition above a third
  of the voting power can pull issuance forward by up to the cap on each block
  it proposes; see Security considerations for why this shifts issuance in
  time rather than adding to it, and for what the cap bounds.

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
- Celestia ADR-019: strict inflation schedule, and celestia-app PR #1578;
  celestia-app `x/mint/keeper/genesis.go` clears `previous_block_time` on
  export and on import
- CometBFT v0.38 `state/validation.go` (block time rules) and
  `consensus/state.go` `voteTime` (precommit timestamp rule)
