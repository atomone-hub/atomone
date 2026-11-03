#!/usr/bin/env bash
# End-to-end test of the spectre Ethereum light client. Builds atomoned and the
# spectre relayer, starts a Kurtosis Ethereum L1, deploys the IBC contracts,
# starts an AtomOne devnet, creates both clients, relays one ICS-20 transfer in
# each direction, then redeems both tokens back to their source chain (voucher
# burn + escrow release) and asserts exact balances and acknowledgements at
# every step.
#
# Requirements: docker, kurtosis, foundry (forge, cast), bun, jq, shasum, nc,
# curl, go, and network access to the Groth16 prover (PROVER_ADDR).
#
# Environment overrides (precedence: environment > tests/e2e/spectre/.env >
# defaults; see .env.example). PROVER_ADDR has no default and must be provided:
# it is the host:port of the Groth16 prover, without scheme.
#   SPECTRE_REPO  git URL of the spectre checkout   (decentrio/spectre-pubblic)
#   SPECTRE_REF   pinned ref tested against         (fc576a4)
#   PROVER_ADDR   prover host:port, no http://      (required)
#   ATOMONE_DIR   atomone checkout                  (sibling of this script's repo)
#   ATOMONED      atomoned binary                   ($ATOMONE_DIR/build/atomoned)
#   SPECTRE_DIR   spectre checkout                  ($ATOMONE_DIR/../spectre)
#   WORK          logs and run environment          (/tmp/atomone-devnet)
set -Eeuo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

load_dotenv() { # environment variables already set take precedence over .env
  local env_file="$SCRIPT_DIR/.env" line k v
  [ -f "$env_file" ] || return 0
  while IFS= read -r line || [ -n "$line" ]; do
    line=${line%$'\r'}
    case "$line" in ''|'#'*) continue ;; esac
    k=${line%%=*}; v=${line#*=}
    case "$k" in ''|[0-9]*|*[!A-Za-z0-9_]*) continue ;; esac
    [ -n "${!k:-}" ] || export "$k=$v"
  done < "$env_file"
}
load_dotenv

