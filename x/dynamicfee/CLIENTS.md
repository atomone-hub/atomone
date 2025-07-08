# Gas Price Queries for Integrations

Because `x/dynamicfee` uses a dynamic fee, end-users will need to query the module
for the current `gasPrice` to use when building a transaction.

A summary for the flow to query this information is as follows:

* Create an RPC connection with the chain
* Create a `dynamicfee` client
* Submit the `GasPrice` query
* Use the `gasPrice` to populate the Tx fee field.

Extensive querying information can be seen in the module [spec](README.md#query).

The specific query for `GasPrices` can be found [here](README.md#gas-prices).

## Code Snippet

Wallet, relayers, and other users will want to add programmatic ways to query
this before building their transactions.  Below is an example of how a user could
implement this lightweight query in Go:

### Create A gRPC Connection

First, a base connection to the chain you are querying must be created.

A chain gRPC (below) or CometBFT ABCI RPC connection can be created:

```go
   // Set up gRPC connection to chain
   cc, err := grpc.NewClient(endpoint, insecure.NewCredentials())
   if err != nil {
      panic(err)
   }
   defer cc.Close()
```

### Create a Dynamicfee Query Client

An `x/dynamicfee` query client can then be created using the created gRPC connection.

This client exposes all [queries](README.md#query) that the `x/dynamicfee` module
exposes.

```go
   // Create DynamicfeeClient with underlying gRPC connection
   dynamicfeeClient := dynamicfeetypes.NewQueryClient(cc)
```

### Query Gas Prices

The `gas price` (as an `sdk.DecCoin`) can be queried using the `GasPrice` query.
This query requires the desired coin denomination for the fee to be paid with.

The query will return an error if the given denomination is not supported.

```go
   gasPrice, err := dynamicfeeClient.GasPrice(ctx, &dynamicfeetypes.GasPriceRequest{
      Denom: denom,
   })
   if err != nil {
      panic(err)
   }
```

### Using `gasPrice` to construct a transaction

There are two ways to construct a transaction with `gasPrice`:

1. Provide the minimum fee: `feeAmount = gasPrice * gasLimit` (`gasLimit` gives
   the maximum amount of gas a transaction can consume. You can obtain
   appropriate `gasLimit` by simulating a transaction to see how much gas
   it consumes under normal conditions).
2. Provide a higher fee with respect to the minimum fee:
   `feeAmount=gasPrice * gasLimit + tip`; this will result in your transaction
   being placed ahead of others with lower tips (or being included in the block
   instead of others when the block is full)

## Examples of Other EIP-1559 Integrations

The original [`skip-mev/feemarket`](https://github.com/skip-mev/feemarket) from
which this module is based on provides a similar implementation.

Also, the [Osmosis](https://github.com/osmosis-labs/osmosis) Blockchain has a similar
EIP-1559 mechanism that has been integrated by wallets and relayers.  Below are
some examples as to how different projects query the dynamic fee for transactions:

* [Keplr Wallet EIP-1559 BaseFee Query](https://github.com/chainapsis/keplr-wallet/blob/b0a96c2c713d8163ce840fcd5abbac4eb612607c/packages/stores/src/query/osmosis/base-fee/index.ts#L18)
* [Cosmos-Relayer EIP-1559 BaseFee Query](https://github.com/cosmos/relayer/blob/9b140b664fe6b10161af1093ccd26627b942742e/relayer/chains/cosmos/fee_market.go#L13)
* [Hermes Relayer EIP-1559 Fee Query](https://github.com/informalsystems/hermes/blob/fc8376ba98e4b595e446b366b736a0c046d6026a/crates/relayer/src/chain/cosmos/eip_base_fee.rs#L15)
  * Note: Hermes also already implements a query `x/feemarket` seen [here](https://github.com/informalsystems/hermes/blob/fc8376ba98e4b595e446b366b736a0c046d6026a/crates/relayer/src/chain/cosmos/eip_base_fee.rs#L33)
