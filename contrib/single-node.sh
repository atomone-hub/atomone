#!/bin/sh

set -o errexit -o nounset

CHAINID=$1
GENACCT=$2

if [ -z "$1" ]; then
  echo "Need to input chain id..."
  exit 1
fi

if [ -z "$2" ]; then
  echo "Need to input genesis account address..."
  exit 1
fi

# Build genesis file incl account for passed address
coins="10000000000stake,100000000000samoleans"
atomoned init --chain-id $CHAINID $CHAINID
atomoned keys add validator --keyring-backend="test"
atomoned add-genesis-account $(atomoned keys show validator -a --keyring-backend="test") $coins
atomoned add-genesis-account $GENACCT $coins
atomoned gentx validator 5000000000stake --keyring-backend="test" --chain-id $CHAINID
atomoned collect-gentxs

# Set proper defaults and change ports
echo "Setting rpc listen address"
sed -i '' 's#"tcp://127.0.0.1:26657"#"tcp://0.0.0.0:26657"#g' ~/.atomone/config/config.toml
echo 2
sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/g' ~/.atomone/config/config.toml
sed -i '' 's/timeout_propose = "3s"/timeout_propose = "1s"/g' ~/.atomone/config/config.toml
sed -i '' 's/index_all_keys = false/index_all_keys = true/g' ~/.atomone/config/config.toml

# Start the atomone
atomoned start --pruning=nothing
