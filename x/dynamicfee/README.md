---
sidebar_position: 1
---

# `x/dynamicfee`

## Abstract

This document describes the specifications of the AtomOne implementation
of the `x/dynamicfee` module. This module is a fork of the
[`skip-mev/feemarket`](https://github.com/skip-mev/feemarket) module
(more specifically, of the [`sdk-47`](https://github.com/skip-mev/feemarket/tree/sdk-47)
branch) and includes changes and adaptations to suit the AtomOne project.

## Contents

- [`x/dynamicfee`](#xdynamicfee)
  - [Abstract](#abstract)
  - [Contents](#contents)
  - [Concepts](#concepts)
    - [Additive Increase Multiplicative Decrease (AIMD) EIP-1559](#additive-increase-multiplicative-decrease-aimd-eip-1559)
    - [Fee deduction and naive Tx prioritization](#fee-deduction-and-naive-tx-prioritization)
    - [Module state updates](#module-state-updates)
  - [State](#state)
    - [GasPrice](#gasprice)
    - [LearningRate](#learningrate)
    - [Window](#window)
    - [Index](#index)
  - [Keeper](#keeper)
  - [Messages](#messages)
    - [MsgParams](#msgparams)
  - [Events](#events)
    - [Tx](#tx)
  - [Parameters](#parameters)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Gamma](#gamma)
    - [MinBaseGasPrice](#minbasegasprice)
    - [TargetBlockUtilization](#targetblockutilization)
    - [MinLearningRate](#minlearningrate)
    - [MaxLearningRate](#maxlearningrate)
    - [Window](#window-1)
    - [FeeDenom](#feedenom)
    - [Enabled](#enabled)
  - [Client](#client)
    - [CLI](#cli)
      - [Query](#query)
        - [params](#params)
        - [state](#state-1)
        - [gas-price](#gas-price)
        - [gas-prices](#gas-prices)
  - [gRPC](#grpc)
    - [Params](#params-1)
    - [State](#state-2)
    - [GasPrice](#gasprice-1)
    - [GasPrices](#gasprices)

## Concepts

### Additive Increase Multiplicative Decrease (AIMD) EIP-1559

Please refer to [AIMD.md](AIMD.md) for a detailed description of the AIMD EIP-1559

### Fee deduction and naive Tx prioritization

Fee deduction is performed in the `anteHandler`. The entire user-set fee is
deducted from the user's account and sent to the `x/distribution` module account.
In order for a transaction to be included in a block, the transaction's gas price
must be at least equat to the current gas price. However, users can also specify
an even higher gas price than the current gas price to increase the priority of
their transaction. A naive form of transactions prioritization is implemented so
that transactions with higher gas prices are included in the block with higher priority.

### Module state updates

The `dynamicfee` module updates the gas consumed in the current block on a per-tx
basis relying on the `postHandler`. Updates to the base fee and learning rate
are instead performed in the `endBlocker`.

## State

The `x/dynamicfee` module keeps state of the following primary objects:

1. Current base-fee
2. Current learning rate
3. Moving window of block gas

In addition, the `x/dynamicfee` module keeps the following indexes to manage the
aforementioned state:

* State: `0x02 |ProtocolBuffer(State)`

### GasPrice

GasPrice is the current gas price. This is denominated in the fee per gas
unit in the base fee denom.

### LearningRate

LearningRate is the current learning rate.

### Window

Window contains a list of the last blocks' gas values. This is used
to calculate the next base fee. This stores the number of units of gas
consumed per block.

### Index

Index is the index of the current block in the block gas window.

```protobuf
// State is utilized to track the current state of the dynamic fee pricer. This
// includes the current base fee, learning rate, and block gas within the
// specified AIMD window.
message State {
  // BaseGasPrice is the current base fee. This is denominated in the fee per gas
  // unit.
  string base_gas_price = 1 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // LearningRate is the current learning rate.
  string learning_rate = 2 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // Window contains a list of the last blocks' gas values. This is used
  // to calculate the next base fee. This stores the number of units of gas
  // consumed per block.
  repeated uint64 window = 3;

  // Index is the index of the current block in the block gas window.
  uint64 index = 4;
}
```

## Keeper

The dynamicfee module provides a keeper interface for accessing the KVStore.

```go
type DynamicfeeKeeper interface {
    // Get the current state from the store.
    GetState(ctx sdk.Context) (types.State, error)

    // Set the state in the store.
    SetState(ctx sdk.Context, state types.State) error

    // Get the current params from the store.
    GetParams(ctx sdk.Context) (types.Params, error)

    // Set the params in the store.
    SetParams(ctx sdk.Context, params types.Params) error

    // Get the minimum gas price for a given denom from the store.
    GetMinGasPrice(ctx sdk.Context, denom string) (sdk.DecCoin, error) {

    // Get the current minimum gas prices from the store.
    GetMinGasPrices(ctx sdk.Context) (sdk.DecCoins, error)
}
```

## Messages

### MsgParams

The `dynamicfee` module params can be updated through `MsgParams`, which can be
done using a governance proposal. The signer will always be the `gov` module
account address.

```protobuf
message MsgParams {
  option (cosmos.msg.v1.signer) = "authority";

  // Params defines the new parameters for the dynamicfee module.
  Params params = 1 [ (gogoproto.nullable) = false ];
  // Authority defines the authority that is updating the dynamicfee module
  // parameters.
  string authority = 2 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];
}
```

The message handling can fail if:

* signer is not the gov module account address.

## Events

The dynamicfee module emits the following events:

### Tx

```json
{
  "type": "tx",
  "attributes": [
    {
      "key": "fee",
      "value": "{{sdk.Coins being payed}}",
      "index": true
    },
    {
      "key": "fee_payer",
      "value": "{{sdk.AccAddress paying the fees}}",
      "index": true
    }
  ]
}
```

## Parameters

The dynamicfee module stores its params in state with the prefix of `0x01`,
which can be updated with governance or the address with authority.

* Params: `0x01 | ProtocolBuffer(Params)`

The dynamicfee module contains the following parameters:

### Alpha

Alpha is the amount we added to the learning rate
when it is above or below the target +/- threshold.

### Beta

Beta is the amount we multiplicatively decrease the learning rate
when it is within the target +/- threshold.

### Gamma

Gamma is the threshold for the learning rate. If the learning rate is
above or below the target +/- threshold, we additively increase the
learning rate by Alpha. Otherwise, we multiplicatively decrease the
learning rate by Beta.

### MinBaseGasPrice

MinBaseGasPrice determines the initial gas price of the module and the global
minimum for the network. This is denominated in fee per gas unit in the `FeeDenom`.

### TargetBlockUtilization

TargetBlockUtilization is the target block utilization expressed as a percentage
of the block gas limit.

### MinLearningRate

MinLearningRate is the lower bound for the learning rate.

### MaxLearningRate

MaxLearningRate is the upper bound for the learning rate.

### Window

Window defines the window size for calculating an adaptive learning rate
over a moving window of blocks. The default EIP1559 implementation uses
a window of size 1.

### FeeDenom

FeeDenom is the denom that will be used for all fee payments.

### Enabled

Enabled is a boolean that determines whether the EIP1559 dynamic fee pricing
is enabled. This can be used to add the dynamicfee module and enable it
through governance at a later time.

```protobuf
// Params contains the required set of parameters for the EIP1559 dynamic fee
// pricing implementation.
message Params {
  // Alpha is the amount we additively increase the learning rate
  // when it is above or below the target +/- threshold.
  //
  // Must be > 0.
  string alpha = 1 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // Beta is the amount we multiplicatively decrease the learning rate
  // when it is within the target +/- threshold.
  //
  // Must be [0, 1].
  string beta = 2 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // Gamma is the threshold for the learning rate. If the learning rate is
  // above or below the target +/- threshold, we additively increase the
  // learning rate by Alpha. Otherwise, we multiplicatively decrease the
  // learning rate by Beta.
  //
  // Must be [0, 0.5].
  string gamma = 3 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // MinBaseGasPrice determines the initial gas price of the module and the
  // global minimum for the network.
  string min_base_gas_price = 5 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // TargetBlockUtilization is the target block utilization expressed as a
  // decimal value between 0 and 1. It is the target percentage utilization
  // of the block in relation to the consensus_params.block.max_gas parameter.
  string target_block_utilization = 6 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // MinLearningRate is the lower bound for the learning rate.
  string min_learning_rate = 7 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // MaxLearningRate is the upper bound for the learning rate.
  string max_learning_rate = 8 [
    (cosmos_proto.scalar) = "cosmos.Dec",
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];

  // Window defines the window size for calculating an adaptive learning rate
  // over a moving window of blocks.
  uint64 window = 9;

  // FeeDenom is the denom that will be used for all fee payments.
  string fee_denom = 10;

  // Enabled is a boolean that determines whether the EIP1559 dynamic fee
  // pricing is enabled.
  bool enabled = 11;
}
```

## Client

### CLI

A user can query and interact with the `dynamicfee` module using the CLI.

#### Query

The `query` commands allow users to query `dynamicfee` state.

```shell
atomoned query dynamicfee --help
```

##### params

The `params` command allows users to query the on-chain parameters.

```shell
atomoned query dynamicfee params [flags]
```

Example:

```shell
atomoned query dynamicfee params
```

Example Output:

```yml
alpha: "0.000000000000000000"
beta: "1.000000000000000000"
enabled: true
fee_denom: uatone
gamma: "0.000000000000000000"
max_learning_rate: "0.125000000000000000"
min_base_gas_price: "1.000000000000000000"
min_learning_rate: "0.125000000000000000"
target_block_utilization: "0.500000000000000000"
window: "1"
```

##### state

The `state` command allows users to query the current on-chain state.

```shell
atomoned query dynamicfee state [flags]
```

Example:

```shell
atomoned query dynamicfee state
```

Example Output:

```yml
base_fee: "1.000000000000000000"
index: "0"
learning_rate: "0.125000000000000000"
window:
  - "0"
```

##### gas-price

The `gas-price` command allows users to query the current gas-price for a given denom.

```shell
atomoned query dynamicfee gas-price [denom] [flags]
```

Example:

```shell
atomoned query dynamicfee gas-price uatone
```

Example Output:

```yml
1000000uatone
```

##### gas-prices

The `gas-prices` command allows users to query the current gas-price for all
supported denoms.

```shell
atomoned query dynamicfee gas-prices [flags]
```

Example:

```shell
atomoned query dynamicfee gas-prices
```

Example Output:

```yml
1000000stake,100000uatone
```

## gRPC

A user can query the `dynamicfee` module using gRPC endpoints.

### Params

The `Params` endpoint allows users to query the on-chain parameters.

```shell
atomone.dynamicfee.v1.Query/Params
```

Example:

```shell
grpcurl -plaintext \
    localhost:9090 \
    atomone.dynamicfee.v1.Query/Params
```

Example Output:

```json
{
  "params": {
    "alpha": "0",
    "beta": "1000000000000000000",
    "gamma": "0",
    "minBaseGasPrice": "1000000",
    "minLearningRate": "125000000000000000",
    "maxLearningRate": "125000000000000000",
    "targetBlockUtilization": "500000000000000000",
    "window": "1",
    "feeDenom": "uatone",
    "enabled": true
  }
}
```

### State

The `State` endpoint allows users to query the current on-chain state.

```shell
atomone.dynamicfee.v1.Query/State
```

Example:

```shell
grpcurl -plaintext \
    localhost:9090 \
    atomone.dynamicfee.v1.Query/State
```

Example Output:

```json
{
  "state": {
    "baseGasPrice": "1000000",
    "learningRate": "125000000000000000",
    "window": [
      "0"
    ]
  }
}
```

### GasPrice

The `GasPrice` endpoint allows users to query the current on-chain gas price for
a given denom.

```shell
atomone.dynamicfee.v1.Query/GasPrice
```

Example:

```shell
grpcurl -plaintext \
    -d '{"denom": "uatone"}' \
    localhost:9090 \
    atomone.dynamicfee.v1.Query/GasPrice/
```

Example Output:

```json
{
  "price": {
      "denom": "uatone",
      "amount": "1000000"
  }
}
```

### GasPrices

The `GasPrices` endpoint allows users to query the current on-chain gas prices
for all denoms.

```shell
atomone.dynamicfee.v1.Query/GasPrices
```

Example:

```shell
grpcurl -plaintext \
    localhost:9090 \
    atomone.dynamicfee.v1.Query/GasPrices
```

Example Output:

```json
{
  "prices": [
    {
      "denom": "uatone",
      "amount": "1000000"
    }
  ]
}
```