SPECTRE_REPO=${SPECTRE_REPO:-https://github.com/decentrio/spectre-pubblic}
SPECTRE_REF=${SPECTRE_REF:-fc576a4}
PROVER_ADDR=${PROVER_ADDR:-}
ATOMONE_DIR=${ATOMONE_DIR:-$(cd "$SCRIPT_DIR/../../.." && pwd)}
ATOMONED=${ATOMONED:-$ATOMONE_DIR/build/atomoned}
export ATOMONED
SPECTRE_DIR=${SPECTRE_DIR:-$ATOMONE_DIR/../spectre}
WORK=${WORK:-/tmp/atomone-devnet}

# Fixed devnet values; must match spectre's scripts (run_atomone_node.sh,
# relayer/config.ethereum.example.json).
ATOMONE_CHAIN_ID=atomone-devnet-1
ATOMONE_NODE=tcp://127.0.0.1:26657
ATOMONE_HOME=$HOME/.atomone-devnet
ICS26_CLIENT_ID=atomone-1     # AtomOne client ID on the ETH router
ETH_CLIENT_ID=spectre-0       # ETH client ID on AtomOne
ENCLAVE=my-testnet            # Kurtosis enclave
TRANSFER_PORT=transfer
AMOUNT=1000000                # transferred in each direction, base units
DEPLOYER=0x8943545177806ED17B9F23F0a21ee5948eCaa776
TEST3_UATONE_AFTER=1999999000000  # after 1,000,000 uatone is escrowed
TEST3_UATONE_FINAL=2000000000000  # after the escrow is released back

ERC20=
ICS20=
ERC20_BAL_BEFORE=            # deployer balance before the first transfer (1e24)
WRAPPED=                     # wrapped uatone (IBCERC20) contract on ETH
TEST3_ADDR=
QUERY_FILE=

log() { printf '\n[spectre-e2e] %s\n' "$*"; }
die() { printf '\n[spectre-e2e] FAIL: %s\n' "$*" >&2; exit 1; }

poll() { # poll <timeout_s> <interval_s> <description> <check cmd...>
  local timeout=$1 interval=$2 desc=$3 waited=0
  shift 3
  while true; do
    if "$@"; then
      log "OK after ${waited}s: $desc"
      return 0
    fi
    if [ "$waited" -ge "$timeout" ]; then
      die "timed out after ${waited}s waiting for: $desc"
    fi
    printf 'waiting (%ss/%ss): %s\n' "$waited" "$timeout" "$desc"
    sleep "$interval"
    waited=$((waited + interval))
  done
}

teardown() {
  local rc=$?
  [ -n "${WORK:-}" ] || return 0
  log "teardown"
  if [ -f "$WORK/relayer.pid" ]; then
    kill -TERM "$(cat "$WORK/relayer.pid")" 2>/dev/null || true
  fi
  if [ -f "$WORK/atomone-script.pid" ]; then
    kill -TERM "$(cat "$WORK/atomone-script.pid")" 2>/dev/null || true
  fi
  pkill -TERM -f 'atomoned start' 2>/dev/null || true
  kurtosis enclave rm -f "$ENCLAVE" >/dev/null 2>&1 || true
  # Keep the final relayer config with the run logs; redact the devnet keys.
  if [ -d "$SPECTRE_DIR/relayer" ]; then
    cp -f "$SPECTRE_DIR/relayer/config.json" "$WORK/config.json.final" 2>/dev/null || true
  fi
  sed -i 's/COSMOS_PRIVATE_KEY=.*/COSMOS_PRIVATE_KEY=<redacted>/' "$WORK/atomone-relayer.env" 2>/dev/null || true
  if [ "$rc" -ne 0 ]; then
    log "run failed (exit $rc); log tails:"
    for f in relayer.log atomone-node.log deploy-eth.log eth-node.log; do
      if [ -f "$WORK/$f" ]; then
        printf '\n===== %s (last 50 lines) =====\n' "$f"
        tail -n 50 "$WORK/$f" || true
      fi
    done
  fi
}
trap teardown EXIT

check_prerequisites() {
  log "checking prerequisites"
  [ -n "$PROVER_ADDR" ] || die "PROVER_ADDR is not set: set it in the environment or in $SCRIPT_DIR/.env (see .env.example), host:port without scheme"
  case "$PROVER_ADDR" in
    http://*|https://*) die "PROVER_ADDR must be host:port without scheme, got '$PROVER_ADDR'" ;;
  esac
  local tool
  for tool in git go docker kurtosis forge cast jq bun shasum nc curl timeout; do
    command -v "$tool" >/dev/null || die "missing tool: $tool"
  done
  docker info >/dev/null 2>&1 || die "docker daemon is not running"
  mkdir -p "$WORK"
  df -h / || true
}

clone_spectre() {
  log "cloning spectre at $SPECTRE_REF"
  if [ ! -d "$SPECTRE_DIR/.git" ]; then
    git clone "$SPECTRE_REPO" "$SPECTRE_DIR"
  fi
  git -C "$SPECTRE_DIR" fetch origin --quiet
  git -C "$SPECTRE_DIR" checkout --quiet --force "$SPECTRE_REF"
  git -C "$SPECTRE_DIR" rev-parse HEAD | tee "$WORK/spectre-commit.txt"
}

build_binaries() {
  log "building atomoned and relayer"
  if [ ! -x "$ATOMONED" ]; then
    make -C "$ATOMONE_DIR" build
  fi
  "$ATOMONED" version
  (cd "$SPECTRE_DIR/relayer" && go build -o relayer ./cmd)
  (cd "$SPECTRE_DIR" && bun install)
}

check_eth_finality() {
  local e
  e=$(curl -s "$ETH_BEACON_API/eth/v1/beacon/states/head/finality_checkpoints" \
    | jq -r '.data.finalized.epoch' 2>/dev/null) || return 1
  [ -n "$e" ] && [ "$e" != "null" ] && [ "$e" -gt 0 ] 2>/dev/null
}

start_eth_l1() {
  log "starting Ethereum L1 with Kurtosis"
  kurtosis enclave rm -f "$ENCLAVE" >/dev/null 2>&1 || true
  (
    cd "$SPECTRE_DIR"
    SKIP_COSMOS_RESET=1 timeout 900 ./scripts/local/run_eth_node.sh
  ) > "$WORK/eth-node.log" 2>&1 || die "run_eth_node.sh failed, see $WORK/eth-node.log"
  set -a
  . "$SPECTRE_DIR/.eth-devnet-run/eth.env" # exports ETH_RPC, ETH_WS, ETH_BEACON_API
  set +a
  log "L1 endpoints: rpc=$ETH_RPC beacon=$ETH_BEACON_API"
  poll 900 10 "finalized.epoch > 0" check_eth_finality
  df -h / || true
}

deploy_contracts() {
  log "deploying IBC contracts on L1"
  (
    cd "$SPECTRE_DIR"
    cp relayer/config.ethereum.example.json relayer/config.json
    cp relayer/config.json "$WORK/config.json.bak"
    PROVER_PAIRING=warn FOUNDRY_EVM_VERSION=osaka timeout 900 ./scripts/local/deploy_eth_contracts.sh
  ) > "$WORK/deploy-eth.log" 2>&1 || die "deploy_eth_contracts.sh failed, see $WORK/deploy-eth.log"
  ERC20=$(grep -E '^ERC20_ADDRESS:' "$WORK/deploy-eth.log" | awk '{print $2}')
  ICS20=$(grep -E '^ICS20_ADDRESS:' "$WORK/deploy-eth.log" | awk '{print $2}')
  [ -n "$ERC20" ] || die "ERC20 address not found in $WORK/deploy-eth.log"
  [ -n "$ICS20" ] || die "ICS20 address not found in $WORK/deploy-eth.log"
  log "ERC20=$ERC20 ICS20=$ICS20"

  cd "$SPECTRE_DIR"
  jq '(.modules[] | select(.name=="cosmos-to-eth") | .config) |= (.ics26_client_id = "'"$ICS26_CLIENT_ID"'" | .cosmos_wasm_client_id = "" | .spectre_client = "")' \
    relayer/config.json > relayer/config.json.tmp
  mv relayer/config.json.tmp relayer/config.json
}

check_atomone_height() {
  local h
  h=$(curl -s localhost:26657/status | jq -r '.result.sync_info.latest_block_height' 2>/dev/null) || return 1
  [ -n "$h" ] && [ "$h" != "null" ] && [ "$h" -ge 3 ] 2>/dev/null
}

start_atomone() {
  log "starting the AtomOne chain"
  export RELAYER_ENV_FILE=$WORK/atomone-relayer.env
  : > "$RELAYER_ENV_FILE"
  (
    cd "$SPECTRE_DIR"
    nohup ./scripts/local/run_atomone_node.sh > "$WORK/atomone-node.log" 2>&1 &
    echo $! > "$WORK/atomone-script.pid"
  )
  poll 240 3 "atomone height >= 3" check_atomone_height
  grep -q '^COSMOS_CHAIN_ID=' "$RELAYER_ENV_FILE" || die "relayer env file not written"
}

create_clients() {
  log "creating clients, AtomOne first"
  cd "$SPECTRE_DIR/relayer"
  set -a
  . "$RELAYER_ENV_FILE" # AtomOne's COSMOS_* overrides relayer/.env
  set +a
  timeout 600 ./relayer create-clients-cosmos --config config.json \
    > "$WORK/create-clients-cosmos.log" 2>&1 || die "create-clients-cosmos failed, see $WORK/create-clients-cosmos.log"
  timeout 600 ./relayer create-clients-eth --config config.json --trust-level 2/3 \
    > "$WORK/create-clients-eth.log" 2>&1 || die "create-clients-eth failed, see $WORK/create-clients-eth.log"

  jq -e '(.modules[] | select(.name=="cosmos-to-eth") | .config
      | .ics26_client_id == "atomone-1"
        and .cosmos_wasm_client_id == "spectre-0"
        and (.spectre_client | test("^0x[0-9a-fA-F]{40}$")))' \
    config.json > /dev/null || die "relayer config not fully wired after client creation"

  "$ATOMONED" q ibc client state "$ETH_CLIENT_ID" --node "$ATOMONE_NODE" --output json \
    | jq -e '.client_state["@type"] == "/ibc.lightclients.spectre.v1.ClientState"' > /dev/null \
    || die "client $ETH_CLIENT_ID on AtomOne is not a spectre client"
  "$ATOMONED" q ibc client status "$ETH_CLIENT_ID" --node "$ATOMONE_NODE" --output json \
    | jq -e '.status == "Active"' > /dev/null \
    || die "client $ETH_CLIENT_ID on AtomOne is not Active"
}

