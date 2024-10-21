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
- New query `ConvertRate`
- New message `MsgBurn` (or `MsgMint`? or `MsgConvert` ? or something else?).
- New [`TxFeeChecker`] implementation to enforce the PHOTON token as the only
  fee token.

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

### `TxFeeChecker` implementation

The [`TxFeeChecker`] is a function definition that is part of the ante handler
`auth/ante.DeductFeeDecorator`. When this ante handler is invoked, it calls the
`TxFeeChecker` to ensure the fee provided in the tx is enough.

Currently, AtomOne uses the default `TxFeeChecker` (namely
[`checkTxFeeWithValidatorMinGasPrices`]), so the photon module must provide an
alternative `TxFeeChecker` implementation, which should:
- enforce that the fee denom is PHOTON, and return a specific error message if
  it does not.
- make exception for some messages, specifically like `MsgBurn`, because
  `MsgBurn` is the only way to get PHOTON, so it should accept ATONE as fee
  token.

### Validator `minimum-gas-prices`

Validators will have to update their `minimum-gas-prices` setting to reflect
this new setup. It should basically allow both ATOM and PHOTON, so the setting
should like:

```toml
minimum-gas-prices = "0.001uatone,0.001uphoton"
```

> [!IMPORTANT]
> In the legacy `TxFeeChecker` implementation
> ([`checkTxFeeWithValidatorMinGasPrices`]), the validator `minimum-gas-prices`
> is checked against *all* mentionned denoms. For the photon module, the
> implementation must be different, it must be checked on at least one denom
> (ATONE or PHOTON, but not both).

If the validator `minimum-gas-prices` does not match the required denom (ATONE
or PHOTON for `MsgBurn, only `PHOTON` for all other messages), an error is
returned and the tx is rejected.

### Params

The `photon` module has the following params:
- `mint_disabled` (default to false): if true, disable the ability to call
  `MsgBurn`.

### State

Aside from its params, the `photon` module do not have any additionnal state,
as the PHOTON balances and supply are handled by the `bank` module.

### Migration

TODO

## Consequences

> This section describes the consequences, after applying the decision. 
> All consequences should be summarized here, not just the "positive" ones.

### Positive

> {positive consequences}

### Negative

- Users will have to choose between PHOTON and ATONE for the fee token when
  signing a transaction for AtomOne. Basically, for `MsgBurn` ATONE and PHOTON
  are accepted, for all other messages, only PHOTON is accepted. While this
  may seem obvious, it can be confusing because as far as we know, wallets do
  not have logic regarding the choice of the fee tokens. Maybe it is time to
  start discussion with some wallets dev regarding that, this would improve the
  UX.

### Neutral

> {neutral consequences}

## References

> Are there any relevant PR comments, issues that led up to this, or articles referrenced for why we made the given design choice? If so link them here!

* {reference link}

[`TxFeeChecker`]: https://github.com/cosmos/cosmos-sdk/blob/44c5d17ca6d9d37fdd6adfa3169c986fbce22b8f/x/auth/ante/fee.go#L11-L13
[`checkTxFeeWithValidatorMinGasPrices`]: https://github.com/cosmos/cosmos-sdk/blob/6e59ad0deea672a21e64fdc83939ca812dcd2b1b/x/auth/ante/validator_tx_fee.go#L17
