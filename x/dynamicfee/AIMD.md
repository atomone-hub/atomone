# Additive Increase Multiplicative Decrease (AIMD) EIP-1559

## Overview

> **Definitions:**
>
> * **`Target Block Gas`**: This is the target block gas consumption.
> * **`Max Block Gas`**: This is the maximum block gas consumption, fetched
>   from the `x/consensus` module.

This plugin implements the AIMD (Additive Increase Multiplicative Decrease)
EIP-1559 dynamic fee pricing as described in this
[AIMD EIP-1559](https://arxiv.org/abs/2110.04753) research publication.

The AIMD EIP-1559 dynamic fee pricing is a slight modification to Ethereum's
EIP-1559 dynamic fee pricing. Specifically it introduces the notion of an
adaptive learning rate which scales the base gas price more aggressively when
the network is congested and less aggressively when the network is not
congested. This is primarily done to address the often cited criticism of
EIP-1559 that it's base fee often lags behind the current demand for block
space. The learning rate on Ethereum is effectively hard-coded to be 12.5%,
which means that between any two blocks the base fee can maximally increase by
12.5% or decrease by 12.5%. Additionally, AIMD EIP-1559 differs from Ethereum's
EIP-1559 by considering a configured time window (number of blocks) to consider
when calculating and comparing target block gas and current block gas.

## Parameters

### Ethereum EIP-1559

Base EIP-1559 currently utilizes the following parameters to compute the base fee:

* **`PreviousBaseGasPrice`**: This is the base gas price from the previous
  block. This must be a value that is greater than `0`.
* **`TargetBlockSize`**: This is the target block size in bytes. This must be a
  value that is greater than `0`.
* **`PreviousBlockSize`**: This is the block size from the previous block.

The calculation for the updated base fee for the next block is as follows:

```golang
currentBaseGasPrice := previousBaseGasPrice * (1 + 0.125 * (currentBlockSize - targetBlockSize) / targetBlockSize)
```

### AIMD EIP-1559

AIMD EIP-1559 introduces a few new parameters to the EIP-1559 dynamic fee pricing:

* **`Alpha`**: This is the amount we additively increase the learning rate when
  the target gas is less than the current gas i.e. the block was
  more full than the target gas. This must be a value that is greater than `0.0`.
* **`Beta`**: This is the amount we multiplicatively decrease the learning rate
  when the target gas is greater than the current gas i.e. the
  block was less full than the target gas. This must be a value that is greater
  than `0.0`.
* **`Window`**: This is the number of blocks we look back to compute the current
  gas. This must be a value that is greater than `0`. Instead of only
  utilizing the previous block's gas, we now consider the gas of
  the previous `Window` blocks.
* **`Gamma`**: This determines whether you are additively increase or
  multiplicatively decreasing the learning rate based on the target and current
  block gas. This must be a value that is between `[0, 1]`. For example,
  if `Gamma = 0.25`, then we multiplicatively decrease the learning rate if the
  average ratio of current block gas to max block gas over some window of
  blocks is within `(0.25, 0.75)` and additively increase it if outside that range.
* **`MaxLearningRate`**: This is the maximum learning rate that can be applied
  to the base fee. This must be a value that is between `[0, 1]`.
* **`MinLearningRate`**: This is the minimum learning rate that can be applied
  to the base fee. This must be a value that is between `[0, 1]`.

The calculation for the updated base fee for the next block is as follows:

```golang

// sumBlockGasInWindow returns the sum of the block gas in the window.
blockConsumption := sumBlockGasInWindow(window) / (window * maxBlockGas)

if blockConsumption <= gamma || blockConsumption >= 1 - gamma {
    // MAX_LEARNING_RATE is a constant that is configured by the chain developer
    newLearningRate := min(MaxLearningRate, alpha + currentLearningRate)
} else {
    // MIN_LEARNING_RATE is a constant that is configured by the chain developer
    newLearningRate := max(MinLearningRate, beta * currentLearningRate)
}

newBaseGasPrice := currentBaseGasPrice * (1 + newLearningRate * (currentBlockGas - targetBlockGas) / targetBlockGas) 
```

The expected behavior is the following: when the current block gas is close to
the `targetBlockGas` (in other words, when `blockConsumption` is in the
`gamma` range), then the base gas price is close to the right value, so the
algorithm reduces the learning rate to reduce the size of the oscillations. By
contrast, if the current block gas is too small or too high
(`blockConsumption` is out of `gamma` range), then the base fee is apparently
far away from its equilibrium value, and the algorithm increases the learning
rate.

## Examples

> **Assume the following parameters:**
>
> * `TargetBlockGas = 50`
> * `MaxBlockGas = 100`
> * `Window = 1`
> * `Alpha = 0.025`
> * `Beta = 0.95`
> * `Gamma = 0.25`
> * `MAX_LEARNING_RATE = 1.0`
> * `MIN_LEARNING_RATE = 0.0125`
> * `Current Learning Rate = 0.125`
> * `Previous Base Fee = 10.0`

### Block is Completely Empty

In this example, we expect the learning rate to additively increase and the base
fee to decrease.

```golang
blockConsumption := sumBlockGasInWindow(1) / (1 * 100) == 0
newLearningRate := min(1.0, 0.025 + 0.125) == 0.15
newBaseGasPrice := 10 * (1 + 0.15 * (0 - 50) / 50) == 8.5
```

As we can see, the base fee decreased by 1.5 and the learning rate increases.

### Block is Completely Full

In this example, we expect the learning rate to additively increase and the base
fee to increase.

```golang
blockConsumption := sumBlockGasInWindow(1) / (1 * 100) == 1
newLearningRate := min(1.0, 0.025 + 0.125) == 0.15
newBaseGasPrice := 10 * (1 + 0.15 * ((100 - 50) / 50)) == 11.5
```

As we can see, the base fee increased by 1.5 and the learning rate increases.

### Block is at Target Gas

In this example, we expect the learning rate to multiplicatively decrease and the
base fee to remain the same.

```golang
blockConsumption := sumBlockGasInWindow(1) / (1 * 100) == 0.5
newLearningRate := max(0.0125, 0.95 * 0.125) == 0.11875
newBaseGasPrice := 10 * (1 + 0.11875 * (50 - 50) / 50) == 10
```

As we can see, the base fee remained the same and the learning rate decreased.

## Default EIP-1559 With AIMD EIP-1559

It is entirely possible to implement the default EIP-1559 dynamic fee pricing
with the AIMD EIP-1559 dynamic fee pricing. This can be done by setting the
following parameters:

* `Alpha = 0.0`
* `Beta = 1.0`
* `Gamma = 1.0`
* `MAX_LEARNING_RATE = 0.125`
* `MIN_LEARNING_RATE = 0.125`
* `Window = 1`
