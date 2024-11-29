# ADR 002: The Photon token

## Changelog

- 15 October 2024: Initial version
- 30 October 2024: Revisions
- 15 November 2024: Replace `TxFeeChecker` by `AnteDecorator`

## Status

DRAFT

## Abstract

This ADR proposes the introduction of the PHOTON token as the only fee token of
AtomOne. The only way to get PHOTONs is to burn ATONEs, with a one-way burn
that is not reversible at protocol level.

The PHOTON denom is `photon`, while the base denom is `uphoton`, with:
```
1 photon = 1,000,000 uphoton
```
Any subsequent formula in this document will use the `photon` denom for
brevity.

## Context

The PHOTON token is specified in the [AtomOne Constitution Article 3 Section 5]:

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

The ADR proposes to create a new `x/photon` module to host the following
features:
- New `ConversionRate` query
- New `MsgMintPhoton` message
- New `AnteDecorator` implementation to enforce the PHOTON token as the only
  fee token.

### `ConversionRate` query

The `ConversionRate` query returns a `sdk.Dec` which represents the current
conversion rate of ATONE to PHOTON. This conversion rate is computed as the
following:

```math
conversion\_rate = \dfrac{photon_{max\_supply} - photon_{supply}}{atone_{supply}}
```
where
```math
photon_{max\_supply} = 1,000,000,000
```

The `conversion_rate` will therefore be a function of the ATONE supply and the
current PHOTON supply. The `conversion_rate` is expected to naturally decrease
over time as the ATONE supply increases by means of inflation.

### `MsgMintPhoton` message

`MsgMintPhoton` takes an amount of ATONEs and returns a corresponding amount
of PHOTONs. The amount of ATONEs is burnt while the corresponding amount of
PHOTONs is minted and sent to the message signer account (who is also the sender
of the ATONEs that are burnt). The number of minted PHOTONs is equal to the
number of burnt ATONEs multiplied by the conversion rate described in the
`ConversionRate` query section below.

```math
photons_{minted} = atones_{burned} \times conversion\_rate
```

The supply of PHOTON, although capped at 1B, will never reach this cap because
the situation in which this happens is if all ATONEs in circulation are burnt.
This is not possible if there are ATONEs staked, which is a requirement to be
able to produce blocks. This also means in practice that this message is
expected to never be able to fail because of insufficient mintable PHOTONs.

There is however still a need to check that after calculations the amount of 
photons to mint is non-zero, which due to rounding is a possibility when 
burning very small fractions of the staking token (ATONE) supply.

The total PHOTON supply will be a constant hard-coded within the `x/photon`
module.

### `AnteDecorator` implementation

The `AnteDecorator` is an interface that must be implemented to add a new
decorator to the `AnteHandler`. Just before the `auth/ante.DeductFeeDecorator`,
we want to add a decorator that should:

- enforce that the fee denom is `uphoton`, and return a specific error message if
  it does not (this to be explicitely separated with the insufficient fee error
  message)
- make exception for some messages, specifically like `MsgMintPhoton`, because
  `MsgMintPhoton` is the only way to get PHOTONs, so it should accept ATONEs as
  fee token. The list of exceptions will in fact be a module parameter.

### Params

The `photon` module has the following params:

- `mint_disabled` (default to `false`): if `true`, disable the ability to call
  `MsgMintPhoton`.
- `txfee_exceptions` (default to `["MsgMintPhoton"]`): list of messages that
  are allowed to have ATONE as fee token as well as PHOTON.

### State

Aside from its params, the `x/photon` module does not have any additionnal state,
as the PHOTON balances and supply are handled by the `x/bank` module.

### Migration

The PHOTON denom metadata has to be added to the `x/bank` module state (XXX while
admittedly this record does not look very usefull, it is only used in
queries...)

```json
{
  "description": "The fee token of AtomOne Hub",
  "denom_units": [
    {
      "denom": "uphoton",
      "exponent": 0,
      "aliases": [
        "microphoton"
      ]
    },
    {
      "denom": "mphoton",
      "exponent": 3,
      "aliases": [
        "milliphoton"
      ]
    },
    {
      "denom": "photon",
      "exponent": 6,
      "aliases": [
        "photon"
      ]
    }
  ],
  "base": "uphoton",
  "display": "photon",
  "name": "AtomOne Photon",
  "symbol": "PHOTON",
  "uri": "",
  "uri_hash": ""
}
```

In contrast, it is not required to provide an initial supply for PHOTON (still
in the `x/bank` module), because the initial supply will be 0.

## Consequences

### Validator `minimum-gas-prices`

Validators will have to update their `minimum-gas-prices` setting to reflect
this new setup. It should basically allow both `uatone` or `uphoton`, so the setting
should look like:

```toml
minimum-gas-prices = "0.001uatone,0.001uphoton"
```

If the validator `minimum-gas-prices` does not match the required denom (`uatone`
or `uphoton` for `MsgMintPhoton` or any message in `txfee_exceptions`, and only
`uphoton` for all other messages), an error is returned and the tx is rejected.

### ICS payments and core shard fees enforcement

The proposed `x/photon` module does not enforce the usage of PHOTON as payment
token for ICS. This enforcement is left to be done by the ICS implementation.

Seemingly, for future core shards the enforcement of PHOTON as the fee token
will have to be done at the shard level. A simple way to do this would be to
use the same `TxFeeChecker` implementation as the root shard (i.e. the one
provided by the `x/photon` module).

### Positive

- Create a clear distinction between fee token (PHOTON) and staking token
  (ATONE). ATONE is meant to be a pure staking token and allow to freely
  inflate between the 7% and 20% bounds,targeting the 2/3 bonding ratio.
  PHOTON as the fee token reinforces this property.

- Having a non-inflationnary fee token (in contrast to ATONE) ensures PHOTON
  will not be subject to the same dilution tax for non-stakers as ATONE does.
  PHOTON holders will be subject to no dilution at all. The more stable nature
  of PHOTON  makes it a perfect candidate for a fee token.

### Negative

- Enforcing PHOTON as the only fee token might be seen as a limitation for
  users who would like to pay fees in other tokens. This is a trade-off to
  ensure the stability of the fee token.

### Neutral

- Dual token model like this has not been experimented at this scale in the
  Cosmos ecosytem, we might experience some unexpected side effect, positive or
  negative.

## References

* [AtomOne Constitution Article 3 Section 5]: The PHOTON Token

[AtomOne Constitution Article 3 Section 5]: https://github.com/atomone-hub/genesis/blob/b84df30364674c3f68b4bc0a43d7ed977ae22226/CONSTITUTION.md#section-5-the-photon-token