check_prover_update() {
  local _
  for _ in 1 2 3; do
    if timeout 300 ./relayer update-client --config config.json > "$WORK/update-client.log" 2>&1; then
      grep -q '\[prover\] connected' "$WORK/update-client.log" && return 0
    fi
    sleep 5
  done
  return 1
}

check_relayer_started() {
  grep -q 'Relayer started' "$WORK/relayer.log"
}

start_relayer() {
  log "checking the prover and starting the relayer"
  cd "$SPECTRE_DIR/relayer"
  nc -z -w 5 "${PROVER_ADDR%:*}" "${PROVER_ADDR#*:}" \
    || die "prover $PROVER_ADDR is unreachable"
  check_prover_update || die "relayer could not connect to the prover, see $WORK/update-client.log"

  nohup ./relayer start --config config.json > "$WORK/relayer.log" 2>&1 &
  echo $! > "$WORK/relayer.pid"
  poll 180 2 "'Relayer started' in relayer.log" check_relayer_started
}

check_voucher() {
  local bal
  bal=$("$ATOMONED" q bank balances "$RECV" --node "$ATOMONE_NODE" --output json 2>/dev/null \
    | jq -r --arg d "$EXPECTED_DENOM" '.balances[] | select(.denom==$d) | .amount') || return 1
  [ "$bal" = "$AMOUNT" ]
}

