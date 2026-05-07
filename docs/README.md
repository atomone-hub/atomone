<!--
parent:
  order: false
layout: home
-->

# AtomOne Documentation

Welcome to the documentation of the **AtomOne application: `atomone`**.

## Generate a key from manual entropy generation

If you don't want to rely on computer-generated randomness, you can provide
your own entropy to generate a key. The following method ensures your private
key's randomness comes from physical sources rather than computer algorithms.

First, generate the mnenonic:

```sh
$ atomoned keys mnemonic --unsafe-entropy
> > WARNING: Generate at least 256-bits of entropy and enter the results here:
```

Use one of the following methods to generate entropy:

- **Dice**: Roll a D20 (20-sided die) exactly 42 times.
Example: `18 7 3 12 5 19 8 2 14 11 20 1 9 15 4 13 6 17 10 16 3 8 12 19 2 7 14 5 11 18 1 20 9 4 15 13 17 6 10 16 3 11`

- **Cards**: Shuffle a standard 52-card deck 20 times, then record the full
deck order.
Example: `AS 2H 7C KD 3S 9H QC 4D JH 10S 5C 8H AC 2D 7S KH 3C 9D QS 4H JS 10C 5D 8S AH

Write the output mnenomic in a safe place, then run the following command to
generate the key:

```sh
$ atomoned keys add <NAME> --recover
> Enter your bip39 mnemonic
```

Copy/paste the mnemonic and you're done.

## Testing Chain Upgrade

Chain upgrade is an important procedure that should be tested carefully. This
section aims to provide a guide for testing chain upgrades in AtomOne using a
localnet. 

1. Git checkout the version of AtomOne you want to upgrade from.
2. Update `contrib/localnet/upgrade_proposal.json` with the correct plan name,
   which means the exact `UpgradeName` used to qualify the upgrade in the
   next version. For instance for the v2 upgrade, the plan name is `v2` (see
   the `app/upgrade` folder).
3. Run `make localnet-start` to start a new localnet.
4. Run `make localnet-submit-upgrade-proposal` to submit the upgrade proposal
   and give it enough yes votes for passing the tally.
5. Wait for 5 minutes (the voting period) and run `atomoned --home ~/.atomone-localnet q gov proposals`
   to check that the proposal has passed.
6. Wait for the block height that was registered in the upgrade proposal. Once
   reached the localnet should stop producing blocks, and return an error like:
   ```
   ERR UPGRADE "v2" NEEDED (...)
   ERR CONSENSUS FAILURE!!!
   ```
   This means it is time to upgrade the binary.
7. Stop the `make localnet-start`
8. Git checkout the version of AtomOne you want to upgrade to.
9. Run `make localnet-restart` (/!\ not `localnet-start` which would delete all
   the chain data). Block production should restart.
10. Check that the upgrade procedure has been executed properly.
11. Restart the node to ensure it continues producing blocks after the upgrade.

## In-place Testnet from Mainnet State

The `in-place-testnet` command takes an existing chain's data directory
(typically a mainnet snapshot) and converts it into a single-validator
testnet running locally. This is useful to:

- Dry-run an upgrade handler against real mainnet state.
- Reproduce a mainnet bug locally with full balances and history.
- Test governance flows that require concentrated voting power (your sole
  validator can pass any active proposal in seconds).

### Setup

1. Initialize a fresh node home:
   ```sh
   atomoned init localnet --home ~/.atomone/validator1 --chain-id atomone-1
   ```

2. Replace the generated genesis with the `atomone-1` mainnet genesis:
   ```sh
   wget -O ~/.atomone/validator1/config/genesis.json \
     https://snapshots.polkachu.com/genesis/atomone/genesis.json
   ```

3. Set `minimum-gas-prices` in `~/.atomone/validator1/config/app.toml` (e.g.
   `0uatone`) and wire `seeds` / `persistent_peers` in `config.toml` from the
   [`atomone-hub/networks`](https://github.com/atomone-hub/networks/tree/main/atomone-1)
   repo.

4. Download a recent mainnet snapshot (e.g. from
   [polkachu](https://polkachu.com/tendermint_snapshots/atomone) or itrocket)
   and extract it. `lz4` is required for decompression.
   ```sh
   rm -rf ~/.atomone/validator1/data && mkdir ~/.atomone/validator1/data
   echo '{ "height": "0", "round": 0, "step": 0 }' \
     > ~/.atomone/validator1/data/priv_validator_state.json
   lz4 -d <snapshot>.tar.lz4 -c | tar -x -C ~/.atomone/validator1
   ```

5. Start the node so it loads the snapshot, then stop it cleanly with
   `Ctrl-C` once `atomoned status` shows the snapshot height. The node does
   not need to be caught up to tip; `in-place-testnet` only needs the latest
   committed block in the blockstore.
   ```sh
   atomoned start --home ~/.atomone/validator1 --x-crisis-skip-assert-invariants
   ```

   The binary version must be compatible with the snapshot height (e.g.
   `v3.3.0` for a current mainnet snapshot). To dry-run a future upgrade,
   load the snapshot with the pre-upgrade binary first, then run
   `in-place-testnet` with the post-upgrade binary and
   `--trigger-testnet-upgrade=<plan-name>`.

### Convert to Testnet

1. Add a local key whose valoper address will become the new sole validator:
   ```sh
   atomoned keys add testnet-val --home ~/.atomone/validator1 --keyring-backend test
   VAL_OPER=$(atomoned keys show testnet-val --bech val -a \
     --home ~/.atomone/validator1 --keyring-backend test)
   ACCOUNT=$(atomoned keys show testnet-val -a \
     --home ~/.atomone/validator1 --keyring-backend test)
   ```

2. Run `in-place-testnet`:
   ```sh
   atomoned in-place-testnet testing-1 "$VAL_OPER" \
     --home ~/.atomone/validator1 \
     --accounts-to-fund="$ACCOUNT"
   ```
   Confirm with `y` when prompted. The node will restart with chain-id
   `testing-1`, your local key as the only validator, and the funded
   accounts holding 1000 ATONE each.

3. Verify:
   ```sh
   atomoned status --home ~/.atomone/validator1 \
     | jq '.NodeInfo.network, .ValidatorInfo'
   atomoned q staking validators --home ~/.atomone/validator1 -o json \
     | jq '.validators | length'
   atomoned q bank balances "$ACCOUNT" --home ~/.atomone/validator1
   ```
   Expect `network: "testing-1"`, exactly one validator, and
   `1000000000 uatone` on the funded account.
