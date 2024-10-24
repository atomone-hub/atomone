# ADR 002: The Photon token

## Changelog

- 15 October 2024: Initial version

## Status

DRAFT

## Abstract

This ADR proposes to introduce the PHOTON token (ticker `photon`) as the only
fee token of AtomOne. The only way to get PHOTONs is to burn ATONEs.

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

The ADR proposes to create a new `photon` module to host the following
features:
- New `ConversionRate` query
- New `MsgBurn` message (XXX or `MsgMint`? or `MsgConvert` ? or something else?).
- New [`TxFeeChecker`] implementation to enforce the PHOTON token as the only
  fee token.

### `ConversionRate` query

The `ConversionRate` query returns a decimal which represents the current
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
it is no longer possible to get PHOTONs, because the conversion rate will return
the 0 value.

### `MsgBurn` message

`MsgBurn` takes an amount of ATONEs and returns an amount of PHOTONs.
The amount of ATONEs is burnt while the amount of PHOTONs is minted and moved
onto the message signer account. The number of minted PHOTONs is equal to the
number of burnt ATONEs multiplied by the conversion rate described in the
`ConversionRate` query section below.

```math
photons_{minted} = atones_{burned} \times conversion\_rate
```

However, if `conversion_rate` is 0, i.e. the maximum supply of PHOTON has been
reached, `MsgBurn` should fail to avoid burning ATONEs without PHOTONs in
return.

### `TxFeeChecker` implementation

The [`TxFeeChecker`] is a function definition that is part of the ante handler
`auth/ante.DeductFeeDecorator`. When this ante handler is invoked, it calls the
`TxFeeChecker` to ensure that the fee specified in the tx is sufficient.

Currently, AtomOne uses the default `TxFeeChecker` (namely
[`checkTxFeeWithValidatorMinGasPrices`]), so the photon module must provide an
alternative `TxFeeChecker` implementation, which should:
- enforce that the fee denom is PHOTON, and return a specific error message if
  it does not (this to be explicitely separated with the insufficient fee error
  message)
- make exception for some messages, specifically like `MsgBurn`, because
  `MsgBurn` is the only way to get PHOTONs, so it should accept ATONEs as fee
  token.

### Params

The `photon` module has the following params:
- `mint_disabled` (default to false): if true, disable the ability to call
  `MsgBurn`. 

> [!NOTE]
> XXX is it really usefull to disable `MsgBurn` ? Looks like it is an urgency
> method when something goes wrong, which is not very compatible with a
> governance proposal to change the parameters... Suggestion: remove it 

> [!NOTE]
> Because the maximum supply of PHOTON is constitutionnal, it must be hosted in
> a constant in the code, and not in the parameters. (XXX true or not?)

### State

Aside from its params, the `photon` module does not have any additionnal state,
as the PHOTON balances and supply are handled by the `bank` module.

### Migration

The PHOTON denom metadata has to be added to the `bank` module state (XXX while
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
in the `bank` module), because the initial supply will be 0.

## Consequences

### Validator `minimum-gas-prices`

Validators will have to update their `minimum-gas-prices` setting to reflect
this new setup. It should basically allow both ATONE and PHOTON, so the setting
should look like:

```toml
minimum-gas-prices = "0.001uatone,0.001uphoton"
```

> [!IMPORTANT]
> In the legacy `TxFeeChecker` implementation
> ([`checkTxFeeWithValidatorMinGasPrices`]), the validator `minimum-gas-prices`
> is checked against *all* mentionned denoms. For the photon module, the
> implementation must be different, it must be checked on at least one of the
> denoms (ATONE or PHOTON, but not both).

If the validator `minimum-gas-prices` does not match the required denom (ATONE
or PHOTON for `MsgBurn, only `PHOTON` for all other messages), an error is
returned and the tx is rejected.

### Positive

- Create a clear distinction between money (PHOTON) and security (ATONE). ATONE
  is not money and should only be thought of as a security token. PHOTON as the
  only fee token reinforces this property.

- Having a non-inflationnary fee token (in contrast to ATONE) ensures PHOTON
  will continue to gain value in line with AtomOne usage. The more
  transactions there are, the more PHOTON is needed, bringing scarcity and
  value.

- The requirement of burning ATONE to get PHOTON also has the side effect of
  bringing more value to ATONE, thanks to the decrease of the total supply of
  ATONE that happens in the process.

### Negative

- Users will have to choose between PHOTON and ATONE for the fee token when
  signing a transaction for AtomOne. Basically, for `MsgBurn` ATONE and PHOTON
  are accepted, for all other messages, only PHOTON is accepted. While this
  may seem obvious, it can be confusing because as far as we know, wallets do
  not have message-based logic regarding the choice of the fee token. Maybe it
  is time to start discussion with some wallets dev regarding that, this would
  improve the UX.

> [!NOTE]
> XXX [`TxFeeChecker`] allows to override the tx fee, one solution for the
> problem above is to override the tx fee denom with PHOTON and keep the
> amount. Note sure this is a good way though, sounds weird to accept a tx and
> deduct from the tx signer balance a different denom than the one specified.

### Neutral

- Dual token model like this has not been experimented at this scale in the
  Cosmos ecosytem, we might experience some unexpected side effect, positive or
  negative.

## References

* [AtomOne Constitution Article 3 Section 5]: The PHOTON Token

[AtomOne Constitution Article 3 Section 5]: https://github.com/atomone-hub/genesis/blob/b84df30364674c3f68b4bc0a43d7ed977ae22226/CONSTITUTION.md#section-5-the-photon-token
[`TxFeeChecker`]: https://github.com/cosmos/cosmos-sdk/blob/44c5d17ca6d9d37fdd6adfa3169c986fbce22b8f/x/auth/ante/fee.go#L11-L13
[`checkTxFeeWithValidatorMinGasPrices`]: https://github.com/cosmos/cosmos-sdk/blob/6e59ad0deea672a21e64fdc83939ca812dcd2b1b/x/auth/ante/validator_tx_fee.go#L17
