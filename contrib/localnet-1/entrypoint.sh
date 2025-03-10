#!/bin/bash

CHAIN_ID="localnet-1"

WALLET_VALIDATOR_ADDR="atone16zau0ntu4xqm58c537e5zwrnkq4s5zpdpxmwz7"
WALLET_VALIDATOR_MNEMONIC="safe permit shrug crumble snap tree law observe melt ramp exit lazy lend few elephant idle now debris soldier split course movie pepper afraid"

WALLET_1_ADDR="atone19j38cmss7m3h4d0fptfwgltfsnhhw583y7h2ay"
WALLET_1_MNEMONIC="enjoy insect mimic number swamp trade obvious swamp stick give stool clarify bulk recipe minimum happy chunk coast suffer dutch high race trumpet sock"

WALLET_2_ADDR="atone1083ytsqvwml9vcytzn79x0dxv027apfthrtsda"
WALLET_2_MNEMONIC="slam tree stamp write hole race unusual erosion express toy pulp domain scan power hawk ball combine that olympic solve loud talk saddle total"

WALLET_3_ADDR="atone1dpspzdsd7nsch65n5dgalqd9ydcuc5nxs9s2x7"
WALLET_3_MNEMONIC="escape daring note begin cattle smooth you forget business vault mirror charge pilot few message corn equip asset nose snake scale sudden practice rib"

atomoned init localnet --chain-id "${CHAIN_ID}" --default-denom="uatone"

atomoned config keyring-backend test

echo "${WALLET_VALIDATOR_MNEMONIC}" | atomoned keys add validator --recover
echo "${WALLET_1_MNEMONIC}" | atomoned keys add wallet-1 --recover
echo "${WALLET_2_MNEMONIC}" | atomoned keys add wallet-2 --recover
echo "${WALLET_3_MNEMONIC}" | atomoned keys add wallet-3 --recover

# Add 100_000_000 $ATONE to all wallets
atomoned genesis add-genesis-account $WALLET_VALIDATOR_ADDR 100000000000uatone

atomoned genesis add-genesis-account $WALLET_1_ADDR 100000000000uatone
atomoned genesis add-genesis-account $WALLET_2_ADDR 100000000000uatone
atomoned genesis add-genesis-account $WALLET_3_ADDR 100000000000uatone

# gentx 10 $ATONE
atomoned genesis gentx validator 10000000uatone --chain-id ${CHAIN_ID}
atomoned genesis collect-gentxs
atomoned genesis validate-genesis

# RPC
sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' $HOME/.atomone/config/config.toml

# REST
sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|g' $HOME/.atomone/config/app.toml
sed -i 's|enable = false|enable = true|g' $HOME/.atomone/config/app.toml

# GRPC
sed -i 's|address = "localhost:9090"|address = "0.0.0.0:9090"|g' $HOME/.atomone/config/app.toml
sed -i 's|address = "localhost:9091"|address = "0.0.0.0:9091"|g' $HOME/.atomone/config/app.toml

exec atomoned start --minimum-gas-prices=0.025uatone