check_ack_on_eth() {
  grep -q 'ackPacket:1' "$WORK/relayer.log"
}

send_eth_to_atomone() {
  log "sending ETH -> AtomOne"
  cd "$SPECTRE_DIR/relayer"
  local eth_key abs_timeout escrowed
  eth_key=$(grep '^ETH_PRIVATE_KEY=' .env | cut -d= -f2 | tr -d '"')
  RECV=$("$ATOMONED" keys show test2 -a --keyring-backend test --home "$ATOMONE_HOME")

  cast send "$ERC20" "approve(address,uint256)" "$ICS20" "$AMOUNT" \
    --private-key "$eth_key" --rpc-url "$ETH_RPC" > /dev/null
  ERC20_BAL_BEFORE=$(cast call "$ERC20" 'balanceOf(address)(uint256)' "$DEPLOYER" \
    --rpc-url "$ETH_RPC" | awk '{print $1}')
  [ -n "$ERC20_BAL_BEFORE" ] || die "could not read the deployer ERC20 balance"
  abs_timeout=$(( $(date +%s) + 3000 ))
  cast send "$ICS20" "sendTransfer((address,uint256,string,string,string,uint64,string))" \
    "($ERC20,$AMOUNT,$RECV,$ICS26_CLIENT_ID,$TRANSFER_PORT,$abs_timeout,\"\")" \
    --private-key "$eth_key" --rpc-url "$ETH_RPC" > /dev/null
  escrowed=$(cast call "$ERC20" 'balanceOf(address)(uint256)' "$DEPLOYER" \
    --rpc-url "$ETH_RPC" | awk '{print $1}')
  [ "$escrowed" != "$ERC20_BAL_BEFORE" ] \
    || die "deployer ERC20 balance unchanged by sendTransfer; expected the transfer to escrow $AMOUNT"

  EXPECTED_DENOM=ibc/$(printf 'transfer/spectre-0/%s' "$ERC20" | shasum -a 256 | awk '{print toupper($1)}')
  log "expected voucher denom: $EXPECTED_DENOM"
  poll 900 10 "voucher $AMOUNT $EXPECTED_DENOM for $RECV" check_voucher
  poll 600 10 "acknowledgement relayed back to ETH" check_ack_on_eth
}

check_wrapped() {
  local w bal
  w=$(cast call "$ICS20" 'ibcERC20Contract(string)(address)' 'transfer/atomone-1/uatone' \
    --rpc-url "$ETH_RPC" 2>/dev/null) || return 1
  [ -n "$w" ] && [ "$w" != "0x0000000000000000000000000000000000000000" ] || return 1
  bal=$(cast call "$w" 'balanceOf(address)(uint256)' "$DEPLOYER" --rpc-url "$ETH_RPC" 2>/dev/null) || return 1
  bal=$(echo "$bal" | awk '{print $1}')
  [ "$bal" -eq "$AMOUNT" ] 2>/dev/null
}

