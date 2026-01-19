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

echo "init with previous atomoned binary"
rm -rf $localnet_home
$localnetd init localnet --default-denom uatone --chain-id localnet
$localnetd config chain-id localnet
$localnetd config keyring-backend test
$localnetd keys add val
$localnetd genesis add-genesis-account val 1000000000000uatone,1000000000uphoton
$localnetd keys add user
$localnetd genesis add-genesis-account user 1000000000uatone,1000000000uphoton
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
jq --rawfile data contrib/localnet/constitution-mock.md '.app_state.gov.constitution=$data' $localnet_home/config/genesis.json > /tmp/gen
mv /tmp/gen $localnet_home/config/genesis.json
echo "start chain"
$localnetd start &
sleep 10
echo "submit upgrade tx"
$localnetd tx gov submit-proposal "$proposal_path" --from user --fees 10000uphoton --yes
sleep 6
$localnetd tx gov deposit 1 10000000uatone --from user --fees 10000uphoton --yes
sleep 6
$localnetd tx gov vote 1 yes --from val --yes
echo "wait for chain halt and restart new binary"
# $localnetd_new start &
# $localnetd_new q tx gov params
