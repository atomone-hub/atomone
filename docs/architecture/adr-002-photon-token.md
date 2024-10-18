# ADR 002: The Photon token

## Changelog

- 15 October 2024: Initial version

## Status

DRAFT

## Abstract

This ADR proposes to introduce the PHOTON token (ticker `photon`) as the only
fee token of AtomOne. The only way to get PHOTON is to burn ATONE.

## Context

The PHOTON token is specified in the AtomOne Constitution Article 3 Section 5:

> ### Section 5: The PHOTON Token
> 
> The PHOTON shall be the only fee token except for ATONE to PHOTON burn
> transactions. This applies for all transactions on the root and core shards,
> and all IBC and ICS payments.
> 
> ATONE tokens may be burned to PHOTON tokens at a conversion rate set by law
> such that the total amount of PHOTONs mintable through burning ATONE tokens
> shall be capped at 1B PHOTON tokens.
> 
> PHOTONs cannot be converted back into ATONE tokens.

## Decision

The ADR proposes to create a new `photon` module to host the following
features:
- new query `ConvertRate`
- new message `MsgBurn` (or `MsgMint`? or `MsgConvert` ? or something else?).

### `ConvertRate` query

The `ConvertRate` query returns a decimal which represents the current
conversion rate of ATONE to PHOTON. This conversion rate is computed as the
following:

```math
conversion\_rate = \dfrac{photon_{max\_supply} - photon_{supply}}{atone_{supply}}
```
where
```math
photon_{max\_supply} = 1,000,000,000
```

Given this formula, when the PHOTON supply reaches the max supply of 1 billion,
it's no longer possible to get PHOTON, because the conversion rate will return
the 0 value.

### `MsgBurn` message

`MsgBurn` takes an amount of ATONE and returns an amount of PHOTON.
The passed ATONEs are burnt while the amount of PHOTONS is minted and moved
onto the message signer wallet. The number of minted PHOTONs is equal to the
number of burnt ATONE multiplied by the conversion rate described in the
`Convert` query section below.

```math
photons_{minted} = atones_{burned} \times conversion\_rate
```

### Params

The `photon` module has the following params:
- `mint_disabled` (default to false): if true, disable the ability to call
  `MsgBurn`.

### State

Aside from its params, the `photon` module doesn't have any additionnal state,
as the PHOTON balances and supply are handled by the `bank` module.

### Migration

TODO

## Consequences

> This section describes the consequences, after applying the decision. 
> All consequences should be summarized here, not just the "positive" ones.

### Positive

> {positive consequences}

### Negative

> {negative consequences}

### Neutral

> {neutral consequences}

## References

> Are there any relevant PR comments, issues that led up to this, or articles referrenced for why we made the given design choice? If so link them here!

* {reference link}