check_ack_on_atomone() {
  local out
  out=$("$ATOMONED" q txs --query "message.action='/ibc.core.channel.v2.MsgAcknowledgement'" \
    --node "$ATOMONE_NODE" --output json 2>/dev/null || echo '{"txs":[]}')
  echo "$out" | jq -e '[.txs[] | select(.code==0)] | length > 0' > /dev/null 2>&1
}

check_tx_committed() { # tx broadcast returns after CheckTx (sync mode); wait for inclusion
  local out code
  local qfile=${QUERY_FILE:-$WORK/send2-query.json}
  out=$("$ATOMONED" q tx "$TXHASH" --node "$ATOMONE_NODE" --output json 2>/dev/null) || return 1
  printf '%s\n' "$out" > "$qfile"
  code=$(jq -r '.code // ""' "$qfile" 2>/dev/null)
  if [ -n "$code" ] && [ "$code" != "0" ]; then
    die "tx $TXHASH failed on deliver: code=$code raw_log=$(jq -r '.raw_log' "$qfile")"
  fi
  jq -e '([.events[]?.type] | index("send_packet")) != null' "$qfile" > /dev/null 2>&1
}

atomone_send_transfer() { # atomone_send_transfer <from-key> <receiver> <coin> <result-json>
  local from=$1 receiver=$2 coin=$3 out=$4 abs_timeout
  abs_timeout=$(( $(date +%s) + 3000 ))
  # The receiver lives on ETH, so the packet payload needs Solidity ABI encoding.
  "$ATOMONED" tx ibc-transfer transfer transfer spectre-0 "$receiver" "$coin" \
    --from "$from" --home "$ATOMONE_HOME" --chain-id "$ATOMONE_CHAIN_ID" --node "$ATOMONE_NODE" \
    --keyring-backend test --gas-prices 0.05uphoton --gas 300000 \
    --absolute-timeouts --packet-timeout-timestamp "$abs_timeout" --generate-only \
    | jq '.body.messages[0].encoding = "application/x-solidity-abi"' \
    | "$ATOMONED" tx sign /dev/stdin --from "$from" --home "$ATOMONE_HOME" \
      --chain-id "$ATOMONE_CHAIN_ID" --keyring-backend test \
    | "$ATOMONED" tx broadcast /dev/stdin --node "$ATOMONE_NODE" -y -o json \
    | tee "$out" > /dev/null
  jq -e '.code == 0 and (.txhash | length > 0)' "$out" > /dev/null \
    || die "AtomOne transfer tx from $from failed: $(jq -c '{code, raw_log}' "$out")"
  TXHASH=$(jq -r .txhash "$out")
  QUERY_FILE="$out-query.json"
  poll 90 2 "tx $TXHASH committed with send_packet event (latest query kept in $QUERY_FILE)" check_tx_committed
}

send_atomone_to_eth() {
  log "sending AtomOne -> ETH"
  TEST3_ADDR=$("$ATOMONED" keys show test3 -a --keyring-backend test --home "$ATOMONE_HOME")
  atomone_send_transfer test3 "$DEPLOYER" "${AMOUNT}uatone" "$WORK/send2.json"

  poll 600 10 "wrapped ${AMOUNT} uatone on ETH for $DEPLOYER" check_wrapped
  poll 600 10 "acknowledgement relayed back to AtomOne" check_ack_on_atomone

  local bal
  bal=$("$ATOMONED" q bank balances "$TEST3_ADDR" \
    --node "$ATOMONE_NODE" --output json | jq -r '.balances[] | select(.denom=="uatone") | .amount')
  [ "$bal" -eq "$TEST3_UATONE_AFTER" ] 2>/dev/null \
    || die "test3 uatone balance is $bal, want $TEST3_UATONE_AFTER (1,000,000 in escrow)"
}

check_voucher_burned() {
  local bal
  bal=$("$ATOMONED" q bank balances "$RECV" --node "$ATOMONE_NODE" --output json 2>/dev/null \
    | jq -r --arg d "$EXPECTED_DENOM" '.balances[] | select(.denom==$d) | .amount') || return 1
  [ -z "$bal" ] || [ "$bal" = "0" ]
}

check_erc20_released() { # string compare: the balance (~1e24) exceeds int64
  local bal
  bal=$(cast call "$ERC20" 'balanceOf(address)(uint256)' "$DEPLOYER" --rpc-url "$ETH_RPC" 2>/dev/null \
    | awk '{print $1}') || return 1
  [ "$bal" = "$ERC20_BAL_BEFORE" ]
}

