#!/bin/bash

set -e

# Usage ATOMONED_BIN=$(which simd) ATOMONED_NEW_BIN=<path to new binary> ./upgrade.sh

if [[ -z $ATOMONED_BIN || -z $ATOMONED_NEW_BIN ]]; then
    echo "ATOMONED_BIN and ATOMONED_NEW_BIN must be set."
    exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
proposal_path="$script_dir/../localnet/proposal_upgrade.json"
localnet_home=~/.atomone-localnet
localnetd="$ATOMONED_BIN --home $localnet_home"
localnetd_new="$ATOMONED_NEW_BIN --home $localnet_home"

echo "Killing any running chain processes..."
pkill -f "atomoned.*start" || true
sleep 2

echo "init with previous atomoned binary"
rm -rf $localnet_home prev.log new.log
$localnetd init localnet --default-denom uatone --chain-id localnet
$localnetd config chain-id localnet
$localnetd config keyring-backend test
$localnetd keys add val
$localnetd genesis add-genesis-account val 1000000000000uatone,1000000000uphoton
$localnetd keys add user
$localnetd genesis add-genesis-account user 100000000000uatone,1000000000uphoton
$localnetd genesis gentx val 1000000000uatone
$localnetd genesis collect-gentxs
# Add treasury DAO address
$localnetd genesis add-genesis-account atone1qqqqqqqqqqqqqqqqqqqqqqqqqqqqp0dqtalx52 5388766663072uatone
# Add CP funds
$localnetd genesis add-genesis-account atone1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8flcml8 5388766663072uatone
jq '.app_state.distribution.fee_pool.community_pool = [ { "denom": "uatone", "amount": "5388766663072.000000000000000000" }]' $localnet_home/config/genesis.json > /tmp/gen
mv /tmp/gen $localnet_home/config/genesis.json
# Previous add-genesis-account call added the auth module account as a BaseAccount, we need to remove it
jq 'del(.app_state.auth.accounts[] | select(.address == "atone1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8flcml8"))' $localnet_home/config/genesis.json > /tmp/gen
mv /tmp/gen $localnet_home/config/genesis.json
# Set validator gas prices
sed -i.bak 's#^minimum-gas-prices = .*#minimum-gas-prices = "0.01uatone,0.01uphoton"#g' $localnet_home/config/app.toml
# enable REST API
sed -i -z 's/# Enable defines if the API server should be enabled.\nenable = false/enable = true/' $localnet_home/config/app.toml
# Decrease voting period to 1min
jq '.app_state.gov.params.voting_period = "60s"' $localnet_home/config/genesis.json > /tmp/gen
mv /tmp/gen $localnet_home/config/genesis.json
# Lower threshold to 0.01
jq '.app_state.gov.params.threshold = "0.010000000000000000"' $localnet_home/config/genesis.json > /tmp/gen
mv /tmp/gen $localnet_home/config/genesis.json
jq --rawfile data contrib/localnet/constitution-mock.md '.app_state.gov.constitution=$data' $localnet_home/config/genesis.json > /tmp/gen
mv /tmp/gen $localnet_home/config/genesis.json
echo "Start old chain..."
$localnetd start &> prev.log &
sleep 10
echo "Stake tokens for user to be able to vote"
stake_tx=$($localnetd tx staking delegate $($localnetd keys show val --bech val -a) 10000000000uatone --from user --fees 10000uphoton --yes --output json)
stake_txhash=$(echo "$stake_tx" | jq -r '.txhash')
echo "Stake tx hash: $stake_txhash"
sleep 6
echo "Querying stake transaction..."
stake_result=$($localnetd q tx "$stake_txhash" --output json)
stake_code=$(echo "$stake_result" | jq -r '.code')
if [ "$stake_code" != "0" ]; then
    echo "Stake transaction failed with code $stake_code"
    echo "$stake_result" | jq
    exit 1
fi
echo "Submitting Upgrade Proposal"
proposal_tx=$($localnetd tx gov submit-proposal "$proposal_path" --from user --fees 10000uphoton --yes --output json)
proposal_txhash=$(echo "$proposal_tx" | jq -r '.txhash')
echo "Proposal tx hash: $proposal_txhash"
sleep 6
echo "Querying proposal transaction..."
proposal_result=$($localnetd q tx "$proposal_txhash" --output json)
proposal_code=$(echo "$proposal_result" | jq -r '.code')
if [ "$proposal_code" != "0" ]; then
    echo "Proposal transaction failed with code $proposal_code"
    echo "$proposal_result" | jq
    exit 1
fi
deposit_tx=$($localnetd tx gov deposit 1 10000000uatone --from val --fees 10000uphoton --yes --output json)
deposit_txhash=$(echo "$deposit_tx" | jq -r '.txhash')
echo "Deposit tx hash: $deposit_txhash"
sleep 6
echo "Querying deposit transaction..."
deposit_result=$($localnetd q tx "$deposit_txhash" --output json)
deposit_code=$(echo "$deposit_result" | jq -r '.code')
if [ "$deposit_code" != "0" ]; then
    echo "Deposit transaction failed with code $deposit_code"
    echo "$deposit_result" | jq
    exit 1
fi
vote_tx=$($localnetd tx gov vote 1 yes --from user --fees 10000uphoton --yes --output json)
vote_txhash=$(echo "$vote_tx" | jq -r '.txhash')
echo "Vote tx hash: $vote_txhash"
sleep 6
echo "Querying vote transaction..."
vote_result=$($localnetd q tx "$vote_txhash" --output json)
vote_code=$(echo "$vote_result" | jq -r '.code')
if [ "$vote_code" != "0" ]; then
    echo "Vote transaction failed with code $vote_code"
    echo "$vote_result" | jq
    exit 1
fi
# Query proposal to get upgrade height
upgrade_height=$($localnetd q gov proposal 1 --output json | jq -r '.messages[0].plan.height')
echo "Proposal should pass! Chain will halt at block height: $upgrade_height"
echo "Waiting for chain to reach upgrade height..."

# Poll chain height until it reaches upgrade height
while true; do
    current_height=$($localnetd status 2>&1 | jq -r '.SyncInfo.latest_block_height' 2>/dev/null || echo "0")
    echo "Current height: $current_height / Upgrade height: $upgrade_height"

    if [ "$current_height" -ge "$upgrade_height" ]; then
        echo "Upgrade height reached!"
        break
    fi

    sleep 2
done

sleep 5

echo "Displaying last 6 lines of previous binary logs:"
tail -n 6 prev.log

echo "Killing old binary..."
pkill -f "$ATOMONED_BIN" || true
sleep 2

echo "Starting new binary..."
$localnetd_new start
