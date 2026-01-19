#!/bin/bash

# Usage ATOMONED_BIN=$(which simd) ATOMONED_NEW_BIN=<path to new binary> ./upgrade.sh

if [[ -z $ATOMONED_BIN || -z $ATOMONED_NEW_BIN ]]; then
    echo "ATOMONED_BIN and ATOMONED_NEW_BIN must be set."
    exit 1
fi

localnet_home=~/.atomone-localnet

echo "init with previous atomoned binary"
$ATOMONED_BIN init localnet --default-denom uatone --chain-id localnet
$ATOMONED_BIN config set client chain-id localnet
$ATOMONED_BIN config set client keyring-backend test
$ATOMONED_BIN keys add val
$ATOMONED_BIN genesis add-genesis-account val 1000000000000uatone,1000000000uphoton
$ATOMONED_BIN keys add user
$ATOMONED_BIN genesis add-genesis-account user 1000000000uatone,1000000000uphoton
$ATOMONED_BIN genesis gentx val 1000000000uatone
$ATOMONED_BIN genesis collect-gentxs
# Add treasury DAO address
$ATOMONED_BIN genesis add-genesis-account atone1qqqqqqqqqqqqqqqqqqqqqqqqqqqqp0dqtalx52 5388766663072uatone
# Add CP funds
$ATOMONED_BIN genesis add-genesis-account atone1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8flcml8 5388766663072uatone
jq '.app_state.distribution.fee_pool.community_pool = [ { "denom": "uatone", "amount": "5388766663072.000000000000000000" }]' $(localnet_home)/config/genesis.json > /tmp/gen
mv /tmp/gen $(localnet_home)/config/genesis.json
# Previous add-genesis-account call added the auth module account as a BaseAccount, we need to remove it
jq 'del(.app_state.auth.accounts[] | select(.address == "atone1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8flcml8"))' $(localnet_home)/config/genesis.json > /tmp/gen
mv /tmp/gen $(localnet_home)/config/genesis.json
# Set validator gas prices
sed -i.bak 's#^minimum-gas-prices = .*#minimum-gas-prices = "0.01uatone,0.01uphoton"#g' $(localnet_home)/config/app.toml
# enable REST API
$ATOMONED_BIN config set app api.enable true
# Decrease voting period to 1min
jq '.app_state.gov.params.voting_period = "60s"' $(localnet_home)/config/genesis.json > /tmp/gen
mv /tmp/gen $(localnet_home)/config/genesis.json
jq --rawfile data contrib/localnet/constitution-mock.md '.app_state.gov.constitution=$$data' $(localnet_home)/config/genesis.json > /tmp/gen
mv /tmp/gen $(localnet_home)/config/genesis.json
echo "start chain"
$ATOMONED_BIN start &
sleep 10
echo "submit upgrade tx"
$ATOMONED_BIN tx gov submit-proposal ../localnet/proposal_upgrade.json --from user --fees 10000uphoton --yes
sleep 6
$ATOMONED_BIN tx gov deposit 1 10000000uatone --from user --fees 10000uphoton --yes
sleep 6
$ATOMONED_BIN tx gov vote 1 yes --from user --yes
echo "wait for chain halt and restart new binary"
# $ATOMONED_NEW_BIN start --home $(localnet_home) &
# $ATOMONED_NEW_BIN q tx gov params