check_second_ack_on_atomone() {
  local out n
  out=$("$ATOMONED" q txs --query "message.action='/ibc.core.channel.v2.MsgAcknowledgement'" \
    --node "$ATOMONE_NODE" --output json 2>/dev/null || echo '{"txs":[]}')
  n=$(echo "$out" | jq '[.txs[] | select(.code==0)] | length' 2>/dev/null)
  [ "${n:-0}" -ge 2 ] 2>/dev/null
}

redeem_voucher_on_eth() {
  log "returning the ETH-origin voucher to ETH (burn + escrow release)"
  atomone_send_transfer test2 "$DEPLOYER" "${AMOUNT}${EXPECTED_DENOM}" "$WORK/send3.json"

  poll 120 5 "voucher $EXPECTED_DENOM burned for $RECV" check_voucher_burned
  poll 600 10 "escrowed ERC20 released to $DEPLOYER (balance back to the pre-send value)" check_erc20_released
  poll 600 10 "second acknowledgement relayed back to AtomOne" check_second_ack_on_atomone
}

check_wrapped_burned() {
  local bal
  bal=$(cast call "$WRAPPED" 'balanceOf(address)(uint256)' "$DEPLOYER" --rpc-url "$ETH_RPC" 2>/dev/null \
    | awk '{print $1}') || return 1
  [ "$bal" -eq 0 ] 2>/dev/null
}

check_uatone_released() {
  local bal
  bal=$("$ATOMONED" q bank balances "$TEST3_ADDR" --node "$ATOMONE_NODE" --output json 2>/dev/null \
    | jq -r '.balances[] | select(.denom=="uatone") | .amount') || return 1
  [ "$bal" -eq "$TEST3_UATONE_FINAL" ] 2>/dev/null
}

check_second_ack_on_eth() { # eth->cosmos packet sequence 2 is the wrapped-token return
  grep -q 'ackPacket:2' "$WORK/relayer.log"
}

redeem_wrapped_on_atomone() {
  log "returning the wrapped uatone to AtomOne (burn + escrow release)"
  cd "$SPECTRE_DIR/relayer"
  local eth_key abs_timeout
  eth_key=$(grep '^ETH_PRIVATE_KEY=' .env | cut -d= -f2 | tr -d '"')
  WRAPPED=$(cast call "$ICS20" 'ibcERC20Contract(string)(address)' 'transfer/atomone-1/uatone' \
    --rpc-url "$ETH_RPC" | awk '{print $1}')
  [ -n "$WRAPPED" ] || die "wrapped uatone contract not found on ICS20"

  cast send "$WRAPPED" "approve(address,uint256)" "$ICS20" "$AMOUNT" \
    --private-key "$eth_key" --rpc-url "$ETH_RPC" > /dev/null
  abs_timeout=$(( $(date +%s) + 3000 ))
  cast send "$ICS20" "sendTransfer((address,uint256,string,string,string,uint64,string))" \
    "($WRAPPED,$AMOUNT,$TEST3_ADDR,$ICS26_CLIENT_ID,$TRANSFER_PORT,$abs_timeout,\"\")" \
    --private-key "$eth_key" --rpc-url "$ETH_RPC" > /dev/null

  poll 120 5 "wrapped token burned for $DEPLOYER" check_wrapped_burned
  poll 900 10 "escrowed uatone released to test3 (balance back to $TEST3_UATONE_FINAL)" check_uatone_released
  poll 600 10 "second acknowledgement relayed back to ETH" check_second_ack_on_eth
}

final_checks() {
  log "final checks"
  local errs
  errs=$(grep -cE 'DROP|"level":"error"' "$WORK/relayer.log" || true)
  [ "$errs" -eq 0 ] || die "relayer logged $errs error/drop lines, see $WORK/relayer.log"
}

main() {
  check_prerequisites
  clone_spectre
  build_binaries
  start_eth_l1
  deploy_contracts
  start_atomone
  create_clients
  start_relayer
  send_eth_to_atomone
  send_atomone_to_eth
  redeem_voucher_on_eth
  redeem_wrapped_on_atomone
  final_checks
  log "spectre e2e PASSED: one transfer and one escrow redemption completed in each direction"
}

main "$@"
