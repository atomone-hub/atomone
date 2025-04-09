---
sidebar_position: 1
---

# `x/photon`

## Abstract

This module manages the PHOTON token (base denom `uphoton`), introduced by
[ADR 002](../../docs/architecture/adr-002-photon-token.md) as the only fee
token in AtomOne. PHOTONs are minted by burning ATONEs at a rate capped by a
maximum PHOTON supply of 1 B. Minting is achieved via a single message
(`MsgMintPhoton`), and the module enforces PHOTON fees with an AnteDecorator.

## Contents

- [`x/photon`](#xphoton)
  - [Abstract](#abstract)
  - [Contents](#contents)
  - [Concepts](#concepts)
    - [ATONE to PHOTON conversion](#atone-to-photon-conversion)
    - [Fee enforcement](#fee-enforcement)
  - [State](#state)
  - [Messages](#messages)
    - [MsgMintPhoton](#msgmintphoton)
  - [Parameters](#parameters)
  - [Client](#client)
    - [gRPC](#grpc)
    - [REST](#rest)
  - [References](#references)

## Concepts

### ATONE to PHOTON conversion

PHOTON is minted by burning ATONE at a conversion rate:

```
photon_minted = atone_burned * (photon_max_supply - photon_supply) / atone_supply
```

This ensures PHOTON’s supply never exceeds 1 B tokens. The module checks for
non-zero PHOTON to mint before completing the transaction. This is because
rounding errors can cause the calculated amount to be zero when burning small
fractions of ATONE’s supply.

### Fee enforcement

An AnteDecorator ensures PHOTON (`uphoton`) is the only fee token for most
transactions. A small set of messages (initially just `MsgMintPhoton`) can
be set as exceptions and accept other fees such ATONE, as defined by the 
`txfee_exceptions` parameter.

## State

`x/photon` stores no extra balance data, and relies on `x/bank`.
The only tracked module data are parameters, such as whether minting is enabled.

## Messages

### MsgMintPhoton

Burns a specified ATONE amount in exchange for newly minted PHOTON. The minted
tokens go to the caller’s account. If `mint_disabled` is `true`, this message fails.

## Parameters

| Key              | Type       | Default               |
|------------------|-----------|-----------------------|
| mint_disabled    | bool       | false                 |
| txfee_exceptions | []string   | ["MsgMintPhoton"]     |

## Client

### gRPC

- Query/ConversionRate: Returns the current conversion rate.  
- Query/Params: Returns `mint_disabled` and `txfee_exceptions`.

### REST

Endpoints mirror the gRPC queries, allowing retrieval of conversion rate and parameters.

- `/atomone/photon/v1/conversion_rate`: Returns the current conversion rate.
- `/atomone/photon/v1/params`: Returns `mint_disabled` and `txfee_exceptions`.

## References

See [ADR 002](../../docs/architecture/adr-002-photon-token.md) and
[AtomOne Constitution Article 3 Section 5](https://github.com/atomone-hub/genesis/blob/b84df30364674c3f68b4bc0a43d7ed977ae22226/CONSTITUTION.md#section-5-the-photon-token)
for more details.
